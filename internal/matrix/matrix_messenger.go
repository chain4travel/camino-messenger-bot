package matrix

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"sync"
	"time"

	"github.com/chain4travel/camino-messenger-bot/internal/compression"

	"github.com/ava-labs/avalanchego/utils/crypto/secp256k1"
	"github.com/ava-labs/avalanchego/utils/formatting"
	"github.com/chain4travel/camino-messenger-bot/config"
	"github.com/chain4travel/camino-messenger-bot/internal/messaging"
	"go.uber.org/zap"
	"maunium.net/go/mautrix"
	"maunium.net/go/mautrix/crypto/cryptohelper"
	"maunium.net/go/mautrix/event"
	"maunium.net/go/mautrix/id"

	_ "github.com/mattn/go-sqlite3" //nolint:revive
)

var _ messaging.Messenger = (*messenger)(nil)

var C4TMessage = event.Type{Type: "m.room.c4t-msg", Class: event.MessageEventType}

type client struct {
	*mautrix.Client
	ctx          context.Context
	cancelSync   context.CancelFunc
	syncStopWait sync.WaitGroup
	cryptoHelper *cryptohelper.CryptoHelper
}
type messenger struct {
	msgChannel   chan messaging.Message
	cfg          *config.MatrixConfig
	logger       *zap.SugaredLogger
	client       client
	roomHandler  RoomHandler
	msgAssembler MessageAssembler
	compressor   compression.Compressor[messaging.Message, []CaminoMatrixMessage]
}

func NewMessenger(cfg *config.MatrixConfig, logger *zap.SugaredLogger) messaging.Messenger {
	c, err := mautrix.NewClient(cfg.Host, "", "")
	if err != nil {
		panic(err)
	}
	return &messenger{
		msgChannel:   make(chan messaging.Message),
		cfg:          cfg,
		logger:       logger,
		client:       client{Client: c},
		roomHandler:  NewRoomHandler(c, logger),
		msgAssembler: NewMessageAssembler(),
		compressor:   &ChunkingCompressor{maxChunkSize: compression.MaxChunkSize},
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
		t := time.Now()
		completeMsg, completed, err := m.msgAssembler.AssembleMessage(msg)
		if err != nil {
			m.logger.Errorf("failed to assemble message: %v", err)
			return
		}
		if !completed {
			return // partial messages are not passed down to the msgChannel
		}
		completeMsg.Metadata.StampOn(fmt.Sprintf("matrix-sent-%s", completeMsg.MsgType), evt.Timestamp)
		completeMsg.Metadata.StampOn(fmt.Sprintf("%s-%s-%s", m.Checkpoint(), "received", completeMsg.MsgType), t.UnixMilli())
		m.msgChannel <- messaging.Message{
			Metadata: completeMsg.Metadata,
			Content:  completeMsg.Content,
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

	cryptoHelper, err := cryptohelper.NewCryptoHelper(m.client.Client, []byte("meow"), m.cfg.Store) // TODO refactor
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

	messages, err := m.compressor.Compress(msg)
	if err != nil {
		return err
	}

	return m.sendMessageEvents(roomID, C4TMessage, messages)
}

func (m *messenger) sendMessageEvents(roomID id.RoomID, eventType event.Type, messages []CaminoMatrixMessage) error {
	// TODO add retry logic?
	for _, msg := range messages {
		_, err := m.client.SendMessageEvent(roomID, eventType, msg)
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
