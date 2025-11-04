// Copyright (C) 2022-2025, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package tests

import (
	"context"
	"math/big"
	"testing"
	"time"

	cancellationv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/cancellation/v1"
	notificationv2 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/notification/v2"
	typesv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v1"
	typesv3 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v3"
	botGenerated "github.com/chain4travel/camino-messenger-bot/v11/internal/rpc/generated"
	"github.com/chain4travel/camino-messenger-bot/v11/pkg/price"
	"github.com/chain4travel/camino-messenger-bot/v11/pp-mock/common"
	cancellation_handlers "github.com/chain4travel/camino-messenger-bot/v11/pp-mock/handlers/cancellation/v1"
	"github.com/chain4travel/camino-messenger-bot/v11/pp-mock/proto/pb/events"
	"github.com/chain4travel/camino-messenger-bot/v11/tests/e2e/bot"
	partnerplugin "github.com/chain4travel/camino-messenger-bot/v11/tests/e2e/partner_plugin"
	"github.com/chain4travel/camino-messenger-bot/v11/tests/e2e/suite"
	ethCommon "github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

var _ suite.Test = (*TestCancellationV1)(nil)

func init() {
	Tests["CancellationV1"] = &TestCancellationV1{}
}

type TestCancellationV1 struct {
	*suite.Environment

	supplierPartnerPlugin *partnerplugin.PartnerPlugin
	supplierPPEventStream events.EventsService_SubscribeClient
	supplierBot           *bot.Bot

	distributorPartnerPlugin *partnerplugin.PartnerPlugin
	distributorPPEventStream events.EventsService_SubscribeClient
	distributorBot           *bot.Bot
}

func (tt *TestCancellationV1) Setup(e *suite.Environment) {
	tt.Environment = e
}

func (tt *TestCancellationV1) Run(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTestTimeout)
	defer cancel()

	tt.prepare(ctx, t)

	t.Run("CheckCancellationV1", func(t *testing.T) {
		tt.testCheckCancellationV1(ctx, t)
	})
	t.Run("Distributor initiates, basic flow", func(t *testing.T) {
		tt.testCancellationV1DistributorInitiatesBasic(ctx, t)
	})
	t.Run("Distributor initiates", func(t *testing.T) {
		tt.testCancellationV1DistributorInitiates(ctx, t)
	})
	t.Run("Supplier initiates", func(t *testing.T) {
		tt.testCancellationV1SupplierInitiates(ctx, t)
	})
}

func (tt *TestCancellationV1) prepare(ctx context.Context, t *testing.T) {
	require.NoError(t, tt.CaminoNetwork.Client.RegisterCMServices(ctx,
		botGenerated.AccommodationSearchServiceV3,
		botGenerated.ValidationServiceV3,
		botGenerated.MintServiceV3,
		botGenerated.CheckCancellationServiceV1,
	))

	var err error

	// supplier bot
	tt.supplierPartnerPlugin = tt.CreatePartnerPlugin(ctx, t)
	tt.supplierBot = tt.CreateBot(ctx, t, true, tt.supplierPartnerPlugin,
		bot.WithServices([]bot.CMService{
			{Name: botGenerated.AccommodationSearchServiceV3, Fee: 120},
			{Name: botGenerated.ValidationServiceV3, Fee: 130},
			{Name: botGenerated.MintServiceV3, Fee: 140},
			{Name: botGenerated.CheckCancellationServiceV1, Fee: 150},
		}),
	)
	tt.supplierPPEventStream, err = tt.supplierPartnerPlugin.SubscribeForEvents(ctx)
	require.NoError(t, err)

	// distributor bot
	tt.distributorPartnerPlugin = tt.CreatePartnerPlugin(ctx, t)
	tt.distributorBot = tt.CreateBot(ctx, t, true, tt.distributorPartnerPlugin)
	tt.distributorPPEventStream, err = tt.distributorPartnerPlugin.SubscribeForEvents(ctx)
	require.NoError(t, err)
}

