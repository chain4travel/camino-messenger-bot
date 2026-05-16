// Copyright (C) 2022-2026, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package tests

import (
	"context"
	"math/big"
	"testing"
	"time"

	cancellationv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/cancellation/v1"
	cancellationv2 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/cancellation/v2"
	notificationv3 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/notification/v3"
	typesv4 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v4"
	"buf.build/go/protovalidate"
	botGenerated "github.com/chain4travel/camino-messenger-bot/v13/internal/rpc/generated"
	"github.com/chain4travel/camino-messenger-bot/v13/pkg/conversion"
	"github.com/chain4travel/camino-messenger-bot/v13/pkg/price"
	"github.com/chain4travel/camino-messenger-bot/v13/pp-mock/common"
	"github.com/chain4travel/camino-messenger-bot/v13/pp-mock/proto/pb/events"
	"github.com/chain4travel/camino-messenger-bot/v13/tests/e2e/bot"
	partnerplugin "github.com/chain4travel/camino-messenger-bot/v13/tests/e2e/partner_plugin"
	"github.com/chain4travel/camino-messenger-bot/v13/tests/e2e/suite"
	ethCommon "github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

var _ suite.Test = (*TestCancellationV2)(nil)

func init() {
	Tests["CancellationV2"] = &TestCancellationV2{}
}

type TestCancellationV2 struct {
	*suite.Environment

	supplierPartnerPlugin *partnerplugin.PartnerPlugin
	supplierPPEventStream events.EventsService_SubscribeClient
	supplierBot           *bot.Bot

	distributorPartnerPlugin *partnerplugin.PartnerPlugin
	distributorPPEventStream events.EventsService_SubscribeClient
	distributorBot           *bot.Bot
}

func (tt *TestCancellationV2) Setup(e *suite.Environment) {
	tt.Environment = e
}

func (tt *TestCancellationV2) Run(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTestTimeout)
	defer cancel()

	tt.prepare(ctx, t)

	t.Run("CheckCancellationV2", func(t *testing.T) {
		tt.testCheckCancellationV2(ctx, t)
	})
	t.Run("Distributor initiates, basic flow", func(t *testing.T) {
		tt.testCancellationV2DistributorInitiatesBasic(ctx, t)
	})
	t.Run("Distributor initiates", func(t *testing.T) {
		tt.testCancellationV2DistributorInitiates(ctx, t)
	})
	t.Run("Supplier initiates", func(t *testing.T) {
		tt.testCancellationV2SupplierInitiates(ctx, t)
	})
}

func (tt *TestCancellationV2) prepare(ctx context.Context, t *testing.T) {
	require.NoError(t, tt.CaminoNetwork.Client.RegisterCMServices(ctx,
		botGenerated.AccommodationSearchServiceV4,
		botGenerated.ValidationServiceV4,
		botGenerated.MintServiceV4,
		botGenerated.CheckCancellationServiceV2,
	))

	tt.supplierPartnerPlugin = tt.CreatePartnerPlugin(ctx, t)

	// bot with partnerPlugin and without rpc server (supplier)
	tt.supplierBot = tt.CreateBot(ctx, t, true, tt.supplierPartnerPlugin,
		bot.WithServices([]bot.CMService{
			{Name: botGenerated.AccommodationSearchServiceV4, Fee: 120},
			{Name: botGenerated.ValidationServiceV4, Fee: 130},
			{Name: botGenerated.MintServiceV4, Fee: 140},
			{Name: botGenerated.CheckCancellationServiceV2, Fee: 150},
		}),
	)

	tt.distributorPartnerPlugin = tt.CreatePartnerPlugin(ctx, t)

	// bot without partnerPlugin and with rpc server (distributor)
	tt.distributorBot = tt.CreateBot(ctx, t, true, tt.distributorPartnerPlugin)

	var err error
	tt.supplierPPEventStream, err = tt.supplierPartnerPlugin.SubscribeForEvents(ctx)
	require.NoError(t, err)
	tt.distributorPPEventStream, err = tt.distributorPartnerPlugin.SubscribeForEvents(ctx)
	require.NoError(t, err)
}

