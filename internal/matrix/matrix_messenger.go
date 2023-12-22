package matrix

import (
	"context"
	"crypto/rsa"
	"errors"
	"fmt"
	aes_util "github.com/chain4travel/camino-messenger-bot/utils/aes"
	rsa_util "github.com/chain4travel/camino-messenger-bot/utils/rsa"
	"reflect"
	"sync"
	"time"

	"github.com/ava-labs/avalanchego/utils/crypto/secp256k1"
	"github.com/ava-labs/avalanchego/utils/formatting"
	"github.com/chain4travel/camino-messenger-bot/config"
	"github.com/chain4travel/camino-messenger-bot/internal/messaging"
	"go.uber.org/zap"
	"maunium.net/go/mautrix"
	"maunium.net/go/mautrix/crypto/cryptohelper"
	"maunium.net/go/mautrix/event"
	"maunium.net/go/mautrix/id"

	_ "github.com/mattn/go-sqlite3"
)

var _ messaging.Messenger = (*messenger)(nil)

var C4TMessageRequest = event.Type{Type: "m.room.c4t-msg-request", Class: event.MessageEventType}
var C4TMessageResponse = event.Type{Type: "m.room.c4t-msg-response", Class: event.MessageEventType}

type client struct {
	*mautrix.Client
	ctx          context.Context
	cancelSync   context.CancelFunc
	syncStopWait sync.WaitGroup
	cryptoHelper *cryptohelper.CryptoHelper
}
type messenger struct {
	msgChannel              chan messaging.Message
	cfg                     *config.MatrixConfig
	logger                  *zap.SugaredLogger
	client                  client
	roomHandler             RoomHandler
	msgAssembler            MessageAssembler
	mu                      sync.Mutex
	privateRSAKey           *rsa.PrivateKey
	encryptionKeyRepository EncryptionKeyRepository
}

func (m *messenger) Encrypt(msg *CaminoMatrixMessage) error {
	pubKey, err := m.encryptionKeyRepository.getPublicKeyForRecipient(msg.Metadata.Recipient)
	if err != nil {
		return err
	}

	symmetricKey := m.encryptionKeyRepository.getSymmetricKeyForRecipient(msg.Metadata.Recipient)
	// encrypt symmetric key with recipient's public key
	msg.EncryptedSymmetricKey, err = rsa_util.EncryptWithPublicKey(symmetricKey, pubKey)
	if err != nil {
		return err
	}
	// encrypt message with symmetric key
	encryptedCompressedContent, err := aes_util.Encrypt(msg.CompressedContent, symmetricKey)
	if err != nil {
		return err
	}
	msg.CompressedContent = nil
	msg.EncryptedCompressedContent = encryptedCompressedContent
	return nil
}

func (m *messenger) Decrypt(msg *CaminoMatrixMessage) error {
	// decrypt symmetric key with private key
	symmetricKey, err := rsa_util.DecryptWithPrivateKey(msg.EncryptedSymmetricKey, m.privateRSAKey)
	if err != nil {
		return err
	}

	m.encryptionKeyRepository.cacheSymmetricKey(msg.Metadata.Sender, symmetricKey)
	// decrypt message with symmetric key
	decryptedCompressedContent, err := aes_util.Decrypt(msg.EncryptedCompressedContent, symmetricKey)
	if err != nil {
		return err
	}
	msg.CompressedContent = decryptedCompressedContent
	return nil
}

func NewMessenger(cfg *config.MatrixConfig, logger *zap.SugaredLogger, privateRSAKey *rsa.PrivateKey) *messenger {
	c, err := mautrix.NewClient(cfg.Host, "", "")
	if err != nil {
		panic(err)
	}
	return &messenger{
		msgChannel:              make(chan messaging.Message),
		cfg:                     cfg,
		logger:                  logger,
		client:                  client{Client: c},
		roomHandler:             NewRoomHandler(c, logger),
		msgAssembler:            NewMessageAssembler(logger),
		privateRSAKey:           privateRSAKey,
		encryptionKeyRepository: *NewEncryptionKeyRepository(),
	}
}
func (m *messenger) Checkpoint() string {
	return "messenger-gateway"
}

