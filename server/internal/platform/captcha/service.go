package captcha

import (
	"fmt"
	"image"
	"sync"

	assetsimages "github.com/wenlng/go-captcha-assets/resources/imagesv2"
	assetsshapes "github.com/wenlng/go-captcha-assets/resources/shapes"
	assetstiles "github.com/wenlng/go-captcha-assets/resources/tiles"
	"github.com/wenlng/go-captcha/v2/base/option"
	"github.com/wenlng/go-captcha/v2/click"
	"github.com/wenlng/go-captcha/v2/rotate"
	"github.com/wenlng/go-captcha/v2/slide"
)

type resources struct {
	backgrounds []image.Image
	shapes      map[string]image.Image
	tiles       []*slide.GraphImage
}

var (
	sharedResources resources
	sharedError     error
	loadOnce        sync.Once
)

type Service struct {
	resources resources
}

func NewService() (*Service, error) {
	loadOnce.Do(func() {
		backgrounds, err := assetsimages.GetImages()
		if err != nil {
			sharedError = fmt.Errorf("load captcha backgrounds: %w", err)
			return
		}
		shapes, err := assetsshapes.GetShapes()
		if err != nil {
			sharedError = fmt.Errorf("load captcha shapes: %w", err)
			return
		}
		tiles, err := assetstiles.GetTiles()
		if err != nil {
			sharedError = fmt.Errorf("load captcha tiles: %w", err)
			return
		}
		graphs := make([]*slide.GraphImage, 0, len(tiles))
		for _, tile := range tiles {
			if tile == nil || tile.OverlayImage == nil || tile.ShadowImage == nil || tile.MaskImage == nil {
				sharedError = errorsForResources("tile")
				return
			}
			graphs = append(graphs, &slide.GraphImage{OverlayImage: tile.OverlayImage, ShadowImage: tile.ShadowImage, MaskImage: tile.MaskImage})
		}
		if len(backgrounds) == 0 || len(shapes) < 4 || len(graphs) == 0 {
			sharedError = errorsForResources("incomplete")
			return
		}
		sharedResources = resources{backgrounds: backgrounds, shapes: shapes, tiles: graphs}
	})
	if sharedError != nil {
		return nil, sharedError
	}
	return &Service{resources: sharedResources}, nil
}

func errorsForResources(name string) error {
	return fmt.Errorf("captcha %s resources are invalid", name)
}

func (service *Service) Generate(kind Type) (PublicChallenge, Solution, error) {
	if service == nil || !kind.Valid() {
		return PublicChallenge{}, Solution{}, ErrInvalidType
	}
	switch kind {
	case TypeClick:
		return service.generateClick()
	case TypeSlide:
		return service.generateSlide(false)
	case TypeDrag:
		return service.generateSlide(true)
	case TypeRotate:
		return service.generateRotate()
	default:
		return PublicChallenge{}, Solution{}, ErrInvalidType
	}
}

func (service *Service) generateClick() (PublicChallenge, Solution, error) {
	builder := click.NewBuilder(
		click.WithRangeLen(option.RangeVal{Min: 6, Max: 6}),
		click.WithRangeVerifyLen(option.RangeVal{Min: 2, Max: 3}),
	)
	builder.SetResources(click.WithShapes(service.resources.shapes), click.WithBackgrounds(service.resources.backgrounds))
	data, err := builder.MakeShape().Generate()
	if err != nil {
		return PublicChallenge{}, Solution{}, fmt.Errorf("generate click captcha: %w", err)
	}
	imageData, err := jpegImage(data.GetMasterImage().Get(), data.GetMasterImage().ToBase64Data)
	if err != nil {
		return PublicChallenge{}, Solution{}, err
	}
	prompt, err := pngImage(data.GetThumbImage().Get(), data.GetThumbImage().ToBase64Data)
	if err != nil {
		return PublicChallenge{}, Solution{}, err
	}
	dots := data.GetData()
	if len(dots) < 1 || len(dots) > maxClickPoints {
		return PublicChallenge{}, Solution{}, ErrInvalidProof
	}
	targets := make([]clickTarget, len(dots))
	for index := 0; index < len(dots); index++ {
		dot, ok := dots[index]
		if !ok || dot == nil {
			return PublicChallenge{}, Solution{}, ErrInvalidProof
		}
		targets[index] = boundedClickTarget(dot.X, dot.Y, dot.Width, dot.Height, imageData.Width, imageData.Height)
	}
	public := PublicChallenge{Type: TypeClick, Image: imageData, PromptImage: &prompt, RequiredPoints: len(targets)}
	solution := Solution{Type: TypeClick, CanvasWidth: imageData.Width, CanvasHeight: imageData.Height, ClickTargets: targets}
	if err = solution.validate(); err != nil {
		return PublicChallenge{}, Solution{}, err
	}
	return public, solution, nil
}

func boundedClickTarget(x, y, width, height, canvasWidth, canvasHeight int) clickTarget {
	return clickTarget{X: x, Y: y, Width: min(width, canvasWidth-x), Height: min(height, canvasHeight-y)}
}

