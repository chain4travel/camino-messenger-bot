<<<<<<<< HEAD:pkg/chequehandler/issued_cheque_record.go
package chequehandler
========
package cheque_handler
>>>>>>>> a1cdaeb (wip):pkg/cheque_handler/issued_cheque_record.go

import (
	"math/big"

	"github.com/chain4travel/camino-messenger-bot/pkg/cheques"
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
