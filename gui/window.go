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
	Trail      []image.Point // recent positions for trajectory trail
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

	// Trajectory trail
	if len(overlay.Trail) > 1 {
		n := len(overlay.Trail)
		for i := 1; i < n; i++ {
			// Fade from dim to bright: older points are dimmer
			alpha := float64(i) / float64(n)
			c := color.RGBA{
				R: uint8(alpha * 255),
				G: uint8((1 - alpha) * 200),
				B: 0,
				A: 0,
			}
			gocv.Line(display, overlay.Trail[i-1], overlay.Trail[i], c, 1)
		}
	}

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

// WaitClickZoom works like WaitClick but after the click, shows a 4x zoomed inset
// of the selected region so the user can confirm. Press Enter/Space to accept, or click again to reselect.
func (w *Window) WaitClickZoom(frame gocv.Mat, prompt string, zoomRadius int) (image.Point, bool) {
	for {
		pt, clicked := w.WaitClick(frame, prompt, nil)
		if !clicked {
			return pt, false
		}

		// Show frame with zoom inset for confirmation
		display := frame.Clone()

		// Draw crosshair at selected point
		green := color.RGBA{0, 255, 0, 0}
		gocv.Line(&display, image.Pt(pt.X-10, pt.Y), image.Pt(pt.X+10, pt.Y), green, 2)
		gocv.Line(&display, image.Pt(pt.X, pt.Y-10), image.Pt(pt.X, pt.Y+10), green, 2)

		// Extract and zoom the region around the click
		fw, fh := frame.Cols(), frame.Rows()
		r := zoomRadius
		x0 := pt.X - r
		y0 := pt.Y - r
		x1 := pt.X + r
		y1 := pt.Y + r
		if x0 < 0 {
			x0 = 0
		}
		if y0 < 0 {
			y0 = 0
		}
		if x1 > fw {
			x1 = fw
		}
		if y1 > fh {
			y1 = fh
		}

		roi := frame.Region(image.Rect(x0, y0, x1, y1))
		zoomSize := 4 * 2 * r // 4x magnification
		zoomed := gocv.NewMat()
		gocv.Resize(roi, &zoomed, image.Pt(zoomSize, zoomSize), 0, 0, gocv.InterpolationNearestNeighbor)
		roi.Close()

		// Draw zoom inset in top-right corner with border
		insetX := fw - zoomSize - 10
		insetY := 10
		if insetX < 0 {
			insetX = 0
		}
		insetRect := image.Rect(insetX, insetY, insetX+zoomSize, insetY+zoomSize)
		gocv.Rectangle(&display, insetRect, color.RGBA{255, 255, 255, 0}, 2)

		insetROI := display.Region(insetRect)
		zoomed.CopyTo(&insetROI)
		insetROI.Close()
		zoomed.Close()

		// Draw crosshair in center of zoom inset
		cx := insetX + zoomSize/2
		cy := insetY + zoomSize/2
		gocv.Line(&display, image.Pt(cx-8, cy), image.Pt(cx+8, cy), green, 1)
		gocv.Line(&display, image.Pt(cx, cy-8), image.Pt(cx, cy+8), green, 1)

		gocv.PutText(&display, "Enter=confirm, Click=reselect", image.Pt(10, 30),
			gocv.FontHersheyPlain, 1.0, green, 1)

		w.win.IMShow(display)
		display.Close()

		// Wait for confirmation (Enter/Space) or a new click to reselect
		select {
		case <-w.clickCh:
		default:
		}
		confirmed := false
		reselect := false
		for !confirmed && !reselect {
			key := w.win.WaitKey(30)
			if key == 13 || key == 10 || key == 32 { // Enter or Space
				confirmed = true
			}
			select {
			case <-w.clickCh:
				reselect = true
			default:
			}
		}
		if confirmed {
			return pt, true
		}
	}
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
