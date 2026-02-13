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

	// If changed-files-only is enabled, detect changes via git and filter stacks
	if cfg.ChangedFilesOnly {
		knownStacks := expander.ExtractStacks(raw)
		if knownStacks == nil {
			// No stack dimension in config — always run, no filtering
			outputs.LogNotice("No stack dimension in config, skipping change detection")
		} else {
			changedFiles, err := gitdetect.DetectChangedFiles()
			if err != nil {
				return fmt.Errorf("failed to detect changed files: %w", err)
			}

			if changedFiles == nil {
				// Event type doesn't support change detection (e.g. workflow_dispatch)
				outputs.LogNotice("Change detection not applicable for this event type, including all stacks")
			} else {
				baseDir, _ := raw["base_dir"].(string)
				changedStacks := expander.FilterChangedStacks(changedFiles, baseDir, knownStacks)
				outputs.LogNotice(fmt.Sprintf("Detected %d changed files, %d/%d stacks with changes: %v", len(changedFiles), len(changedStacks), len(knownStacks), changedStacks))

				if len(changedStacks) == 0 {
					outputs.SetOutput("matrix", "[]")
					outputs.SetOutput("any_changed", "false")
					outputs.LogNotice("No stacks with changes, matrix is empty")
					return nil
				}

				// Merge with existing stack filter (intersect)
				if len(opts.StackFilter) > 0 {
					existing := make(map[string]bool, len(opts.StackFilter))
					for _, s := range opts.StackFilter {
						existing[s] = true
					}
					var merged []string
					for _, s := range changedStacks {
						if existing[s] {
							merged = append(merged, s)
						}
					}
					opts.StackFilter = merged
				} else {
					opts.StackFilter = changedStacks
				}
			}
		}
	}

	entries, err := expander.Expand(raw, opts)
	if err != nil {
		return fmt.Errorf("failed to expand configuration: %w", err)
	}

	matrixJSON, err := json.Marshal(entries)
	if err != nil {
		return fmt.Errorf("failed to marshal matrix: %w", err)
	}

	outputs.SetOutput("matrix", string(matrixJSON))

	if cfg.ChangedFilesOnly {
		if len(entries) > 0 {
			outputs.SetOutput("any_changed", "true")
		} else {
			outputs.SetOutput("any_changed", "false")
		}
	}

	// Log filters
	if len(opts.StackFilter) > 0 {
		outputs.LogNotice(fmt.Sprintf("Filtered by stack: %v", opts.StackFilter))
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
