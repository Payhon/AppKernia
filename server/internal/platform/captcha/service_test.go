package captcha

import (
	"encoding/base64"
	"fmt"
	"sync"
	"testing"
)

func TestServiceGeneratesAndValidatesEverySupportedType(t *testing.T) {
	service, err := NewService()
	if err != nil {
		t.Fatalf("create service: %v", err)
	}
	for _, kind := range []Type{TypeClick, TypeSlide, TypeDrag, TypeRotate} {
		t.Run(string(kind), func(t *testing.T) {
			public, solution, generateErr := service.Generate(kind)
			if generateErr != nil {
				t.Fatalf("generate: %v", generateErr)
			}
			if public.Type != kind || public.Image.Width < 1 || public.Image.Height < 1 || public.Image.MimeType == "" {
				t.Fatalf("invalid public challenge: %+v", public)
			}
			if _, decodeErr := base64.StdEncoding.DecodeString(public.Image.Base64); decodeErr != nil {
				t.Fatalf("master image is not raw base64: %v", decodeErr)
			}
			switch kind {
			case TypeClick:
				if public.PromptImage == nil || public.RequiredPoints != len(solution.ClickTargets) || public.TileImage != nil || public.InitialPoint != nil || public.ThumbImage != nil {
					t.Fatalf("invalid click payload: %+v", public)
				}
			case TypeSlide, TypeDrag:
				if public.TileImage == nil || public.InitialPoint == nil || public.PromptImage != nil || public.RequiredPoints != 0 || public.ThumbImage != nil {
					t.Fatalf("invalid %s payload: %+v", kind, public)
				}
			case TypeRotate:
				if public.ThumbImage == nil || public.PromptImage != nil || public.RequiredPoints != 0 || public.TileImage != nil || public.InitialPoint != nil {
					t.Fatalf("invalid rotate payload: %+v", public)
				}
			}
			valid, validateErr := Validate(solution, validResponse(solution))
			if validateErr != nil || !valid {
				t.Fatalf("generated solution must validate: valid=%v err=%v", valid, validateErr)
			}
		})
	}
}

func TestServiceSupportsConcurrentGeneration(t *testing.T) {
	service, err := NewService()
	if err != nil {
		t.Fatal(err)
	}
	kinds := []Type{TypeClick, TypeSlide, TypeDrag, TypeRotate}
	errorsFound := make(chan error, len(kinds)*2)
	var wait sync.WaitGroup
	for index := 0; index < len(kinds)*2; index++ {
		kind := kinds[index%len(kinds)]
		wait.Add(1)
		go func() {
			defer wait.Done()
			public, solution, generateErr := service.Generate(kind)
			if generateErr != nil {
				errorsFound <- generateErr
				return
			}
			valid, validateErr := Validate(solution, validResponse(solution))
			if validateErr != nil || !valid || public.Type != kind {
				errorsFound <- fmt.Errorf("type %s: valid=%v err=%v public_type=%s", kind, valid, validateErr, public.Type)
			}
		}()
	}
	wait.Wait()
	close(errorsFound)
	for generateErr := range errorsFound {
		t.Errorf("concurrent generation failed: %v", generateErr)
	}
}

func TestClickValidationPreservesPromptOrderAndRejectsMalformedUnion(t *testing.T) {
	solution := Solution{
		Type: TypeClick, CanvasWidth: 200, CanvasHeight: 100,
		ClickTargets: []clickTarget{
			{X: 10, Y: 10, Width: 20, Height: 20},
			{X: 140, Y: 50, Width: 20, Height: 20},
		},
	}
	ordered := Response{Type: TypeClick, Points: []Point{{X: 20, Y: 20}, {X: 150, Y: 60}}}
	valid, err := Validate(solution, ordered)
	if err != nil || !valid {
		t.Fatalf("ordered points must validate: valid=%v err=%v", valid, err)
	}
	swapped := Response{Type: TypeClick, Points: []Point{{X: 150, Y: 60}, {X: 20, Y: 20}}}
	if valid, err = Validate(solution, swapped); err != nil || valid {
		t.Fatalf("swapped points must fail without becoming malformed: valid=%v err=%v", valid, err)
	}
	extra := 1
	ordered.Angle = &extra
	if _, err = Validate(solution, ordered); err != ErrInvalidResponse {
		t.Fatalf("mode-specific response union must be strict, got %v", err)
	}
}

