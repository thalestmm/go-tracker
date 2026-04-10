package gui

import (
	"fmt"
	"image"
	"image/color"
	"math"

	"gocv.io/x/gocv"
)

const (
	graphWidth     = 600
	graphMargin    = 30
	plotHeightUnit = 120
)

type GraphWindow struct {
	win         *gocv.Window
	canvas      gocv.Mat
	derivatives bool
}

func NewGraphWindow(title string, derivatives bool) *GraphWindow {
	win := gocv.NewWindow(title)
	canvas := gocv.NewMat()
	return &GraphWindow{win: win, canvas: canvas, derivatives: derivatives}
}

func (g *GraphWindow) Update(times []float64, xs, ys []int) {
	if len(times) < 2 {
		return
	}

	// Determine number of plots
	numPlots := 2 // X(t), Y(t)
	if g.derivatives {
		numPlots = 6 // + Vx(t), Vy(t), Ax(t), Ay(t)
	}

	canvasH := numPlots*plotHeightUnit + (numPlots+1)*graphMargin
	g.canvas.Close()
	g.canvas = gocv.NewMatWithSize(canvasH, graphWidth, gocv.MatTypeCV8UC3)
	fillMat(&g.canvas, color.RGBA{30, 30, 30, 0})

	white := color.RGBA{200, 200, 200, 0}
	dimWhite := color.RGBA{80, 80, 80, 0}

	tMin, tMax := times[0], times[len(times)-1]
	tRange := tMax - tMin

	plotLeft := graphMargin + 10
	plotRight := graphWidth - graphMargin/2
	plotW := plotRight - plotLeft

	// Prepare series
	type series struct {
		label string
		color color.RGBA
		vals  []float64
	}

	xf := intsToFloats(xs)
	yf := intsToFloats(ys)

	plots := []series{
		{"X(t)", color.RGBA{255, 255, 0, 0}, xf},
		{"Y(t)", color.RGBA{255, 0, 255, 0}, yf},
	}

	if g.derivatives {
		vx, vy, ax, ay := computeRealtimeDerivatives(times, xf, yf)
		plots = append(plots,
			series{"Vx(t)", color.RGBA{0, 200, 255, 0}, vx},
			series{"Vy(t)", color.RGBA{0, 255, 200, 0}, vy},
			series{"Ax(t)", color.RGBA{0, 128, 255, 0}, ax},
			series{"Ay(t)", color.RGBA{0, 255, 128, 0}, ay},
		)
	}

	// Draw each plot
	for pi, s := range plots {
		plotTop := graphMargin + pi*(plotHeightUnit+graphMargin)
		plotBot := plotTop + plotHeightUnit

		gocv.Rectangle(&g.canvas, image.Rect(plotLeft, plotTop, plotRight, plotBot), dimWhite, 1)

		// Label
		gocv.PutText(&g.canvas, s.label, image.Pt(plotLeft, plotTop-6),
			gocv.FontHersheyPlain, 0.9, s.color, 1)

		// Compute value range
		vMin, vMax := minMaxFloat(s.vals)
		if vMin == vMax {
			vMin -= 1
			vMax += 1
		}
		vRange := vMax - vMin

		// Axis value labels
		gocv.PutText(&g.canvas, fmtVal(vMax), image.Pt(2, plotTop+12),
			gocv.FontHersheyPlain, 0.65, white, 1)
		gocv.PutText(&g.canvas, fmtVal(vMin), image.Pt(2, plotBot-4),
			gocv.FontHersheyPlain, 0.65, white, 1)

		// Data lines
		for i := 1; i < len(times); i++ {
			t0 := (times[i-1] - tMin) / tRange
			t1 := (times[i] - tMin) / tRange
			px0 := plotLeft + int(t0*float64(plotW))
			px1 := plotLeft + int(t1*float64(plotW))

			n0 := 1.0 - (s.vals[i-1]-vMin)/vRange
			n1 := 1.0 - (s.vals[i]-vMin)/vRange
			py0 := plotTop + int(n0*float64(plotHeightUnit))
			py1 := plotTop + int(n1*float64(plotHeightUnit))

			gocv.Line(&g.canvas, image.Pt(px0, py0), image.Pt(px1, py1), s.color, 1)
		}
	}

	// Time labels at bottom
	lastPlotBot := graphMargin + numPlots*(plotHeightUnit+graphMargin) - graphMargin
	gocv.PutText(&g.canvas, fmt.Sprintf("%.1fs", tMin), image.Pt(plotLeft, lastPlotBot+15),
		gocv.FontHersheyPlain, 0.7, white, 1)
	gocv.PutText(&g.canvas, fmt.Sprintf("%.1fs", tMax), image.Pt(plotRight-40, lastPlotBot+15),
		gocv.FontHersheyPlain, 0.7, white, 1)

	g.win.IMShow(g.canvas)
}

