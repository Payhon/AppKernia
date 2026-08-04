package repository

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"fmt"
)

type AESGCMSealer struct {
	aead    cipher.AEAD
	version int32
}

func NewAESGCMSealer(key []byte, version int32) (*AESGCMSealer, error) {
	if len(key) != 32 || version < 1 {
		return nil, fmt.Errorf("config secret key must be 32 bytes with a positive version")
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	return &AESGCMSealer{aead: aead, version: version}, nil
}

func (s *AESGCMSealer) Seal(plaintext []byte, aad string) ([]byte, int32, error) {
	nonce := make([]byte, s.aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, 0, err
	}
	sealed := s.aead.Seal(nil, nonce, plaintext, []byte(aad))
	return append(nonce, sealed...), s.version, nil
}

func (s *AESGCMSealer) Open(ciphertext []byte, aad string) ([]byte, error) {
	nonceSize := s.aead.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, fmt.Errorf("encrypted value is truncated")
	}
	return s.aead.Open(nil, ciphertext[:nonceSize], ciphertext[nonceSize:], []byte(aad))
}
