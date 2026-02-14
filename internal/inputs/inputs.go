// Package inputs handles parsing GitHub Action inputs from environment variables.
package inputs

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/dnd-it/action-config/internal/expander"
)

// Config holds all parsed input values.
type Config struct {
	ConfigPath      string
	Target          string
	Environment     string
	Exclude         string
	Include         string
	ChangeDetection bool
}

// Parse reads inputs from environment variables.
func Parse() *Config {
	return &Config{
		ConfigPath:      getEnv("CONFIG_PATH", ".github/matrix-config.json"),
		Target:          getEnv("TARGET", ""),
		Environment:      getEnv("ENVIRONMENT", ""),
		Exclude:          getEnv("EXCLUDE", ""),
		Include:          getEnv("INCLUDE", ""),
		ChangeDetection: getEnv("CHANGE_DETECTION", "false") == "true",
	}
}

// BuildExpanderOptions converts raw input strings to typed expander.Options.
func (c *Config) BuildExpanderOptions() (expander.Options, error) {
	opts := expander.Options{
		FilterValues:      parseList(c.Target, ","),
		EnvironmentFilter: parseList(c.Environment, ","),
	}

	if c.Exclude != "" {
		if err := json.Unmarshal([]byte(c.Exclude), &opts.InputExclude); err != nil {
			return opts, fmt.Errorf("invalid exclude JSON: %w", err)
		}
	}

	if c.Include != "" {
		if err := json.Unmarshal([]byte(c.Include), &opts.InputInclude); err != nil {
			return opts, fmt.Errorf("invalid include JSON: %w", err)
		}
	}

	return opts, nil
}

func getEnv(name, defaultValue string) string {
	key := "INPUT_" + strings.ToUpper(name)
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultValue
}

func parseList(s, sep string) []string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	var result []string
	for _, item := range strings.Split(s, sep) {
		item = strings.TrimSpace(item)
		if item != "" {
			result = append(result, item)
		}
	}
	return result
}
