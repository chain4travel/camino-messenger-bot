<<<<<<<< HEAD:pkg/chequehandler/cheque_record.go
package chequehandler
========
package cheque_handler
>>>>>>>> a1cdaeb (wip):pkg/cheque_handler/cheque_record.go

import (
	"fmt"

	"github.com/chain4travel/camino-messenger-bot/pkg/cheques"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
)

type ChequeTxStatus uint8

const (
	ChequeTxStatusUnknown ChequeTxStatus = iota
	ChequeTxStatusPending
	ChequeTxStatusAccepted
	ChequeTxStatusRejected
)

func (c ChequeTxStatus) String() string {
	switch c {
	case ChequeTxStatusUnknown:
		return "unknown"
	case ChequeTxStatusPending:
		return "pending"
	case ChequeTxStatusAccepted:
		return "accepted"
	case ChequeTxStatusRejected:
		return "rejected"
	default:
		return fmt.Sprintf("unknown status: %d", c)
	}
}

func ChequeTxStatusFromTxStatus(txStatus uint64) ChequeTxStatus {
	if txStatus == types.ReceiptStatusSuccessful {
		return ChequeTxStatusAccepted
	}
	return ChequeTxStatusRejected
}

type ChequeRecord struct {
	cheques.SignedCheque
	ChequeRecordID common.Hash
	TxID           common.Hash
	Status         ChequeTxStatus
}

func (c ChequeRecord) String() string {
	return fmt.Sprintf("{ID: %s, txID %s, status: %s, cheque: %+v}", c.ChequeRecordID.Hex(), c.TxID.Hex(), c.Status, c.Cheque)
}

func ChequeRecordFromCheque(chequeRecordID common.Hash, cheque *cheques.SignedCheque) *ChequeRecord {
	return &ChequeRecord{
		SignedCheque: cheques.SignedCheque{
			Cheque: cheques.Cheque{
				FromCMAccount: cheque.FromCMAccount,
				ToCMAccount:   cheque.ToCMAccount,
				ToBot:         cheque.ToBot,
				Counter:       cheque.Counter,
				Amount:        cheque.Amount,
				CreatedAt:     cheque.CreatedAt,
				ExpiresAt:     cheque.ExpiresAt,
			},
			Signature: cheque.Signature,
		},
		ChequeRecordID: chequeRecordID,
	}
}

func ChequeRecordID(cheque *cheques.Cheque) common.Hash {
	return crypto.Keccak256Hash(
		cheque.FromCMAccount.Bytes(),
		cheque.ToCMAccount.Bytes(),
		cheque.ToBot.Bytes(),
	)
}