func (tt *TestCancellationV2) testCancellationV2DistributorInitiatesBasic(ctx context.Context, t *testing.T) {
	// reasons are selected randomly, just to be unique among requests

	tokenID, _, bookingPrice := mintBuyAccommodationTokenV4(ctx, t, tt.Environment, tt.supplierPPEventStream, tt.distributorBot, tt.supplierBot)
	refundAmount := common.CloneProto(bookingPrice)
	time.Sleep(2 * time.Second)

	cancellationHelper := newCancellationV2Helper(
		ctx,
		t,
		tt.Environment,
		tt.distributorBot,
		tt.supplierBot,
		tt.distributorPPEventStream,
		tt.supplierPPEventStream,
		tokenID,
	)

	// distributor initiates cancellation
	cancellationHelper.initiateCancellation(
		Distributor,
		refundAmount,
		cancellationv1.CancellationReason_CANCELLATION_REASON_AMENITY_REQUIREMENT_CHANGE,
	)

	// supplier finalizes (implicit accept)
	cancellationHelper.finalizeCancellation(
		refundAmount,
	)
}

func (tt *TestCancellationV2) testCancellationV2DistributorInitiates(ctx context.Context, t *testing.T) {
	// reasons are selected randomly, just to be unique among requests

	tokenID, _, bookingPrice := mintBuyAccommodationTokenV4(ctx, t, tt.Environment, tt.supplierPPEventStream, tt.distributorBot, tt.supplierBot)
	refundAmount := common.CloneProto(bookingPrice)
	time.Sleep(2 * time.Second)

	cancellationHelper := newCancellationV2Helper(
		ctx,
		t,
		tt.Environment,
		tt.distributorBot,
		tt.supplierBot,
		tt.distributorPPEventStream,
		tt.supplierPPEventStream,
		tokenID,
	)

	// attempt 1 (distributor, reject)
	// distributor initiates cancellation
	cancellationHelper.initiateCancellation(
		Distributor,
		refundAmount,
		cancellationv1.CancellationReason_CANCELLATION_REASON_AMENITY_REQUIREMENT_CHANGE,
	)

	// supplier rejects
	cancellationHelper.rejectCancellation(
		Supplier,
		cancellationv1.RejectionReason_REJECTION_REASON_INVALID_SERVICE_OR_BOOKING_REFERENCE,
	)

	// attempt 2 (distributor, withdraw)
	// distributor initiates cancellation again
	cancellationHelper.initiateCancellation(
		Distributor,
		refundAmount,
		cancellationv1.CancellationReason_CANCELLATION_REASON_CALENDAR_SYNC_ERROR,
	)

	// distributor withdraws
	cancellationHelper.withdrawCancellation(
		Distributor,
		cancellationv1.WithdrawalReason_WITHDRAWAL_REASON_FEE_TOO_HIGH,
	)

	// attempt 3 (supplier, reject)
	// supplier initiates cancellation
	cancellationHelper.initiateCancellation(
		Supplier,
		refundAmount,
		cancellationv1.CancellationReason_CANCELLATION_REASON_BOOKING_SYSTEM_ERROR,
	)

	// distributor rejects
	cancellationHelper.rejectCancellation(
		Distributor,
		cancellationv1.RejectionReason_REJECTION_REASON_REFUND_CURRENCY_NOT_SUPPORTED,
	)

	// attempt 4 (supplier, withdraw)
	// supplier initiates cancellation again
	cancellationHelper.initiateCancellation(
		Supplier,
		refundAmount,
		cancellationv1.CancellationReason_CANCELLATION_REASON_DOUBLE_BOOKING,
	)

	// supplier withdraws
	cancellationHelper.withdrawCancellation(
		Supplier,
		cancellationv1.WithdrawalReason_WITHDRAWAL_REASON_PAYMENT_ISSUE_RESOLVED,
	)

	// attempt 5 (distributor, counter, counter, success)
	// distributor initiates cancellation 3rd time
	cancellationHelper.initiateCancellation(
		Distributor,
		refundAmount,
		cancellationv1.CancellationReason_CANCELLATION_REASON_OVERBOOKING,
	)

	// supplier counters
	cancellationHelper.counterCancellation(
		Supplier,
		refundAmount,
		cancellationv1.CounterReason_COUNTER_REASON_CUSTOMER_HISTORY_CONSIDERATION,
	)

	// distributor counters back
	cancellationHelper.counterCancellation(
		Distributor,
		refundAmount,
		cancellationv1.CounterReason_COUNTER_REASON_BOOKING_TERMS_REQUIREMENT,
	)

	// supplier finalizes (implicit accept)
	cancellationHelper.finalizeCancellation(
		refundAmount,
	)
}

