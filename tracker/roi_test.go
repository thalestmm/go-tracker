package tracker

import (
	"image"
	"math"
	"testing"

	"gocv.io/x/gocv"
)

func TestClampRectInside(t *testing.T) {
	r := image.Rect(10, 10, 50, 50)
	result := clampRect(r, 100, 100)
	if result != r {
		t.Errorf("got %v, want %v", result, r)
	}
}

func TestClampRectOutsideLeft(t *testing.T) {
	r := image.Rect(-10, 10, 50, 50)
	result := clampRect(r, 100, 100)
	if result.Min.X != 0 {
		t.Errorf("Min.X: got %d, want 0", result.Min.X)
	}
}

func TestClampRectOutsideRight(t *testing.T) {
	r := image.Rect(10, 10, 110, 50)
	result := clampRect(r, 100, 100)
	if result.Max.X != 100 {
		t.Errorf("Max.X: got %d, want 100", result.Max.X)
	}
}

func TestClampRectOutsideTop(t *testing.T) {
	r := image.Rect(10, -5, 50, 50)
	result := clampRect(r, 100, 100)
	if result.Min.Y != 0 {
		t.Errorf("Min.Y: got %d, want 0", result.Min.Y)
	}
}

func TestClampRectOutsideBottom(t *testing.T) {
	r := image.Rect(10, 10, 50, 150)
	result := clampRect(r, 100, 100)
	if result.Max.Y != 100 {
		t.Errorf("Max.Y: got %d, want 100", result.Max.Y)
	}
}

func TestClampRectAllEdges(t *testing.T) {
	r := image.Rect(-10, -10, 200, 200)
	result := clampRect(r, 100, 80)
	expected := image.Rect(0, 0, 100, 80)
	if result != expected {
		t.Errorf("got %v, want %v", result, expected)
	}
}

func TestAdaptiveMarginZeroVelocity(t *testing.T) {
	result := AdaptiveMargin(40, 0, 0, 120)
	if result != 40 {
		t.Errorf("got %d, want 40", result)
	}
}

func TestAdaptiveMarginModerateVelocity(t *testing.T) {
	// velocity = sqrt(3^2 + 4^2) = 5, margin = 40 + 2*5 = 50
	result := AdaptiveMargin(40, 3, 4, 120)
	if result != 50 {
		t.Errorf("got %d, want 50", result)
	}
}

func TestAdaptiveMarginCapped(t *testing.T) {
	// Large velocity, should be capped at max
	result := AdaptiveMargin(40, 100, 100, 120)
	if result != 120 {
		t.Errorf("got %d, want 120 (capped)", result)
	}
}

func TestExtractTemplate(t *testing.T) {
	// Create 100x100 grayscale frame
	frame := gocv.NewMatWithSize(100, 100, gocv.MatTypeCV8UC1)
	defer frame.Close()

	tmpl, rect := ExtractTemplate(frame, 50, 50, 10)
	defer tmpl.Close()

	// Template should be 21x21 (2*10+1)
	if tmpl.Cols() != 21 || tmpl.Rows() != 21 {
		t.Errorf("template size: got %dx%d, want 21x21", tmpl.Cols(), tmpl.Rows())
	}
	if rect.Min.X != 40 || rect.Min.Y != 40 {
		t.Errorf("rect min: got (%d,%d), want (40,40)", rect.Min.X, rect.Min.Y)
	}
}

func TestExtractTemplateNearEdge(t *testing.T) {
	frame := gocv.NewMatWithSize(100, 100, gocv.MatTypeCV8UC1)
	defer frame.Close()

	// Near top-left corner: should be clamped
	tmpl, rect := ExtractTemplate(frame, 3, 3, 10)
	defer tmpl.Close()

	if rect.Min.X != 0 || rect.Min.Y != 0 {
		t.Errorf("rect min: got (%d,%d), want (0,0)", rect.Min.X, rect.Min.Y)
	}
	// Template will be smaller than 21x21 due to clamping
	if tmpl.Cols() > 21 || tmpl.Rows() > 21 {
		t.Errorf("template too large: %dx%d", tmpl.Cols(), tmpl.Rows())
	}
}

func TestExtractSearchRegion(t *testing.T) {
	frame := gocv.NewMatWithSize(200, 200, gocv.MatTypeCV8UC1)
	defer frame.Close()

	region, rect := ExtractSearchRegion(frame, 100, 100, 10, 30)
	defer region.Close()

	// Region should be centered on (100,100) with margin 10+30=40 in each direction
	expectedSize := 2*40 + 1 // 81
	if region.Cols() != expectedSize || region.Rows() != expectedSize {
		t.Errorf("region size: got %dx%d, want %dx%d", region.Cols(), region.Rows(), expectedSize, expectedSize)
	}
	_ = rect
	_ = math.Abs // suppress unused import
}
