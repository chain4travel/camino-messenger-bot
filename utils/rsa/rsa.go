/*
 * Copyright (C) 2022-2023, Chain4Travel AG. All rights reserved.
 * See the file LICENSE for licensing terms.
 */

package rsa_util

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha512"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"os"
)

func main() {
	// Replace these file paths with your own private and public key file paths
	privateKeyFilePath := "private_key2.pem"
	publicKeyFilePath := "public_key2.pem"

	// Read private key
	_, err := ParseRSAPrivateKeyFromFile(privateKeyFilePath)
	if err != nil {
		fmt.Printf("Error reading private key file: %v\n", err)
		return
	}

	publicKeyBytes, err := os.ReadFile(publicKeyFilePath)
	if err != nil {
		fmt.Printf("Error reading public key file: %v\n", err)
		return
	}

	// Use the keys as needed
	//fmt.Println("Private Key:", privateKey)

	b := make([]byte, base64.StdEncoding.EncodedLen(len(publicKeyBytes)))
	base64.StdEncoding.Encode(b, publicKeyBytes)
	fmt.Println("Public Key:", string(b))
	dec := make([]byte, base64.StdEncoding.DecodedLen(len(b)))
	base64.StdEncoding.Decode(dec, b)
	fmt.Println("Public Key:", string(dec))

	p, err := ParseRSAPublicKey(dec)
	if err != nil {
		fmt.Printf("Error parsing public key: %v\n", err)
		return
	}
	fmt.Println("Public Key:", p)
}

func ParseRSAPrivateKeyFromFile(keyFilePath string) (*rsa.PrivateKey, error) {
	privateKeyBytes, err := os.ReadFile(keyFilePath)
	if err != nil {
		fmt.Printf("Error reading private key file: %v\n", err)
		return nil, err
	}
	return ParseRSAPrivateKey(privateKeyBytes)
}
func ParseRSAPrivateKey(keyBytes []byte) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode(keyBytes)
	if block == nil || block.Type != "RSA PRIVATE KEY" {
		return nil, fmt.Errorf("failed to decode PEM block containing private key")
	}

	privateKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse private key: %v", err)
	}

	return privateKey, nil
}

func ParseRSAPublicKey(keyBytes []byte) (*rsa.PublicKey, error) {
	block, _ := pem.Decode(keyBytes)
	if block == nil || block.Type != "RSA PUBLIC KEY" {
		return nil, fmt.Errorf("failed to decode PEM block containing public key")
	}

	publicKey, err := x509.ParsePKCS1PublicKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse public key: %v", err)
	}

	return publicKey, nil
}

// EncryptWithPublicKey encrypts data with public key
func EncryptWithPublicKey(msg []byte, pub *rsa.PublicKey) ([]byte, error) {
	hash := sha512.New()
	return rsa.EncryptOAEP(hash, rand.Reader, pub, msg, nil)
}

// DecryptWithPrivateKey decrypts data with private key
func DecryptWithPrivateKey(ciphertext []byte, priv *rsa.PrivateKey) ([]byte, error) {
	hash := sha512.New()
	return rsa.DecryptOAEP(hash, rand.Reader, priv, ciphertext, nil)
}
