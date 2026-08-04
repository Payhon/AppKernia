package application

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1" // #nosec G505 -- RFC 6238 interoperability requires HMAC-SHA1, not collision resistance.
	"crypto/sha256"
	"encoding/base32"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"
)

func randomToken(size int) ([]byte, error) {
	value := make([]byte, size)
	_, err := rand.Read(value)
	return value, err
}

func opaqueToken(size int) (string, error) {
	raw, err := randomToken(size)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(raw), nil
}

func totpSecret() (string, error) {
	raw, err := randomToken(20)
	if err != nil {
		return "", err
	}
	return base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(raw), nil
}

func totpURI(secret, account string) string {
	label := url.PathEscape("AppKernia:" + account)
	values := url.Values{"secret": {secret}, "issuer": {"AppKernia"}, "algorithm": {"SHA1"}, "digits": {"6"}, "period": {"30"}}
	return "otpauth://totp/" + label + "?" + values.Encode()
}

func totpAt(secret string, timestamp time.Time) (string, error) {
	raw, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(strings.ToUpper(strings.TrimSpace(secret)))
	if err != nil {
		return "", err
	}
	counter := uint64(timestamp.Unix() / 30) // #nosec G115 -- Unix time is non-negative for supported runtime dates.
	message := make([]byte, 8)
	binary.BigEndian.PutUint64(message, counter)
	mac := hmac.New(sha1.New, raw)
	_, _ = mac.Write(message)
	sum := mac.Sum(nil)
	offset := sum[len(sum)-1] & 0x0f
	value := (uint32(sum[offset])&0x7f)<<24 | uint32(sum[offset+1])<<16 | uint32(sum[offset+2])<<8 | uint32(sum[offset+3])
	return fmt.Sprintf("%06d", value%1_000_000), nil
}

func verifyTOTP(secret, candidate string, now time.Time) bool {
	if len(candidate) != 6 {
		return false
	}
	if _, err := strconv.Atoi(candidate); err != nil {
		return false
	}
	for offset := -1; offset <= 1; offset++ {
		expected, err := totpAt(secret, now.Add(time.Duration(offset)*30*time.Second))
		if err == nil && hmac.Equal([]byte(expected), []byte(candidate)) {
			return true
		}
	}
	return false
}

func recoveryCodes() ([]string, [][]byte, error) {
	codes := make([]string, 10)
	hashes := make([][]byte, 10)
	for index := range codes {
		raw, err := randomToken(8)
		if err != nil {
			return nil, nil, err
		}
		encoded := strings.ToUpper(base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(raw))
		codes[index] = encoded[:6] + "-" + encoded[6:12]
		digest := sha256.Sum256([]byte(codes[index]))
		hashes[index] = digest[:]
	}
	return codes, hashes, nil
}

func sha256Bytes(value string) []byte {
	digest := sha256.Sum256([]byte(value))
	return digest[:]
}

func pkceChallenge(verifier string) string {
	return base64.RawURLEncoding.EncodeToString(sha256Bytes(verifier))
}
