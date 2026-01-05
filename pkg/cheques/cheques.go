// Copyright (C) 2022-2026, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package cheques

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

var (
	ErrChequeAlreadyExpired                = errors.New("cheque already expired")
	ErrChequeExpiresTooSoon                = errors.New("cheque expires too soon")
	ErrChequeAmountLessThanPrevious        = errors.New("new cheque amount less than previous cheque amount")
	ErrChequeCounterNotGreaterThanPrevious = errors.New("new cheque counter not greater than previous cheque counter")
)

type SignedCheque struct {
	Cheque
	Signature []byte
}

type Cheque struct {
	FromCMAccount common.Address
	ToCMAccount   common.Address
	ToBot         common.Address
	Counter       *big.Int
	Amount        *big.Int
	CreatedAt     *big.Int
	ExpiresAt     *big.Int
	PaymentToken  common.Address
}

type signedChequeJSON struct {
	FromCMAccount string `json:"fromCMAccount"`
	ToCMAccount   string `json:"toCMAccount"`
	ToBot         string `json:"toBot"`
	Counter       string `json:"counter"`
	Amount        string `json:"amount"`
	CreatedAt     string `json:"createdAt"`
	ExpiresAt     string `json:"expiresAt"`
	PaymentToken  string `json:"paymentToken"`
	Signature     string `json:"signature"`
}

func (sc SignedCheque) MarshalJSON() ([]byte, error) {
	return json.Marshal(&signedChequeJSON{
		FromCMAccount: sc.FromCMAccount.Hex(),
		ToCMAccount:   sc.ToCMAccount.Hex(),
		ToBot:         sc.ToBot.Hex(),
		Counter:       hexutil.EncodeBig(sc.Counter),
		Amount:        hexutil.EncodeBig(sc.Amount),
		CreatedAt:     hexutil.EncodeBig(sc.CreatedAt),
		ExpiresAt:     hexutil.EncodeBig(sc.ExpiresAt),
		PaymentToken:  sc.PaymentToken.Hex(),
		Signature:     hex.EncodeToString(sc.Signature),
	})
}

func (sc *SignedCheque) UnmarshalJSON(data []byte) error {
	var raw signedChequeJSON

	err := json.Unmarshal(data, &raw)
	if err != nil {
		return err
	}

	sc.Counter, err = hexutil.DecodeBig(raw.Counter)
	if err != nil {
		return err
	}

	sc.Amount, err = hexutil.DecodeBig(raw.Amount)
	if err != nil {
		return err
	}

	sc.CreatedAt, err = hexutil.DecodeBig(raw.CreatedAt)
	if err != nil {
		return err
	}

	sc.ExpiresAt, err = hexutil.DecodeBig(raw.ExpiresAt)
	if err != nil {
		return err
	}

	sc.Signature, err = hex.DecodeString(raw.Signature)
	if err != nil {
		return fmt.Errorf("invalid signature hex string: %w", err)
	}

	sc.FromCMAccount = common.HexToAddress(raw.FromCMAccount)
	sc.ToCMAccount = common.HexToAddress(raw.ToCMAccount)
	sc.ToBot = common.HexToAddress(raw.ToBot)
	sc.PaymentToken = common.HexToAddress(raw.PaymentToken)

	return nil
}

func VerifyCheque(previousCheque, newCheque *SignedCheque, timestamp, minDurationUntilExpiration *big.Int) error {
	durationUntilExpiration := big.NewInt(0).Sub(newCheque.ExpiresAt, timestamp)

	switch {
	case newCheque.ExpiresAt.Cmp(timestamp) < 1: // cheque.ExpiresAt <= timestamp
		return fmt.Errorf("cheque expired at %s; %w", newCheque.ExpiresAt, ErrChequeAlreadyExpired)
	case durationUntilExpiration.Cmp(minDurationUntilExpiration) < 0: // durationUntilExpiration < minDurationUntilExpiration
		return fmt.Errorf("duration until expiration less than min (%s < %s): %w",
			durationUntilExpiration, minDurationUntilExpiration, ErrChequeExpiresTooSoon)
	case newCheque.FromCMAccount == newCheque.ToCMAccount:
		return errors.New("cheque FromCMAccount and ToCMAccount are the same")
	case previousCheque == nil:
		return nil
	case previousCheque.PaymentToken != newCheque.PaymentToken:
		return fmt.Errorf("mismatched payment token: previous %s vs new %s",
			previousCheque.PaymentToken, newCheque.PaymentToken)
	case previousCheque.Amount.Cmp(newCheque.Amount) > 0: // previous.Amount > new.Amount
		return fmt.Errorf("new cheque amount (%s) < (%s) previous cheque amount: %w",
			newCheque.Amount, previousCheque.Amount, ErrChequeAmountLessThanPrevious)
	case previousCheque.Counter.Cmp(newCheque.Counter) > -1: // previous.Counter >= new.Counter
		return fmt.Errorf("new cheque counter (%s) <= (%s) previous cheque counter: %w",
			newCheque.Counter, previousCheque.Counter, ErrChequeCounterNotGreaterThanPrevious)
	}
	return nil
}
