package gui

import (
	"math"
	"testing"
)

func TestMovingAverageIdentity(t *testing.T) {
	vals := []float64{1, 2, 3, 4, 5}
	result := movingAverage(vals, 1)
	for i, v := range result {
		if v != vals[i] {
			t.Errorf("window=1: result[%d] = %f, want %f", i, v, vals[i])
		}
	}
}

func TestMovingAverageWindow3(t *testing.T) {
	vals := []float64{1, 2, 3, 4, 5}
	result := movingAverage(vals, 3)

	// Middle elements: avg of 3 neighbors
	// result[1] = (1+2+3)/3 = 2.0
	// result[2] = (2+3+4)/3 = 3.0
	// result[3] = (3+4+5)/3 = 4.0
	expected := []float64{1.5, 2.0, 3.0, 4.0, 4.5}
	for i, v := range result {
		if math.Abs(v-expected[i]) > 0.01 {
			t.Errorf("result[%d] = %f, want %f", i, v, expected[i])
		}
	}
}

func TestMovingAverageLargeWindow(t *testing.T) {
	vals := []float64{10, 20, 30}
	result := movingAverage(vals, 100)

	// Window larger than array: all elements should be the global average
	avg := 20.0
	for i, v := range result {
		if math.Abs(v-avg) > 0.01 {
			t.Errorf("result[%d] = %f, want %f", i, v, avg)
		}
	}
}

func TestMovingAverageEmpty(t *testing.T) {
	result := movingAverage([]float64{}, 5)
	if len(result) != 0 {
		t.Errorf("expected empty result, got %v", result)
	}
}

func TestMinMaxFloat(t *testing.T) {
	mn, mx := minMaxFloat([]float64{3.0, 1.0, 4.0, 1.5, 9.0, 2.6})
	if mn != 1.0 {
		t.Errorf("min: got %f, want 1.0", mn)
	}
	if mx != 9.0 {
		t.Errorf("max: got %f, want 9.0", mx)
	}
}

func TestMinMaxFloatSingleElement(t *testing.T) {
	mn, mx := minMaxFloat([]float64{42.0})
	if mn != 42.0 || mx != 42.0 {
		t.Errorf("got min=%f max=%f, want both 42.0", mn, mx)
	}
}

func TestMinMaxInt(t *testing.T) {
	mn, mx := minMaxInt([]int{3, 1, 4, 1, 9, 2})
	if mn != 1 {
		t.Errorf("min: got %d, want 1", mn)
	}
	if mx != 9 {
		t.Errorf("max: got %d, want 9", mx)
	}
}

func TestFmtVal(t *testing.T) {
	tests := []struct {
		input float64
		want  string
	}{
		{1234.5, "1234"}, // large: no decimals (%.0f rounds to even)
		{42.7, "42.7"},   // medium: 1 decimal
		{0.123, "0.12"},  // small: 2 decimals
		{-500.0, "-500"}, // negative large
		{0.0, "0.00"},    // zero
	}

	for _, tc := range tests {
		got := fmtVal(tc.input)
		if got != tc.want {
			t.Errorf("fmtVal(%f) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

func TestIntsToFloats(t *testing.T) {
	input := []int{1, 2, 3}
	result := intsToFloats(input)
	expected := []float64{1.0, 2.0, 3.0}
	for i, v := range result {
		if v != expected[i] {
			t.Errorf("result[%d] = %f, want %f", i, v, expected[i])
		}
	}
}

func TestComputeRealtimeDerivativesLinear(t *testing.T) {
	// Linear motion: x = 10*t, y = 0
	times := []float64{0, 1, 2, 3, 4}
	xs := []float64{0, 10, 20, 30, 40}
	ys := []float64{0, 0, 0, 0, 0}

	vx, vy, ax, ay := computeRealtimeDerivatives(times, xs, ys)

	// vx should be ~10 for central points
	for i := 1; i < len(vx)-1; i++ {
		if math.Abs(vx[i]-10.0) > 0.01 {
			t.Errorf("vx[%d] = %f, want 10.0", i, vx[i])
		}
	}
	// vy should be ~0
	for i := range vy {
		if math.Abs(vy[i]) > 0.01 {
			t.Errorf("vy[%d] = %f, want 0.0", i, vy[i])
		}
	}
	// ax should be ~0 (constant velocity)
	for i := 1; i < len(ax)-1; i++ {
		if math.Abs(ax[i]) > 0.01 {
			t.Errorf("ax[%d] = %f, want ~0", i, ax[i])
		}
	}
	// ay should be ~0
	for i := range ay {
		if math.Abs(ay[i]) > 0.01 {
			t.Errorf("ay[%d] = %f, want 0.0", i, ay[i])
		}
	}
}
