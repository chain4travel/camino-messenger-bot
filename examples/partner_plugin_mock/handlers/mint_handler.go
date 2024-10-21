package handlers_mock

import (
	"context"
	"crypto/ecdsa"
	"log"

	bookv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/book/v1"
	bookv2 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/book/v2"
	"go.uber.org/zap"
	"google.golang.org/protobuf/reflect/protoreflect"

	"github.com/chain4travel/camino-messenger-bot/internal/messaging/types"
	"github.com/chain4travel/camino-messenger-bot/internal/rpc/generated"
	"github.com/chain4travel/camino-messenger-bot/pkg/booking"
	cmaccounts "github.com/chain4travel/camino-messenger-bot/pkg/cm_accounts"
	"github.com/chain4travel/camino-messenger-bot/pkg/erc20"
	"github.com/chain4travel/camino-messenger-bot/pkg/events"
	"github.com/chain4travel/camino-messenger-contracts/go/contracts/bookingtoken"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
)

func NewResponseHandler(
	botKey *ecdsa.PrivateKey,
	ethClient *ethclient.Client,
	logger *zap.SugaredLogger,
	cmAccountAddress common.Address,
	bookingTokenAddress common.Address,
	serviceRegistry ServiceRegistry,
	cmAccounts cmaccounts.Service,
	tokenCacheSize int,
) (ResponseHandler, error) {
	erc20, err := erc20.NewERC20Service(ethClient, tokenCacheSize)
	if err != nil {
		return nil, err
	}

	bookingService, err := booking.NewService(cmAccountAddress, botKey, ethClient, logger, erc20, cmAccounts)
	if err != nil {
		log.Printf("%v", err)
		return nil, err
	}

	bookingToken, err := bookingtoken.NewBookingtoken(bookingTokenAddress, ethClient)
	if err != nil {
		log.Printf("%v", err)
		return nil, err
	}

	return &evmResponseHandler{
		ethClient:           ethClient,
		logger:              logger,
		cmAccountAddress:    cmAccountAddress,
		bookingTokenAddress: bookingTokenAddress,
		bookingService:      *bookingService,
		bookingToken:        *bookingToken,
		serviceRegistry:     serviceRegistry,
		evmEventListener:    events.NewEventListener(ethClient, logger),
		erc20:               erc20,
	}, nil
}

type evmResponseHandler struct {
	ethClient           *ethclient.Client
	logger              *zap.SugaredLogger
	cmAccountAddress    common.Address
	bookingTokenAddress common.Address
	bookingService      booking.Service
	bookingToken        bookingtoken.Bookingtoken
	serviceRegistry     ServiceRegistry
	evmEventListener    *events.EventListener
	erc20               erc20.Service
}

func (h *evmResponseHandler) HandleResponse(ctx context.Context, msgType types.MessageType, request protoreflect.ProtoMessage, response protoreflect.ProtoMessage) {
	switch msgType {
	case generated.MintServiceV1Request: // distributor will post-process a mint request to buy the returned NFT
		if h.handleMintRequestV1(ctx, response) {
			return // TODO @evlekht we don't need this if true/false then do nothing
		}
	case generated.MintServiceV1Response: // supplier will act upon receiving a mint response by minting an NFT
		if h.handleMintResponseV1(ctx, response, request) {
			return // TODO @evlekht we don't need this if true/false then do nothing
		}
	case generated.MintServiceV2Request: // distributor will post-process a mint request to buy the returned NFT
		if h.handleMintRequestV2(ctx, response) {
			return // TODO @evlekht we don't need this if true/false then do nothing
		}
	case generated.MintServiceV2Response: // supplier will act upon receiving a mint response by minting an NFT
		if h.handleMintResponseV2(ctx, response, request) {
			return // TODO @evlekht we don't need this if true/false then do nothing
		}
	}
}

func (h *evmResponseHandler) HandleRequest(_ context.Context, msgType types.MessageType, request protoreflect.ProtoMessage) error {
	switch msgType {
	case generated.MintServiceV2Request:
		mintReq, ok := request.(*bookv2.MintRequest)
		if !ok {
			return nil
		}
		mintReq.BuyerAddress = h.cmAccountAddress.Hex()
	case generated.MintServiceV1Request:
		mintReq, ok := request.(*bookv1.MintRequest)
		if !ok {
			return nil
		}
		mintReq.BuyerAddress = h.cmAccountAddress.Hex()
	}
	return nil
}
