// Copyright (C) 2022-2025, Chain4Travel AG. All rights reserved.
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
	botGenerated "github.com/chain4travel/camino-messenger-bot/v12/internal/rpc/generated"
	"github.com/chain4travel/camino-messenger-bot/v12/pkg/conversion"
	"github.com/chain4travel/camino-messenger-bot/v12/pkg/price"
	"github.com/chain4travel/camino-messenger-bot/v12/pp-mock/common"
	"github.com/chain4travel/camino-messenger-bot/v12/pp-mock/proto/pb/events"
	"github.com/chain4travel/camino-messenger-bot/v12/tests/e2e/bot"
	partnerplugin "github.com/chain4travel/camino-messenger-bot/v12/tests/e2e/partner_plugin"
	"github.com/chain4travel/camino-messenger-bot/v12/tests/e2e/suite"
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
	require.Equal(t, typesv4.StatusType_STATUS_TYPE_SUCCESS, resp.Header.Status, "unexpected response status")
	require.Empty(t, resp.Header.Alerts, "unexpected response alerts")
	require.Equal(t, req.TokenId, resp.TokenId, "unexpected response TokenID")
	require.Equal(t, common.BookingTokenPriceValue, resp.RefundAmount.Value, "unexpected response RefundAmount.Value")
	require.Equal(t, uint32(price.NativeTokenDecimals), resp.RefundAmount.Decimals, "unexpected response RefundAmount.Decimals")
	require.IsType(t, &typesv4.Currency_NativeToken{}, resp.RefundAmount.Currency.Currency, "unexpected response RefundAmount.Currency type")
	require.Equal(t, common.CancellationPolicyID, resp.PolicyIdApplied, "unexpected response PolicyIdApplied")
	require.Equal(t, cancellationv2.CancellationCheckStatus_CANCELLATION_CHECK_STATUS_CONFIRM, resp.Status, "unexpected response status")
	require.Equal(t, cancellationv1.RejectionReason_REJECTION_REASON_UNSPECIFIED, resp.RejectionReason, "unexpected response RejectionReason")
	require.WithinRange(t, resp.Timestamp.AsTime(), reqTime, time.Now(), "unexpected response Timestamp, expected it to be within the request time and now")
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

	initiateCancellationResp, err := initiatorBot.CancellationServiceV2.InitiateCancellation(h.ctx, &cancellationv2.InitiateCancellationRequest{
		Header:       &typesv4.RequestHeader{BaseHeader: &typesv4.Header{Version: &typesv4.Version{}}},
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

func (h *cancellationV2Helper) counterCancellation(
	counterer SupplierOrDistributor,
	refundAmount *typesv4.Price,
	reason cancellationv1.CounterReason,
) {
	refundAmountBig, err := price.ToBigInt(refundAmount.Value, conversion.MustUInt32ToInt32(refundAmount.Decimals), price.NativeTokenDecimals)
	h.require.NoError(err)

	countererBot := h.getBot(counterer)

	counterCancellationResp, err := countererBot.CancellationServiceV2.CounterCancellation(h.ctx, &cancellationv2.CounterCancellationRequest{
		Header:       &typesv4.RequestHeader{BaseHeader: &typesv4.Header{Version: &typesv4.Version{}}},
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

func (h *cancellationV2Helper) acceptCancellation(
	accepter SupplierOrDistributor,
	refundAmount *typesv4.Price,
) {
	refundAmountBig, err := price.ToBigInt(refundAmount.Value, conversion.MustUInt32ToInt32(refundAmount.Decimals), price.NativeTokenDecimals)
	h.require.NoError(err)

	accepterBot := h.getBot(accepter)

	acceptCancellationResp, err := accepterBot.CancellationServiceV2.AcceptCancellation(h.ctx, &cancellationv2.AcceptCancellationRequest{
		Header:       &typesv4.RequestHeader{BaseHeader: &typesv4.Header{Version: &typesv4.Version{}}},
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

func (h *cancellationV2Helper) rejectCancellation(
	rejecter SupplierOrDistributor,
	reason cancellationv1.RejectionReason,
) {
	rejecterBot := h.getBot(rejecter)

	rejectCancellationResp, err := rejecterBot.CancellationServiceV2.RejectCancellation(h.ctx, &cancellationv2.RejectCancellationRequest{
		Header:  &typesv4.RequestHeader{BaseHeader: &typesv4.Header{Version: &typesv4.Version{}}},
		TokenId: h.tokenID,
		Reason:  reason,
	})
	h.require.NoError(err)

	h.timesRejected++
	h.rejectionReason = reason

	h.expectCancellationRejectedNotification(h.distributorPPEventStream, rejectCancellationResp.TransactionId.Hash)
	h.expectCancellationRejectedNotification(h.supplierPPEventStream, rejectCancellationResp.TransactionId.Hash)
}

func (h *cancellationV2Helper) withdrawCancellation(
	withdrawer SupplierOrDistributor,
	reason cancellationv1.WithdrawalReason,
) {
	withdrawerBot := h.getBot(withdrawer)

	withdrawCancellationResp, err := withdrawerBot.CancellationServiceV2.WithdrawCancellation(h.ctx, &cancellationv2.WithdrawCancellationRequest{
		Header:  &typesv4.RequestHeader{BaseHeader: &typesv4.Header{Version: &typesv4.Version{}}},
		TokenId: h.tokenID,
		Reason:  reason,
	})
	h.require.NoError(err)

	h.withdrawalReason = reason

	h.expectCancellationWithdrawnNotification(h.distributorPPEventStream, withdrawCancellationResp.TransactionId.Hash)
	h.expectCancellationWithdrawnNotification(h.supplierPPEventStream, withdrawCancellationResp.TransactionId.Hash)
}

func (h *cancellationV2Helper) finalizeCancellation(refundAmount *typesv4.Price) {
	refundAmountBig, err := price.ToBigInt(refundAmount.Value, conversion.MustUInt32ToInt32(refundAmount.Decimals), price.NativeTokenDecimals)
	h.require.NoError(err)

	supplierBalance, err := h.e.CaminoNetwork.Client.BalanceOf(h.ctx, h.supplierBot.CMAccountAddress())
	h.require.NoError(err)
	distributorBalance, err := h.e.CaminoNetwork.Client.BalanceOf(h.ctx, h.distributorBot.CMAccountAddress())
	h.require.NoError(err)

	expectedSupplierBalance := big.NewInt(0).Sub(supplierBalance, refundAmountBig)
	expectedDistributorBalance := big.NewInt(0).Add(distributorBalance, refundAmountBig)

	finalizeCancellationResp, err := h.supplierBot.CancellationServiceV2.FinalizeCancellation(h.ctx, &cancellationv2.FinalizeCancellationRequest{
		Header:  &typesv4.RequestHeader{BaseHeader: &typesv4.Header{Version: &typesv4.Version{}}},
		TokenId: h.tokenID,
		RefundAmount: &typesv4.Price{
			Value:    refundAmount.Value,
			Decimals: refundAmount.Decimals,
			Currency: &typesv4.Currency{Currency: &typesv4.Currency_NativeToken{}},
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

func (h *cancellationV2Helper) expectCancellationRejectedNotification(
	ppEventsStream events.EventsService_SubscribeClient,
	txHash string,
) {
	eventMsg, err := ppEventsStream.Recv()
	h.require.NoError(err)
	h.e.DebugPrintProtoMessage(eventMsg)
	cancellationRejectedNotification := &notificationv3.CancellationRejected{}
	h.require.NoError(proto.Unmarshal(eventMsg.Data, cancellationRejectedNotification))
	h.require.Equal(h.tokenID, cancellationRejectedNotification.TokenId)
	h.require.Equal(h.rejectionReason, cancellationRejectedNotification.Reason)
	h.require.Equal(txHash, cancellationRejectedNotification.TxId.Hash)
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
	h.require.Equal(h.tokenID, cancellationWithdrawnNotification.TokenId)
	h.require.Equal(h.withdrawalReason, cancellationWithdrawnNotification.Reason)
	h.require.Equal(txHash, cancellationWithdrawnNotification.TxId.Hash)
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
	h.require.Equal(h.tokenID, cancellationFinalizedNotification.TokenId)
	h.require.Equal(txHash, cancellationFinalizedNotification.TxId.Hash)
}
