package matrix

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"sync"

	"github.com/ava-labs/avalanchego/utils/crypto/secp256k1"
	"github.com/ava-labs/avalanchego/utils/formatting"
	"github.com/chain4travel/camino-messenger-bot/config"
	"github.com/chain4travel/camino-messenger-bot/internal/compression"
	"github.com/chain4travel/camino-messenger-bot/internal/messaging"
	"github.com/golang/protobuf/proto"
	"go.uber.org/zap"
	"maunium.net/go/mautrix"
	"maunium.net/go/mautrix/crypto/cryptohelper"
	"maunium.net/go/mautrix/event"
	"maunium.net/go/mautrix/id"

	_ "github.com/mattn/go-sqlite3"
)

var _ messaging.Messenger = (*messenger)(nil)

var C4TMessage = event.Type{Type: "m.room.c4t-msg", Class: event.MessageEventType}

type messageEventPayload struct {
	roomID      id.RoomID
	eventType   event.Type
	contentJSON interface{}
}

type client struct {
	*mautrix.Client
	ctx          context.Context
	cancelSync   context.CancelFunc
	syncStopWait sync.WaitGroup
	cryptoHelper *cryptohelper.CryptoHelper
}
type messenger struct {
	msgChannel  chan messaging.Message
	cfg         *config.MatrixConfig
	logger      *zap.SugaredLogger
	client      client
	roomHandler RoomHandler
}

func NewMessenger(cfg *config.MatrixConfig, logger *zap.SugaredLogger) *messenger {
	c, err := mautrix.NewClient(cfg.Host, "", "")
	if err != nil {
		panic(err)
	}
	return &messenger{
		msgChannel:  make(chan messaging.Message),
		cfg:         cfg,
		logger:      logger,
		client:      client{Client: c},
		roomHandler: NewRoomHandler(c, logger),
	}
}
func (m *messenger) Checkpoint() string {
	return "messenger-gateway"
}

func (m *messenger) StartReceiver() (string, error) {
	syncer := m.client.Syncer.(*mautrix.DefaultSyncer)
	event.TypeMap[C4TMessage] = reflect.TypeOf(CaminoMatrixMessage{}) // custom message event types have to be registered properly

	syncer.OnEventType(C4TMessage, func(source mautrix.EventSource, evt *event.Event) {
		msg := evt.Content.Parsed.(*CaminoMatrixMessage)
		msg.Metadata.Sender = evt.Sender.String() // overwrite sender with actual sender
		msg.Metadata.Stamp(fmt.Sprintf("%s-%s", m.Checkpoint(), "received"))

		decompressedContent, err := compression.Decompress(msg.CompressedContent)

		switch messaging.MessageType(msg.MsgType).Category() {
		case messaging.Request:
			requestContent := &messaging.RequestContent{}
			err = proto.Unmarshal(decompressedContent, requestContent)
			msg.Content = messaging.MessageContent{
				RequestContent: *requestContent,
			}
		case messaging.Response:
			responseContent := &messaging.ResponseContent{}
			err = proto.Unmarshal(decompressedContent, responseContent)
			msg.Content = messaging.MessageContent{
				ResponseContent: *responseContent,
			}
		default:
			m.logger.Errorf("unknown message category for type: %v", messaging.MessageType(msg.MsgType))
			return
		}

		if err != nil {
			m.logger.Error("failed to decompress msg", zap.Error(err))
			return
		}

		m.msgChannel <- messaging.Message{
			Metadata: msg.Metadata,
			Content:  msg.Content,
			Type:     messaging.MessageType(msg.MsgType),
		}
	})
	syncer.OnEventType(event.StateMember, func(source mautrix.EventSource, evt *event.Event) {
		if evt.GetStateKey() == m.client.UserID.String() && evt.Content.AsMember().Membership == event.MembershipInvite {
			_, err := m.client.JoinRoomByID(evt.RoomID)
			if err == nil {
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

	messageEvents, err := compressAndSplitCaminoMatrixMsg(roomID, msg)
	if err != nil {
		return err
	}

	return m.sendMessageEvents(messageEvents)
}

func (m *messenger) sendMessageEvents(messageEvents []messageEventPayload) error {
	//TODO add retry logic?
	for _, me := range messageEvents {
		_, err := m.client.SendMessageEvent(me.roomID, me.eventType, me.contentJSON)
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
