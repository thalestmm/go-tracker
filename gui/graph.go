package gui

import (
	"fmt"
	"image"
	"image/color"
	"math"

	"gocv.io/x/gocv"
)

const (
	graphWidth  = 600
	graphHeight = 400
	graphMargin = 40
	plotHeight  = (graphHeight - 3*graphMargin) / 2
)

type GraphWindow struct {
	win    *gocv.Window
	canvas gocv.Mat
}

func NewGraphWindow(title string) *GraphWindow {
	win := gocv.NewWindow(title)
	canvas := gocv.NewMatWithSize(graphHeight, graphWidth, gocv.MatTypeCV8UC3)
	return &GraphWindow{win: win, canvas: canvas}
}

func (g *GraphWindow) Update(times []float64, xs, ys []int) {
	if len(times) < 2 {
		return
	}

	// Clear canvas
	g.canvas.Close()
	g.canvas = gocv.NewMatWithSize(graphHeight, graphWidth, gocv.MatTypeCV8UC3)

	bg := color.RGBA{30, 30, 30, 0}
	fillMat(&g.canvas, bg)

	white := color.RGBA{200, 200, 200, 0}
	cyan := color.RGBA{255, 255, 0, 0}
	magenta := color.RGBA{255, 0, 255, 0}
	dimWhite := color.RGBA{80, 80, 80, 0}

	// Compute ranges
	tMin, tMax := times[0], times[len(times)-1]
	xMin, xMax := minMaxInt(xs)
	yMin, yMax := minMaxInt(ys)

	// Pad ranges to avoid division by zero
	if xMin == xMax {
		xMin--
		xMax++
	}
	if yMin == yMax {
		yMin--
		yMax++
	}

	// Plot areas
	plotLeft := graphMargin
	plotRight := graphWidth - graphMargin/2
	plotW := plotRight - plotLeft

	topPlotTop := graphMargin
	topPlotBot := graphMargin + plotHeight
	botPlotTop := 2*graphMargin + plotHeight
	botPlotBot := 2*graphMargin + 2*plotHeight

	// Draw grid lines and borders
	gocv.Rectangle(&g.canvas, image.Rect(plotLeft, topPlotTop, plotRight, topPlotBot), dimWhite, 1)
	gocv.Rectangle(&g.canvas, image.Rect(plotLeft, botPlotTop, plotRight, botPlotBot), dimWhite, 1)

	// Labels
	gocv.PutText(&g.canvas, "X(t)", image.Pt(plotLeft, topPlotTop-8),
		gocv.FontHersheyPlain, 0.9, cyan, 1)
	gocv.PutText(&g.canvas, "Y(t)", image.Pt(plotLeft, botPlotTop-8),
		gocv.FontHersheyPlain, 0.9, magenta, 1)

	// Axis value labels
	gocv.PutText(&g.canvas, fmt.Sprintf("%d", xMax), image.Pt(2, topPlotTop+12),
		gocv.FontHersheyPlain, 0.7, white, 1)
	gocv.PutText(&g.canvas, fmt.Sprintf("%d", xMin), image.Pt(2, topPlotBot-4),
		gocv.FontHersheyPlain, 0.7, white, 1)
	gocv.PutText(&g.canvas, fmt.Sprintf("%d", yMax), image.Pt(2, botPlotTop+12),
		gocv.FontHersheyPlain, 0.7, white, 1)
	gocv.PutText(&g.canvas, fmt.Sprintf("%d", yMin), image.Pt(2, botPlotBot-4),
		gocv.FontHersheyPlain, 0.7, white, 1)

	// Time labels
	gocv.PutText(&g.canvas, fmt.Sprintf("%.1fs", tMin), image.Pt(plotLeft, botPlotBot+15),
		gocv.FontHersheyPlain, 0.7, white, 1)
	gocv.PutText(&g.canvas, fmt.Sprintf("%.1fs", tMax), image.Pt(plotRight-40, botPlotBot+15),
		gocv.FontHersheyPlain, 0.7, white, 1)

	// Draw data points as connected lines
	tRange := tMax - tMin
	for i := 1; i < len(times); i++ {
		// Normalize time to [0, 1]
		t0 := (times[i-1] - tMin) / tRange
		t1 := (times[i] - tMin) / tRange

		px0 := plotLeft + int(t0*float64(plotW))
		px1 := plotLeft + int(t1*float64(plotW))

		// X plot (top)
		xn0 := 1.0 - float64(xs[i-1]-xMin)/float64(xMax-xMin)
		xn1 := 1.0 - float64(xs[i]-xMin)/float64(xMax-xMin)
		py0x := topPlotTop + int(xn0*float64(plotHeight))
		py1x := topPlotTop + int(xn1*float64(plotHeight))
		gocv.Line(&g.canvas, image.Pt(px0, py0x), image.Pt(px1, py1x), cyan, 1)

		// Y plot (bottom)
		yn0 := 1.0 - float64(ys[i-1]-yMin)/float64(yMax-yMin)
		yn1 := 1.0 - float64(ys[i]-yMin)/float64(yMax-yMin)
		py0y := botPlotTop + int(yn0*float64(plotHeight))
		py1y := botPlotTop + int(yn1*float64(plotHeight))
		gocv.Line(&g.canvas, image.Pt(px0, py0y), image.Pt(px1, py1y), magenta, 1)
	}

	g.win.IMShow(g.canvas)
}

func (g *GraphWindow) Close() {
	g.canvas.Close()
	g.win.Close()
}

func fillMat(mat *gocv.Mat, c color.RGBA) {
	gocv.Rectangle(mat, image.Rect(0, 0, mat.Cols(), mat.Rows()), c, -1)
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