func (m *messenger) StartReceiver(botMode uint) (string, error) {
	syncer := m.client.Syncer.(*mautrix.DefaultSyncer)
	event.TypeMap[C4TMessageRequest] = reflect.TypeOf(CaminoMatrixMessage{})  // custom message event types have to be registered properly
	event.TypeMap[C4TMessageResponse] = reflect.TypeOf(CaminoMatrixMessage{}) // custom message event types have to be registered properly

	processCamMsg := func(source mautrix.EventSource, evt *event.Event) {
		msg := evt.Content.Parsed.(*CaminoMatrixMessage)
		go func() {
			t := time.Now()
			if msg.EncryptedSymmetricKey == nil { // if no symmetric key is provided, it should have been exchanged and cached already
				key := m.encryptionKeyRepository.fetchSymmetricKeyFromCache(msg.Metadata.Sender)
				if key == nil {
					m.logger.Errorf("no symmetric key found for sender: %s [request-id:%s]", msg.Metadata.Sender, msg.Metadata.RequestID)
					return
				} else {
					msg.EncryptedSymmetricKey = key
				}
			}
			err := m.Decrypt(msg)
			if err != nil {
				m.logger.Errorf("failed to decrypt message: %v", err)
				return
			}
			fmt.Printf("%d|decrypted-message|%s|%d\n", t.UnixMilli(), evt.ID.String(), time.Since(t).Milliseconds())
			t = time.Now()
			completeMsg, err, completed := m.msgAssembler.AssembleMessage(*msg)
			if err != nil {
				m.logger.Errorf("failed to assemble message: %v", err)
				return
			}
			if !completed {
				return // partial messages are not passed down to the msgChannel
			}
			completeMsg.Metadata.StampOn(fmt.Sprintf("matrix-sent-%s", completeMsg.MsgType), evt.Timestamp)
			completeMsg.Metadata.StampOn(fmt.Sprintf("%s-%s-%s", m.Checkpoint(), "received", completeMsg.MsgType), t.UnixMilli())

			t = time.Now()
			m.mu.Lock()
			m.msgChannel <- messaging.Message{
				Metadata: completeMsg.Metadata,
				Content:  completeMsg.Content,
				Type:     messaging.MessageType(msg.MsgType),
			}
			m.mu.Unlock()
		}()
	}
	switch botMode {
	case 0:
		syncer.OnEventType(C4TMessageResponse, processCamMsg)
		syncer.OnEventType(C4TMessageRequest, processCamMsg)
	case 1:
		syncer.OnEventType(C4TMessageRequest, processCamMsg)
	case 2:
		syncer.OnEventType(C4TMessageResponse, processCamMsg)
	default:
		return "", fmt.Errorf("invalid bot mode: %d", botMode)
	}

	syncer.OnEventType(event.StateMember, func(source mautrix.EventSource, evt *event.Event) {
		if evt.GetStateKey() == m.client.UserID.String() && evt.Content.AsMember().Membership == event.MembershipInvite && !m.roomHandler.HasAlreadyJoined(id.UserID(evt.Sender.String()), evt.RoomID) {
			_, err := m.client.JoinRoomByID(evt.RoomID)
			if err == nil {
				m.roomHandler.CacheRoom(id.UserID(evt.Sender.String()), evt.RoomID) // add room to pubKeyCache
				m.logger.Info("Joined room after invite",
					zap.String("room_id", evt.RoomID.String()),
					zap.String("inviter", evt.Sender.String()))
			} else {
				m.logger.Error("Failed to join room after invite",
					zap.String("room_id", evt.RoomID.String()),
					zap.String("inviter", evt.Sender.String()))
			}
		}
	})

	cryptoHelper, err := cryptohelper.NewCryptoHelper(m.client.Client, []byte("meow"), m.cfg.Store) //TODO refactor
	if err != nil {
		return "", err
	}

	camioPrivateKey, err := readPrivateKey(m.cfg.Key)
	if err != nil {
		return "", err
	}

	signature, message, err := signPublicKey(camioPrivateKey)
	if err != nil {
		return "", err
	}

	cryptoHelper.LoginAs = &mautrix.ReqLogin{
		Type:      mautrix.AuthTypeCamino,
		PublicKey: message[2:],   // removing 0x prefix
		Signature: signature[2:], // removing 0x prefix
	}

	err = cryptoHelper.Init()
	if err != nil {
		return "", err
	}
	// Set the client crypto helper in order to automatically encrypt outgoing messages
	m.client.Crypto = cryptoHelper
	m.client.cryptoHelper = cryptoHelper // nikos: we need the struct cause stop method is not available on the interface level

	m.logger.Infof("Successfully logged in as: %s", m.client.UserID.String())
	syncCtx, cancelSync := context.WithCancel(context.Background())
	m.client.ctx = syncCtx
	m.client.cancelSync = cancelSync
	m.client.syncStopWait.Add(1)

	m.roomHandler.Init()
	go func() {
		err = m.client.SyncWithContext(syncCtx)
		defer m.client.syncStopWait.Done()
		if err != nil && !errors.Is(err, context.Canceled) {
			panic(err)
		}
	}()

	return m.client.UserID.String(), nil
}
func (m *messenger) StopReceiver() error {
	m.logger.Info("Stopping matrix syncer...")
	if m.client.cancelSync != nil {
		m.client.cancelSync()
	}
	m.client.syncStopWait.Wait()
	return m.client.cryptoHelper.Close()
}

