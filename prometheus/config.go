package prometheus

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/consize-oss/consize/pkg/plugin"
)

// JSONConfig is the configuration accepted by the protocol 1 subprocess.
type JSONConfig struct {
	BaseURL        string            `json:"base_url"`
	Window         string            `json:"window"`
	Step           string            `json:"step"`
	Queries        map[string]string `json:"queries"`
	Units          map[string]string `json:"units,omitempty"`
	Transport      TransportConfig   `json:"transport,omitempty"`
	Mapping        MappingConfig     `json:"mapping,omitempty"`
	Percentile     int               `json:"percentile,omitempty"`
	MinimumSamples int               `json:"minimum_samples,omitempty"`
}

// FromJSON constructs a plugin from explicit subprocess configuration.
func FromJSON(raw json.RawMessage) (plugin.Plugin, error) {
	var cfg JSONConfig
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&cfg); err != nil {
		return nil, fmt.Errorf("invalid Prometheus plugin configuration")
	}
	var extra any
	if err := dec.Decode(&extra); err != io.EOF {
		return nil, fmt.Errorf("expected one config object")
	}
	window, err := duration(cfg.Window, 24*time.Hour)
	if err != nil {
		return nil, fmt.Errorf("window: %w", err)
	}
	step, err := duration(cfg.Step, 5*time.Minute)
	if err != nil || step < time.Second || step%time.Second != 0 {
		return nil, fmt.Errorf("step must be a positive whole number of seconds")
	}
	if step > window {
		return nil, fmt.Errorf("step must not exceed window")
	}
	// The host deliberately strips ambient environment secrets. File references
	// are usable by the subprocess; environment references would silently fail.
	if cfg.Transport.Auth.BearerTokenEnv != "" || cfg.Transport.Auth.UsernameEnv != "" || cfg.Transport.Auth.PasswordEnv != "" || len(cfg.Transport.HeaderEnv) != 0 {
		return nil, fmt.Errorf("external plugins require file-based authentication; environment references are unavailable")
	}
	return New(Config{BaseURL: cfg.BaseURL, Window: window, Step: step, Queries: cfg.Queries, Units: cfg.Units, Transport: cfg.Transport, Mapping: cfg.Mapping, Percentile: cfg.Percentile, MinimumSamples: cfg.MinimumSamples})
}

func duration(raw string, fallback time.Duration) (time.Duration, error) {
	if raw == "" {
		return fallback, nil
	}
	value, err := time.ParseDuration(raw)
	if err != nil || value <= 0 {
		return 0, fmt.Errorf("expected a positive duration")
	}
	return value, nil
}
