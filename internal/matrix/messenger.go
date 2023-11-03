package matrix

import (
	"context"
	"errors"
	"sync"

	"camino-messenger-bot/config"
	"camino-messenger-bot/internal/messaging"
	"go.uber.org/zap"
	"maunium.net/go/mautrix"
	"maunium.net/go/mautrix/crypto/cryptohelper"
	"maunium.net/go/mautrix/event"

	_ "github.com/mattn/go-sqlite3"
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
	msgChannel chan messaging.Message
	cfg        *config.MatrixConfig
	logger     *zap.SugaredLogger
	client     client
}

func NewMessenger(cfg *config.MatrixConfig, logger *zap.SugaredLogger) *messenger {
	c, err := mautrix.NewClient(cfg.MatrixHost, "", "")
	if err != nil {
		panic(err)
	}
	return &messenger{
		msgChannel: make(chan messaging.Message),
		cfg:        cfg,
		logger:     logger,
		client:     client{Client: c},
	}
}

func (m *messenger) StartReceiver() error {
	syncer := m.client.Syncer.(*mautrix.DefaultSyncer)
	// TODO: custom message event types have to be registered properly . see also event.TypeMap[event.MessageEventType] = event.MessageEventContent{}
	syncer.OnEventType(event.EventMessage, func(source mautrix.EventSource, evt *event.Event) {
		m.logger.Info("Received msg",
			zap.String("sender", evt.Sender.String()),
			zap.String("type", evt.Type.String()),
			zap.String("id", evt.ID.String()),
			zap.String("body", evt.Content.AsMessage().Body))
		m.msgChannel <- messaging.Message{
			Metadata: messaging.Metadata{
				Sender: evt.Sender.String(),
			},
			RequestID: evt.ID.String(),
			Body:      evt.Content.AsMessage().Body,
			Type:      messaging.MessageType(evt.Content.AsMessage().MsgType),
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

	cryptoHelper, err := cryptohelper.NewCryptoHelper(m.client.Client, []byte("meow"), "mautrix.db") //TODO refactor
	if err != nil {
		return err
	}

	cryptoHelper.LoginAs = &mautrix.ReqLogin{
		Type:       mautrix.AuthTypePassword,
		Identifier: mautrix.UserIdentifier{Type: mautrix.IdentifierTypeThirdParty, Medium: "camino", Address: m.cfg.Username},
		Password:   m.cfg.Password,
	}
	err = cryptoHelper.Init()
	if err != nil {
		return err
	}
	// Set the client crypto helper in order to automatically encrypt outgoing messages
	m.client.Crypto = cryptoHelper
	m.client.cryptoHelper = cryptoHelper // nikos: we need the struct cause stop method is not available on the interface level

	m.logger.Info("Now running")
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

	return nil
}
func (m *messenger) StopReceiver() error {
	m.logger.Info("Stopping matrix syncer...")
	m.client.cancelSync()
	m.client.syncStopWait.Wait()
	return m.client.cryptoHelper.Close()
}

func (m *messenger) Send(msg messaging.Message, rc chan<- messaging.APIMessageResponse) error {
	//TODO implement me
	m.logger.Info("Sending message", zap.Any("msg", msg))
	response := messaging.APIMessageResponse{}
	rc <- response
	return nil
}

func (m *messenger) SendAsync(msg messaging.Message) error {
	m.logger.Info("Sending async message", zap.Any("msg", msg))
	return nil
}

func (m *messenger) Inbound() chan messaging.Message {
	return m.msgChannel
}
