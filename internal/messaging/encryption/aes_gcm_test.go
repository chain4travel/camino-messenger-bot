// Copyright (C) 2022-2025, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package encryption

import (
	"crypto/ecdsa"
	"crypto/rand"
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/crypto/ecies"
	"github.com/stretchr/testify/require"
)

func TestEncryptionKey(t *testing.T) {
	data := []byte("test data")

	encryptionKey1, err := NewKey()
	require.NoError(t, err)

	encryptionKey2, err := KeyFromBytes(encryptionKey1.Bytes())
	require.NoError(t, err)
	require.Equal(t, encryptionKey1, encryptionKey2)

	encryptedData1, err := encryptionKey1.Encrypt(data)
	require.NoError(t, err)
	encryptedData2, err := encryptionKey2.Encrypt(data)
	require.NoError(t, err)

	decryptedData11, err := encryptionKey1.Decrypt(encryptedData1)
	require.NoError(t, err)
	decryptedData12, err := encryptionKey1.Decrypt(encryptedData2)
	require.NoError(t, err)

	decryptedData21, err := encryptionKey2.Decrypt(encryptedData1)
	require.NoError(t, err)
	decryptedData22, err := encryptionKey2.Decrypt(encryptedData2)
	require.NoError(t, err)

	require.Equal(t, data, decryptedData11)
	require.Equal(t, data, decryptedData12)
	require.Equal(t, data, decryptedData21)
	require.Equal(t, data, decryptedData22)
}

var sink any // sink is used to avoid compiler optimizations in benchmarks

// Benchmarks shared key generation and encryption
func BenchmarkSharedKey(b *testing.B) {
	key, err := ecdsa.GenerateKey(crypto.S256(), rand.Reader)
	if err != nil {
		panic(err)
	}
	for range b.N {
		sessionKey := make([]byte, 32)
		_, _ = rand.Read(sessionKey)
		sink, _ = aeadFromSessionKey(sessionKey)
		eciesPubKey := ecies.ImportECDSAPublic(&key.PublicKey)
		sink, _ = ecies.Encrypt(rand.Reader, eciesPubKey, sessionKey, nil, nil)
	}
}
