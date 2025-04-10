// Copyright (C) 2022-2025, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package messaging

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"
)

var (
	errMissingPrice    = errors.New("missing price")
	errUnknownCurrency = errors.New("unknown currency type")
	errMissingMintTxID = errors.New("missing mint transaction id")
)

func (h *evmResponseHandler) subscribeForTokenBoughtEvent(tokenID *big.Int, mintID string, buyableUntil *timestamppb.Timestamp) {
	tokenBoughtTimeout := time.Unix(buyableUntil.Seconds, 0)
	if err := h.eventListener.SubscribeForTokenBoughtEvent(tokenID, mintID, tokenBoughtTimeout); err != nil {
		h.logger.Errorf("error subscribing for token bought event (tokenID: %d, mintID: %s, timeout: %d): %v",
			tokenID.Int64(), mintID, tokenBoughtTimeout, err)
	}
}

type hotelAttribute struct {
	TraitType string `json:"trait_type"`
	Value     string `json:"value"`
}

type hotelJSON struct {
	Name        string           `json:"name"`
	Description string           `json:"description,omitempty"`
	Date        string           `json:"date,omitempty"`
	ExternalURL string           `json:"external_url,omitempty"`
	Image       string           `json:"image,omitempty"`
	Attributes  []hotelAttribute `json:"attributes,omitempty"`
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

	attributes := []hotelAttribute{
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

func generateAndEncodeJSON(name, description, date, externalURL, image string, attributes []hotelAttribute) (string, string, error) {
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
