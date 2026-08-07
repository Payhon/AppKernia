package application

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

var ErrInvalidAccessToken = errors.New("invalid access token")

type AccessClaims struct {
	SessionID    uuid.UUID `json:"sid"`
	TenantID     uuid.UUID `json:"tid"`
	AppID        uuid.UUID `json:"aid,omitempty"`
	TokenVersion int32     `json:"ver"`
	jwt.RegisteredClaims
}

type TokenIssuer struct {
	issuer     string
	keyID      string
	privateKey ed25519.PrivateKey
	publicKey  ed25519.PublicKey
	lifetime   time.Duration
	clock      func() time.Time
}

func NewTokenIssuer(issuer, keyID string, privateKey ed25519.PrivateKey, lifetime time.Duration) (*TokenIssuer, error) {
	if len(privateKey) != ed25519.PrivateKeySize || issuer == "" || keyID == "" || lifetime <= 0 {
		return nil, errors.New("invalid access token issuer configuration")
	}
	return &TokenIssuer{
		issuer: issuer, keyID: keyID, privateKey: privateKey,
		publicKey: privateKey.Public().(ed25519.PublicKey), lifetime: lifetime, clock: time.Now,
	}, nil
}

func NewDevelopmentTokenIssuer() (*TokenIssuer, error) {
	_, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("generate development signing key: %w", err)
	}
	return NewTokenIssuer("appkernia-development", "development-ephemeral", privateKey, 15*time.Minute)
}

func NewTokenIssuerFromBase64(issuer, keyID, encodedPrivateKey string, lifetime time.Duration) (*TokenIssuer, error) {
	key, err := base64.StdEncoding.DecodeString(encodedPrivateKey)
	if err != nil {
		return nil, fmt.Errorf("decode Ed25519 private key: %w", err)
	}
	if len(key) != ed25519.PrivateKeySize {
		return nil, errors.New("Ed25519 private key must be 64 bytes")
	}
	return NewTokenIssuer(issuer, keyID, ed25519.PrivateKey(key), lifetime)
}

func (issuer *TokenIssuer) Issue(userID, tenantID, sessionID uuid.UUID, audience string, version int32) (string, time.Time, error) {
	return issuer.IssueForApp(userID, tenantID, sessionID, audience, version, uuid.Nil)
}

func (issuer *TokenIssuer) IssueForApp(userID, tenantID, sessionID uuid.UUID, audience string, version int32, appID uuid.UUID) (string, time.Time, error) {
	now := issuer.clock().UTC()
	expiresAt := now.Add(issuer.lifetime)
	claims := AccessClaims{
		SessionID: sessionID, TenantID: tenantID, AppID: appID, TokenVersion: version,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer: issuer.issuer, Subject: userID.String(), Audience: jwt.ClaimStrings{audience},
			IssuedAt: jwt.NewNumericDate(now), NotBefore: jwt.NewNumericDate(now.Add(-5 * time.Second)),
			ExpiresAt: jwt.NewNumericDate(expiresAt), ID: uuid.NewString(),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodEdDSA, claims)
	token.Header["kid"] = issuer.keyID
	signed, err := token.SignedString(issuer.privateKey)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("sign access token: %w", err)
	}
	return signed, expiresAt, nil
}

func (issuer *TokenIssuer) Verify(rawToken, audience string) (AccessClaims, error) {
	claims := AccessClaims{}
	token, err := jwt.ParseWithClaims(rawToken, &claims, func(token *jwt.Token) (any, error) {
		if token.Header["kid"] != issuer.keyID {
			return nil, ErrInvalidAccessToken
		}
		return issuer.publicKey, nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodEdDSA.Alg()}), jwt.WithIssuer(issuer.issuer), jwt.WithAudience(audience))
	if err != nil || !token.Valid {
		return AccessClaims{}, ErrInvalidAccessToken
	}
	if claims.SessionID == uuid.Nil || claims.TenantID == uuid.Nil || claims.Subject == "" || claims.TokenVersion <= 0 {
		return AccessClaims{}, ErrInvalidAccessToken
	}
	if _, err = uuid.Parse(claims.Subject); err != nil {
		return AccessClaims{}, ErrInvalidAccessToken
	}
	return claims, nil
}

func NewOpaqueToken() (plainText string, hash []byte, err error) {
	value := make([]byte, 32)
	if _, err = rand.Read(value); err != nil {
		return "", nil, fmt.Errorf("generate opaque token: %w", err)
	}
	plainText = base64.RawURLEncoding.EncodeToString(value)
	digest := sha256.Sum256([]byte(plainText))
	return plainText, digest[:], nil
}

func HashOpaqueToken(plainText string) []byte {
	digest := sha256.Sum256([]byte(plainText))
	return digest[:]
}
