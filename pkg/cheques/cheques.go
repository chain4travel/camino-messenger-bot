package cheques

import (
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
)

type Cheque struct {
	FromCMAccount common.Address `json:"fromCMAccount"`
	ToCMAccount   common.Address `json:"fromCMAccount"`
	ToBot         common.Address `json:"fromCMAccount"`
	Counter       *big.Int
	Amount        *big.Int
	CreatedAt     time.Time
	ExpiresAt     time.Time
}
