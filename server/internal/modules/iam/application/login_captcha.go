package application

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"net/netip"
	"strings"
	"time"

	"github.com/appkernia/appkernia/server/internal/modules/iam/domain"
	"github.com/google/uuid"
)

const (
	loginCaptchaThreshold = 3
	loginFailureWindow    = 30 * time.Minute
	loginCaptchaLifetime  = 5 * time.Minute
	loginCaptchaLength    = 6
)

var captchaDigits = []byte("23456789")

var captchaGlyphs = map[byte][7]byte{
	'2': {0b11111, 0b00001, 0b00001, 0b11111, 0b10000, 0b10000, 0b11111},
	'3': {0b11111, 0b00001, 0b00001, 0b01111, 0b00001, 0b00001, 0b11111},
	'4': {0b10001, 0b10001, 0b10001, 0b11111, 0b00001, 0b00001, 0b00001},
	'5': {0b11111, 0b10000, 0b10000, 0b11111, 0b00001, 0b00001, 0b11111},
	'6': {0b11111, 0b10000, 0b10000, 0b11111, 0b10001, 0b10001, 0b11111},
	'7': {0b11111, 0b00001, 0b00010, 0b00100, 0b01000, 0b01000, 0b01000},
	'8': {0b11111, 0b10001, 0b10001, 0b11111, 0b10001, 0b10001, 0b11111},
	'9': {0b11111, 0b10001, 0b10001, 0b11111, 0b00001, 0b00001, 0b11111},
}

type LoginCaptcha struct {
	ID           uuid.UUID
	ImageBase64  string
	MimeType     string
	ExpiresInSec int64
}

func (service *AuthService) CreateLoginCaptcha(ctx context.Context, email, audience string, client ClientMetadata) (LoginCaptcha, error) {
	normalizedEmail := strings.ToLower(strings.TrimSpace(email))
	if normalizedEmail == "" || len(normalizedEmail) > 320 || !strings.Contains(normalizedEmail, "@") {
		return LoginCaptcha{}, ErrProfileValidation
	}
	answer, err := randomCaptchaAnswer()
	if err != nil {
		return LoginCaptcha{}, err
	}
	salt := make([]byte, 16)
	if _, err = rand.Read(salt); err != nil {
		return LoginCaptcha{}, fmt.Errorf("create captcha salt: %w", err)
	}
	imageBytes, err := renderCaptchaPNG(answer)
	if err != nil {
		return LoginCaptcha{}, err
	}
	now := service.clock().UTC()
	expiresAt := now.Add(loginCaptchaLifetime)
	id, err := service.identities.CreateLoginCaptcha(ctx, domain.LoginCaptchaChallenge{
		ScopeHash:  loginScopeHash(service.loginProtectionKey, normalizedEmail, audience, client.IPAddress),
		AnswerSalt: salt, AnswerHash: hashCaptchaAnswer(salt, answer), CreatedAt: now, ExpiresAt: expiresAt,
	})
	if err != nil {
		return LoginCaptcha{}, fmt.Errorf("store login captcha: %w", err)
	}
	return LoginCaptcha{
		ID: id, ImageBase64: base64.StdEncoding.EncodeToString(imageBytes), MimeType: "image/png",
		ExpiresInSec: int64(expiresAt.Sub(now).Seconds()),
	}, nil
}

func loginScopeHash(key []byte, email, audience string, ipAddress *netip.Addr) []byte {
	ip := "unknown"
	if ipAddress != nil {
		ip = ipAddress.Unmap().String()
	}
	value := strings.ToLower(strings.TrimSpace(email)) + "\n" + strings.TrimSpace(audience) + "\n" + ip
	hasher := hmac.New(sha256.New, key)
	hasher.Write([]byte(value))
	return hasher.Sum(nil)
}

func hashCaptchaAnswer(salt []byte, answer string) []byte {
	hasher := sha256.New()
	hasher.Write(salt)
	hasher.Write([]byte(strings.TrimSpace(answer)))
	return hasher.Sum(nil)
}

func randomCaptchaAnswer() (string, error) {
	answer := make([]byte, loginCaptchaLength)
	buffer := make([]byte, loginCaptchaLength)
	if _, err := rand.Read(buffer); err != nil {
		return "", fmt.Errorf("create captcha answer: %w", err)
	}
	for index := range answer {
		answer[index] = captchaDigits[int(buffer[index])%len(captchaDigits)]
	}
	return string(answer), nil
}

func renderCaptchaPNG(answer string) ([]byte, error) {
	const width, height = 176, 56
	canvas := image.NewRGBA(image.Rect(0, 0, width, height))
	fill(canvas, color.RGBA{R: 245, G: 248, B: 252, A: 255})
	noise := make([]byte, 256)
	if _, err := rand.Read(noise); err != nil {
		return nil, fmt.Errorf("create captcha noise: %w", err)
	}
	for index := 0; index < 96; index++ {
		x := int(noise[(index*2)%len(noise)]) % width
		y := int(noise[(index*2+1)%len(noise)]) % height
		canvas.Set(x, y, color.RGBA{R: 120, G: 150, B: 180, A: 170})
	}
	for index := 0; index < 5; index++ {
		offset := index * 8
		drawCaptchaLine(canvas,
			int(noise[offset])%width, int(noise[offset+1])%height,
			int(noise[offset+2])%width, int(noise[offset+3])%height,
			color.RGBA{R: 90, G: 130, B: 170, A: 120})
	}
	for index := 0; index < len(answer); index++ {
		glyph := captchaGlyphs[answer[index]]
		scale := 4
		x := 12 + index*27 + int(noise[80+index])%3
		y := 10 + int(noise[90+index])%6
		ink := color.RGBA{R: 22 + noise[100+index]%30, G: 52 + noise[110+index]%35, B: 92 + noise[120+index]%35, A: 255}
		for row, bits := range glyph {
			for column := 0; column < 5; column++ {
				if bits&(1<<uint(4-column)) == 0 {
					continue
				}
				for dy := 0; dy < scale; dy++ {
					for dx := 0; dx < scale; dx++ {
						canvas.Set(x+column*scale+dx, y+row*scale+dy, ink)
					}
				}
			}
		}
	}
	var output bytes.Buffer
	if err := png.Encode(&output, canvas); err != nil {
		return nil, fmt.Errorf("encode captcha png: %w", err)
	}
	return output.Bytes(), nil
}

func fill(target *image.RGBA, value color.RGBA) {
	for y := target.Rect.Min.Y; y < target.Rect.Max.Y; y++ {
		for x := target.Rect.Min.X; x < target.Rect.Max.X; x++ {
			target.SetRGBA(x, y, value)
		}
	}
}

func drawCaptchaLine(target *image.RGBA, x0, y0, x1, y1 int, value color.RGBA) {
	dx, sx := abs(x1-x0), -1
	if x0 < x1 {
		sx = 1
	}
	dy, sy := -abs(y1-y0), -1
	if y0 < y1 {
		sy = 1
	}
	err := dx + dy
	for {
		if image.Pt(x0, y0).In(target.Rect) {
			target.SetRGBA(x0, y0, value)
		}
		if x0 == x1 && y0 == y1 {
			break
		}
		e2 := 2 * err
		if e2 >= dy {
			err += dy
			x0 += sx
		}
		if e2 <= dx {
			err += dx
			y0 += sy
		}
	}
}

func abs(value int) int {
	if value < 0 {
		return -value
	}
	return value
}
