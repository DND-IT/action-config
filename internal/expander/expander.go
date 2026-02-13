// Package expander handles config parsing and matrix expansion.
package expander

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// MatrixEntry represents a single entry in the expanded matrix.
type MatrixEntry map[string]any

// RawConfig represents the parsed configuration file.
type RawConfig map[string]any

// Options controls the expansion behavior.
type Options struct {
	StackFilter       []string
	EnvironmentFilter []string
	InputExclude      []MatrixEntry
	InputInclude      []MatrixEntry
}

// reservedKeys are top-level keys that are never treated as dimensions.
var reservedKeys = map[string]bool{
	"config":  true,
	"exclude": true,
	"include": true,
}

// ParseConfigFile reads and validates a JSON or YAML configuration file.
func ParseConfigFile(path string) (RawConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("configuration file not found: %s", path)
		}
		return nil, fmt.Errorf("failed to read configuration file: %w", err)
	}

	ext := strings.ToLower(filepath.Ext(path))

	var raw RawConfig

	switch ext {
	case ".json":
		if err := json.Unmarshal(data, &raw); err != nil {
			return nil, fmt.Errorf("invalid JSON in %s: %w", path, err)
		}
	case ".yaml", ".yml":
		if err := yaml.Unmarshal(data, &raw); err != nil {
			return nil, fmt.Errorf("invalid YAML in %s: %w", path, err)
		}
	default:
		return nil, fmt.Errorf("unsupported file type. Use .json, .yaml, or .yml")
	}

	if raw == nil {
		return nil, fmt.Errorf("configuration must be an object")
	}

	// Normalize: yaml.v3 may produce named map types (RawConfig) for nested
	// objects, which won't match plain map[string]any type assertions.
	// Round-trip through JSON to ensure uniform types.
	normalized, err := normalizeViaJSON(raw)
	if err != nil {
		return nil, fmt.Errorf("failed to normalize config: %w", err)
	}

	return normalized, nil
}

// dimension represents a named list of values for cartesian product.
type dimension struct {
	key    string
	values []any
}

// FilterChangedStacks returns the subset of knownStacks that have at least one
// matching changed file. A stack matches if any file starts with "{baseDir}/{stack}/"
// (or "{stack}/" if baseDir is empty).
func FilterChangedStacks(changedFiles []string, baseDir string, knownStacks []string) []string {
	var changed []string
	for _, stack := range knownStacks {
		prefix := stack + "/"
		if baseDir != "" {
			prefix = baseDir + "/" + stack + "/"
		}
		for _, f := range changedFiles {
			if strings.HasPrefix(strings.TrimSpace(f), prefix) {
				changed = append(changed, stack)
				break
			}
		}
	}
	return changed
}

// ExtractStacks returns the stack dimension values from a raw config, or nil if
// no "stack" dimension is defined.
func ExtractStacks(raw RawConfig) []string {
	val, ok := raw["stack"]
	if !ok {
		return nil
	}
	arr, ok := toSlice(val)
	if !ok {
		return nil
	}
	stacks := make([]string, 0, len(arr))
	for _, v := range arr {
		stacks = append(stacks, fmt.Sprintf("%v", v))
	}
	return stacks
}

// Expand takes a raw config and options, producing the expanded matrix.
func Expand(raw RawConfig, opts Options) ([]MatrixEntry, error) {
	dimensions := extractDimensions(raw)
	baseConfig := extractBaseConfig(raw)
	configObj := extractConfigObj(raw)

	var entries []MatrixEntry

	if len(dimensions) == 0 {
		// No dimensions: return single entry with non-reserved scalars
		entry := make(MatrixEntry)
		for k, v := range raw {
			if !reservedKeys[k] {
				entry[k] = v
			}
		}
		entries = []MatrixEntry{entry}
	} else {
		// Build cartesian product
		entries = cartesianProduct(dimensions)

		// Merge base config and config lookups
		entries = mergeConfig(entries, baseConfig, configObj)
	}

	// Apply config-level exclude
	if exc, ok := raw["exclude"]; ok {
		if patterns, err := toMatrixEntries(exc); err == nil {
			entries = applyExclude(entries, patterns)
		}
	}

	// Apply config-level include
	if inc, ok := raw["include"]; ok {
		if includes, err := toMatrixEntries(inc); err == nil {
			entries = applyInclude(entries, includes)
		}
	}

	// Apply input-level filters
	if len(opts.StackFilter) > 0 {
		entries = applyFilter(entries, "stack", opts.StackFilter)
	}
	if len(opts.EnvironmentFilter) > 0 {
		entries = applyFilter(entries, "environment", opts.EnvironmentFilter)
	}

	// Apply input-level exclude
	if len(opts.InputExclude) > 0 {
		entries = applyExclude(entries, opts.InputExclude)
	}

	// Apply input-level include
	if len(opts.InputInclude) > 0 {
		entries = applyInclude(entries, opts.InputInclude)
	}

	return entries, nil
}

