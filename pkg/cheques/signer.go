package cheques

import (
	"crypto/ecdsa"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/signer/core/apitypes"
)

const (
	domainType = "EIP712Domain"
	chequeType = "MessengerCheque"
)

var (
	hashPrefix = []byte{0x19, 0x01}
	types      = apitypes.Types{
		domainType: {
			{Name: "name", Type: "string"},
			{Name: "version", Type: "string"},
			{Name: "chainId", Type: "uint256"},
		},
		chequeType: {
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

type ChequeSigner struct {
	privateKey      *ecdsa.PrivateKey
	domainSeparator []byte
	chequeTypeHash  []byte
	domain          *apitypes.TypedDataDomain
}

func NewChequeSigner(privateKey *ecdsa.PrivateKey, chainID *big.Int) (*ChequeSigner, error) {
	domain := apitypes.TypedDataDomain{
		Name:    "CaminoMessenger",
		Version: "1",
		ChainId: (*math.HexOrDecimal256)(chainID),
	}

	data := apitypes.TypedData{
		Domain: domain,
		Types:  types,
	}

	domainSeparator, err := data.HashStruct(domainType, apitypes.TypedDataMessage{
		"name":    domain.Name,
		"version": domain.Version,
		"chainId": domain.ChainId,
	})
	if err != nil {
		return nil, err
	}

	return &ChequeSigner{
		privateKey:      privateKey,
		domainSeparator: domainSeparator,
		chequeTypeHash:  data.TypeHash(chequeType),
		domain:          &domain,
	}, nil
}

func (cs *ChequeSigner) SignCheque(cheque *Cheque) (*SignedCheque, error) {
	message := apitypes.TypedDataMessage{
		"fromCMAccount": cheque.FromCMAccount.Hex(),
		"toCMAccount":   cheque.ToCMAccount.Hex(),
		"toBot":         cheque.ToBot.Hex(),
		"counter":       cheque.Counter,
		"amount":        cheque.Amount,
		"createdAt":     cheque.CreatedAt,
		"expiresAt":     cheque.ExpiresAt,
	}

	data := &apitypes.TypedData{
		Types:       types,
		Domain:      *cs.domain,
		Message:     message,
		PrimaryType: chequeType,
	}

	typedDataHash, err := hashStructWithTypeHash(data, chequeType, cs.chequeTypeHash)
	if err != nil {
		return nil, fmt.Errorf("failed to hash struct: %w", err)
	}

	finalHash := crypto.Keccak256(
		hashPrefix,
		cs.domainSeparator,
		typedDataHash,
	)

	signature, err := crypto.Sign(finalHash, cs.privateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to sign the hash: %w", err)
	}

	// adjust recovery byte for compatibility
	signature[64] += 27

	return &SignedCheque{
		Cheque:    *cheque,
		Signature: signature,
	}, nil
}
