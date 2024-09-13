package cheques

import (
	"math/big"
	"time"

	"crypto/ecdsa"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/math"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/signer/core/apitypes"
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
	CreatedAt     time.Time      `json:"createdAt"`
	ExpiresAt     time.Time      `json:"expiresAt"`
}

var (
	types = apitypes.Types{
		"EIP712Domain": {
			{Name: "name", Type: "string"},
			{Name: "version", Type: "string"},
			{Name: "chainId", Type: "uint256"},
		},
		"Cheque": {
			{Name: "fromCMAccount", Type: "address"},
			{Name: "toCMAccount", Type: "address"},
			{Name: "toBot", Type: "address"},
			{Name: "counter", Type: "uint256"},
			{Name: "amount", Type: "uint256"},
			{Name: "createdAt", Type: "uint256"},
			{Name: "expiresAt", Type: "uint256"},
		},
	}
)

type chequeSigner struct {
	privateKey *ecdsa.PrivateKey
	domainHash []byte
}

func NewChequeSigner(privateKey *ecdsa.PrivateKey, chainID *big.Int) (*chequeSigner, error) {
	domain := apitypes.TypedDataDomain{
		Name:    "CaminoMessenger",
		Version: "1",
		ChainId: (*math.HexOrDecimal256)(chainID),
	}

	domainData := apitypes.TypedData{
		Domain: domain,
		Types:  types,
	}
	domainMessage := apitypes.TypedDataMessage{
		"name":    domain.Name,
		"version": domain.Version,
		"chainId": domain.ChainId,
	}

	domainHash, err := domainData.HashStruct("EIP712Domain", domainMessage)
	if err != nil {
		return nil, err
	}

	return &chequeSigner{
		privateKey: privateKey,
		domainHash: domainHash,
	}, nil
}

func (cs *chequeSigner) SignCheque(cheque *Cheque) (*SignedCheque, error) {
	message := apitypes.TypedDataMessage{
		"fromCMAccount": cheque.FromCMAccount.Hex(),
		"toCMAccount":   cheque.ToCMAccount.Hex(),
		"toBot":         cheque.ToBot.Hex(),
		"counter":       cheque.Counter,
		"amount":        cheque.Amount,
		"createdAt":     cheque.CreatedAt.Unix(),
		"expiresAt":     cheque.ExpiresAt.Unix(),
	}

	data := apitypes.TypedData{
		Types:       types,
		Message:     message,
		PrimaryType: "Cheque",
	}

	messageHash, err := data.HashStruct(data.PrimaryType, message)
	if err != nil {
		return nil, fmt.Errorf("failed to hash struct: %v", err)
	}

	// Calculate the final hash (EIP-712 final hash: keccak256("\x19\x01", domainHash, messageHash))
	finalHash := crypto.Keccak256(
		[]byte{0x19, 0x01},
		cs.domainHash,
		messageHash,
	)

	// Sign the final hash
	signature, err := crypto.Sign(finalHash, cs.privateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to sign the hash: %v", err)
	}

	return &SignedCheque{
		Cheque:    *cheque,
		Signature: signature,
	}, nil
}