// extractDimensions finds all dimension arrays from the config.
// It also derives environments from config keys if not explicitly present.
func extractDimensions(raw RawConfig) []dimension {
	// Collect explicit dimensions (top-level arrays, excluding reserved keys)
	var explicit []dimension
	explicitKeys := make(map[string]bool)

	// Sort keys for determinism
	keys := sortedKeys(raw)

	for _, k := range keys {
		if reservedKeys[k] {
			continue
		}
		if arr, ok := toSlice(raw[k]); ok {
			explicit = append(explicit, dimension{key: k, values: arr})
			explicitKeys[k] = true
		}
	}

	// Derive environment dimension from config keys
	if configObj, ok := raw["config"].(map[string]any); ok {
		envKey := "environment"
		if !explicitKeys[envKey] {
			envKeys := sortedKeys(configObj)
			values := make([]any, len(envKeys))
			for i, k := range envKeys {
				values[i] = k
			}
			explicit = append(explicit, dimension{key: envKey, values: values})
		}
	}

	return explicit
}

// extractBaseConfig returns non-array, non-reserved top-level values.
func extractBaseConfig(raw RawConfig) MatrixEntry {
	base := make(MatrixEntry)
	for k, v := range raw {
		if reservedKeys[k] {
			continue
		}
		if _, isArr := toSlice(v); isArr {
			continue
		}
		base[k] = v
	}
	return base
}

// extractConfigObj extracts the "config" object if present.
func extractConfigObj(raw RawConfig) map[string]any {
	if cfg, ok := raw["config"].(map[string]any); ok {
		return cfg
	}
	return nil
}

// cartesianProduct computes the cartesian product of all dimensions.
func cartesianProduct(dims []dimension) []MatrixEntry {
	result := []MatrixEntry{{}}

	for _, dim := range dims {
		var next []MatrixEntry
		for _, entry := range result {
			for _, val := range dim.values {
				newEntry := make(MatrixEntry)
				for k, v := range entry {
					newEntry[k] = v
				}
				newEntry[dim.key] = val
				next = append(next, newEntry)
			}
		}
		result = next
	}

	return result
}

// mergeConfig merges base config and config lookups into each entry.
func mergeConfig(entries []MatrixEntry, baseConfig MatrixEntry, configObj map[string]any) []MatrixEntry {
	result := make([]MatrixEntry, len(entries))

	for i, combo := range entries {
		entry := make(MatrixEntry)

		// Start with base config
		for k, v := range baseConfig {
			entry[k] = v
		}

		// Add combo values
		for k, v := range combo {
			entry[k] = v
		}

		// Merge config lookups for ALL combo values
		if configObj != nil {
			for _, v := range combo {
				strVal := fmt.Sprintf("%v", v)
				if cfgEntry, ok := configObj[strVal]; ok {
					if cfgMap, ok := cfgEntry.(map[string]any); ok {
						for ck, cv := range cfgMap {
							entry[ck] = cv
						}
					}
				}
			}
		}

		result[i] = entry
	}

	return result
}

// applyExclude removes entries matching all key/value pairs in any pattern.
func applyExclude(entries []MatrixEntry, patterns []MatrixEntry) []MatrixEntry {
	var result []MatrixEntry
	for _, entry := range entries {
		excluded := false
		for _, pattern := range patterns {
			if matchesPattern(entry, pattern) {
				excluded = true
				break
			}
		}
		if !excluded {
			result = append(result, entry)
		}
	}
	return result
}

// applyInclude appends entries to the matrix.
func applyInclude(entries []MatrixEntry, includes []MatrixEntry) []MatrixEntry {
	return append(entries, includes...)
}

// applyFilter keeps only entries where the given key's value is in the allowed list.
func applyFilter(entries []MatrixEntry, key string, allowed []string) []MatrixEntry {
	allowedSet := make(map[string]bool, len(allowed))
	for _, v := range allowed {
		allowedSet[v] = true
	}

	var result []MatrixEntry
	for _, entry := range entries {
		if val, ok := entry[key]; ok {
			if allowedSet[fmt.Sprintf("%v", val)] {
				result = append(result, entry)
			}
		}
	}
	return result
}

// matchesPattern checks if an entry matches all key/value pairs in a pattern.
func matchesPattern(entry, pattern MatrixEntry) bool {
	for k, pv := range pattern {
		ev, ok := entry[k]
		if !ok {
			return false
		}
		// Use string comparison for cross-type matching
		if fmt.Sprintf("%v", ev) != fmt.Sprintf("%v", pv) {
			return false
		}
	}
	return true
}

// toSlice converts an interface{} to []any if it's a slice.
func toSlice(v any) ([]any, bool) {
	if val, ok := v.([]any); ok {
		return val, true
	}
	return nil, false
}

// toMatrixEntries converts an interface{} to []MatrixEntry.
func toMatrixEntries(v any) ([]MatrixEntry, error) {
	arr, ok := toSlice(v)
	if !ok {
		return nil, fmt.Errorf("expected array")
	}

	result := make([]MatrixEntry, 0, len(arr))
	for _, item := range arr {
		if m, ok := item.(map[string]any); ok {
			result = append(result, MatrixEntry(m))
		}
	}
	return result, nil
}

// normalizeViaJSON round-trips through JSON to ensure all nested maps are
// plain map[string]any and all slices are []any, regardless of the source
// parser's named types.
func normalizeViaJSON(raw RawConfig) (RawConfig, error) {
	data, err := json.Marshal(raw)
	if err != nil {
		return nil, err
	}
	var result RawConfig
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// sortedKeys returns the keys of a map sorted alphabetically.
func sortedKeys[M ~map[string]V, V any](m M) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