func (tt *TestCancellationV2) testCancellationV2SupplierInitiates(ctx context.Context, t *testing.T) {
	// reasons are selected randomly, just to be unique among requests

	// making new booking so we can test different flow
	tokenID, _, bookingPrice := mintBuyAccommodationTokenV4(ctx, t, tt.Environment, tt.supplierPPEventStream, tt.distributorBot, tt.supplierBot)
	refundAmount := common.CloneProto(bookingPrice)
	time.Sleep(2 * time.Second)

	cancellationHelper := newCancellationV2Helper(
		ctx,
		t,
		tt.Environment,
		tt.distributorBot,
		tt.supplierBot,
		tt.distributorPPEventStream,
		tt.supplierPPEventStream,
		tokenID,
	)

	// supplier initiates cancellation
	cancellationHelper.initiateCancellation(
		Supplier,
		refundAmount,
		cancellationv1.CancellationReason_CANCELLATION_REASON_CONTRACT_TERMINATION,
	)

	// distributor counters
	cancellationHelper.counterCancellation(
		Distributor,
		refundAmount,
		cancellationv1.CounterReason_COUNTER_REASON_LONG_STAY_DISCOUNT_APPLIED,
	)

	// supplier counters back
	cancellationHelper.counterCancellation(
		Supplier,
		refundAmount,
		cancellationv1.CounterReason_COUNTER_REASON_PENALTY_CALCULATION_ERROR,
	)

	// distributor accepts
	cancellationHelper.acceptCancellation(
		Distributor,
		refundAmount,
	)

	// supplier finalizes
	cancellationHelper.finalizeCancellation(
		refundAmount,
	)
}

func (tt *TestCancellationV2) testCheckCancellationV2(ctx context.Context, t *testing.T) {
	reqTime := time.Now()
	req := &cancellationv2.CheckCancellationRequest{
		Header:  &typesv4.RequestHeader{BaseHeader: &typesv4.Header{Version: &typesv4.Version{}}},
		TokenId: 1,
		Reason:  cancellationv1.CancellationReason_CANCELLATION_REASON_AMENITY_REQUIREMENT_CHANGE,
	}
	resp, err := tt.distributorBot.CheckCancellationServiceV2.CheckCancellation(
		requestContext(ctx, tt.supplierBot.CMAccountAddress()),
		req,
	)
	require.NoError(t, err)
	_, err = tt.supplierPPEventStream.Recv()
	require.NoError(t, err)

	tt.DebugPrintRequestResponse(req, resp)

	require.NoError(t, protovalidate.Validate(resp))

	successResp := resp.GetSuccessResponse()
	require.NotNil(t, successResp, "unexpected response status")

	require.WithinRange(t, successResp.Timestamp.AsTime(), reqTime, time.Now(), "unexpected response Timestamp, expected it to be within the request time and now")
	successResp.Timestamp = nil
	successResp.Header.BaseHeader.Version = nil

	requireProtoEqual(t, &cancellationv2.CheckCancellationResponse{
		Response: &cancellationv2.CheckCancellationResponse_SuccessResponse{
			SuccessResponse: &cancellationv2.CheckCancellationSuccessResponse{
				Header: &typesv4.SuccessResponseHeader{
					BaseHeader: &typesv4.Header{},
				},
				TokenId: req.TokenId,
				RefundAmount: &typesv4.Price{
					Value:    common.BookingTokenPriceValue,
					Decimals: uint32(price.NativeTokenDecimals),
					Currency: &typesv4.Currency{Currency: &typesv4.Currency_NativeToken{}},
				},
				PolicyIdApplied: common.CancellationPolicyID,
				Status:          cancellationv2.CancellationCheckStatus_CANCELLATION_CHECK_STATUS_CONFIRM,
				RejectionReason: cancellationv1.RejectionReason_REJECTION_REASON_UNSPECIFIED,
			},
		},
	}, resp)
}

