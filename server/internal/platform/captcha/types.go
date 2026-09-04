package captcha

import (
	"errors"
	"fmt"
)

// Type is a compile-time CAPTCHA capability. Values are part of the API
// contract and must not be extended from runtime configuration.
type Type string

const (
	TypeClick  Type = "click"
	TypeSlide  Type = "slide"
	TypeDrag   Type = "drag"
	TypeRotate Type = "rotate"
)

const (
	maxCoordinate  = 4096
	maxClickPoints = 4
	clickPadding   = 8
	slidePadding   = 6
	rotatePadding  = 6
)

var (
	ErrInvalidType     = errors.New("captcha type is invalid")
	ErrInvalidResponse = errors.New("captcha response is invalid")
	ErrInvalidProof    = errors.New("captcha proof is invalid")
)

func ParseType(raw string) (Type, error) {
	value := Type(raw)
	if !value.Valid() {
		return "", ErrInvalidType
	}
	return value, nil
}

func (value Type) Valid() bool {
	switch value {
	case TypeClick, TypeSlide, TypeDrag, TypeRotate:
		return true
	default:
		return false
	}
}

type Point struct {
	X int `json:"x"`
	Y int `json:"y"`
}

type Image struct {
	Base64   string `json:"base64"`
	MimeType string `json:"mime_type"`
	Width    int    `json:"width"`
	Height   int    `json:"height"`
}

// PublicChallenge contains only browser-safe rendering data. Secret target
// coordinates and angles live exclusively in Solution and an encrypted proof.
type PublicChallenge struct {
	Type           Type   `json:"type"`
	Image          Image  `json:"image"`
	PromptImage    *Image `json:"prompt_image,omitempty"`
	RequiredPoints int    `json:"required_points,omitempty"`
	TileImage      *Image `json:"tile_image,omitempty"`
	InitialPoint   *Point `json:"initial_point,omitempty"`
	ThumbImage     *Image `json:"thumb_image,omitempty"`
}

// Response is the typed browser answer. Exactly one mode-specific field is
// accepted and Type must match the server-authenticated proof.
type Response struct {
	Type   Type    `json:"type"`
	Points []Point `json:"points,omitempty"`
	Point  *Point  `json:"point,omitempty"`
	Angle  *int    `json:"angle,omitempty"`
}

type clickTarget struct {
	X      int `json:"x"`
	Y      int `json:"y"`
	Width  int `json:"width"`
	Height int `json:"height"`
}

// Solution is never serialized into an API response. It is encrypted into an
// opaque proof token by Codec and is safe to persist only as a token hash.
type Solution struct {
	Type         Type          `json:"type"`
	CanvasWidth  int           `json:"canvas_width"`
	CanvasHeight int           `json:"canvas_height"`
	ClickTargets []clickTarget `json:"click_targets,omitempty"`
	Point        *Point        `json:"point,omitempty"`
	Angle        *int          `json:"angle,omitempty"`
}

func (value Point) valid(width, height int) bool {
	return value.X >= 0 && value.Y >= 0 && value.X < width && value.Y < height && value.X <= maxCoordinate && value.Y <= maxCoordinate
}

func (value Solution) validate() error {
	if !value.Type.Valid() || value.CanvasWidth < 1 || value.CanvasHeight < 1 || value.CanvasWidth > maxCoordinate || value.CanvasHeight > maxCoordinate {
		return ErrInvalidProof
	}
	switch value.Type {
	case TypeClick:
		if len(value.ClickTargets) < 1 || len(value.ClickTargets) > maxClickPoints || value.Point != nil || value.Angle != nil {
			return ErrInvalidProof
		}
		for _, target := range value.ClickTargets {
			if target.Width < 1 || target.Height < 1 || target.X < 0 || target.Y < 0 || target.X+target.Width > value.CanvasWidth || target.Y+target.Height > value.CanvasHeight {
				return ErrInvalidProof
			}
		}
	case TypeSlide, TypeDrag:
		if value.Point == nil || len(value.ClickTargets) != 0 || value.Angle != nil || !value.Point.valid(value.CanvasWidth, value.CanvasHeight) {
			return ErrInvalidProof
		}
	case TypeRotate:
		if value.Angle == nil || *value.Angle < 0 || *value.Angle > 360 || len(value.ClickTargets) != 0 || value.Point != nil {
			return ErrInvalidProof
		}
	default:
		return fmt.Errorf("%w: %q", ErrInvalidType, value.Type)
	}
	return nil
}
