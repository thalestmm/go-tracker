package tracker

import (
	"testing"

	"gocv.io/x/gocv"
)

// fillRect sets pixels in a grayscale Mat to a given value.
func fillRect(mat *gocv.Mat, x0, y0, x1, y1 int, val uint8) {
	for y := y0; y < y1; y++ {
		for x := x0; x < x1; x++ {
			mat.SetUCharAt(y, x, val)
		}
	}
}

func TestMatcherFindsKnownPosition(t *testing.T) {
	// Create a 100x100 gray frame (all black)
	frame := gocv.NewMatWithSize(100, 100, gocv.MatTypeCV8UC1)
	defer func() { _ = frame.Close() }()

	// Draw a white 20x20 square at position (40,40)-(60,60)
	fillRect(&frame, 40, 40, 60, 60, 255)

	// Extract template centered at (50,50) with half-size 10 → 21x21
	tmpl, _ := ExtractTemplate(frame, 50, 50, 10)
	defer func() { _ = tmpl.Close() }()

	m := NewMatcher()
	defer m.Close()

	x, y, conf := m.Match(frame, tmpl)

	// MatchTemplate returns top-left of best match
	if x < 38 || x > 42 || y < 38 || y > 42 {
		t.Errorf("match position (%d,%d) too far from expected (~40,40)", x, y)
	}
	if conf < 0.9 {
		t.Errorf("confidence %f too low, expected > 0.9", conf)
	}
}

func TestMatcherFindsMovedTarget(t *testing.T) {
	frame1 := gocv.NewMatWithSize(100, 100, gocv.MatTypeCV8UC1)
	defer func() { _ = frame1.Close() }()
	frame2 := gocv.NewMatWithSize(100, 100, gocv.MatTypeCV8UC1)
	defer func() { _ = frame2.Close() }()

	// Frame 1: square at (30,30)-(50,50)
	fillRect(&frame1, 30, 30, 50, 50, 255)
	// Frame 2: square at (50,50)-(70,70)
	fillRect(&frame2, 50, 50, 70, 70, 255)

	// Template from frame 1 centered at (40,40)
	tmpl, _ := ExtractTemplate(frame1, 40, 40, 10)
	defer func() { _ = tmpl.Close() }()

	m := NewMatcher()
	defer m.Close()

	x, y, conf := m.Match(frame2, tmpl)

	// Should find the square near its new position
	if x < 48 || x > 52 || y < 48 || y > 52 {
		t.Errorf("match position (%d,%d) too far from expected (~50,50)", x, y)
	}
	if conf < 0.8 {
		t.Errorf("confidence %f too low", conf)
	}
}

func TestMatcherLowConfidenceOnBlank(t *testing.T) {
	frame := gocv.NewMatWithSize(100, 100, gocv.MatTypeCV8UC1)
	defer func() { _ = frame.Close() }()
	fillRect(&frame, 40, 40, 60, 60, 255)

	tmpl, _ := ExtractTemplate(frame, 50, 50, 10)
	defer func() { _ = tmpl.Close() }()

	// Search in a uniform gray frame — no matching pattern
	gray := gocv.NewMatWithSize(100, 100, gocv.MatTypeCV8UC1)
	defer func() { _ = gray.Close() }()
	fillRect(&gray, 0, 0, 100, 100, 128) // uniform mid-gray

	m := NewMatcher()
	defer m.Close()

	_, _, conf := m.Match(gray, tmpl)

	if conf > 0.5 {
		t.Errorf("expected low confidence on uniform frame, got %f", conf)
	}
}
