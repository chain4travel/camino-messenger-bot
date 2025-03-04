// Copyright (C) 2022-2025, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package messaging

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"time"

	notificationv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/notification/v1"
	typesv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v1"
	tokenstorage "github.com/chain4travel/camino-messenger-bot/pkg/tokens"
	"github.com/chain4travel/camino-messenger-contracts/go/contracts/bookingtoken"
	"github.com/ethereum/go-ethereum/common"
	"google.golang.org/grpc"
	grpc_metadata "google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var (
	errMissingPrice    = errors.New("missing price")
	errUnknownCurrency = errors.New("unknown currency type")
	errMissingMintTxID = errors.New("missing mint transaction id")
)

// Mints a BookingToken with the supplier private key and reserves it for the buyer address
// For testing you can use this uri: "data:application/json;base64,eyJuYW1lIjoiQ2FtaW5vIE1lc3NlbmdlciBCb29raW5nVG9rZW4gVGVzdCJ9Cg=="
func (h *evmResponseHandler) mint(
	ctx context.Context,
	reservedFor common.Address,
	uri string,
	expiration *big.Int,
	price *big.Int,
	paymentToken common.Address,
	offchainPaymentCurrency *big.Int,
) (string, *big.Int, error) {
	receipt, err := h.bookingService.MintBookingToken(
		ctx,
		reservedFor,
		uri,
		expiration,
		price,
		paymentToken,
		offchainPaymentCurrency,
	)
	if err != nil {
		return "", nil, err
	}

	tokenID := big.NewInt(0)

	for _, mLog := range receipt.Logs {
		event, err := h.bookingToken.ParseTokenReserved(*mLog)
		if err == nil {
			tokenID = event.TokenId
			h.logger.Infof("[TokenReserved] TokenID: %s ReservedFor: %s Price: %s, PaymentToken: %s", event.TokenId, event.ReservedFor, event.Price, event.PaymentToken)
		}
	}

	return receipt.TxHash.Hex(), tokenID, nil
}

func registerTokenListeners(h *evmResponseHandler, tokenID *big.Int, mintID *typesv1.UUID, buyableUntil time.Time) {
	notificationClient := h.serviceRegistry.NotificationClient()
	expirationTimer := &time.Timer{}

	unsubscribeTokenBought, err := h.evmEventListener.RegisterTokenBoughtHandler(
		h.bookingTokenAddress,
		[]*big.Int{tokenID},
		nil,
		func(e any) {
			expirationTimer.Stop()
			h.logger.Infof("Token bought event received for token %s", tokenID.String())
			event := e.(*bookingtoken.BookingtokenTokenBought)

			if _, err := notificationClient.TokenBoughtNotification(
				context.Background(),
				&notificationv1.TokenBought{
					TokenId: tokenID.Uint64(),
					TxId:    event.Raw.TxHash.Hex(),
					MintId:  mintID,
				},
				grpc.Header(&grpc_metadata.MD{}),
			); err != nil {
				h.logger.Errorf("error calling partner plugin TokenBoughtNotification service: %v", err)
			}

			// Mark the token as bought
			session, err := h.evmEventStorage.NewSession(context.Background())
			if err != nil {
				h.logger.Errorf("failed to create session: %v", err)
				return
			}

			err = h.evmEventStorage.UpdateTokenRecord(context.Background(), session, &tokenstorage.TokenRecord{
				TokenID: tokenID.String(),
				Bought:  true,
				Expired: false,
			})
			if err != nil {
				h.logger.Errorf("failed to update token record: %v", err)
				return
			}
		},
	)
	if err != nil {
		h.logger.Errorf("failed to register handler: %v", err)
		// TODO @evlekht send some notification to partner plugin
		return
	}

	// Adding 60 seconds to the buyableUntil time to avoid race condition between the token being bought and the token being expired
	timeToExpire := time.Until(buyableUntil.Add(60 * time.Second))

	expirationTimer = time.AfterFunc(timeToExpire, func() {
		unsubscribeTokenBought()
		h.logger.Infof("Token %s expired", tokenID.String())

		// check the status of the token -> if it is not bought, then we can just expire the token
		tx, err := h.bookingToken.GetBookingStatus(nil, tokenID)
		if err != nil {
			h.logger.Errorf("failed to get booking status: %v", err)
			return
		}

		// if the token is bought or canceled, then we don't need to expire the token
		if tx == uint8(BookingStatusBought) || tx == uint8(BookingStatusExpired) {
			return
		}

		// Mark the token as bought
		session, err := h.evmEventStorage.NewSession(context.Background())
		if err != nil {
			h.logger.Errorf("Failed to create session: %v", err)
			return
		}

		// check if config has "record_expiration" set to true
		// Update the token record to expired on chain
		if h.recordExpiration {
			_, err = h.bookingService.RecordExpiration(context.Background(), tokenID)
			if err != nil {
				h.logger.Errorf("failed to record expiration: %v", err)
			}
		}

		// update the token record to expired on chain
		err = h.evmEventStorage.UpdateTokenRecord(context.Background(), session, &tokenstorage.TokenRecord{
			TokenID: tokenID.String(),
			Bought:  false,
			Expired: true,
		})

		if _, err := notificationClient.TokenExpiredNotification(
			context.Background(),
			&notificationv1.TokenExpired{
				TokenId: tokenID.Uint64(),
				MintId:  mintID,
			},
			grpc.Header(&grpc_metadata.MD{}),
		); err != nil {
			h.logger.Errorf("error calling partner plugin TokenExpiredNotification service: %v", err)
		}
	})
}

