package export

import (
	"encoding/csv"
	"fmt"
	"math"
	"os"
)

type TrackPoint struct {
	Time       float64
	X          int
	Y          int
	Confidence float64
}

type CSVOptions struct {
	IncludeConfidence bool
	Scale             float64 // pixels per unit; 0 means no calibration
	ScaleUnit         string  // e.g. "m", "cm"
	Derivatives       bool    // include vx, vy, ax, ay columns
}

func WriteCSV(path string, points []TrackPoint, opts CSVOptions) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("export: failed to create %s: %w", path, err)
	}
	defer func() { _ = f.Close() }()

	w := csv.NewWriter(f)
	defer w.Flush()

	calibrated := opts.Scale > 0
	unit := opts.ScaleUnit
	velUnit := "px/s"
	accUnit := "px/s2"
	if calibrated {
		velUnit = unit + "/s"
		accUnit = unit + "/s2"
	}

	// Build header
	header := []string{"time", "x", "y"}
	if opts.IncludeConfidence {
		header = append(header, "confidence")
	}
	if calibrated {
		header = append(header, "x_"+unit, "y_"+unit)
	}
	if opts.Derivatives {
		header = append(header, "vx_"+velUnit, "vy_"+velUnit, "ax_"+accUnit, "ay_"+accUnit)
	}
	if err := w.Write(header); err != nil {
		return err
	}

	// Precompute derivatives if needed
	var vx, vy, ax, ay []float64
	if opts.Derivatives && len(points) > 0 {
		vx, vy = computeVelocity(points, opts.Scale)
		ax, ay = computeAcceleration(points, vx, vy)
	}

	for i, p := range points {
		row := []string{
			fmt.Sprintf("%.6f", p.Time),
			fmt.Sprintf("%d", p.X),
			fmt.Sprintf("%d", p.Y),
		}
		if opts.IncludeConfidence {
			row = append(row, fmt.Sprintf("%.4f", p.Confidence))
		}
		if calibrated {
			row = append(row,
				fmt.Sprintf("%.6f", float64(p.X)/opts.Scale),
				fmt.Sprintf("%.6f", float64(p.Y)/opts.Scale))
		}
		if opts.Derivatives {
			row = append(row,
				fmt.Sprintf("%.6f", vx[i]),
				fmt.Sprintf("%.6f", vy[i]),
				fmt.Sprintf("%.6f", ax[i]),
				fmt.Sprintf("%.6f", ay[i]))
		}
		if err := w.Write(row); err != nil {
			return err
		}
	}

	return nil
}

// computeVelocity returns vx, vy arrays using central finite differences.
// If scale > 0, positions are converted to calibrated units first.
// Units: calibrated_unit/s or px/s.
func computeVelocity(points []TrackPoint, scale float64) ([]float64, []float64) {
	n := len(points)
	vx := make([]float64, n)
	vy := make([]float64, n)

	pos := func(i int) (float64, float64) {
		x, y := float64(points[i].X), float64(points[i].Y)
		if scale > 0 {
			x /= scale
			y /= scale
		}
		return x, y
	}

	for i := 0; i < n; i++ {
		switch {
		case i == 0 && n > 1:
			// Forward difference
			dt := points[1].Time - points[0].Time
			if dt > 0 {
				x0, y0 := pos(0)
				x1, y1 := pos(1)
				vx[0] = (x1 - x0) / dt
				vy[0] = (y1 - y0) / dt
			}
		case i == n-1 && n > 1:
			// Backward difference
			dt := points[n-1].Time - points[n-2].Time
			if dt > 0 {
				x0, y0 := pos(n - 2)
				x1, y1 := pos(n - 1)
				vx[n-1] = (x1 - x0) / dt
				vy[n-1] = (y1 - y0) / dt
			}
		default:
			// Central difference
			dt := points[i+1].Time - points[i-1].Time
			if dt > 0 {
				x0, y0 := pos(i - 1)
				x1, y1 := pos(i + 1)
				vx[i] = (x1 - x0) / dt
				vy[i] = (y1 - y0) / dt
			}
		}
	}

	return vx, vy
}

// computeAcceleration returns ax, ay from velocity arrays using central differences.
func computeAcceleration(points []TrackPoint, vx, vy []float64) ([]float64, []float64) {
	n := len(points)
	ax := make([]float64, n)
	ay := make([]float64, n)

	for i := 0; i < n; i++ {
		switch {
		case i == 0 && n > 1:
			dt := points[1].Time - points[0].Time
			if dt > 0 {
				ax[0] = (vx[1] - vx[0]) / dt
				ay[0] = (vy[1] - vy[0]) / dt
			}
		case i == n-1 && n > 1:
			dt := points[n-1].Time - points[n-2].Time
			if dt > 0 {
				ax[n-1] = (vx[n-1] - vx[n-2]) / dt
				ay[n-1] = (vy[n-1] - vy[n-2]) / dt
			}
		default:
			dt := points[i+1].Time - points[i-1].Time
			if dt > 0 {
				ax[i] = (vx[i+1] - vx[i-1]) / dt
				ay[i] = (vy[i+1] - vy[i-1]) / dt
			}
		}
	}

	return ax, ay
}

// ComputeScale returns pixels-per-unit from two points and a known distance.
func ComputeScale(p1, p2 [2]int, realDistance float64) float64 {
	dx := float64(p2[0] - p1[0])
	dy := float64(p2[1] - p1[1])
	pixelDist := math.Sqrt(dx*dx + dy*dy)
	return pixelDist / realDistance
}
