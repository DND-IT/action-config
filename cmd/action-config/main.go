package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/dnd-it/action-config/internal/expander"
	gitdetect "github.com/dnd-it/action-config/internal/git"
	"github.com/dnd-it/action-config/internal/inputs"
	"github.com/dnd-it/action-config/internal/outputs"
)

func main() {
	if err := run(); err != nil {
		outputs.LogError(err.Error())
		os.Exit(1)
	}
}

func run() error {
	cfg := inputs.Parse()

	opts, err := cfg.BuildExpanderOptions()
	if err != nil {
		return fmt.Errorf("invalid inputs: %w", err)
	}

	raw, err := expander.ParseConfigFile(cfg.ConfigPath)
	if err != nil {
		return err
	}

	optsCfg, dimensions := expander.ParseOptions(raw)

	// Set the filter key from the config's dimension_key
	opts.FilterKey = optsCfg.DimensionKey

	// Resolve dimension selection (explicit input or target shorthand)
	expander.ResolveTarget(dimensions, &optsCfg, &opts, cfg.DimensionKey)

	// If change detection is enabled, detect changes via git and filter
	if cfg.ChangeDetection {
		knownValues := expander.ExtractDimensionValues(dimensions, optsCfg.DimensionKey)
		if knownValues == nil {
			outputs.LogNotice(fmt.Sprintf("No %s dimension in config, skipping change detection", optsCfg.DimensionKey))
		} else {
			changedFiles, err := gitdetect.DetectChangedFiles()
			if err != nil {
				return fmt.Errorf("failed to detect changed files: %w", err)
			}

			if changedFiles == nil {
				outputs.LogNotice("Change detection not applicable for this event type, including all entries")
			} else {
				changedValues := expander.FilterChanged(changedFiles, optsCfg.BaseDir, knownValues)
				outputs.LogNotice(fmt.Sprintf("Detected %d changed files, %d/%d %s(s) with changes: %v", len(changedFiles), len(changedValues), len(knownValues), optsCfg.DimensionKey, changedValues))

				if len(changedValues) == 0 {
					outputs.SetOutput("matrix", "[]")
					outputs.SetOutput("changes_detected", "false")
					outputs.LogNotice("No entries with changes, matrix is empty")
					return nil
				}

				// Merge with existing filter (intersect)
				if len(opts.FilterValues) > 0 {
					existing := make(map[string]bool, len(opts.FilterValues))
					for _, s := range opts.FilterValues {
						existing[s] = true
					}
					var merged []string
					for _, s := range changedValues {
						if existing[s] {
							merged = append(merged, s)
						}
					}
					opts.FilterValues = merged
				} else {
					opts.FilterValues = changedValues
				}
			}
		}
	}

	entries, err := expander.Expand(dimensions, optsCfg, opts)
	if err != nil {
		return fmt.Errorf("failed to expand configuration: %w", err)
	}

	matrixJSON, err := json.Marshal(entries)
	if err != nil {
		return fmt.Errorf("failed to marshal matrix: %w", err)
	}

	outputs.SetOutput("matrix", string(matrixJSON))

	if cfg.ChangeDetection {
		if len(entries) > 0 {
			outputs.SetOutput("changes_detected", "true")
		} else {
			outputs.SetOutput("changes_detected", "false")
		}
	}

	// Log filters
	if len(opts.FilterValues) > 0 {
		outputs.LogNotice(fmt.Sprintf("Filtered by %s: %v", opts.FilterKey, opts.FilterValues))
	}
	if len(opts.EnvironmentFilter) > 0 {
		outputs.LogNotice(fmt.Sprintf("Filtered by environment: %v", opts.EnvironmentFilter))
	}
	if len(opts.InputExclude) > 0 {
		outputs.LogNotice("Applied input exclude filter")
	}
	if len(opts.InputInclude) > 0 {
		outputs.LogNotice("Applied input include filter")
	}

	// Pretty-print matrix to logs
	outputs.LogNotice("Matrix configuration loaded successfully:")
	prettyJSON, err := json.MarshalIndent(entries, "", "  ")
	if err == nil {
		outputs.LogInfo(string(prettyJSON))
	}

	return nil
}
