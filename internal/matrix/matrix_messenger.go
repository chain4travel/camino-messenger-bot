package matrix

import (
	"context"
	"crypto/ecdsa"
	"crypto/sha256"
	"errors"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/chain4travel/camino-messenger-bot/config"
	"github.com/chain4travel/camino-messenger-bot/internal/messaging"
	"github.com/chain4travel/camino-messenger-bot/internal/messaging/types"
	"github.com/chain4travel/camino-messenger-bot/internal/metadata"
	"github.com/chain4travel/camino-messenger-bot/pkg/matrix"
	"github.com/ethereum/go-ethereum/crypto"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"maunium.net/go/mautrix"
	"maunium.net/go/mautrix/crypto/cryptohelper"
	"maunium.net/go/mautrix/event"
	"maunium.net/go/mautrix/id"

	_ "github.com/mattn/go-sqlite3" //nolint:revive
)

var _ messaging.Messenger = (*messenger)(nil)

type client struct {
	*mautrix.Client
	ctx          context.Context
	cancelSync   context.CancelFunc
	syncStopWait sync.WaitGroup
	cryptoHelper *cryptohelper.CryptoHelper
}
type messenger struct {
	msgChannel chan types.Message

	cfg    *config.MatrixConfig
	logger *zap.SugaredLogger
	tracer trace.Tracer

	client       client
	roomHandler  RoomHandler
	msgAssembler MessageAssembler
}

func NewMessenger(cfg *config.MatrixConfig, logger *zap.SugaredLogger) messaging.Messenger {
	c, err := mautrix.NewClient(cfg.Host, "", "")
	if err != nil {
		panic(err)
	}
	return &messenger{
		msgChannel:   make(chan types.Message),
		cfg:          cfg,
		logger:       logger,
		tracer:       otel.GetTracerProvider().Tracer(""),
		client:       client{Client: c},
		roomHandler:  NewRoomHandler(NewClient(c), logger),
		msgAssembler: NewMessageAssembler(),
	}
}

func (m *messenger) Checkpoint() string {
	return "messenger-gateway"
}