func newCancellationV2Helper(
	ctx context.Context,
	t *testing.T,
	e *suite.Environment,
	distributorBot *bot.Bot,
	supplierBot *bot.Bot,
	distributorPPEventStream events.EventsService_SubscribeClient,
	supplierPPEventStream events.EventsService_SubscribeClient,
	tokenID uint64,
) *cancellationV2Helper {
	return &cancellationV2Helper{
		ctx:                      ctx,
		t:                        t,
		e:                        e,
		require:                  require.New(t),
		distributorBot:           distributorBot,
		supplierBot:              supplierBot,
		distributorPPEventStream: distributorPPEventStream,
		supplierPPEventStream:    supplierPPEventStream,
		tokenID:                  tokenID,
	}
}

type cancellationV2Helper struct {
	ctx                      context.Context
	t                        *testing.T
	e                        *suite.Environment
	require                  *require.Assertions
	distributorBot           *bot.Bot
	supplierBot              *bot.Bot
	distributorPPEventStream events.EventsService_SubscribeClient
	supplierPPEventStream    events.EventsService_SubscribeClient
	tokenID                  uint64

	initialProposer    ethCommon.Address
	currentProposer    ethCommon.Address
	timesRejected      uint64
	timesCountered     uint64
	supplierAccepted   bool
	ownerAccepted      bool
	cancellationReason cancellationv1.CancellationReason
	counterReason      cancellationv1.CounterReason
	rejectionReason    cancellationv1.RejectionReason
	withdrawalReason   cancellationv1.WithdrawalReason
}

func (h *cancellationV2Helper) getBot(
	supplierOrDistributor SupplierOrDistributor,
) *bot.Bot {
	switch supplierOrDistributor {
	case Supplier: // from supplier to distributor
		return h.supplierBot
	case Distributor: // from distributor to supplier
		return h.distributorBot
	default:
		h.require.FailNow("invalid supplierOrDistributor value: %v", supplierOrDistributor)
		return nil
	}
}

func (h *cancellationV2Helper) initiateCancellation(
	initiator SupplierOrDistributor,
	refundAmount *typesv4.Price,
	reason cancellationv1.CancellationReason,
) {
	refundAmountBig, err := price.ToBigInt(refundAmount.Value, conversion.MustUInt32ToInt32(refundAmount.Decimals), price.NativeTokenDecimals)
	h.require.NoError(err)

	initiatorBot := h.getBot(initiator)

	resp, err := initiatorBot.CancellationServiceV2.InitiateCancellation(h.ctx, &cancellationv2.InitiateCancellationRequest{
		Header:       &typesv4.RequestHeader{BaseHeader: &typesv4.Header{Version: &typesv4.Version{}}},
		TokenId:      h.tokenID,
		RefundAmount: refundAmount,
		Reason:       reason,
	})
	h.require.NoError(err)
	h.require.NoError(protovalidate.Validate(resp))

	successResp := resp.GetSuccessResponse()
	h.require.NotNil(successResp, "unexpected response status")

	if h.initialProposer.Cmp(ethCommon.Address{}) == 0 {
		h.initialProposer = initiatorBot.CMAccountAddress()
	}
	h.currentProposer = initiatorBot.CMAccountAddress()
	h.cancellationReason = reason
	h.rejectionReason = cancellationv1.RejectionReason_REJECTION_REASON_UNSPECIFIED
	h.withdrawalReason = cancellationv1.WithdrawalReason_WITHDRAWAL_REASON_UNSPECIFIED
	h.counterReason = cancellationv1.CounterReason_COUNTER_REASON_UNSPECIFIED

	switch initiator {
	case Supplier:
		h.supplierAccepted = true
		h.ownerAccepted = false
	case Distributor:
		h.ownerAccepted = true
		h.supplierAccepted = false
	}

	h.expectCancellationPendingNotification(h.distributorPPEventStream, refundAmountBig, successResp.TransactionId.Hash)
	h.expectCancellationPendingNotification(h.supplierPPEventStream, refundAmountBig, successResp.TransactionId.Hash)
}