func (m *messenger) SendAsync(_ context.Context, msg messaging.Message) error {
	m.logger.Info("Sending async message", zap.String("msg", msg.Metadata.RequestID))

	roomID, err := m.roomHandler.GetOrCreateRoomForRecipient(id.UserID(msg.Metadata.Recipient))
	if err != nil {
		return err
	}

	messages, err := compressAndSplitCaminoMatrixMsg(msg)
	if err != nil {
		return err
	}

	switch msg.Type.Category() {
	case messaging.Request:
		return m.sendMessageEvents(roomID, C4TMessageRequest, messages)
	case messaging.Response:
		return m.sendMessageEvents(roomID, C4TMessageResponse, messages)
	default:
		return fmt.Errorf("no message category defined for type: %s", msg.Type)
	}
}

func (m *messenger) sendMessageEvents(roomID id.RoomID, eventType event.Type, messages []CaminoMatrixMessage) error {
	//TODO add retry logic?
	for _, msg := range messages {
		t := time.Now()
		err := m.Encrypt(&msg)
		if err != nil {
			return err
		}
		fmt.Printf("%d|encrypted-message|%d\n", t.UnixMilli(), time.Since(t).Milliseconds())
		_, err = m.client.SendMessageEvent(roomID, eventType, msg)
		if err != nil {
			return err
		}
	}
	return nil
}

func (m *messenger) Inbound() chan messaging.Message {
	return m.msgChannel
}

func readPrivateKey(keyStr string) (*secp256k1.PrivateKey, error) {
	key := new(secp256k1.PrivateKey)
	if err := key.UnmarshalText([]byte("\"" + keyStr + "\"")); err != nil {
		return nil, err
	}
	return key, nil
}

func signPublicKey(key *secp256k1.PrivateKey) (signature string, message string, err error) {
	signatureBytes, err := key.Sign(key.PublicKey().Bytes())
	if err != nil {
		return "", "", err
	}
	signature, err = formatting.Encode(formatting.Hex, signatureBytes)
	if err != nil {
		return "", "", err
	}
	message, err = formatting.Encode(formatting.Hex, key.PublicKey().Bytes())
	if err != nil {
		return "", "", err
	}
	return signature, message, nil
}
