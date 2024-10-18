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

func (h *evmResponseHandler) handleMintResponseV2(ctx context.Context, response protoreflect.ProtoMessage, request protoreflect.ProtoMessage) bool {
	mintResp, ok := response.(*bookv2.MintResponse)
	if !ok {
		return false
	}
	mintReq, ok := request.(*bookv2.MintRequest)
	if !ok {
		return false
	}
	if mintResp.Header == nil {
		mintResp.Header = &typesv1.ResponseHeader{}
	}

	// TODO: @VjeraTurk check if CMAccount exists

	// TODO @evlekht ensure that mintReq.BuyerAddress is c-chain address format,
	// TODO not x/p/t chain or anything else. Currently it will not error
	// TODO if address is invalid and will just get zero addr
	buyerAddress := common.HexToAddress(mintReq.BuyerAddress)

	if mintResp.BookingTokenUri == "" {
		jsonPlain, tokenURI, err := createTokenURIforMintResponse(
			mintResp.MintId.Value,
			mintReq.BookingReference,
		)
		if err != nil {
			errMsg := fmt.Sprintf("Failed to mint token: failed to generate tokenURI:  %s", err)
			h.logger.Error(errMsg)
			h.AddErrorToResponseHeader(response, errMsg)
			return true
		}
		h.logger.Debugf("Token URI JSON: %s\n", jsonPlain)
		mintResp.BookingTokenUri = tokenURI
	}
	h.logger.Debugf("Token URI: %s\n", mintResp.BookingTokenUri)

	buyableUntil, err := verifyAndFixBuyableUntil(mintResp.BuyableUntil, time.Now())
	if err != nil {
		h.logger.Error(err)
		h.AddErrorToResponseHeader(response, err.Error())
		return true
	}
	mintResp.BuyableUntil = buyableUntil

	price, paymentToken, err := h.getPriceAndTokenV2(ctx, mintResp.Price)
	if err != nil {
		errMessage := fmt.Sprintf("error getting price and payment token: %v", err)
		h.logger.Errorf(errMessage)
		h.AddErrorToResponseHeader(response, errMessage)
		return true
	}

	// MINT TOKEN
	txID, tokenID, err := h.mint(
		ctx,
		buyerAddress,
		mintResp.BookingTokenUri,
		big.NewInt(mintResp.BuyableUntil.Seconds),
		price,
		paymentToken,
	)
	if err != nil {
		errMessage := fmt.Sprintf("error minting NFT: %v", err)
		h.logger.Errorf(errMessage)
		h.AddErrorToResponseHeader(response, errMessage)
		return true
	}

	h.logger.Infof("NFT minted with txID: %s\n", txID)

	h.onBookingTokenMint(tokenID, mintResp.MintId, mintResp.BuyableUntil.AsTime())

	mintResp.Header.Status = typesv1.StatusType_STATUS_TYPE_SUCCESS
	mintResp.BookingTokenId = tokenID.Uint64()
	mintResp.MintTransactionId = txID
	return false
}

func (h *evmResponseHandler) handleMintRequestV2(ctx context.Context, response protoreflect.ProtoMessage) bool {
	mintResp, ok := response.(*bookv2.MintResponse)
	if !ok {
		return false
	}
	if mintResp.Header == nil {
		mintResp.Header = &typesv1.ResponseHeader{}
	}
	if mintResp.MintTransactionId == "" {
		h.AddErrorToResponseHeader(response, "missing mint transaction id")
		return true
	}

	tokenID := new(big.Int).SetUint64(mintResp.BookingTokenId)

	receipt, err := h.bookingService.BuyBookingToken(ctx, tokenID)
	if err != nil {
		errMessage := fmt.Sprintf("error buying NFT: %v", err)
		h.logger.Errorf(errMessage)
		h.AddErrorToResponseHeader(response, errMessage)
		return true
	}

	h.logger.Infof("Bought NFT (txID=%s) with ID: %s\n", receipt, mintResp.MintTransactionId)
	mintResp.BuyTransactionId = receipt.TxHash.Hex()
	return false
}

func (h *evmResponseHandler) getPriceAndTokenV2(ctx context.Context, price *typesv2.Price) (*big.Int, common.Address, error) {
	priceBigInt := big.NewInt(0)
	paymentToken := zeroAddress
	var err error
	switch currency := price.Currency.Currency.(type) {
	case *typesv2.Currency_NativeToken:
		priceBigInt, err = h.bookingService.ConvertPriceToBigInt(price.Value, price.Decimals, int32(18)) // CAM uses 18 decimals
		if err != nil {
			return nil, zeroAddress, fmt.Errorf("error minting NFT: %w", err)
		}
	case *typesv2.Currency_TokenCurrency:
		if !common.IsHexAddress(currency.TokenCurrency.ContractAddress) {
			return nil, zeroAddress, fmt.Errorf("invalid contract address: %s", currency.TokenCurrency.ContractAddress)
		}
		contractAddress := common.HexToAddress(currency.TokenCurrency.ContractAddress)
		tokenDecimals, err := h.erc20.Decimals(ctx, contractAddress)
		if err != nil {
			return nil, zeroAddress, fmt.Errorf("failed to fetch token decimals: %w", err)
		}
		priceBigInt, err = h.bookingService.ConvertPriceToBigInt(price.Value, price.Decimals, tokenDecimals)
		if err != nil {
			return nil, zeroAddress, err
		}
		paymentToken = contractAddress
	case *typesv2.Currency_IsoCurrency:
		// For IsoCurrency, keep price as 0 and paymentToken as zeroAddress
	}
	return priceBigInt, paymentToken, nil
}
