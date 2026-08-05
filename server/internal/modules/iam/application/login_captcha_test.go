package application

import (
	"bytes"
	"image/png"
	"net/netip"
	"testing"
)

func TestLoginCaptchaRenderingProducesRasterPNG(t *testing.T) {
	answer, err := randomCaptchaAnswer()
	if err != nil {
		t.Fatalf("create answer: %v", err)
	}
	if len(answer) != loginCaptchaLength {
		t.Fatalf("unexpected answer length: %d", len(answer))
	}
	for _, value := range []byte(answer) {
		if !bytes.ContainsRune(captchaDigits, rune(value)) {
			t.Fatalf("answer contains ambiguous digit: %q", value)
		}
	}
	encoded, err := renderCaptchaPNG(answer)
	if err != nil {
		t.Fatalf("render captcha: %v", err)
	}
	if !bytes.HasPrefix(encoded, []byte{0x89, 'P', 'N', 'G'}) {
		t.Fatal("captcha response is not a PNG")
	}
	decoded, err := png.Decode(bytes.NewReader(encoded))
	if err != nil {
		t.Fatalf("decode captcha: %v", err)
	}
	if decoded.Bounds().Dx() != 176 || decoded.Bounds().Dy() != 56 {
		t.Fatalf("unexpected captcha dimensions: %v", decoded.Bounds())
	}
}

func TestLoginScopeHashNormalizesEmailAndIPAddress(t *testing.T) {
	key := []byte("test-login-protection-key")
	ipv4Mapped := netip.MustParseAddr("::ffff:127.0.0.1")
	ipv4 := netip.MustParseAddr("127.0.0.1")
	first := loginScopeHash(key, " Admin@Example.Test ", "ak-admin", &ipv4Mapped)
	second := loginScopeHash(key, "admin@example.test", "ak-admin", &ipv4)
	if !bytes.Equal(first, second) {
		t.Fatal("equivalent identifiers must have the same protection scope")
	}
	apiScope := loginScopeHash(key, "admin@example.test", "ak-api", &ipv4)
	if bytes.Equal(first, apiScope) {
		t.Fatal("token audiences must have isolated protection scopes")
	}
}
