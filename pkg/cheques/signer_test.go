package cheques

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"
)

func TestSignCheque(t *testing.T) {
	chainID := big.NewInt(1)
	privateKey, err := crypto.GenerateKey()
	require.NoError(t, err)

	signer, err := NewSigner(privateKey, chainID)
	require.NoError(t, err)

	cheque := &Cheque{
		FromCMAccount: common.HexToAddress("0x0"),
		ToCMAccount:   common.HexToAddress("0x1"),
		ToBot:         common.HexToAddress("0x2"),
		Counter:       big.NewInt(1),
		Amount:        big.NewInt(100),
		CreatedAt:     big.NewInt(1633024800),
		ExpiresAt:     big.NewInt(1633111200),
	}

	signedCheque, err := signer.SignCheque(cheque)
	require.NoError(t, err)
	require.NotNil(t, signedCheque)
	require.Equal(t, cheque, &signedCheque.Cheque)

	pubKey, err := signer.RecoverPublicKey(signedCheque)
	require.NoError(t, err)

	recoveredAddress := crypto.PubkeyToAddress(*pubKey)
	expectedAddress := crypto.PubkeyToAddress(privateKey.PublicKey)
	require.Equal(t, expectedAddress, recoveredAddress)
}
