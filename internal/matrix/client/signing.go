// Copyright (C) 2022-2025, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package client

import (
	"crypto/ecdsa"
	"crypto/sha256"
	"errors"
	"fmt"
	"math"

	"github.com/ethereum/go-ethereum/crypto"
)

func SignPublicKey(key *ecdsa.PrivateKey) (signature string, message string, err error) {
	pubKeyBytes := crypto.FromECDSAPub(&key.PublicKey)
	signatureBytes, err := sign(pubKeyBytes, key)
	if err != nil {
		return "", "", err
	}

	signature, err = hexWithChecksum(signatureBytes)
	if err != nil {
		return "", "", err
	}
	message, err = hexWithChecksum(pubKeyBytes)
	if err != nil {
		return "", "", err
	}
	return signature, message, nil
}

func sign(msg []byte, key *ecdsa.PrivateKey) ([]byte, error) {
	// TODO @evlekht use crypto.keccak256 in conduit, here and in asb
	hash256 := sha256.Sum256(msg)

	signature, err := crypto.Sign(hash256[:], key)
	if err != nil {
		return nil, err
	}

	return signature, nil
}

func hexWithChecksum(bytes []byte) (string, error) {
	const checksumLen = 4
	bytesLen := len(bytes)
	if bytesLen > math.MaxInt32-checksumLen {
		return "", errors.New("encoding overflow")
	}
	checked := make([]byte, bytesLen+checksumLen)
	copy(checked, bytes)
	hash := sha256.Sum256(bytes)
	copy(checked[len(bytes):], hash[len(hash)-checksumLen:])
	bytes = checked
	return fmt.Sprintf("0x%x", bytes), nil
}