func (h *cancellationV2Helper) counterCancellation(
	counterer SupplierOrDistributor,
	refundAmount *typesv4.Price,
	reason cancellationv1.CounterReason,
) {
	refundAmountBig, err := price.ToBigInt(refundAmount.Value, conversion.MustUInt32ToInt32(refundAmount.Decimals), price.NativeTokenDecimals)
	h.require.NoError(err)

	countererBot := h.getBot(counterer)

	resp, err := countererBot.CancellationServiceV2.CounterCancellation(h.ctx, &cancellationv2.CounterCancellationRequest{
		Header:       &typesv4.RequestHeader{BaseHeader: &typesv4.Header{Version: &typesv4.Version{}}},
		TokenId:      h.tokenID,
		RefundAmount: refundAmount,
		Reason:       reason,
	})
	h.require.NoError(err)
	h.require.NoError(protovalidate.Validate(resp))

	successResp := resp.GetSuccessResponse()
	h.require.NotNil(successResp, "unexpected response status")

	h.currentProposer = countererBot.CMAccountAddress()
	h.counterReason = reason
	h.timesCountered++

	switch counterer {
	case Supplier:
		h.supplierAccepted = true
		h.ownerAccepted = false
	case Distributor:
		h.ownerAccepted = true
		h.supplierAccepted = false
	}

	h.expectCancellationPendingNotification(h.distributorPPEventStream, refundAmountBig, successResp.TransactionId.Hash)
	h.expectCancellationPendingNotification(h.supplierPPEventStream, refundAmountBig, successResp.TransactionId.Hash)
}

func (h *cancellationV2Helper) acceptCancellation(
	accepter SupplierOrDistributor,
	refundAmount *typesv4.Price,
) {
	refundAmountBig, err := price.ToBigInt(refundAmount.Value, conversion.MustUInt32ToInt32(refundAmount.Decimals), price.NativeTokenDecimals)
	h.require.NoError(err)

	accepterBot := h.getBot(accepter)

	resp, err := accepterBot.CancellationServiceV2.AcceptCancellation(h.ctx, &cancellationv2.AcceptCancellationRequest{
		Header:       &typesv4.RequestHeader{BaseHeader: &typesv4.Header{Version: &typesv4.Version{}}},
		TokenId:      h.tokenID,
		RefundAmount: refundAmount,
	})
	h.require.NoError(err)
	h.require.NoError(protovalidate.Validate(resp))

	successResp := resp.GetSuccessResponse()
	h.require.NotNil(successResp, "unexpected response status")

	switch accepter {
	case Supplier:
		h.supplierAccepted = true
	case Distributor:
		h.ownerAccepted = true
	}

	h.expectCancellationPendingNotification(h.distributorPPEventStream, refundAmountBig, successResp.TransactionId.Hash)
	h.expectCancellationPendingNotification(h.supplierPPEventStream, refundAmountBig, successResp.TransactionId.Hash)
}

