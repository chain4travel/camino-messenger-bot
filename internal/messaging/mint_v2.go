package messaging

import (
	"context"
	"fmt"
	"math/big"
	"time"

	bookv2 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/book/v2"
	typesv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v1"
	typesv2 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v2"
	"github.com/ethereum/go-ethereum/common"
	"google.golang.org/protobuf/reflect/protoreflect"
)

func (h *evmResponseHandler) prepareMintResponseV2(
	ctx context.Context,
	response *bookv2.MintResponse,
	requestIntf protoreflect.ProtoMessage,
) {
	request, ok := requestIntf.(*bookv2.MintRequest)
	if !ok {
		err := fmt.Errorf("%w: expected *bookv2.MintRequest, got %T", errUnexpectedRequestType, requestIntf)
		h.logger.Error(err)
		h.AddErrorToResponseHeader(response, err.Error())
		return
	}

	ensureHeaderV1(&response.Header)

	// TODO: @VjeraTurk check if CMAccount exists

	// TODO @evlekht ensure that mintReq.BuyerAddress is c-chain address format,
	// TODO not x/p/t chain or anything else. Currently it will not error
	// TODO if address is invalid and will just get zero addr
	buyerAddress := common.HexToAddress(request.BuyerAddress)

	if response.BookingTokenUri == "" {
		jsonPlain, tokenURI, err := createTokenURIforMintResponse(
			response.MintId.Value,
			request.BookingReference,
		)
		if err != nil {
			errMsg := fmt.Sprintf("Failed to mint token: failed to generate tokenURI:  %s", err)
			h.logger.Error(errMsg)
			h.AddErrorToResponseHeader(response, errMsg)
			return
		}
		h.logger.Debugf("Token URI JSON: %s\n", jsonPlain)
		response.BookingTokenUri = tokenURI
	}
	h.logger.Debugf("Token URI: %s\n", response.BookingTokenUri)

	buyableUntil, err := verifyAndFixBuyableUntil(response.BuyableUntil, time.Now())
	if err != nil {
		h.logger.Error(err)
		h.AddErrorToResponseHeader(response, err.Error())
		return
	}
	response.BuyableUntil = buyableUntil

	price, paymentToken, err := h.getPriceAndTokenV2(response.Price)
	if err != nil {
		errMessage := fmt.Sprintf("error minting NFT: %v", err)
		h.logger.Errorf(errMessage)
		h.AddErrorToResponseHeader(response, errMessage)
		return
	}

	// MINT TOKEN
	txID, tokenID, err := h.mint(
		ctx,
		buyerAddress,
		response.BookingTokenUri,
		big.NewInt(response.BuyableUntil.Seconds),
		price,
		paymentToken,
	)
	if err != nil {
		errMessage := fmt.Sprintf("error minting NFT: %v", err)
		h.logger.Errorf(errMessage)
		h.AddErrorToResponseHeader(response, errMessage)
		return
	}

	h.logger.Infof("NFT minted with txID: %s\n", txID)

	h.onBookingTokenMint(tokenID, response.MintId, response.BuyableUntil.AsTime())

	response.Header.Status = typesv1.StatusType_STATUS_TYPE_SUCCESS
	response.BookingTokenId = tokenID.Uint64()
	response.MintTransactionId = txID
}

func (h *evmResponseHandler) processMintResponseV2(ctx context.Context, responseIntf protoreflect.ProtoMessage) {
	response, ok := responseIntf.(*bookv2.MintResponse)
	if !ok {
		err := fmt.Errorf("%w: expected *bookv2.MintResponse, got %T", errUnexpectedResponseType, responseIntf)
		h.logger.Error(err)
		h.AddErrorToResponseHeader(response, err.Error())
		return
	}

	ensureHeaderV1(&response.Header)

	if response.MintTransactionId == "" {
		h.logger.Error(errMissingMintTxID)
		h.AddErrorToResponseHeader(response, errMissingMintTxID.Error())
		return
	}

	tokenID := new(big.Int).SetUint64(response.BookingTokenId)

	txID, err := h.buy(ctx, tokenID)
	if err != nil {
		errMessage := fmt.Sprintf("error buying NFT: %v", err)
		h.logger.Errorf(errMessage)
		h.AddErrorToResponseHeader(response, errMessage)
		return
	}

	h.logger.Infof("Bought NFT (txID=%s) with ID: %s\n", txID, response.MintTransactionId)
	response.BuyTransactionId = txID
}

func (h *evmResponseHandler) getPriceAndTokenV2(price *typesv2.Price) (*big.Int, common.Address, error) {
	priceBigInt := big.NewInt(0)
	paymentToken := zeroAddress
	switch price.Currency.Currency.(type) {
	case *typesv2.Currency_NativeToken:
		var err error
		priceBigInt, err = h.bookingService.ConvertPriceToBigInt(price.Value, price.Decimals, int32(18)) // CAM uses 18 decimals
		if err != nil {
			return nil, zeroAddress, fmt.Errorf("error minting NFT: %w", err)
		}
	case *typesv2.Currency_TokenCurrency:
		// Add logic to handle TokenCurrency
		// if contract address is zeroAddress, then it is native token
		return nil, zeroAddress, fmt.Errorf("TokenCurrency not supported yet")
	case *typesv2.Currency_IsoCurrency:
		// For IsoCurrency, keep price as 0 and paymentToken as zeroAddress
	}
	return priceBigInt, paymentToken, nil
}
