//go:build cuda

package tracker

import (
	"gocv.io/x/gocv"
	"gocv.io/x/gocv/cuda"
)

type cudaMatcher struct {
	tmatcher    cuda.TemplateMatching
	gpuTemplate cuda.GpuMat
	gpuFrame    cuda.GpuMat
	gpuResult   cuda.GpuMat
	cpuResult   gocv.Mat
	cpuFallback *cpuMatcher
}

func NewCUDAMatcher(template gocv.Mat) Matcher {
	if cuda.GetCudaEnabledDeviceCount() == 0 {
		return NewMatcher()
	}

	tm := cuda.NewTemplateMatching(gocv.MatTypeCV8UC1, gocv.TmCcoeffNormed)

	gpuTmpl := cuda.NewGpuMat()
	gpuTmpl.Upload(template)

	return &cudaMatcher{
		tmatcher:    tm,
		gpuTemplate: gpuTmpl,
		gpuFrame:    cuda.NewGpuMat(),
		gpuResult:   cuda.NewGpuMat(),
		cpuResult:   gocv.NewMat(),
		cpuFallback: &cpuMatcher{result: gocv.NewMat(), mask: gocv.NewMat()},
	}
}

func (m *cudaMatcher) Match(searchRegion, template gocv.Mat) (int, int, float64) {
	// Fall back to CPU for small regions where GPU overhead dominates
	if searchRegion.Cols()*searchRegion.Rows() < 10000 {
		return m.cpuFallback.Match(searchRegion, template)
	}

	m.gpuFrame.Upload(searchRegion)
	m.tmatcher.Match(m.gpuFrame, m.gpuTemplate, &m.gpuResult)
	m.gpuResult.Download(&m.cpuResult)

	_, maxVal, _, maxLoc := gocv.MinMaxLoc(m.cpuResult)
	return maxLoc.X, maxLoc.Y, maxVal
}

func (m *cudaMatcher) Close() {
	m.tmatcher.Close()
	m.gpuTemplate.Close()
	m.gpuFrame.Close()
	m.gpuResult.Close()
	m.cpuResult.Close()
	m.cpuFallback.Close()
}
