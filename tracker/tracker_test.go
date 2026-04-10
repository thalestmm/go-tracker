package tracker

import (
	"image"
	"testing"

	"gocv.io/x/gocv"
)

// makeFrame creates a BGR frame with a white square at the given position.
func makeFrame(width, height, squareX, squareY, squareSize int) gocv.Mat {
	frame := gocv.NewMatWithSize(height, width, gocv.MatTypeCV8UC3)
	half := squareSize / 2
	for y := squareY - half; y < squareY+half; y++ {
		for x := squareX - half; x < squareX+half; x++ {
			if x >= 0 && x < width && y >= 0 && y < height {
				frame.SetUCharAt(y, x*3, 255)   // B
				frame.SetUCharAt(y, x*3+1, 255) // G
				frame.SetUCharAt(y, x*3+2, 255) // R
			}
		}
	}
	return frame
}

func TestInitialize(t *testing.T) {
	cfg := DefaultConfig()
	tr := New(cfg, 30.0)
	defer tr.Close()

	frame := makeFrame(200, 200, 100, 100, 20)
	defer frame.Close()

	tr.Initialize(frame, 100, 100)

	if tr.State() != StateTracking {
		t.Errorf("expected StateTracking, got %d", tr.State())
	}
	if tr.LastPos() != image.Pt(100, 100) {
		t.Errorf("expected last pos (100,100), got %v", tr.LastPos())
	}
}

func TestProcessFrameTracksMovingObject(t *testing.T) {
	cfg := DefaultConfig()
	tr := New(cfg, 30.0)
	defer tr.Close()

	// Initialize with square at (100, 100)
	frame1 := makeFrame(200, 200, 100, 100, 20)
	defer frame1.Close()
	tr.Initialize(frame1, 100, 100)

	// Move square to (105, 100) — small displacement
	frame2 := makeFrame(200, 200, 105, 100, 20)
	defer frame2.Close()

	state, tp := tr.ProcessFrame(frame2, 1)

	if state != StateTracking {
		t.Errorf("expected StateTracking, got %d", state)
	}
	if tp == nil {
		t.Fatal("expected non-nil TrackPoint")
	}
	// Should find the object near (105, 100) — allow some tolerance
	if abs(tp.X-105) > 5 || abs(tp.Y-100) > 5 {
		t.Errorf("tracked position (%d,%d) too far from expected (105,100)", tp.X, tp.Y)
	}
}

func TestProcessFrameLostTrack(t *testing.T) {
	cfg := DefaultConfig()
	cfg.ConfidenceThreshold = 0.8 // high threshold to trigger loss easily
	tr := New(cfg, 30.0)
	defer tr.Close()

	// Initialize with a distinctive pattern
	frame1 := makeFrame(200, 200, 100, 100, 20)
	defer frame1.Close()
	tr.Initialize(frame1, 100, 100)

	// Completely blank frame — should lose track
	blank := gocv.NewMatWithSize(200, 200, gocv.MatTypeCV8UC3)
	defer blank.Close()

	state, tp := tr.ProcessFrame(blank, 1)

	if state != StatePausedForRealignment {
		t.Errorf("expected StatePausedForRealignment, got %d", state)
	}
	if tp != nil {
		t.Error("expected nil TrackPoint on lost track")
	}
}

func TestResume(t *testing.T) {
	cfg := DefaultConfig()
	tr := New(cfg, 30.0)
	defer tr.Close()

	frame := makeFrame(200, 200, 100, 100, 20)
	defer frame.Close()
	tr.Initialize(frame, 100, 100)

	// Force paused state
	blank := gocv.NewMatWithSize(200, 200, gocv.MatTypeCV8UC3)
	defer blank.Close()
	tr.ProcessFrame(blank, 1)

	tr.Resume()
	if tr.State() != StateTracking {
		t.Errorf("expected StateTracking after Resume, got %d", tr.State())
	}
}

func TestRealign(t *testing.T) {
	cfg := DefaultConfig()
	tr := New(cfg, 30.0)
	defer tr.Close()

	frame := makeFrame(200, 200, 100, 100, 20)
	defer frame.Close()
	tr.Initialize(frame, 100, 100)

	// Realign to new position
	frame2 := makeFrame(200, 200, 150, 150, 20)
	defer frame2.Close()
	tr.Realign(frame2, 150, 150)

	if tr.State() != StateTracking {
		t.Errorf("expected StateTracking after Realign, got %d", tr.State())
	}
	if tr.LastPos() != image.Pt(150, 150) {
		t.Errorf("expected last pos (150,150), got %v", tr.LastPos())
	}
}

func TestPointsAccumulate(t *testing.T) {
	cfg := DefaultConfig()
	tr := New(cfg, 30.0)
	defer tr.Close()

	frame := makeFrame(200, 200, 100, 100, 20)
	defer frame.Close()
	tr.Initialize(frame, 100, 100)

	// Process same frame a few times
	for i := 1; i <= 3; i++ {
		tr.ProcessFrame(frame, i)
	}

	points := tr.Points()
	if len(points) != 3 {
		t.Errorf("expected 3 points, got %d", len(points))
	}
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
