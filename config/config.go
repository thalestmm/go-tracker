package config

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	Output           string
	TemplateSize     int
	SearchMargin     int
	Confidence       float64
	StartFrame       int
	StartTime        float64
	Axes             bool
	Turbo            bool
	ExportConfidence bool
	Calibrate        bool
	Unit             string
	Graph            bool
	Trail            int
	ExportVideo      string
	Derivatives      bool
	Smooth           int
}

func Defaults() Config {
	return Config{
		Output:       "tracking.csv",
		TemplateSize: 15,
		SearchMargin: 40,
		Confidence:   0.6,
		Unit:         "m",
	}
}

// Load reads a simple TOML-like config file (key = value pairs, # comments).
func Load(path string) (Config, error) {
	cfg := Defaults()

	f, err := os.Open(path)
	if err != nil {
		return cfg, err
	}
	defer func() { _ = f.Close() }()

	scanner := bufio.NewScanner(f)
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "[") {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])
		// Strip inline comments
		if idx := strings.Index(val, "#"); idx >= 0 {
			val = strings.TrimSpace(val[:idx])
		}
		// Strip quotes
		val = strings.Trim(val, "\"'")

		if err := cfg.set(key, val); err != nil {
			return cfg, fmt.Errorf("config line %d: %w", lineNum, err)
		}
	}

	return cfg, scanner.Err()
}

func (c *Config) set(key, val string) error {
	switch strings.ReplaceAll(key, "-", "_") {
	case "output":
		c.Output = val
	case "template_size":
		v, err := strconv.Atoi(val)
		if err != nil {
			return fmt.Errorf("invalid template_size: %s", val)
		}
		c.TemplateSize = v
	case "search_margin":
		v, err := strconv.Atoi(val)
		if err != nil {
			return fmt.Errorf("invalid search_margin: %s", val)
		}
		c.SearchMargin = v
	case "confidence":
		v, err := strconv.ParseFloat(val, 64)
		if err != nil {
			return fmt.Errorf("invalid confidence: %s", val)
		}
		c.Confidence = v
	case "start_frame":
		v, err := strconv.Atoi(val)
		if err != nil {
			return fmt.Errorf("invalid start_frame: %s", val)
		}
		c.StartFrame = v
	case "start_time":
		v, err := strconv.ParseFloat(val, 64)
		if err != nil {
			return fmt.Errorf("invalid start_time: %s", val)
		}
		c.StartTime = v
	case "axes":
		c.Axes = parseBool(val)
	case "turbo":
		c.Turbo = parseBool(val)
	case "export_confidence":
		c.ExportConfidence = parseBool(val)
	case "calibrate":
		c.Calibrate = parseBool(val)
	case "unit":
		c.Unit = val
	case "graph":
		c.Graph = parseBool(val)
	case "trail":
		v, err := strconv.Atoi(val)
		if err != nil {
			return fmt.Errorf("invalid trail: %s", val)
		}
		c.Trail = v
	case "export_video":
		c.ExportVideo = val
	case "derivatives":
		c.Derivatives = parseBool(val)
	case "smooth":
		v, err := strconv.Atoi(val)
		if err != nil {
			return fmt.Errorf("invalid smooth: %s", val)
		}
		c.Smooth = v
	}
	return nil
}

func parseBool(val string) bool {
	v := strings.ToLower(val)
	return v == "true" || v == "1" || v == "yes"
}
