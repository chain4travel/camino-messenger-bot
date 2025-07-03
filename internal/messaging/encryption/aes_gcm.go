// Copyright (C) 2022-2025, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package encryption

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"fmt"
)

var (
	_ Key = NopKey{}
	_ Key = aead{}
)

type Key interface {
	Encrypt(data []byte) ([]byte, error)
	Decrypt(encryptedData []byte) ([]byte, error)
	Bytes() []byte

	private()
}

func KeyFromBytes(sessionKey []byte) (Key, error) {
	return aeadFromSessionKey(sessionKey)
}

func NewKey() (Key, error) {
	sessionKey := make([]byte, 32) // AES-256 requires a 32-byte key
	if _, err := rand.Read(sessionKey); err != nil {
		return nil, fmt.Errorf("failed to generate random session key: %w", err)
	}

	aead, err := aeadFromSessionKey(sessionKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create AEAD from session key: %w", err)
	}

	return aead, nil
}

type aead struct {
	sessionKey []byte
	aead       cipher.AEAD
}

func aeadFromSessionKey(sessionKey []byte) (aead, error) {
	block, err := aes.NewCipher(sessionKey)
	if err != nil {
		return aead{}, fmt.Errorf("failed to create AES cipher: %w", err)
	}
	cipherAEAD, err := cipher.NewGCM(block)
	if err != nil {
		return aead{}, fmt.Errorf("failed to create AES-GCM: %w", err)
	}
	return aead{
		sessionKey: sessionKey,
		aead:       cipherAEAD,
	}, nil
}

func (k aead) Encrypt(data []byte) ([]byte, error) {
	nonce := make([]byte, k.aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}
	return k.aead.Seal(nonce, nonce, data, nil), nil
}

func (k aead) Bytes() []byte {
	return k.sessionKey
}

func (k aead) Decrypt(encryptedData []byte) ([]byte, error) {
	nonceSize := k.aead.NonceSize()
	if len(encryptedData) < nonceSize {
		return nil, fmt.Errorf("encryptedData is too short: %d < %d", len(encryptedData), nonceSize)
	}
	nonce, cipherText := encryptedData[:nonceSize], encryptedData[nonceSize:]
	return k.aead.Open(nil, nonce, cipherText, nil)
}

func (k aead) private() {}

type NopKey struct {
	SessionKey []byte
}

func (NopKey) Encrypt(data []byte) ([]byte, error) {
	return data, nil
}

func (NopKey) Decrypt(encryptedData []byte) ([]byte, error) {
	return encryptedData, nil
}

func (k NopKey) Bytes() []byte {
	return k.SessionKey
}

func (NopKey) private() {}
