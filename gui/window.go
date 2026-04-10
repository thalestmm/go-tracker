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

// WaitClick displays a frame with a prompt and blocks until the user clicks.
func (w *Window) WaitClick(frame gocv.Mat, prompt string) image.Point {
	display := frame.Clone()
	defer display.Close()

	gocv.PutText(&display, prompt, image.Pt(10, 30),
		gocv.FontHersheyPlain, 1.0,
		color.RGBA{0, 255, 0, 0}, 2)

	w.win.IMShow(display)

	// Drain any stale clicks
	select {
	case <-w.clickCh:
	default:
	}

	for {
		w.win.WaitKey(30)
		select {
		case pt := <-w.clickCh:
			return pt
		default:
		}
	}
}

// ShowFrame displays the frame with optional tracking overlay.
// Returns the key pressed (or -1 if none).
func (w *Window) ShowFrame(frame gocv.Mat, overlay *Overlay, waitMs int) int {
	display := frame.Clone()
	defer display.Close()

	if overlay != nil {
		green := color.RGBA{0, 255, 0, 0}
		yellow := color.RGBA{255, 255, 0, 0}
		red := color.RGBA{0, 0, 255, 0}

		// Crosshair at track position
		p := overlay.TrackPos
		gocv.Line(&display, image.Pt(p.X-10, p.Y), image.Pt(p.X+10, p.Y), green, 2)
		gocv.Line(&display, image.Pt(p.X, p.Y-10), image.Pt(p.X, p.Y+10), green, 2)

		// ROI rectangle
		if !overlay.ROIRect.Empty() {
			gocv.Rectangle(&display, overlay.ROIRect, yellow, 1)
		}

		// Confidence text
		confColor := green
		if overlay.Confidence < 0.7 {
			confColor = red
		}
		confStr := fmt.Sprintf("Conf: %.2f", overlay.Confidence)
		gocv.PutText(&display, confStr, image.Pt(10, 25),
			gocv.FontHersheyPlain, 0.8, confColor, 1)

		// Status text
		if overlay.Status != "" {
			gocv.PutText(&display, overlay.Status, image.Pt(10, 50),
				gocv.FontHersheyPlain, 0.8, yellow, 1)
		}
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

func (w *Window) Close() {
	w.win.Close()
}
