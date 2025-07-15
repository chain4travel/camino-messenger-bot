// Copyright (C) 2022-2025, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package tests

import (
	"context"
	"math/big"
	"testing"

	bookv2 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/book/v2"
	bookv3 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/book/v3"
	notificationv2 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/notification/v2"
	typesv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v1"
	typesv2 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v2"
	typesv3 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v3"
	"github.com/chain4travel/camino-messenger-bot/v11/pkg/booking"
	"github.com/chain4travel/camino-messenger-bot/v11/pkg/price"
	"github.com/chain4travel/camino-messenger-bot/v11/pp-mock/proto/pb/events"
	"github.com/chain4travel/camino-messenger-bot/v11/tests/e2e/bot"
	"github.com/chain4travel/camino-messenger-bot/v11/tests/e2e/suite"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

func mintBuyTokenV3(
	ctx context.Context,
	t *testing.T,
	e *suite.Environment,
	supplierPPEventStream events.EventsService_SubscribeClient,
	distributorBot *bot.Bot,
	supplierBot *bot.Bot,
) (uint64, *typesv3.Price) {
	searchID, resultID, totalPrice := testAccommodationV3SearchServiceWithTravelPeriod(ctx, t, e, distributorBot, supplierBot) // see test_accommodation_v3.go
	_, err := supplierPPEventStream.Recv()                                                                                     // skip AccommodationSearchRequest
	require.NoError(t, err)

	validationID := testValidateV2(ctx, t, e, distributorBot, supplierBot, searchID, resultID, totalPrice)
	_, err = supplierPPEventStream.Recv() // skip ValidateRequest
	require.NoError(t, err)

	tokenID, bookingPrice, _ := testMintV3(ctx, t, e, distributorBot, supplierBot, validationID)
	_, err = supplierPPEventStream.Recv() // skip MintRequest
	require.NoError(t, err)

	eventMsg, err := supplierPPEventStream.Recv()
	require.NoError(t, err)
	e.DebugPrintProtoMessage(eventMsg)
	tokenBoughtNotification := &notificationv2.TokenBought{}
	require.NoError(t, proto.Unmarshal(eventMsg.Data, tokenBoughtNotification))

	return tokenID, bookingPrice
}

// validate

func testValidateV2(
	ctx context.Context,
	t *testing.T,
	e *suite.Environment,
	distributorBot *bot.Bot,
	supplierBot *bot.Bot,
	searchID string,
	resultID int32,
	expectedTotalPrice *big.Int,
) (validateID string) {
	req := &bookv2.ValidationRequest{
		ValidationObject: &bookv2.ValidationObject{
			SearchIdentifier: &typesv2.SearchIdentifier{
				SearchId: &typesv1.UUID{Value: searchID},
				ResultId: resultID,
			},
		},
	}
	resp, err := distributorBot.ValidationServiceV2.Validation(
		requestContext(ctx, supplierBot.CMAccountAddress()),
		req,
	)
	require.NoError(t, err)
	e.DebugPrintRequestResponse(req, resp)

	require.Equal(t, typesv1.StatusType_STATUS_TYPE_SUCCESS, resp.Header.Status, "unexpected response status")
	require.Empty(t, resp.Header.Alerts, "unexpected response alerts")

	// Check if the validationObject is correct in the response
	require.NotEmpty(t, resp.ValidationObject, "unexpected empty response ValidationObject")
	require.NotEmpty(t, resp.ValidationObject.SearchIdentifier, "unexpected empty response ValidationObject.SearchIdentifier")
	require.NotEmpty(t, resp.ValidationObject.SearchIdentifier.SearchId, "unexpected empty response ValidationObject.SearchIdentifier.SearchId")
	require.NotEmpty(t, resp.ValidationObject.SearchIdentifier.SearchId.Value, "unexpected empty response ValidationObject.SearchIdentifier.SearchId.Value")
	require.Equal(t, searchID, resp.ValidationObject.SearchIdentifier.SearchId.Value, "unexpected searchID in response")
	require.Equal(t, resultID, resp.ValidationObject.SearchIdentifier.ResultId, "unexpected resultID in response")

	// Check if the price per night is as expected
	require.NotEmpty(t, resp.PriceDetail, "unexpected empty response PriceDetail")
	require.NotEmpty(t, resp.PriceDetail.Price, "unexpected empty response PriceDetail.Price")
	require.NotEmpty(t, resp.PriceDetail.Price.Value, "unexpected empty response PriceDetail.Price.Value")

	totalPrice := priceBigV2(t, resp.PriceDetail.Price)
	require.True(t, totalPrice.Cmp(expectedTotalPrice) == 0, "unexpected total price")

	// Last check if the validationID is set and if yes extract it and pass it back for the mint step
	require.NotEmpty(t, resp.ValidationId, "unexpected empty response validationID")
	require.NotEmpty(t, resp.ValidationId.Value, "unexpected empty response validationID.Value")
	return resp.ValidationId.Value
}

// mint

func testMintV2(
	ctx context.Context,
	t *testing.T,
	e *suite.Environment,
	distributorBot *bot.Bot,
	supplierBot *bot.Bot,
	validationID string,
) (
	tokenID uint64,
	price *typesv2.Price,
	mintID string,
) {
	req := &bookv2.MintRequest{
		Header:       &typesv1.RequestHeader{BaseHeader: &typesv1.Header{}},
		ValidationId: &typesv1.UUID{Value: validationID},
	}
	resp, err := distributorBot.MintServiceV2.Mint(
		requestContext(ctx, supplierBot.CMAccountAddress()),
		req,
	)
	require.NoError(t, err)
	e.DebugPrintRequestResponse(req, resp)

	require.Equal(t, typesv1.StatusType_STATUS_TYPE_SUCCESS, resp.Header.Status, "unexpected response status")

	// Check if the MintId is set
	require.NotEmpty(t, resp.MintId, "unexpected empty response MintId")
	require.NotEmpty(t, resp.MintId.Value, "unexpected empty response MintId.Value")

	// check if the transaction ids are set and return them for further tests
	require.NotEmpty(t, resp.MintTransactionId, "unexpected empty response MintTransactionId")
	require.NotEmpty(t, resp.BuyTransactionId, "unexpected empty response BuyTransactionId")

	return resp.BookingTokenId, resp.Price, resp.MintId.Value
}

func testMintV3(
	ctx context.Context,
	t *testing.T,
	e *suite.Environment,
	distributorBot *bot.Bot,
	supplierBot *bot.Bot,
	validationID string,
) (
	tokenID uint64,
	price *typesv3.Price,
	mintID string,
) {
	req := &bookv3.MintRequest{
		Header:       &typesv1.RequestHeader{BaseHeader: &typesv1.Header{}},
		ValidationId: &typesv1.UUID{Value: validationID},
	}
	resp, err := distributorBot.MintServiceV3.Mint(
		requestContext(ctx, supplierBot.CMAccountAddress()),
		req,
	)
	require.NoError(t, err)
	e.DebugPrintRequestResponse(req, resp)

	require.Equal(t, typesv1.StatusType_STATUS_TYPE_SUCCESS, resp.Header.Status, "unexpected response status")

	// Check if the MintId is set
	require.NotEmpty(t, resp.MintId, "unexpected empty response MintId")
	require.NotEmpty(t, resp.MintId.Value, "unexpected empty response MintId.Value")

	// check if the transaction ids are set and return them for further tests
	require.NotEmpty(t, resp.MintTransactionId, "unexpected empty response MintTransactionId")
	require.NotEmpty(t, resp.BuyTransactionId, "unexpected empty response BuyTransactionId")

	return resp.BookingTokenId, resp.Price, resp.MintId.Value
}

// verify blockchain state

func verifyBookingTokenStateWithPriceV2(
	ctx context.Context,
	t *testing.T,
	e *suite.Environment,
	distributorBot *bot.Bot,
	tokenID uint64,
	tokenPrice *typesv2.Price,
) {
	require.Equal(t, booking.NativePaymentToken, getPaymentTokenFromPriceV2(t, tokenPrice))
	expectedReservationPrice, err := price.ToBigInt(tokenPrice.Value, tokenPrice.Decimals, price.NativeTokenDecimals)
	require.NoError(t, err)
	verifyBookingTokenState(ctx, t, e, distributorBot, tokenID, expectedReservationPrice)
}

func verifyBookingTokenState(
	ctx context.Context,
	t *testing.T,
	e *suite.Environment,
	distributorBot *bot.Bot,
	tokenID uint64,
	expectedReservationPrice *big.Int,
) {
	bigTokenID := big.NewInt(0).SetUint64(tokenID)
	callOpts := &bind.CallOpts{Context: ctx}

	reservationPrice, err := e.CaminoNetwork.Client.BookingToken.GetReservationPrice(callOpts, bigTokenID)
	require.NoError(t, err)
	require.Equal(t, booking.NativePaymentToken, reservationPrice.PaymentToken)
	require.Equal(t, expectedReservationPrice, reservationPrice.Price)

	ownerAddr, err := e.CaminoNetwork.Client.BookingToken.OwnerOf(callOpts, bigTokenID)
	require.NoError(t, err)
	require.Equal(t, distributorBot.CMAccountAddress(), ownerAddr)

	tokenStatus, err := e.CaminoNetwork.Client.BookingToken.GetBookingStatus(callOpts, bigTokenID)
	require.NoError(t, err)
	require.Equal(t, booking.StatusBought, booking.Status(tokenStatus))
}
