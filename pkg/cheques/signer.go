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
	domainType   = "EIP712Domain"
	chequeType   = "MessengerCheque"
	signatureLen = 65

	// from the decred library:
	// compactSigMagicOffset is a value used when creating the compact signature
	// recovery code inherited from Bitcoin and has no meaning, but has been
	// retained for compatibility.  For historical purposes, it was originally
	// picked to avoid a binary representation that would allow compact
	// signatures to be mistaken for other components.
	compactSigMagicOffset = 27
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

type Signer interface {
	SignCheque(cheque *Cheque) (*SignedCheque, error)
	RecoverPublicKey(cheque *SignedCheque) (*ecdsa.PublicKey, error)
}

type signer struct {
	privateKey      *ecdsa.PrivateKey
	domainSeparator []byte
	chequeTypeHash  []byte
	domain          *apitypes.TypedDataDomain
}

func NewSigner(privateKey *ecdsa.PrivateKey, chainID *big.Int) (Signer, error) {
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

	return &signer{
		privateKey:      privateKey,
		domainSeparator: domainSeparator,
		chequeTypeHash:  data.TypeHash(chequeType),
		domain:          &domain,
	}, nil
}

func (cs *signer) SignCheque(cheque *Cheque) (*SignedCheque, error) {
	finalHash, err := cs.getFinalHash(cheque)
	if err != nil {
		return nil, fmt.Errorf("failed to get final hash: %w", err)
	}

	signature, err := crypto.Sign(finalHash, cs.privateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to sign the hash: %w", err)
	}

	if len(signature) != signatureLen {
		return nil, fmt.Errorf("invalid signature length: %d", len(signature))
	}

	// adjust recovery byte
	signature[signatureLen-1] += compactSigMagicOffset

	return &SignedCheque{
		Cheque:    *cheque,
		Signature: signature,
	}, nil
}

func (cs *signer) RecoverPublicKey(cheque *SignedCheque) (*ecdsa.PublicKey, error) {
	finalHash, err := cs.getFinalHash(&cheque.Cheque)
	if err != nil {
		return nil, fmt.Errorf("failed to get final hash: %w", err)
	}

	if len(cheque.Signature) != signatureLen {
		return nil, fmt.Errorf("invalid signature length: %d", len(cheque.Signature))
	}

	// adjust recovery byte
	signature := make([]byte, signatureLen)
	copy(signature, cheque.Signature)
	signature[signatureLen-1] -= compactSigMagicOffset

	pubKey, err := crypto.SigToPub(finalHash, signature)
	if err != nil {
		return nil, fmt.Errorf("failed to recover public key: %w", err)
	}

	return pubKey, nil
}

func (cs *signer) getFinalHash(cheque *Cheque) ([]byte, error) {
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

	return crypto.Keccak256(
		hashPrefix,
		cs.domainSeparator,
		typedDataHash,
	), nil
}