func (g *GraphWindow) Close() {
	g.canvas.Close()
	g.win.Close()
}

// computeRealtimeDerivatives returns per-axis velocity and acceleration arrays.
func computeRealtimeDerivatives(times, xs, ys []float64) (vxOut, vyOut, axOut, ayOut []float64) {
	n := len(times)
	vx := make([]float64, n)
	vy := make([]float64, n)
	ax := make([]float64, n)
	ay := make([]float64, n)

	// Velocity via central differences
	for i := 1; i < n-1; i++ {
		dt := times[i+1] - times[i-1]
		if dt > 0 {
			vx[i] = (xs[i+1] - xs[i-1]) / dt
			vy[i] = (ys[i+1] - ys[i-1]) / dt
		}
	}
	if n > 1 {
		dt := times[1] - times[0]
		if dt > 0 {
			vx[0] = (xs[1] - xs[0]) / dt
			vy[0] = (ys[1] - ys[0]) / dt
		}
		dt = times[n-1] - times[n-2]
		if dt > 0 {
			vx[n-1] = (xs[n-1] - xs[n-2]) / dt
			vy[n-1] = (ys[n-1] - ys[n-2]) / dt
		}
	}

	// Acceleration via central differences on velocity
	for i := 1; i < n-1; i++ {
		dt := times[i+1] - times[i-1]
		if dt > 0 {
			ax[i] = (vx[i+1] - vx[i-1]) / dt
			ay[i] = (vy[i+1] - vy[i-1]) / dt
		}
	}
	if n > 1 {
		dt := times[1] - times[0]
		if dt > 0 {
			ax[0] = (vx[1] - vx[0]) / dt
			ay[0] = (vy[1] - vy[0]) / dt
		}
		dt = times[n-1] - times[n-2]
		if dt > 0 {
			ax[n-1] = (vx[n-1] - vx[n-2]) / dt
			ay[n-1] = (vy[n-1] - vy[n-2]) / dt
		}
	}

	return vx, vy, ax, ay
}

func fillMat(mat *gocv.Mat, c color.RGBA) {
	gocv.Rectangle(mat, image.Rect(0, 0, mat.Cols(), mat.Rows()), c, -1)
}

func intsToFloats(vals []int) []float64 {
	out := make([]float64, len(vals))
	for i, v := range vals {
		out[i] = float64(v)
	}
	return out
}

func minMaxInt(vals []int) (int, int) {
	mn, mx := math.MaxInt, math.MinInt
	for _, v := range vals {
		if v < mn {
			mn = v
		}
		if v > mx {
			mx = v
		}
	}
	return mn, mx
}

func minMaxFloat(vals []float64) (float64, float64) {
	mn, mx := math.MaxFloat64, -math.MaxFloat64
	for _, v := range vals {
		if v < mn {
			mn = v
		}
		if v > mx {
			mx = v
		}
	}
	return mn, mx
}

func fmtVal(v float64) string {
	if math.Abs(v) >= 100 {
		return fmt.Sprintf("%.0f", v)
	}
	if math.Abs(v) >= 1 {
		return fmt.Sprintf("%.1f", v)
	}
	return fmt.Sprintf("%.2f", v)
}
