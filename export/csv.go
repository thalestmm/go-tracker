package export

import (
	"encoding/csv"
	"fmt"
	"os"
)

type TrackPoint struct {
	Time       float64
	X          int
	Y          int
	Confidence float64
}

func WriteCSV(path string, points []TrackPoint) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("export: failed to create %s: %w", path, err)
	}
	defer f.Close()

	w := csv.NewWriter(f)
	defer w.Flush()

	if err := w.Write([]string{"time", "x", "y"}); err != nil {
		return err
	}

	for _, p := range points {
		row := []string{
			fmt.Sprintf("%.6f", p.Time),
			fmt.Sprintf("%d", p.X),
			fmt.Sprintf("%d", p.Y),
		}
		if err := w.Write(row); err != nil {
			return err
		}
	}

	return nil
}
