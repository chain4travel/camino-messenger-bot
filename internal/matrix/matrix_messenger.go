package matrix

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/chain4travel/camino-messenger-bot/config"
	"github.com/chain4travel/camino-messenger-bot/internal/compression"
	"github.com/chain4travel/camino-messenger-bot/internal/messaging"
	"github.com/chain4travel/camino-messenger-bot/pkg/matrix"
	"github.com/decred/dcrd/dcrec/secp256k1/v4"
	"github.com/decred/dcrd/dcrec/secp256k1/v4/ecdsa"
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
	msgChannel chan messaging.Message

	cfg    *config.MatrixConfig
	logger *zap.SugaredLogger
	tracer trace.Tracer

	client       client
	roomHandler  RoomHandler
	msgAssembler MessageAssembler
	compressor   compression.Compressor[messaging.Message, []matrix.CaminoMatrixMessage]
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
		tracer:       otel.GetTracerProvider().Tracer(""),
		client:       client{Client: c},
		roomHandler:  NewRoomHandler(NewClient(c), logger),
		msgAssembler: NewMessageAssembler(),
		compressor:   &ChunkingCompressor{maxChunkSize: compression.MaxChunkSize},
	}
}

func (m *messenger) Checkpoint() string {
	return "messenger-gateway"
}

func (m *messenger) StartReceiver() (string, error) {
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
		m.msgChannel <- messaging.Message{
			Metadata: completeMsg.Metadata,
			Content:  completeMsg.Content,
			Type:     messaging.MessageType(msg.MsgType),
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

	err = cryptoHelper.Init(context.TODO())
	if err != nil {
		return "", err
	}
	// Set the wrappedClient crypto helper in order to automatically encrypt outgoing messages
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

func (m *messenger) SendAsync(ctx context.Context, msg messaging.Message) error {
	m.logger.Info("Sending async message", zap.String("msg", msg.Metadata.RequestID))
	ctx, span := m.tracer.Start(ctx, "messenger.SendAsync", trace.WithSpanKind(trace.SpanKindProducer), trace.WithAttributes(attribute.String("type", string(msg.Type))))
	defer span.End()

	ctx, roomSpan := m.tracer.Start(ctx, "roomHandler.GetOrCreateRoom", trace.WithAttributes(attribute.String("type", string(msg.Type))))
	roomID, err := m.roomHandler.GetOrCreateRoomForRecipient(ctx, id.UserID(msg.Metadata.Recipient))
	if err != nil {
		return err
	}
	roomSpan.End()

	ctx, compressSpan := m.tracer.Start(ctx, "messenger.Compress", trace.WithAttributes(attribute.String("type", string(msg.Type))))
	messages, err := m.compressor.Compress(msg)
	if err != nil {
		return err
	}
	compressSpan.End()

	return m.sendMessageEvents(ctx, roomID, matrix.EventTypeC4TMessage, messages)
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

func (m *messenger) Inbound() chan messaging.Message {
	return m.msgChannel
}

func readPrivateKey(keyStr string) (*secp256k1.PrivateKey, error) {
	privateKeyBytes, err := hex.DecodeString(keyStr)
	if err != nil {
		return nil, err
	}

	return secp256k1.PrivKeyFromBytes(privateKeyBytes), nil
}

func signPublicKey(key *secp256k1.PrivateKey) (signature string, message string, err error) {
	pubKeyBytes := key.PubKey().SerializeCompressed()
	signatureBytes := sign(pubKeyBytes, key)
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

func sign(msg []byte, key *secp256k1.PrivateKey) []byte {
	hash := sha256.Sum256(msg)
	signature := ecdsa.SignCompact(key, hash[:], false)

	// fix signature format
	recoveryCode := signature[0]
	copy(signature, signature[1:])
	signature[len(signature)-1] = recoveryCode - 27
	return signature
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
