package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaults(t *testing.T) {
	cfg := Defaults()

	if cfg.Output != "tracking.csv" {
		t.Errorf("expected output 'tracking.csv', got %q", cfg.Output)
	}
	if cfg.TemplateSize != 15 {
		t.Errorf("expected template_size 15, got %d", cfg.TemplateSize)
	}
	if cfg.SearchMargin != 40 {
		t.Errorf("expected search_margin 40, got %d", cfg.SearchMargin)
	}
	if cfg.Confidence != 0.6 {
		t.Errorf("expected confidence 0.6, got %f", cfg.Confidence)
	}
	if cfg.Unit != "m" {
		t.Errorf("expected unit 'm', got %q", cfg.Unit)
	}
}

func TestLoad(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.toml")

	content := `# test config
output = "data.csv"
template_size = 25
search_margin = 60
confidence = 0.8
unit = "cm"
axes = true
turbo = true
trail = 50
derivatives = true
smooth = 10
start_frame = 100
start_time = 3.5
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if cfg.Output != "data.csv" {
		t.Errorf("output: got %q, want 'data.csv'", cfg.Output)
	}
	if cfg.TemplateSize != 25 {
		t.Errorf("template_size: got %d, want 25", cfg.TemplateSize)
	}
	if cfg.SearchMargin != 60 {
		t.Errorf("search_margin: got %d, want 60", cfg.SearchMargin)
	}
	if cfg.Confidence != 0.8 {
		t.Errorf("confidence: got %f, want 0.8", cfg.Confidence)
	}
	if cfg.Unit != "cm" {
		t.Errorf("unit: got %q, want 'cm'", cfg.Unit)
	}
	if !cfg.Axes {
		t.Error("axes: expected true")
	}
	if !cfg.Turbo {
		t.Error("turbo: expected true")
	}
	if cfg.Trail != 50 {
		t.Errorf("trail: got %d, want 50", cfg.Trail)
	}
	if !cfg.Derivatives {
		t.Error("derivatives: expected true")
	}
	if cfg.Smooth != 10 {
		t.Errorf("smooth: got %d, want 10", cfg.Smooth)
	}
	if cfg.StartFrame != 100 {
		t.Errorf("start_frame: got %d, want 100", cfg.StartFrame)
	}
	if cfg.StartTime != 3.5 {
		t.Errorf("start_time: got %f, want 3.5", cfg.StartTime)
	}
}

func TestLoadMissing(t *testing.T) {
	_, err := Load("/nonexistent/path.toml")
	if err == nil {
		t.Error("expected error for missing file")
	}
}

func TestLoadWithComments(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.toml")

	content := `# Full line comment
[section_header]  # section headers are ignored
template_size = 20  # inline comment
# confidence = 0.9  # commented out line
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if cfg.TemplateSize != 20 {
		t.Errorf("template_size: got %d, want 20", cfg.TemplateSize)
	}
	// confidence should remain at default since the line is commented out
	if cfg.Confidence != 0.6 {
		t.Errorf("confidence: got %f, want 0.6 (default)", cfg.Confidence)
	}
}

func TestParseBool(t *testing.T) {
	trueCases := []string{"true", "True", "TRUE", "1", "yes", "Yes", "YES"}
	falseCases := []string{"false", "False", "0", "no", "No", "anything"}

	for _, v := range trueCases {
		if !parseBool(v) {
			t.Errorf("parseBool(%q) = false, want true", v)
		}
	}
	for _, v := range falseCases {
		if parseBool(v) {
			t.Errorf("parseBool(%q) = true, want false", v)
		}
	}
}

func TestSetKeys(t *testing.T) {
	cfg := Defaults()

	// String key
	if err := cfg.set("output", "test.csv"); err != nil {
		t.Fatalf("set output failed: %v", err)
	}
	if cfg.Output != "test.csv" {
		t.Errorf("output: got %q", cfg.Output)
	}

	// Int key
	if err := cfg.set("template_size", "30"); err != nil {
		t.Fatalf("set template_size failed: %v", err)
	}
	if cfg.TemplateSize != 30 {
		t.Errorf("template_size: got %d", cfg.TemplateSize)
	}

	// Float key
	if err := cfg.set("confidence", "0.75"); err != nil {
		t.Fatalf("set confidence failed: %v", err)
	}
	if cfg.Confidence != 0.75 {
		t.Errorf("confidence: got %f", cfg.Confidence)
	}

	// Bool key
	if err := cfg.set("axes", "true"); err != nil {
		t.Fatalf("set axes failed: %v", err)
	}
	if !cfg.Axes {
		t.Error("axes: expected true")
	}

	// Hyphenated key (should work same as underscore)
	if err := cfg.set("search-margin", "50"); err != nil {
		t.Fatalf("set search-margin failed: %v", err)
	}
	if cfg.SearchMargin != 50 {
		t.Errorf("search_margin: got %d", cfg.SearchMargin)
	}

	// Invalid int
	err := cfg.set("template_size", "abc")
	if err == nil {
		t.Error("expected error for invalid int")
	}

	// Invalid float
	err = cfg.set("confidence", "abc")
	if err == nil {
		t.Error("expected error for invalid float")
	}
}
