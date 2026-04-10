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
}

func WriteCSV(path string, points []TrackPoint, opts CSVOptions) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("export: failed to create %s: %w", path, err)
	}
	defer f.Close()

	w := csv.NewWriter(f)
	defer w.Flush()

	header := []string{"time", "x", "y"}
	if opts.IncludeConfidence {
		header = append(header, "confidence")
	}
	if opts.Scale > 0 {
		unit := opts.ScaleUnit
		header = append(header, "x_"+unit, "y_"+unit)
	}
	if err := w.Write(header); err != nil {
		return err
	}

	for _, p := range points {
		row := []string{
			fmt.Sprintf("%.6f", p.Time),
			fmt.Sprintf("%d", p.X),
			fmt.Sprintf("%d", p.Y),
		}
		if opts.IncludeConfidence {
			row = append(row, fmt.Sprintf("%.4f", p.Confidence))
		}
		if opts.Scale > 0 {
			row = append(row,
				fmt.Sprintf("%.6f", float64(p.X)/opts.Scale),
				fmt.Sprintf("%.6f", float64(p.Y)/opts.Scale))
		}
		if err := w.Write(row); err != nil {
			return err
		}
	}

	return nil
}

// ComputeScale returns pixels-per-unit from two points and a known distance.
func ComputeScale(p1, p2 [2]int, realDistance float64) float64 {
	dx := float64(p2[0] - p1[0])
	dy := float64(p2[1] - p1[1])
	pixelDist := math.Sqrt(dx*dx + dy*dy)
	return pixelDist / realDistance
}
