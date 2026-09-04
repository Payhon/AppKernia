package captcha

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"time"
)

const (
	proofVersion       = 1
	maximumProofLength = 4096
	maximumProofTTL    = 10 * time.Minute
)

var proofAAD = []byte("appkernia:interactive-captcha-proof:v1")

type Proof struct {
	Version     int      `json:"v"`
	ChallengeID string   `json:"challenge_id"`
	ScopeHash   []byte   `json:"scope_hash"`
	IssuedAt    int64    `json:"issued_at"`
	ExpiresAt   int64    `json:"expires_at"`
	Solution    Solution `json:"solution"`
}

type Codec struct {
	aead   cipher.AEAD
	random io.Reader
}

// NewProof builds the authenticated server-side payload that is returned only
// after being sealed by Codec. The scope hash is copied so callers cannot
// mutate a proof after construction.
func NewProof(challengeID string, scopeHash []byte, issuedAt, expiresAt time.Time, solution Solution) (Proof, error) {
	proof := Proof{
		Version: proofVersion, ChallengeID: challengeID, ScopeHash: append([]byte(nil), scopeHash...),
		IssuedAt: issuedAt.UTC().Unix(), ExpiresAt: expiresAt.UTC().Unix(), Solution: solution,
	}
	if err := validateProof(proof); err != nil {
		return Proof{}, err
	}
	return proof, nil
}

func NewCodec(masterKey []byte) (*Codec, error) {
	if len(masterKey) != sha256.Size {
		return nil, fmt.Errorf("%w: master key must contain 32 bytes", ErrInvalidProof)
	}
	deriver := hmac.New(sha256.New, masterKey)
	_, _ = deriver.Write([]byte("appkernia:interactive-captcha:aes-gcm:v1"))
	block, err := aes.NewCipher(deriver.Sum(nil))
	if err != nil {
		return nil, fmt.Errorf("create captcha proof cipher: %w", err)
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("create captcha proof AEAD: %w", err)
	}
	return &Codec{aead: aead, random: rand.Reader}, nil
}

func (codec *Codec) Seal(proof Proof) (string, error) {
	if codec == nil || codec.aead == nil || validateProof(proof) != nil {
		return "", ErrInvalidProof
	}
	plaintext, err := json.Marshal(proof)
	if err != nil {
		return "", fmt.Errorf("encode captcha proof: %w", err)
	}
	nonce := make([]byte, codec.aead.NonceSize())
	if _, err = io.ReadFull(codec.random, nonce); err != nil {
		return "", fmt.Errorf("create captcha proof nonce: %w", err)
	}
	sealed := codec.aead.Seal(nonce, nonce, plaintext, proofAAD)
	return base64.RawURLEncoding.EncodeToString(sealed), nil
}

func (codec *Codec) Open(token string) (Proof, error) {
	if codec == nil || codec.aead == nil || token == "" || len(token) > maximumProofLength {
		return Proof{}, ErrInvalidProof
	}
	sealed, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil || len(sealed) <= codec.aead.NonceSize() {
		return Proof{}, ErrInvalidProof
	}
	nonce, ciphertext := sealed[:codec.aead.NonceSize()], sealed[codec.aead.NonceSize():]
	plaintext, err := codec.aead.Open(nil, nonce, ciphertext, proofAAD)
	if err != nil {
		return Proof{}, ErrInvalidProof
	}
	decoder := json.NewDecoder(bytes.NewReader(plaintext))
	decoder.DisallowUnknownFields()
	var proof Proof
	if err = decoder.Decode(&proof); err != nil {
		return Proof{}, ErrInvalidProof
	}
	if err = ensureJSONEOF(decoder); err != nil || validateProof(proof) != nil {
		return Proof{}, ErrInvalidProof
	}
	return proof, nil
}

func TokenHash(token string) []byte {
	digest := sha256.Sum256([]byte(token))
	return digest[:]
}

func validateProof(proof Proof) error {
	if proof.Version != proofVersion || len(proof.ChallengeID) < 1 || len(proof.ChallengeID) > 64 || len(proof.ScopeHash) != sha256.Size || proof.IssuedAt <= 0 || proof.ExpiresAt <= proof.IssuedAt || time.Duration(proof.ExpiresAt-proof.IssuedAt)*time.Second > maximumProofTTL {
		return ErrInvalidProof
	}
	return proof.Solution.validate()
}

func ensureJSONEOF(decoder *json.Decoder) error {
	var trailing any
	err := decoder.Decode(&trailing)
	if errors.Is(err, io.EOF) {
		return nil
	}
	return ErrInvalidProof
}
