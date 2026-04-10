package tracker

type Config struct {
	TemplateSize        int
	SearchMargin        int
	ConfidenceThreshold float64
	AdaptiveSearch      bool
	MaxSearchMargin     int
}

func DefaultConfig() Config {
	return Config{
		TemplateSize:        15,
		SearchMargin:        40,
		ConfidenceThreshold: 0.6,
		AdaptiveSearch:      true,
		MaxSearchMargin:     120,
	}
}
