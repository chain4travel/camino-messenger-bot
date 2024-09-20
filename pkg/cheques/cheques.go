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
	Cheque    `json:"cheque"`
	Signature []byte `json:"signature"`
}

type Cheque struct {
	FromCMAccount common.Address `json:"fromCMAccount"`
	ToCMAccount   common.Address `json:"toCMAccount"`
	ToBot         common.Address `json:"toBot"`
	Counter       *big.Int       `json:"counter"`
	Amount        *big.Int       `json:"amount"`
	CreatedAt     *big.Int       `json:"createdAt"`
	ExpiresAt     *big.Int       `json:"expiresAt"`
}

type signedChequeJSON struct {
	Cheque    chequeJSON `json:"cheque"`
	Signature string     `json:"signature"`
}

type chequeJSON struct {
	FromCMAccount string `json:"fromCMAccount"`
	ToCMAccount   string `json:"toCMAccount"`
	ToBot         string `json:"toBot"`
	Counter       string `json:"counter"`
	Amount        string `json:"amount"`
	CreatedAt     uint64 `json:"createdAt"`
	ExpiresAt     uint64 `json:"expiresAt"`
}

func (sc *SignedCheque) MarshalJSON() ([]byte, error) {
	return json.Marshal(&signedChequeJSON{
		Cheque: chequeJSON{
			FromCMAccount: sc.Cheque.FromCMAccount.Hex(),
			ToCMAccount:   sc.Cheque.ToCMAccount.Hex(),
			ToBot:         sc.Cheque.ToBot.Hex(),
			Counter:       hexutil.EncodeBig(sc.Cheque.Counter),
			Amount:        hexutil.EncodeBig(sc.Cheque.Amount),
			CreatedAt:     sc.Cheque.CreatedAt.Uint64(),
			ExpiresAt:     sc.Cheque.ExpiresAt.Uint64(),
		},
		Signature: hex.EncodeToString(sc.Signature),
	})
}

func (sc *SignedCheque) UnmarshalJSON(data []byte) error {
	var raw signedChequeJSON
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	counter, err := hexutil.DecodeBig(raw.Cheque.Counter)
	if err != nil {
		return err
	}
	sc.Cheque.Counter = counter

	amount, err := hexutil.DecodeBig(raw.Cheque.Amount)
	if err != nil {
		return err
	}
	sc.Cheque.Amount = amount

	signatureBytes, err := hex.DecodeString(raw.Signature)
	if err != nil {
		return fmt.Errorf("invalid signature hex string: %w", err)
	}
	sc.Signature = signatureBytes

	sc.Cheque.FromCMAccount = common.HexToAddress(raw.Cheque.FromCMAccount)
	sc.Cheque.ToCMAccount = common.HexToAddress(raw.Cheque.ToCMAccount)
	sc.Cheque.ToBot = common.HexToAddress(raw.Cheque.ToBot)
	sc.Cheque.CreatedAt = big.NewInt(0).SetUint64(raw.Cheque.CreatedAt)
	sc.Cheque.ExpiresAt = big.NewInt(0).SetUint64(raw.Cheque.ExpiresAt)

	return nil
}

func VerifyCheque(previousCheque, newCheque *SignedCheque, timestamp, minDurationUntilExpiration *big.Int) error {
	if newCheque.ExpiresAt.Cmp(timestamp) < 1 { // cheque.ExpiresAt <= timestamp
		return fmt.Errorf("cheque expired at %s; %w", newCheque.ExpiresAt, ErrChequeAlreadyExpired)
	}
	durationUntilExpiration := big.NewInt(0).Sub(newCheque.ExpiresAt, timestamp)
	if durationUntilExpiration.Cmp(minDurationUntilExpiration) < 0 { // durationUntilExpiration < minDurationUntilExpiration
		return fmt.Errorf("duration until expiration less than min (%s < %s): %w", durationUntilExpiration, minDurationUntilExpiration, ErrChequeExpiresTooSoon)
	}
	switch {
	case previousCheque == nil:
		return nil
	case previousCheque.Amount.Cmp(newCheque.Amount) > 0: // previous.Amount > new.Amount
		return fmt.Errorf("new cheque amount (%s) < (%s) previous cheque amount: %w", newCheque.Amount, previousCheque.Amount, ErrChequeAmountLessThanPrevious)
	case previousCheque.Counter.Cmp(newCheque.Counter) > -1: // previous.Counter >= new.Counter
		return fmt.Errorf("new cheque counter (%s) <= (%s) previous cheque counter: %w", newCheque.Counter, previousCheque.Counter, ErrChequeCounterNotGreaterThanPrevious)
	}
	return nil
}
