package tracker

import (
	"image"

	"gocv.io/x/gocv"

	"github.com/thalesmeier/go-tracker/export"
)

type State int

const (
	StateIdle State = iota
	StateTracking
	StatePausedForRealignment
	StateDone
)

type Tracker struct {
	config  Config
	matcher Matcher
	state   State

	template gocv.Mat
	lastPos  image.Point
	prevPos  image.Point
	gray     gocv.Mat
	fps      float64

	points []export.TrackPoint
}

func New(cfg Config, fps float64) *Tracker {
	return &Tracker{
		config:  cfg,
		matcher: NewMatcher(),
		state:   StateIdle,
		gray:    gocv.NewMat(),
		fps:     fps,
	}
}

func (t *Tracker) Initialize(frame gocv.Mat, x, y int) {
	gocv.CvtColor(frame, &t.gray, gocv.ColorBGRToGray)
	tmpl, _ := ExtractTemplate(t.gray, x, y, t.config.TemplateSize)
	t.template = tmpl
	t.lastPos = image.Pt(x, y)
	t.prevPos = t.lastPos
	t.state = StateTracking
}

func (t *Tracker) ProcessFrame(frame gocv.Mat, frameNum int) (State, *export.TrackPoint) {
	if t.state != StateTracking {
		return t.state, nil
	}

	gocv.CvtColor(frame, &t.gray, gocv.ColorBGRToGray)

	margin := t.config.SearchMargin
	if t.config.AdaptiveSearch {
		vx := float64(t.lastPos.X - t.prevPos.X)
		vy := float64(t.lastPos.Y - t.prevPos.Y)
		margin = AdaptiveMargin(t.config.SearchMargin, vx, vy, t.config.MaxSearchMargin)
	}

	searchRegion, searchRect := ExtractSearchRegion(
		t.gray, t.lastPos.X, t.lastPos.Y,
		t.config.TemplateSize, margin,
	)

	relX, relY, confidence := t.matcher.Match(searchRegion, t.template)
	searchRegion.Close()

	absX := searchRect.Min.X + relX + t.config.TemplateSize
	absY := searchRect.Min.Y + relY + t.config.TemplateSize

	if confidence < t.config.ConfidenceThreshold {
		t.state = StatePausedForRealignment
		return t.state, nil
	}

	tp := export.TrackPoint{
		Time:       float64(frameNum) / t.fps,
		X:          absX,
		Y:          absY,
		Confidence: confidence,
	}
	t.points = append(t.points, tp)

	t.prevPos = t.lastPos
	t.lastPos = image.Pt(absX, absY)

	return t.state, &tp
}

func (t *Tracker) Realign(frame gocv.Mat, x, y int) {
	gocv.CvtColor(frame, &t.gray, gocv.ColorBGRToGray)
	if !t.template.Empty() {
		t.template.Close()
	}
	tmpl, _ := ExtractTemplate(t.gray, x, y, t.config.TemplateSize)
	t.template = tmpl
	t.lastPos = image.Pt(x, y)
	t.prevPos = t.lastPos
	t.state = StateTracking
}

func (t *Tracker) Resume() {
	t.state = StateTracking
}

func (t *Tracker) State() State {
	return t.state
}

func (t *Tracker) LastPos() image.Point {
	return t.lastPos
}

func (t *Tracker) Points() []export.TrackPoint {
	return t.points
}

func (t *Tracker) Confidence() float64 {
	return t.config.ConfidenceThreshold
}

func (t *Tracker) Close() {
	if !t.template.Empty() {
		t.template.Close()
	}
	t.gray.Close()
	t.matcher.Close()
}
