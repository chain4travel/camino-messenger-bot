/*
 * Copyright (C) 2022-2023, Chain4Travel AG. All rights reserved.
 * See the file LICENSE for licensing terms.
 */

package matrix

import (
	"crypto/rsa"
	"encoding/base64"
	"fmt"
	aes_util "github.com/chain4travel/camino-messenger-bot/utils/aes"
	rsa_util "github.com/chain4travel/camino-messenger-bot/utils/rsa"
	"sync"
)

type EncryptionKeyRepository struct {
	pubKeyCache       map[string]*rsa.PublicKey
	symmetricKeyCache map[string][]byte
	mu                sync.Mutex
}

func NewEncryptionKeyRepository() *EncryptionKeyRepository {
	return &EncryptionKeyRepository{pubKeyCache: make(map[string]*rsa.PublicKey), symmetricKeyCache: make(map[string][]byte)}
}
func (p *EncryptionKeyRepository) getPublicKeyForRecipient(recipient string) (*rsa.PublicKey, error) {
	pkey := p.fetchKeyFromCache(recipient)
	if pkey != nil {
		return pkey, nil
	}
	var encodedPubKey string
	//TODO for now it's all keys are hardcoded. Later we need to get the keys from the key server
	switch recipient {
	case "@t-kopernikus1tyewqsap6v8r8wghg7qn7dyfzg2prtcrw04ke3:matrix.camino.network":
		encodedPubKey = "LS0tLS1CRUdJTiBSU0EgUFVCTElDIEtFWS0tLS0tCk1JSUJpZ0tDQVlFQWpwR2R4NlloMkJnR3V2QUdQQTZiMnAzd1dBWnZYalQvWTBQR1ViRlR0U1lVY1hydEc1eEIKcy9DYVo4NGtmTEpHRnhEV3d3d3E4bzRoNHBzOGd0aHJkaG5QMUFIOEtGRFFCTzJNbDY5ZmFZYWd4ajdtVUxnSQpqTWIzUEorVjQzZUQrRktHZis2R0E5aHpXd09RZDhvVWdZVnhSQ2xMU0tMYi82WXFqaU81LzFxK3plTWowZWF2CmJQY2VsK2V6UVlpQmE3UzNJcHFGUFdhL0N0TTd3Qi90UWI2MnAzWFRkU0pnenR1SlJpTU5MeFI2NFU3WWlHcGsKR0VYSjFyd2lPZFhPMjJaRyt4UkxmOEl3ZFF2dEUxR1VnL1llTEtSOWd5blI5WTNiZzA5UWErRkpYQ1FrTjVHUApJY2E0S2R4UGpjQ0xHTklGVlVSTnNkTjFrZnJzcXpLTXNQOVgwQkFUMHNWYTk3WTd5RnAxUTFKTmU4Uy96T1FWCm9XOHJpSVFvWGRqSDNES2Q3cERQekN2TEpQRm50dzF5YWRUZ1pLbGs5Y21tT0dDbXh5SUZwMW5mTXk1R1FDM20KS1AxZ2NIV3J5UmFBcG4reG9BSFdIcHErcVNicmpka0h2MEt1MDRaMTRYcWhaK2Ezc3FtM3oreWpNYTF2OExDUApvK2I3OFI4OGpqVDFBZ01CQUFFPQotLS0tLUVORCBSU0EgUFVCTElDIEtFWS0tLS0tCg=="
	case "@t-kopernikus15ss2mvy86h0hcwdhfukdx7y3cvuwdqxm6a0aqm:matrix.camino.network":
		encodedPubKey = "LS0tLS1CRUdJTiBSU0EgUFVCTElDIEtFWS0tLS0tCk1JSUJpZ0tDQVlFQXI5a1RrWHkyNWlIaTNhai9ib2VER3VFTmNJZ1dqVmlYRHVmUFJUQ1FSV0I0TEt1eCtaWXoKaElWckdPb2l6eDNoR29NTnMwOUlvODFzb2wyd3crQ0tENUtTYTVHVHNJelh6ZytGcEErTmsrcDFOOGlsWFVINQp4d2NFTlRBclVCK2Y0SmU4Vkl0dEc5ZVhHQW9aQ1RYc2FTRWNmVG1Lc24vVUdsSHVQdGs5WHVpTlNTb3k0ZTMvCnpGcGFjZngyaVR6TVJOMjc1Ky9aZjllZ3RtSnVXS0JKcnNOcC9iQ245Q2ErcURheDNMTmJpdG55TUF2eG5rUmgKTWZrQ1lHTHZmSkFoRVVlSVJUUkRLT0xCSEtRVFJpQ1F4SHlXSVVEQkswbkZtbkt5Ti80RUI1RWkzTkg4RkpFKwpaK1NobmExdmlkdWV0R2NtMjhKRFRweXhGRStyZXZQWWs3aXVJZGF3VEZtTUlabkRrTnpRRkxlSStHaXFPN2JNCkRlT0NSa2FBRDhnSkEzT29OeXBmUlRuaEMvVHFvMWk1VjZ1RlV5RU9LT3dvMHk4cEFCSmNTRzBoUVRxQUh3blAKZkZFLzI2REtsMzQzZ1oxV3lBa29QcUUyVk1ESklSVFVUcHhBR09IMk9qZDRnWjBJWk1QTks0RDYyMWk4V2NrZApNTlI5ZEZRQW1mOS9BZ01CQUFFPQotLS0tLUVORCBSU0EgUFVCTElDIEtFWS0tLS0tCg=="
	default:
		return nil, fmt.Errorf("no public key found for recipient: %s", recipient)
	}
	pubKeyBytes, err := base64.StdEncoding.DecodeString(encodedPubKey)
	if err != nil {
		return nil, err
	}
	pKey, err := rsa_util.ParseRSAPublicKey(pubKeyBytes)
	if err != nil {
		return nil, err
	}
	p.cachePublicKey(recipient, pKey)
	return pKey, nil
}

func (p *EncryptionKeyRepository) cachePublicKey(recipient string, key *rsa.PublicKey) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.pubKeyCache[recipient] = key
}

func (p *EncryptionKeyRepository) fetchKeyFromCache(recipient string) *rsa.PublicKey {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.pubKeyCache[recipient]
}

func (p *EncryptionKeyRepository) getSymmetricKeyForRecipient(recipient string) []byte {
	key := p.fetchSymmetricKeyFromCache(recipient)
	if key != nil {
		return key
	}
	key = aes_util.GenerateAESKey()
	p.cacheSymmetricKey(recipient, key)
	return key
}

func (p *EncryptionKeyRepository) cacheSymmetricKey(recipient string, key []byte) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.symmetricKeyCache[recipient] = key
}

func (p *EncryptionKeyRepository) fetchSymmetricKeyFromCache(recipient string) []byte {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.symmetricKeyCache[recipient]
}