func TestClickTargetIsClippedToGeneratedCanvas(t *testing.T) {
	imageWidth, imageHeight := 300, 220
	target := boundedClickTarget(103, 175, 48, 47, imageWidth, imageHeight)
	solution := Solution{Type: TypeClick, CanvasWidth: imageWidth, CanvasHeight: imageHeight, ClickTargets: []clickTarget{target}}
	if err := solution.validate(); err != nil || target.Width != 48 || target.Height != 45 {
		t.Fatalf("clipped target must stay inside the canvas: target=%+v err=%v", target, err)
	}
}

func TestValidationHonorsConfiguredToleranceBoundaries(t *testing.T) {
	clickSolution := Solution{
		Type: TypeClick, CanvasWidth: 200, CanvasHeight: 120,
		ClickTargets: []clickTarget{{X: 50, Y: 40, Width: 20, Height: 20}},
	}
	for _, test := range []struct {
		name     string
		solution Solution
		response Response
		want     bool
	}{
		{
			name: "click boundary", solution: clickSolution,
			response: Response{Type: TypeClick, Points: []Point{{X: 86, Y: 76}}}, want: true,
		},
		{
			name: "click outside", solution: clickSolution,
			response: Response{Type: TypeClick, Points: []Point{{X: 87, Y: 76}}},
		},
		{
			name: "slide boundary", solution: pointSolution(TypeSlide),
			response: Response{Type: TypeSlide, Point: &Point{X: 106, Y: 66}}, want: true,
		},
		{
			name: "slide outside", solution: pointSolution(TypeSlide),
			response: Response{Type: TypeSlide, Point: &Point{X: 107, Y: 66}},
		},
		{
			name: "drag boundary", solution: pointSolution(TypeDrag),
			response: Response{Type: TypeDrag, Point: &Point{X: 94, Y: 54}}, want: true,
		},
		{
			name: "drag outside", solution: pointSolution(TypeDrag),
			response: Response{Type: TypeDrag, Point: &Point{X: 93, Y: 54}},
		},
		{
			name: "rotate boundary", solution: rotateSolution(100),
			response: Response{Type: TypeRotate, Angle: intPointer(254)}, want: true,
		},
		{
			name: "rotate outside", solution: rotateSolution(100),
			response: Response{Type: TypeRotate, Angle: intPointer(253)},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			valid, err := Validate(test.solution, test.response)
			if err != nil || valid != test.want {
				t.Fatalf("valid=%v want=%v err=%v", valid, test.want, err)
			}
		})
	}
}

func pointSolution(kind Type) Solution {
	target := Point{X: 100, Y: 60}
	return Solution{Type: kind, CanvasWidth: 200, CanvasHeight: 120, Point: &target}
}

func rotateSolution(angle int) Solution {
	return Solution{Type: TypeRotate, CanvasWidth: 220, CanvasHeight: 220, Angle: &angle}
}

func intPointer(value int) *int { return &value }

func TestValidationRejectsMismatchedTypeAndOutOfBoundsPoint(t *testing.T) {
	target := Point{X: 50, Y: 20}
	solution := Solution{Type: TypeSlide, CanvasWidth: 100, CanvasHeight: 50, Point: &target}
	if _, err := Validate(solution, Response{Type: TypeDrag, Point: &target}); err != ErrInvalidResponse {
		t.Fatalf("mismatched response type must fail, got %v", err)
	}
	outOfBounds := Point{X: 100, Y: 20}
	if _, err := Validate(solution, Response{Type: TypeSlide, Point: &outOfBounds}); err != ErrInvalidResponse {
		t.Fatalf("out-of-bounds point must fail, got %v", err)
	}
}

func validResponse(solution Solution) Response {
	response := Response{Type: solution.Type}
	switch solution.Type {
	case TypeClick:
		response.Points = make([]Point, len(solution.ClickTargets))
		for index, target := range solution.ClickTargets {
			response.Points[index] = Point{X: target.X + target.Width/2, Y: target.Y + target.Height/2}
		}
	case TypeSlide, TypeDrag:
		point := *solution.Point
		response.Point = &point
	case TypeRotate:
		angle := (360 - *solution.Angle) % 360
		response.Angle = &angle
	}
	return response
}
