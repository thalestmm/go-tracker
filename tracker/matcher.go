package tracker

import (
	"gocv.io/x/gocv"
)

type Matcher interface {
	Match(searchRegion, template gocv.Mat) (x, y int, confidence float64)
	Close()
}

type cpuMatcher struct {
	result gocv.Mat
	mask   gocv.Mat
}

func NewMatcher() Matcher {
	return &cpuMatcher{
		result: gocv.NewMat(),
		mask:   gocv.NewMat(),
	}
}

func (m *cpuMatcher) Match(searchRegion, template gocv.Mat) (int, int, float64) {
	_ = gocv.MatchTemplate(searchRegion, template, &m.result, gocv.TmCcoeffNormed, m.mask)
	_, maxVal, _, maxLoc := gocv.MinMaxLoc(m.result)
	return maxLoc.X, maxLoc.Y, float64(maxVal)
}

func (m *cpuMatcher) Close() {
	_ = m.result.Close()
	_ = m.mask.Close()
}
