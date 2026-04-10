package tracker

import (
	"image"
	"math"

	"gocv.io/x/gocv"
)

func clampRect(r image.Rectangle, w, h int) image.Rectangle {
	if r.Min.X < 0 {
		r.Min.X = 0
	}
	if r.Min.Y < 0 {
		r.Min.Y = 0
	}
	if r.Max.X > w {
		r.Max.X = w
	}
	if r.Max.Y > h {
		r.Max.Y = h
	}
	return r
}

// ExtractTemplate clones a (2*halfSize+1)² region centered on (cx, cy).
func ExtractTemplate(frame gocv.Mat, cx, cy, halfSize int) (gocv.Mat, image.Rectangle) {
	rect := image.Rect(cx-halfSize, cy-halfSize, cx+halfSize+1, cy+halfSize+1)
	rect = clampRect(rect, frame.Cols(), frame.Rows())
	region := frame.Region(rect)
	tmpl := region.Clone()
	region.Close()
	return tmpl, rect
}

// ExtractSearchRegion returns a zero-copy region view for matching.
// The caller must NOT close the returned Mat (it shares memory with frame).
func ExtractSearchRegion(frame gocv.Mat, cx, cy, templateHalf, searchMargin int) (gocv.Mat, image.Rectangle) {
	margin := templateHalf + searchMargin
	rect := image.Rect(cx-margin, cy-margin, cx+margin+1, cy+margin+1)
	rect = clampRect(rect, frame.Cols(), frame.Rows())
	return frame.Region(rect), rect
}

func AdaptiveMargin(baseMargin int, vx, vy float64, maxMargin int) int {
	speed := math.Sqrt(vx*vx + vy*vy)
	margin := float64(baseMargin) + 2.0*speed
	if margin > float64(maxMargin) {
		margin = float64(maxMargin)
	}
	return int(margin)
}