func (m *messenger) StartReceiver() (id.UserID, error) {
	syncer := m.client.Syncer.(*mautrix.DefaultSyncer)

	syncer.OnEventType(matrix.EventTypeC4TMessage, func(ctx context.Context, evt *event.Event) {
		msg := evt.Content.Parsed.(*matrix.CaminoMatrixMessage)
		traceID, err := trace.TraceIDFromHex(msg.Metadata.RequestID)
		if err != nil {
			m.logger.Warnf("failed to parse traceID from hex [requestID:%s]: %v", msg.Metadata.RequestID, err)
		}
		ctx = trace.ContextWithRemoteSpanContext(ctx, trace.NewSpanContext(trace.SpanContextConfig{TraceID: traceID}))
		_, span := m.tracer.Start(ctx, "messenger.OnC4TMessageReceive", trace.WithSpanKind(trace.SpanKindConsumer), trace.WithAttributes(attribute.String("type", evt.Type.Type)))
		defer span.End()
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
		m.msgChannel <- types.Message{
			Metadata: completeMsg.Metadata,
			Content:  completeMsg.Content,
			Type:     types.MessageType(msg.MsgType),
			Sender:   evt.Sender,
		}
	})
	syncer.OnEventType(event.StateMember, func(ctx context.Context, evt *event.Event) {
		if evt.GetStateKey() == m.client.UserID.String() && evt.Content.AsMember().Membership == event.MembershipInvite {
			_, err := m.client.JoinRoomByID(ctx, evt.RoomID)
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

	caminoPrivateKey, err := crypto.HexToECDSA(m.cfg.Key)
	if err != nil {
		return "", err
	}

	signature, message, err := signPublicKey(caminoPrivateKey)
	if err != nil {
		return "", err
	}

	cryptoHelper.LoginAs = &mautrix.ReqLogin{
		Type:      mautrix.AuthTypeCamino,
		PublicKey: message[2:],   // removing 0x prefix
		Signature: signature[2:], // removing 0x prefix
	}

	err = cryptoHelper.Init(context.TODO())
	if err != nil {
		return "", err
	}
	// Set the wrappedClient crypto helper in order to automatically encrypt outgoing messages
	m.client.Crypto = cryptoHelper
	m.client.cryptoHelper = cryptoHelper // nikos: we need the struct cause stop method is not available on the interface level

	m.logger.Infof("Successfully logged in as: %s", m.client.UserID)
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

	return m.client.UserID, nil
}

func (m *messenger) StopReceiver() error {
	m.logger.Info("Stopping matrix syncer...")
	if m.client.cancelSync != nil {
		m.client.cancelSync()
	}
	m.client.syncStopWait.Wait()
	return m.client.cryptoHelper.Close()
}

func (m *messenger) SendAsync(ctx context.Context, msg types.Message, content [][]byte, sendTo id.UserID) error {
	m.logger.Info("Sending async message", zap.String("msg", msg.Metadata.RequestID))
	ctx, span := m.tracer.Start(ctx, "messenger.SendAsync", trace.WithSpanKind(trace.SpanKindProducer), trace.WithAttributes(attribute.String("type", string(msg.Type))))
	defer span.End()

	ctx, roomSpan := m.tracer.Start(ctx, "roomHandler.GetOrCreateRoom", trace.WithAttributes(attribute.String("type", string(msg.Type))))
	roomID, err := m.roomHandler.GetOrCreateRoomForRecipient(ctx, sendTo)
	if err != nil {
		return err
	}
	roomSpan.End()

	return m.sendMessageEvents(ctx, roomID, matrix.EventTypeC4TMessage, createMatrixMessages(&msg, content))
}

func (m *messenger) sendMessageEvents(ctx context.Context, roomID id.RoomID, eventType event.Type, messages []matrix.CaminoMatrixMessage) error {
	// TODO add retry logic?
	for _, msg := range messages {
		_, err := m.client.SendMessageEvent(ctx, roomID, eventType, msg)
		if err != nil {
			return err
		}
	}
	return nil
}

func (m *messenger) Inbound() chan types.Message {
	return m.msgChannel
}

func signPublicKey(key *ecdsa.PrivateKey) (signature string, message string, err error) {
	pubKeyBytes := crypto.FromECDSAPub(&key.PublicKey)
	signatureBytes, err := sign(pubKeyBytes, key)
	if err != nil {
		return "", "", err
	}

	signature, err = hexWithChecksum(signatureBytes)
	if err != nil {
		return "", "", err
	}
	message, err = hexWithChecksum(pubKeyBytes)
	if err != nil {
		return "", "", err
	}
	return signature, message, nil
}

func sign(msg []byte, key *ecdsa.PrivateKey) ([]byte, error) {
	// TODO: Why don't we use crypto.keccak256 both on ASB and here?
	hash256 := sha256.Sum256(msg)

	signature, err := crypto.Sign(hash256[:], key)
	if err != nil {
		return nil, err
	}

	return signature, nil
}

func hexWithChecksum(bytes []byte) (string, error) {
	const checksumLen = 4
	bytesLen := len(bytes)
	if bytesLen > math.MaxInt32-checksumLen {
		return "", errors.New("encoding overflow")
	}
	checked := make([]byte, bytesLen+checksumLen)
	copy(checked, bytes)
	hash := sha256.Sum256(bytes)
	copy(checked[len(bytes):], hash[len(hash)-checksumLen:])
	bytes = checked
	return fmt.Sprintf("0x%x", bytes), nil
}

func createMatrixMessages(msg *types.Message, content [][]byte) []matrix.CaminoMatrixMessage {
	messages := make([]matrix.CaminoMatrixMessage, 0, len(content))

	// add first chunk to messages slice
	caminoMatrixMsg := matrix.CaminoMatrixMessage{
		MessageEventContent: event.MessageEventContent{MsgType: event.MessageType(msg.Type)},
		Metadata:            msg.Metadata,
	}
	caminoMatrixMsg.Metadata.NumberOfChunks = uint64(len(content))
	caminoMatrixMsg.Metadata.ChunkIndex = 0
	caminoMatrixMsg.CompressedContent = content[0]
	messages = append(messages, caminoMatrixMsg)

	// if multiple chunks were produced upon compression, add them to messages slice
	for i, chunk := range content[1:] {
		messages = append(messages, matrix.CaminoMatrixMessage{
			MessageEventContent: event.MessageEventContent{MsgType: event.MessageType(msg.Type)},
			Metadata:            metadata.Metadata{RequestID: msg.Metadata.RequestID, NumberOfChunks: uint64(len(content)), ChunkIndex: uint64(i + 1)},
			CompressedContent:   chunk,
		})
	}

	return messages
}
