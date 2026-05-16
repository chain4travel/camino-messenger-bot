// Copyright (C) 2022-2026, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package chequehandler

import (
	"math/big"

	"github.com/chain4travel/camino-messenger-bot/v13/pkg/cheques"
	"github.com/ethereum/go-ethereum/common"
)

type IssuedChequeRecord struct {
	ChequeRecordID common.Hash
	Counter        *big.Int
	Amount         *big.Int
}

func IssuedChequeRecordCheque(chequeRecordID common.Hash, cheque *cheques.SignedCheque) *IssuedChequeRecord {
	return &IssuedChequeRecord{
		ChequeRecordID: chequeRecordID,
		Counter:        cheque.Counter,
		Amount:         cheque.Amount,
	}
}