func (tt *TestCancellationV1) testCancellationV1DistributorInitiatesBasic(ctx context.Context, t *testing.T) {
	// reasons are selected randomly, just to be unique among requests

	tokenID, _, bookingPrice := mintBuyAccommodationTokenV3(ctx, t, tt.Environment, tt.supplierPPEventStream, tt.distributorBot, tt.supplierBot)
	refundAmount := common.CloneProto(bookingPrice)
	time.Sleep(2 * time.Second)

	cancellationHelper := newCancellationV1Helper(
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

func (tt *TestCancellationV1) testCancellationV1DistributorInitiates(ctx context.Context, t *testing.T) {
	// reasons are selected randomly, just to be unique among requests

	tokenID, _, bookingPrice := mintBuyAccommodationTokenV3(ctx, t, tt.Environment, tt.supplierPPEventStream, tt.distributorBot, tt.supplierBot)
	refundAmount := common.CloneProto(bookingPrice)
	time.Sleep(2 * time.Second)

	cancellationHelper := newCancellationV1Helper(
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

func (tt *TestCancellationV1) testCancellationV1SupplierInitiates(ctx context.Context, t *testing.T) {
	// reasons are selected randomly, just to be unique among requests

	// making new booking so we can test different flow
	tokenID, _, bookingPrice := mintBuyAccommodationTokenV3(ctx, t, tt.Environment, tt.supplierPPEventStream, tt.distributorBot, tt.supplierBot)
	refundAmount := common.CloneProto(bookingPrice)
	time.Sleep(2 * time.Second)

	cancellationHelper := newCancellationV1Helper(
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

func (tt *TestCancellationV1) testCheckCancellationV1(ctx context.Context, t *testing.T) {
	reqTime := time.Now()
	req := &cancellationv1.CheckCancellationRequest{
		Header:  &typesv1.RequestHeader{BaseHeader: &typesv1.Header{}},
		TokenId: 1,
		Reason:  cancellationv1.CancellationReason_CANCELLATION_REASON_AMENITY_REQUIREMENT_CHANGE,
	}
	resp, err := tt.distributorBot.CheckCancellationServiceV1.CheckCancellation(
		requestContext(ctx, tt.supplierBot.CMAccountAddress()),
		req,
	)
	require.NoError(t, err)
	_, err = tt.supplierPPEventStream.Recv()
	require.NoError(t, err)

	tt.DebugPrintRequestResponse(req, resp)
	require.Equal(t, typesv1.StatusType_STATUS_TYPE_SUCCESS, resp.Header.Status, "unexpected response status")
	require.Empty(t, resp.Header.Alerts, "unexpected response alerts")
	require.Equal(t, req.TokenId, resp.TokenId, "unexpected response TokenID")
	require.Equal(t, common.BookingTokenPriceValue, resp.RefundAmount.Value, "unexpected response RefundAmount.Value")
	require.Equal(t, price.NativeTokenDecimals, resp.RefundAmount.Decimals, "unexpected response RefundAmount.Decimals")
	require.IsType(t, &typesv3.Currency_NativeToken{}, resp.RefundAmount.Currency.Currency, "unexpected response RefundAmount.Currency type")
	require.Equal(t, cancellation_handlers.PolicyID, resp.PolicyIdApplied, "unexpected response PolicyIdApplied")
	require.Equal(t, cancellationv1.CancellationCheckStatus_CANCELLATION_CHECK_STATUS_CONFIRM, resp.Status, "unexpected response status")
	require.Equal(t, cancellationv1.RejectionReason_REJECTION_REASON_UNSPECIFIED, resp.RejectionReason, "unexpected response RejectionReason")
	require.WithinRange(t, resp.Timestamp.AsTime(), reqTime, time.Now(), "unexpected response Timestamp, expected it to be within the request time and now")
}

func newCancellationV1Helper(
	ctx context.Context,
	t *testing.T,
	e *suite.Environment,
	distributorBot *bot.Bot,
	supplierBot *bot.Bot,
	distributorPPEventStream events.EventsService_SubscribeClient,
	supplierPPEventStream events.EventsService_SubscribeClient,
	tokenID uint64,
) *cancellationV1Helper {
	return &cancellationV1Helper{
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

type cancellationV1Helper struct {
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

func (h *cancellationV1Helper) getBot(
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

func (h *cancellationV1Helper) initiateCancellation(
	initiator SupplierOrDistributor,
	refundAmount *typesv3.Price,
	reason cancellationv1.CancellationReason,
) {
	refundAmountBig, err := price.ToBigInt(refundAmount.Value, refundAmount.Decimals, price.NativeTokenDecimals)
	h.require.NoError(err)

	initiatorBot := h.getBot(initiator)

	initiateCancellationResp, err := initiatorBot.CancellationServiceV1.InitiateCancellation(h.ctx, &cancellationv1.InitiateCancellationRequest{
		Header:       &typesv1.RequestHeader{BaseHeader: &typesv1.Header{}},
		TokenId:      h.tokenID,
		RefundAmount: refundAmount,
		Reason:       reason,
	})
	h.require.NoError(err)

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

	h.expectCancellationPendingNotification(h.distributorPPEventStream, refundAmountBig, initiateCancellationResp.TransactionId.Hash)
	h.expectCancellationPendingNotification(h.supplierPPEventStream, refundAmountBig, initiateCancellationResp.TransactionId.Hash)
}

func (h *cancellationV1Helper) counterCancellation(
	counterer SupplierOrDistributor,
	refundAmount *typesv3.Price,
	reason cancellationv1.CounterReason,
) {
	refundAmountBig, err := price.ToBigInt(refundAmount.Value, refundAmount.Decimals, price.NativeTokenDecimals)
	h.require.NoError(err)

	countererBot := h.getBot(counterer)

	counterCancellationResp, err := countererBot.CancellationServiceV1.CounterCancellation(h.ctx, &cancellationv1.CounterCancellationRequest{
		Header:       &typesv1.RequestHeader{BaseHeader: &typesv1.Header{}},
		TokenId:      h.tokenID,
		RefundAmount: refundAmount,
		Reason:       reason,
	})
	h.require.NoError(err)

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

	h.expectCancellationPendingNotification(h.distributorPPEventStream, refundAmountBig, counterCancellationResp.TransactionId.Hash)
	h.expectCancellationPendingNotification(h.supplierPPEventStream, refundAmountBig, counterCancellationResp.TransactionId.Hash)
}

func (h *cancellationV1Helper) acceptCancellation(
	accepter SupplierOrDistributor,
	refundAmount *typesv3.Price,
) {
	refundAmountBig, err := price.ToBigInt(refundAmount.Value, refundAmount.Decimals, price.NativeTokenDecimals)
	h.require.NoError(err)

	accepterBot := h.getBot(accepter)

	acceptCancellationResp, err := accepterBot.CancellationServiceV1.AcceptCancellation(h.ctx, &cancellationv1.AcceptCancellationRequest{
		Header:       &typesv1.RequestHeader{BaseHeader: &typesv1.Header{}},
		TokenId:      h.tokenID,
		RefundAmount: refundAmount,
	})
	h.require.NoError(err)

	switch accepter {
	case Supplier:
		h.supplierAccepted = true
	case Distributor:
		h.ownerAccepted = true
	}

	h.expectCancellationPendingNotification(h.distributorPPEventStream, refundAmountBig, acceptCancellationResp.TransactionId.Hash)
	h.expectCancellationPendingNotification(h.supplierPPEventStream, refundAmountBig, acceptCancellationResp.TransactionId.Hash)
}

func (h *cancellationV1Helper) rejectCancellation(
	rejecter SupplierOrDistributor,
	reason cancellationv1.RejectionReason,
) {
	rejecterBot := h.getBot(rejecter)

	rejectCancellationResp, err := rejecterBot.CancellationServiceV1.RejectCancellation(h.ctx, &cancellationv1.RejectCancellationRequest{
		Header:  &typesv1.RequestHeader{BaseHeader: &typesv1.Header{}},
		TokenId: h.tokenID,
		Reason:  reason,
	})
	h.require.NoError(err)

	h.timesRejected++
	h.rejectionReason = reason

	h.expectCancellationRejectedNotification(h.distributorPPEventStream, rejectCancellationResp.TransactionId.Hash)
	h.expectCancellationRejectedNotification(h.supplierPPEventStream, rejectCancellationResp.TransactionId.Hash)
}

func (h *cancellationV1Helper) withdrawCancellation(
	withdrawer SupplierOrDistributor,
	reason cancellationv1.WithdrawalReason,
) {
	withdrawerBot := h.getBot(withdrawer)

	withdrawCancellationResp, err := withdrawerBot.CancellationServiceV1.WithdrawCancellation(h.ctx, &cancellationv1.WithdrawCancellationRequest{
		Header:  &typesv1.RequestHeader{BaseHeader: &typesv1.Header{}},
		TokenId: h.tokenID,
		Reason:  reason,
	})
	h.require.NoError(err)

	h.withdrawalReason = reason

	h.expectCancellationWithdrawnNotification(h.distributorPPEventStream, withdrawCancellationResp.TransactionId.Hash)
	h.expectCancellationWithdrawnNotification(h.supplierPPEventStream, withdrawCancellationResp.TransactionId.Hash)
}

func (h *cancellationV1Helper) finalizeCancellation(refundAmount *typesv3.Price) {
	refundAmountBig, err := price.ToBigInt(refundAmount.Value, refundAmount.Decimals, price.NativeTokenDecimals)
	h.require.NoError(err)

	supplierBalance, err := h.e.CaminoNetwork.Client.BalanceOf(h.ctx, h.supplierBot.CMAccountAddress())
	h.require.NoError(err)
	distributorBalance, err := h.e.CaminoNetwork.Client.BalanceOf(h.ctx, h.distributorBot.CMAccountAddress())
	h.require.NoError(err)

	expectedSupplierBalance := big.NewInt(0).Sub(supplierBalance, refundAmountBig)
	expectedDistributorBalance := big.NewInt(0).Add(distributorBalance, refundAmountBig)

	finalizeCancellationResp, err := h.supplierBot.CancellationServiceV1.FinalizeCancellation(h.ctx, &cancellationv1.FinalizeCancellationRequest{
		Header:  &typesv1.RequestHeader{BaseHeader: &typesv1.Header{}},
		TokenId: h.tokenID,
		RefundAmount: &typesv3.Price{
			Value:    refundAmount.Value,
			Decimals: refundAmount.Decimals,
			Currency: &typesv3.Currency{Currency: &typesv3.Currency_NativeToken{}},
		},
	})
	h.require.NoError(err)

	h.expectCancellationFinalizedNotification(h.supplierPPEventStream, finalizeCancellationResp.TransactionId.Hash)
	h.expectCancellationFinalizedNotification(h.distributorPPEventStream, finalizeCancellationResp.TransactionId.Hash)

	supplierBalanceAfter, err := h.e.CaminoNetwork.Client.BalanceOf(h.ctx, h.supplierBot.CMAccountAddress())
	h.require.NoError(err)
	distributorBalanceAfter, err := h.e.CaminoNetwork.Client.BalanceOf(h.ctx, h.distributorBot.CMAccountAddress())
	h.require.NoError(err)

	h.require.Truef(
		supplierBalanceAfter.Cmp(expectedSupplierBalance) == 0,
		"unexpected supplier balance after cancellation: expected %s, actual %s",
		expectedSupplierBalance.String(), supplierBalanceAfter.String(),
	)
	h.require.Truef(
		distributorBalanceAfter.Cmp(expectedDistributorBalance) == 0,
		"unexpected distributor balance after cancellation: expected %s, actual %s",
		expectedDistributorBalance.String(), distributorBalanceAfter.String(),
	)
}

func (h *cancellationV1Helper) expectCancellationPendingNotification(
	ppEventsStream events.EventsService_SubscribeClient,
	refundAmount *big.Int,
	txHash string,
) {
	eventMsg, err := ppEventsStream.Recv()
	h.require.NoError(err)
	h.e.DebugPrintProtoMessage(eventMsg)
	cancellationPendingNotification := &notificationv2.CancellationPending{}
	h.require.NoError(proto.Unmarshal(eventMsg.Data, cancellationPendingNotification))
	h.require.Equal(h.tokenID, cancellationPendingNotification.TokenId)
	h.require.Equal(h.initialProposer.Hex(), cancellationPendingNotification.InitialProposer.Address)
	h.require.Equal(h.currentProposer.Hex(), cancellationPendingNotification.CurrentProposer.Address)
	h.require.Equal(refundAmount.Uint64(), cancellationPendingNotification.RefundAmount)
	h.require.Equal(h.timesCountered, cancellationPendingNotification.TimesCountered)
	h.require.Equal(h.timesRejected, cancellationPendingNotification.TimesRejected)
	h.require.Equal(h.cancellationReason, cancellationPendingNotification.CancellationReason)
	h.require.Equal(h.rejectionReason, cancellationPendingNotification.RejectionReason)
	h.require.Equal(h.counterReason, cancellationPendingNotification.CounterReason)
	h.require.Equal(h.withdrawalReason, cancellationPendingNotification.WithdrawalReason)
	h.require.Equal(h.ownerAccepted, cancellationPendingNotification.OwnerAccepted)
	h.require.Equal(h.supplierAccepted, cancellationPendingNotification.SupplierAccepted)
	h.require.Equal(txHash, cancellationPendingNotification.TxId.Hash)
}

func (h *cancellationV1Helper) expectCancellationRejectedNotification(
	ppEventsStream events.EventsService_SubscribeClient,
	txHash string,
) {
	eventMsg, err := ppEventsStream.Recv()
	h.require.NoError(err)
	h.e.DebugPrintProtoMessage(eventMsg)
	cancellationRejectedNotification := &notificationv2.CancellationRejected{}
	h.require.NoError(proto.Unmarshal(eventMsg.Data, cancellationRejectedNotification))
	h.require.Equal(h.tokenID, cancellationRejectedNotification.TokenId)
	h.require.Equal(h.rejectionReason, cancellationRejectedNotification.Reason)
	h.require.Equal(txHash, cancellationRejectedNotification.TxId.Hash)
}

func (h *cancellationV1Helper) expectCancellationWithdrawnNotification(
	ppEventsStream events.EventsService_SubscribeClient,
	txHash string,
) {
	eventMsg, err := ppEventsStream.Recv()
	h.require.NoError(err)
	h.e.DebugPrintProtoMessage(eventMsg)
	cancellationWithdrawnNotification := &notificationv2.CancellationWithdrawn{}
	h.require.NoError(proto.Unmarshal(eventMsg.Data, cancellationWithdrawnNotification))
	h.require.Equal(h.tokenID, cancellationWithdrawnNotification.TokenId)
	h.require.Equal(h.withdrawalReason, cancellationWithdrawnNotification.Reason)
	h.require.Equal(txHash, cancellationWithdrawnNotification.TxId.Hash)
}

func (h *cancellationV1Helper) expectCancellationFinalizedNotification(
	ppEventsStream events.EventsService_SubscribeClient,
	txHash string,
) {
	eventMsg, err := ppEventsStream.Recv()
	h.require.NoError(err)
	h.e.DebugPrintProtoMessage(eventMsg)
	cancellationFinalizedNotification := &notificationv2.CancellationFinalized{}
	h.require.NoError(proto.Unmarshal(eventMsg.Data, cancellationFinalizedNotification))
	h.require.Equal(h.tokenID, cancellationFinalizedNotification.TokenId)
	h.require.Equal(txHash, cancellationFinalizedNotification.TxId.Hash)
}