func (h *evmResponseHandler) onBookingTokenMint(tokenID *big.Int, mintID *typesv1.UUID, buyableUntil time.Time) {
	registerTokenListeners(h, tokenID, mintID, buyableUntil)

	session, err := h.evmEventStorage.NewSession(context.Background())
	if err != nil {
		h.logger.Errorf("failed to create session: %v", err)
		return
	}

	err = h.evmEventStorage.SaveTokenRecord(context.Background(), session, &tokenstorage.TokenRecord{
		TokenID:   tokenID.String(),
		Bought:    false,
		Expired:   false,
		MintID:    mintID,
		CreatedAt: big.NewInt(time.Now().Unix()).Bytes(),
		ExpiresAt: big.NewInt(buyableUntil.Unix()).Bytes(),
	})
	if err != nil {
		h.logger.Errorf("Failed to save token record: %v", err)
		return
	}
}

// TODO @evlekht check if those structs are needed as exported here, otherwise make them private or move to another pkg
type hotelAtrribute struct {
	TraitType string `json:"trait_type"`
	Value     string `json:"value"`
}

type hotelJSON struct {
	Name        string           `json:"name"`
	Description string           `json:"description,omitempty"`
	Date        string           `json:"date,omitempty"`
	ExternalURL string           `json:"external_url,omitempty"`
	Image       string           `json:"image,omitempty"`
	Attributes  []hotelAtrribute `json:"attributes,omitempty"`
}

// Generates a token data URI from a MintResponse object. Returns jsonPlain and a
// data URI with base64 encoded json data.
//
// TODO: @havan: We need decide what data needs to be in the tokenURI JSON and add
// those fields to the MintResponse. These will be shown in the UI of wallets,
// explorers etc.
func createTokenURIforMintResponse(mintID, bookingReference string) (string, string, error) {
	// TODO: What should we use for a token name? This will be shown in the UI of wallets, explorers etc.
	name := "CM Booking Token"

	// TODO: What should we use for a token description? This will be shown in the UI of wallets, explorers etc.
	description := "This NFT represents the booking with the specified attributes."

	// Dummy data
	date := "2024-09-27"

	externalURL := "https://camino.network"

	// Placeholder Image
	image := "https://camino.network/static/images/N9IkxmG-Sg-1800.webp"

	attributes := []hotelAtrribute{
		{
			TraitType: "Mint ID",
			Value:     mintID,
		},
		{
			TraitType: "Reference",
			Value:     bookingReference,
		},
	}

	jsonPlain, jsonEncoded, err := generateAndEncodeJSON(
		name,
		description,
		date,
		externalURL,
		image,
		attributes,
	)
	if err != nil {
		return "", "", err
	}

	// Add data URI scheme
	tokenURI := "data:application/json;base64," + jsonEncoded

	return jsonPlain, tokenURI, nil
}

func generateAndEncodeJSON(name, description, date, externalURL, image string, attributes []hotelAtrribute) (string, string, error) {
	hotel := hotelJSON{
		Name:        name,
		Description: description,
		Date:        date,
		ExternalURL: externalURL,
		Image:       image,
		Attributes:  attributes,
	}

	jsonData, err := json.Marshal(hotel)
	if err != nil {
		return "", "", err
	}

	encoded := base64.StdEncoding.EncodeToString(jsonData)
	return string(jsonData), encoded, nil
}

func verifyAndFixBuyableUntil(buyableUntil *timestamppb.Timestamp, currentTime time.Time) (*timestamppb.Timestamp, error) {
	switch {
	case buyableUntil == nil || buyableUntil.Seconds == 0:
		// BuyableUntil not set
		return timestamppb.New(currentTime.Add(buyableUntilDurationDefault)), nil

	case buyableUntil.Seconds < timestamppb.New(currentTime).Seconds:
		// BuyableUntil in the past
		return nil, fmt.Errorf("refused to mint token - BuyableUntil in the past:  %v", buyableUntil)

	case buyableUntil.Seconds < timestamppb.New(currentTime.Add(buyableUntilDurationMinimal)).Seconds:
		// BuyableUntil too early
		return timestamppb.New(currentTime.Add(buyableUntilDurationMinimal)), nil

	case buyableUntil.Seconds > timestamppb.New(currentTime.Add(buyableUntilDurationMaximal)).Seconds:
		// BuyableUntil too late
		return timestamppb.New(currentTime.Add(buyableUntilDurationMaximal)), nil
	}

	return buyableUntil, nil
}