func (h *cancellationV2Helper) rejectCancellation(
	rejecter SupplierOrDistributor,
	reason cancellationv1.RejectionReason,
) {
	rejecterBot := h.getBot(rejecter)

	resp, err := rejecterBot.CancellationServiceV2.RejectCancellation(h.ctx, &cancellationv2.RejectCancellationRequest{
		Header:  &typesv4.RequestHeader{BaseHeader: &typesv4.Header{Version: &typesv4.Version{}}},
		TokenId: h.tokenID,
		Reason:  reason,
	})
	h.require.NoError(err)
	h.require.NoError(protovalidate.Validate(resp))

	successResp := resp.GetSuccessResponse()
	h.require.NotNil(successResp, "unexpected response status")

	h.timesRejected++
	h.rejectionReason = reason

	h.expectCancellationRejectedNotification(h.distributorPPEventStream, successResp.TransactionId.Hash)
	h.expectCancellationRejectedNotification(h.supplierPPEventStream, successResp.TransactionId.Hash)
}

func (h *cancellationV2Helper) withdrawCancellation(
	withdrawer SupplierOrDistributor,
	reason cancellationv1.WithdrawalReason,
) {
	withdrawerBot := h.getBot(withdrawer)

	resp, err := withdrawerBot.CancellationServiceV2.WithdrawCancellation(h.ctx, &cancellationv2.WithdrawCancellationRequest{
		Header:  &typesv4.RequestHeader{BaseHeader: &typesv4.Header{Version: &typesv4.Version{}}},
		TokenId: h.tokenID,
		Reason:  reason,
	})
	h.require.NoError(err)
	h.require.NoError(protovalidate.Validate(resp))

	successResp := resp.GetSuccessResponse()
	h.require.NotNil(successResp, "unexpected response status")

	h.withdrawalReason = reason

	h.expectCancellationWithdrawnNotification(h.distributorPPEventStream, successResp.TransactionId.Hash)
	h.expectCancellationWithdrawnNotification(h.supplierPPEventStream, successResp.TransactionId.Hash)
}

func (h *cancellationV2Helper) finalizeCancellation(refundAmount *typesv4.Price) {
	refundAmountBig, err := price.ToBigInt(refundAmount.Value, conversion.MustUInt32ToInt32(refundAmount.Decimals), price.NativeTokenDecimals)
	h.require.NoError(err)

	supplierBalance := h.e.Balance(h.ctx, h.t, h.supplierBot)
	distributorBalance := h.e.Balance(h.ctx, h.t, h.distributorBot)

	expectedSupplierBalance := big.NewInt(0).Sub(supplierBalance, refundAmountBig)
	expectedDistributorBalance := big.NewInt(0).Add(distributorBalance, refundAmountBig)

	resp, err := h.supplierBot.CancellationServiceV2.FinalizeCancellation(h.ctx, &cancellationv2.FinalizeCancellationRequest{
		Header:  &typesv4.RequestHeader{BaseHeader: &typesv4.Header{Version: &typesv4.Version{}}},
		TokenId: h.tokenID,
		RefundAmount: &typesv4.Price{
			Value:    refundAmount.Value,
			Decimals: refundAmount.Decimals,
			Currency: &typesv4.Currency{Currency: &typesv4.Currency_NativeToken{}},
		},
	})
	h.require.NoError(err)
	h.require.NoError(protovalidate.Validate(resp))

	successResp := resp.GetSuccessResponse()
	h.require.NotNil(successResp, "unexpected response status")

	h.expectCancellationFinalizedNotification(h.supplierPPEventStream, successResp.TransactionId.Hash)
	h.expectCancellationFinalizedNotification(h.distributorPPEventStream, successResp.TransactionId.Hash)

	supplierBalanceAfter := h.e.Balance(h.ctx, h.t, h.supplierBot)
	distributorBalanceAfter := h.e.Balance(h.ctx, h.t, h.distributorBot)

	h.require.Equalf(expectedSupplierBalance, supplierBalanceAfter,
		"unexpected supplier balance after cancellation: expected %s, actual %s",
		expectedSupplierBalance.String(), supplierBalanceAfter.String(),
	)
	h.require.Equalf(expectedDistributorBalance, distributorBalanceAfter,
		"unexpected distributor balance after cancellation: expected %s, actual %s",
		expectedDistributorBalance.String(), distributorBalanceAfter.String(),
	)
}