func (service *Service) generateSlide(drag bool) (PublicChallenge, Solution, error) {
	builder := slide.NewBuilder()
	builder.SetResources(slide.WithGraphImages(service.resources.tiles), slide.WithBackgrounds(service.resources.backgrounds))
	instance := builder.Make()
	kind := TypeSlide
	if drag {
		instance = builder.MakeDragDrop()
		kind = TypeDrag
	}
	data, err := instance.Generate()
	if err != nil {
		return PublicChallenge{}, Solution{}, fmt.Errorf("generate %s captcha: %w", kind, err)
	}
	imageData, err := jpegImage(data.GetMasterImage().Get(), data.GetMasterImage().ToBase64Data)
	if err != nil {
		return PublicChallenge{}, Solution{}, err
	}
	tile, err := pngImage(data.GetTileImage().Get(), data.GetTileImage().ToBase64Data)
	if err != nil {
		return PublicChallenge{}, Solution{}, err
	}
	block := data.GetData()
	if block == nil {
		return PublicChallenge{}, Solution{}, ErrInvalidProof
	}
	initial := Point{X: block.DX, Y: block.DY}
	target := Point{X: block.X, Y: block.Y}
	public := PublicChallenge{Type: kind, Image: imageData, TileImage: &tile, InitialPoint: &initial}
	solution := Solution{Type: kind, CanvasWidth: imageData.Width, CanvasHeight: imageData.Height, Point: &target}
	if err = solution.validate(); err != nil || !initial.valid(imageData.Width, imageData.Height) {
		return PublicChallenge{}, Solution{}, ErrInvalidProof
	}
	return public, solution, nil
}

func (service *Service) generateRotate() (PublicChallenge, Solution, error) {
	builder := rotate.NewBuilder()
	builder.SetResources(rotate.WithImages(service.resources.backgrounds))
	data, err := builder.Make().Generate()
	if err != nil {
		return PublicChallenge{}, Solution{}, fmt.Errorf("generate rotate captcha: %w", err)
	}
	imageData, err := pngImage(data.GetMasterImage().Get(), data.GetMasterImage().ToBase64Data)
	if err != nil {
		return PublicChallenge{}, Solution{}, err
	}
	thumb, err := pngImage(data.GetThumbImage().Get(), data.GetThumbImage().ToBase64Data)
	if err != nil {
		return PublicChallenge{}, Solution{}, err
	}
	block := data.GetData()
	if block == nil {
		return PublicChallenge{}, Solution{}, ErrInvalidProof
	}
	angle := block.Angle
	public := PublicChallenge{Type: TypeRotate, Image: imageData, ThumbImage: &thumb}
	solution := Solution{Type: TypeRotate, CanvasWidth: imageData.Width, CanvasHeight: imageData.Height, Angle: &angle}
	if err = solution.validate(); err != nil {
		return PublicChallenge{}, Solution{}, err
	}
	return public, solution, nil
}

func jpegImage(value image.Image, encode func() (string, error)) (Image, error) {
	return encodedImage(value, "image/jpeg", encode)
}

func pngImage(value image.Image, encode func() (string, error)) (Image, error) {
	return encodedImage(value, "image/png", encode)
}

func encodedImage(value image.Image, mimeType string, encode func() (string, error)) (Image, error) {
	if value == nil {
		return Image{}, ErrInvalidProof
	}
	bounds := value.Bounds()
	if bounds.Dx() < 1 || bounds.Dy() < 1 || bounds.Dx() > maxCoordinate || bounds.Dy() > maxCoordinate {
		return Image{}, ErrInvalidProof
	}
	encoded, err := encode()
	if err != nil {
		return Image{}, fmt.Errorf("encode captcha image: %w", err)
	}
	if encoded == "" {
		return Image{}, ErrInvalidProof
	}
	return Image{Base64: encoded, MimeType: mimeType, Width: bounds.Dx(), Height: bounds.Dy()}, nil
}

func Validate(solution Solution, response Response) (bool, error) {
	if err := solution.validate(); err != nil || response.Type != solution.Type {
		return false, ErrInvalidResponse
	}
	switch solution.Type {
	case TypeClick:
		if len(response.Points) != len(solution.ClickTargets) || response.Point != nil || response.Angle != nil {
			return false, ErrInvalidResponse
		}
		for index, point := range response.Points {
			if !point.valid(solution.CanvasWidth, solution.CanvasHeight) {
				return false, ErrInvalidResponse
			}
			target := solution.ClickTargets[index]
			if !click.Validate(point.X, point.Y, target.X, target.Y, target.Width, target.Height, clickPadding) {
				return false, nil
			}
		}
		return true, nil
	case TypeSlide, TypeDrag:
		if response.Point == nil || len(response.Points) != 0 || response.Angle != nil || !response.Point.valid(solution.CanvasWidth, solution.CanvasHeight) {
			return false, ErrInvalidResponse
		}
		return slide.Validate(response.Point.X, response.Point.Y, solution.Point.X, solution.Point.Y, slidePadding), nil
	case TypeRotate:
		if response.Angle == nil || *response.Angle < 0 || *response.Angle > 360 || len(response.Points) != 0 || response.Point != nil {
			return false, ErrInvalidResponse
		}
		return rotate.Validate(*response.Angle, *solution.Angle, rotatePadding), nil
	default:
		return false, ErrInvalidType
	}
}
