package export

import (
	"encoding/csv"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"testing"
)

func TestWriteCSVBasic(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.csv")

	points := []TrackPoint{
		{Time: 0.0, X: 100, Y: 200, Confidence: 0.95},
		{Time: 0.033333, X: 102, Y: 198, Confidence: 0.92},
	}

	err := WriteCSV(path, points, CSVOptions{})
	if err != nil {
		t.Fatalf("WriteCSV failed: %v", err)
	}

	rows := readCSV(t, path)
	if len(rows) != 3 { // header + 2 data rows
		t.Fatalf("expected 3 rows, got %d", len(rows))
	}
	if rows[0][0] != "time" || rows[0][1] != "x" || rows[0][2] != "y" {
		t.Errorf("unexpected header: %v", rows[0])
	}
	if len(rows[0]) != 3 {
		t.Errorf("expected 3 columns, got %d", len(rows[0]))
	}
	if rows[1][1] != "100" || rows[1][2] != "200" {
		t.Errorf("unexpected data: %v", rows[1])
	}
}

func TestWriteCSVWithConfidence(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.csv")

	points := []TrackPoint{
		{Time: 0.0, X: 100, Y: 200, Confidence: 0.95},
	}

	err := WriteCSV(path, points, CSVOptions{IncludeConfidence: true})
	if err != nil {
		t.Fatalf("WriteCSV failed: %v", err)
	}

	rows := readCSV(t, path)
	if len(rows[0]) != 4 {
		t.Fatalf("expected 4 columns, got %d", len(rows[0]))
	}
	if rows[0][3] != "confidence" {
		t.Errorf("expected 'confidence' header, got %q", rows[0][3])
	}
	if rows[1][3] != "0.9500" {
		t.Errorf("expected confidence '0.9500', got %q", rows[1][3])
	}
}

func TestWriteCSVWithScale(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.csv")

	points := []TrackPoint{
		{Time: 0.0, X: 500, Y: 1000, Confidence: 0.9},
	}

	// 100 pixels per meter
	opts := CSVOptions{Scale: 100.0, ScaleUnit: "m"}
	err := WriteCSV(path, points, opts)
	if err != nil {
		t.Fatalf("WriteCSV failed: %v", err)
	}

	rows := readCSV(t, path)
	if rows[0][3] != "x_m" || rows[0][4] != "y_m" {
		t.Errorf("unexpected headers: %v", rows[0])
	}
	// 500/100 = 5.0, 1000/100 = 10.0
	if rows[1][3] != "5.000000" {
		t.Errorf("expected x_m '5.000000', got %q", rows[1][3])
	}
	if rows[1][4] != "10.000000" {
		t.Errorf("expected y_m '10.000000', got %q", rows[1][4])
	}
}

func TestWriteCSVWithDerivatives(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.csv")

	// Linear motion: x increases by 10px per 0.1s = 100 px/s
	points := []TrackPoint{
		{Time: 0.0, X: 0, Y: 0},
		{Time: 0.1, X: 10, Y: 0},
		{Time: 0.2, X: 20, Y: 0},
		{Time: 0.3, X: 30, Y: 0},
	}

	opts := CSVOptions{Derivatives: true}
	err := WriteCSV(path, points, opts)
	if err != nil {
		t.Fatalf("WriteCSV failed: %v", err)
	}

	rows := readCSV(t, path)
	// Header: time, x, y, vx_px/s, vy_px/s, ax_px/s2, ay_px/s2
	if len(rows[0]) != 7 {
		t.Fatalf("expected 7 columns, got %d: %v", len(rows[0]), rows[0])
	}
	if rows[0][3] != "vx_px/s" {
		t.Errorf("expected 'vx_px/s', got %q", rows[0][3])
	}

	// Central difference for point[1]: vx = (20 - 0) / 0.2 = 100
	vx, _ := strconv.ParseFloat(rows[2][3], 64)
	if math.Abs(vx-100.0) > 0.01 {
		t.Errorf("expected vx ~100, got %f", vx)
	}
}

func TestWriteCSVWithDerivativesCalibratedUnits(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.csv")

	points := []TrackPoint{
		{Time: 0.0, X: 0, Y: 0},
		{Time: 0.1, X: 100, Y: 0},
		{Time: 0.2, X: 200, Y: 0},
	}

	// 100 pixels per meter, so 100px = 1m
	opts := CSVOptions{Derivatives: true, Scale: 100.0, ScaleUnit: "m"}
	err := WriteCSV(path, points, opts)
	if err != nil {
		t.Fatalf("WriteCSV failed: %v", err)
	}

	rows := readCSV(t, path)
	// Should have m/s units
	if rows[0][5] != "vx_m/s" {
		t.Errorf("expected 'vx_m/s', got %q", rows[0][5])
	}
}

func TestComputeScale(t *testing.T) {
	// Two points 300 pixels apart, known distance 1.5 meters
	p1 := [2]int{0, 0}
	p2 := [2]int{300, 0}
	scale := ComputeScale(p1, p2, 1.5)
	if math.Abs(scale-200.0) > 0.01 {
		t.Errorf("expected scale 200.0, got %f", scale)
	}

	// Diagonal: sqrt(300^2 + 400^2) = 500 pixels, 2.5m
	p3 := [2]int{0, 0}
	p4 := [2]int{300, 400}
	scale = ComputeScale(p3, p4, 2.5)
	if math.Abs(scale-200.0) > 0.01 {
		t.Errorf("expected scale 200.0, got %f", scale)
	}
}

func TestComputeVelocityConstant(t *testing.T) {
	// Constant velocity: x increases by 10 each step
	points := []TrackPoint{
		{Time: 0.0, X: 0, Y: 0},
		{Time: 1.0, X: 10, Y: 0},
		{Time: 2.0, X: 20, Y: 0},
		{Time: 3.0, X: 30, Y: 0},
		{Time: 4.0, X: 40, Y: 0},
	}

	vx, vy := computeVelocity(points, 0)

	// Central points should all have vx = 10
	for i := 1; i < len(vx)-1; i++ {
		if math.Abs(vx[i]-10.0) > 0.01 {
			t.Errorf("vx[%d] = %f, want 10.0", i, vx[i])
		}
		if math.Abs(vy[i]) > 0.01 {
			t.Errorf("vy[%d] = %f, want 0.0", i, vy[i])
		}
	}
}

func TestComputeAccelerationZero(t *testing.T) {
	// Constant velocity → zero acceleration
	points := []TrackPoint{
		{Time: 0.0, X: 0, Y: 0},
		{Time: 1.0, X: 10, Y: 0},
		{Time: 2.0, X: 20, Y: 0},
		{Time: 3.0, X: 30, Y: 0},
		{Time: 4.0, X: 40, Y: 0},
	}

	vx, vy := computeVelocity(points, 0)
	ax, ay := computeAcceleration(points, vx, vy)

	for i := 1; i < len(ax)-1; i++ {
		if math.Abs(ax[i]) > 0.01 {
			t.Errorf("ax[%d] = %f, want ~0", i, ax[i])
		}
		if math.Abs(ay[i]) > 0.01 {
			t.Errorf("ay[%d] = %f, want ~0", i, ay[i])
		}
	}
}

func readCSV(t *testing.T, path string) [][]string {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("failed to open %s: %v", path, err)
	}
	defer f.Close()
	r := csv.NewReader(f)
	rows, err := r.ReadAll()
	if err != nil {
		t.Fatalf("failed to read CSV: %v", err)
	}
	return rows
}