func (h *cancellationV2Helper) expectCancellationPendingNotification(
	ppEventsStream events.EventsService_SubscribeClient,
	refundAmount *big.Int,
	txHash string,
) {
	eventMsg, err := ppEventsStream.Recv()
	h.require.NoError(err)
	h.e.DebugPrintProtoMessage(eventMsg)
	cancellationPendingNotification := &notificationv3.CancellationPending{}
	h.require.NoError(proto.Unmarshal(eventMsg.Data, cancellationPendingNotification))
	h.require.NoError(protovalidate.Validate(cancellationPendingNotification))
	requireProtoEqual(h.t, &notificationv3.CancellationPending{
		TokenId:            h.tokenID,
		InitialProposer:    &typesv4.EVMAddress{Address: h.initialProposer.Hex()},
		CurrentProposer:    &typesv4.EVMAddress{Address: h.currentProposer.Hex()},
		RefundAmount:       refundAmount.Uint64(),
		TimesCountered:     h.timesCountered,
		TimesRejected:      h.timesRejected,
		OwnerAccepted:      h.ownerAccepted,
		SupplierAccepted:   h.supplierAccepted,
		CancellationReason: h.cancellationReason,
		RejectionReason:    h.rejectionReason,
		CounterReason:      h.counterReason,
		WithdrawalReason:   h.withdrawalReason,
		TxId:               &typesv4.EVMTransactionID{Hash: txHash},
	}, cancellationPendingNotification)
}

func (h *cancellationV2Helper) expectCancellationRejectedNotification(
	ppEventsStream events.EventsService_SubscribeClient,
	txHash string,
) {
	eventMsg, err := ppEventsStream.Recv()
	h.require.NoError(err)
	h.e.DebugPrintProtoMessage(eventMsg)
	cancellationRejectedNotification := &notificationv3.CancellationRejected{}
	h.require.NoError(proto.Unmarshal(eventMsg.Data, cancellationRejectedNotification))
	h.require.NoError(protovalidate.Validate(cancellationRejectedNotification))
	requireProtoEqual(h.t, &notificationv3.CancellationRejected{
		TokenId: h.tokenID,
		Reason:  h.rejectionReason,
		TxId:    &typesv4.EVMTransactionID{Hash: txHash},
	}, cancellationRejectedNotification)
}

func (h *cancellationV2Helper) expectCancellationWithdrawnNotification(
	ppEventsStream events.EventsService_SubscribeClient,
	txHash string,
) {
	eventMsg, err := ppEventsStream.Recv()
	h.require.NoError(err)
	h.e.DebugPrintProtoMessage(eventMsg)
	cancellationWithdrawnNotification := &notificationv3.CancellationWithdrawn{}
	h.require.NoError(proto.Unmarshal(eventMsg.Data, cancellationWithdrawnNotification))
	h.require.NoError(protovalidate.Validate(cancellationWithdrawnNotification))
	requireProtoEqual(h.t, &notificationv3.CancellationWithdrawn{
		TokenId: h.tokenID,
		Reason:  h.withdrawalReason,
		TxId:    &typesv4.EVMTransactionID{Hash: txHash},
	}, cancellationWithdrawnNotification)
}

func (h *cancellationV2Helper) expectCancellationFinalizedNotification(
	ppEventsStream events.EventsService_SubscribeClient,
	txHash string,
) {
	eventMsg, err := ppEventsStream.Recv()
	h.require.NoError(err)
	h.e.DebugPrintProtoMessage(eventMsg)
	cancellationFinalizedNotification := &notificationv3.CancellationFinalized{}
	h.require.NoError(proto.Unmarshal(eventMsg.Data, cancellationFinalizedNotification))
	h.require.NoError(protovalidate.Validate(cancellationFinalizedNotification))
	requireProtoEqual(h.t, &notificationv3.CancellationFinalized{
		TokenId: h.tokenID,
		TxId:    &typesv4.EVMTransactionID{Hash: txHash},
	}, cancellationFinalizedNotification)
}
