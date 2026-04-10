package gui

import (
	"fmt"
	"image"
	"image/color"

	"gocv.io/x/gocv"
)

type Overlay struct {
	TrackPos   image.Point
	ROIRect    image.Rectangle
	Confidence float64
	Status     string
	ShowAxes   bool
}

type Window struct {
	win     *gocv.Window
	clickCh chan image.Point
}

func New(title string) *Window {
	win := gocv.NewWindow(title)
	w := &Window{
		win:     win,
		clickCh: make(chan image.Point, 1),
	}
	win.SetMouseHandler(func(event int, x int, y int, flags int, userdata interface{}) {
		if event == 1 { // EVENT_LBUTTONDOWN
			select {
			case w.clickCh <- image.Pt(x, y):
			default:
			}
		}
	}, nil)
	return w
}

// WaitClick displays a frame with a prompt and blocks until the user clicks or presses Space to resume.
// If overlay is non-nil, it is drawn on the frame (useful for showing axes at last position during pause).
// Returns the clicked point and true, or zero point and false if the user pressed Space to resume.
func (w *Window) WaitClick(frame gocv.Mat, prompt string, overlay *Overlay) (image.Point, bool) {
	display := frame.Clone()
	defer display.Close()

	if overlay != nil {
		w.drawOverlay(&display, overlay)
	}

	gocv.PutText(&display, prompt, image.Pt(10, 30),
		gocv.FontHersheyPlain, 1.0,
		color.RGBA{0, 255, 0, 0}, 1)

	w.win.IMShow(display)

	// Drain any stale clicks
	select {
	case <-w.clickCh:
	default:
	}

	for {
		key := w.win.WaitKey(30)
		if key == 32 || key == 'p' || key == 'P' { // Space or P to resume
			return image.Point{}, false
		}
		select {
		case pt := <-w.clickCh:
			return pt, true
		default:
		}
	}
}

func (w *Window) drawOverlay(display *gocv.Mat, overlay *Overlay) {
	green := color.RGBA{0, 255, 0, 0}
	yellow := color.RGBA{255, 255, 0, 0}
	red := color.RGBA{0, 0, 255, 0}

	// Crosshair at track position
	p := overlay.TrackPos
	gocv.Line(display, image.Pt(p.X-10, p.Y), image.Pt(p.X+10, p.Y), green, 2)
	gocv.Line(display, image.Pt(p.X, p.Y-10), image.Pt(p.X, p.Y+10), green, 2)

	// Full-frame axes through tracking point
	if overlay.ShowAxes {
		cyan := color.RGBA{255, 255, 0, 0}
		fw := display.Cols()
		fh := display.Rows()
		gocv.Line(display, image.Pt(0, p.Y), image.Pt(fw, p.Y), cyan, 1)
		gocv.Line(display, image.Pt(p.X, 0), image.Pt(p.X, fh), cyan, 1)
	}

	// ROI rectangle
	if !overlay.ROIRect.Empty() {
		gocv.Rectangle(display, overlay.ROIRect, yellow, 1)
	}

	// Confidence text
	confColor := green
	if overlay.Confidence < 0.7 {
		confColor = red
	}
	confStr := fmt.Sprintf("Conf: %.2f", overlay.Confidence)
	gocv.PutText(display, confStr, image.Pt(10, 25),
		gocv.FontHersheyPlain, 0.8, confColor, 1)

	// Status text
	if overlay.Status != "" {
		gocv.PutText(display, overlay.Status, image.Pt(10, 50),
			gocv.FontHersheyPlain, 0.8, yellow, 1)
	}
}

// ShowFrame displays the frame with optional tracking overlay.
// Returns the key pressed (or -1 if none).
func (w *Window) ShowFrame(frame gocv.Mat, overlay *Overlay, waitMs int) int {
	display := frame.Clone()
	defer display.Close()

	if overlay != nil {
		w.drawOverlay(&display, overlay)
	}

	w.win.IMShow(display)
	return w.win.WaitKey(waitMs)
}

// CheckClick returns a click if one happened, without blocking.
func (w *Window) CheckClick() (image.Point, bool) {
	select {
	case pt := <-w.clickCh:
		return pt, true
	default:
		return image.Point{}, false
	}
}

// WaitTwoClicks displays a frame and collects two clicks for calibration.
// Both points are shown with a connecting line so the user can see the measured distance.
func (w *Window) WaitTwoClicks(frame gocv.Mat, prompt1, prompt2 string) (image.Point, image.Point) {
	cyan := color.RGBA{255, 255, 0, 0}
	magenta := color.RGBA{255, 0, 255, 0}

	// First click
	p1, _ := w.WaitClick(frame, prompt1, nil)

	// Show frame with first point marked, wait for second
	display := frame.Clone()
	defer display.Close()
	gocv.Circle(&display, p1, 5, cyan, 2)
	p2, _ := w.WaitClick(display, prompt2, nil)

	// Show both points and the line between them
	confirm := frame.Clone()
	defer confirm.Close()
	gocv.Circle(&confirm, p1, 5, cyan, 2)
	gocv.Circle(&confirm, p2, 5, magenta, 2)
	gocv.Line(&confirm, p1, p2, color.RGBA{255, 255, 255, 0}, 1)
	gocv.PutText(&confirm, "Calibration set. Enter distance in terminal.", image.Pt(10, 30),
		gocv.FontHersheyPlain, 1.0, color.RGBA{0, 255, 0, 0}, 1)
	w.win.IMShow(confirm)
	w.win.WaitKey(1)

	return p1, p2
}

// ShowTurboLabel shows the frame with a "TURBO MODE" label in the top-left.
func (w *Window) ShowTurboLabel(frame gocv.Mat) {
	display := frame.Clone()
	defer display.Close()

	gocv.PutText(&display, "TURBO MODE", image.Pt(10, 25),
		gocv.FontHersheyPlain, 1.0, color.RGBA{255, 240, 31, 0}, 1)

	w.win.IMShow(display)
}

// PollKey checks for key presses without rendering. Used in turbo mode.
func (w *Window) PollKey(waitMs int) int {
	return w.win.WaitKey(waitMs)
}

func (w *Window) Close() {
	w.win.Close()
}
