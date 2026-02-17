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

// OptionsConfig holds the parsed "global" block from the config file.
type OptionsConfig struct {
	DimensionKey string
	BaseDir      string
	SortBy       []string
	GlobalConfig map[string]any
	Exclude      []MatrixEntry
	Include      []MatrixEntry
}

// Options controls the expansion behavior.
type Options struct {
	FilterKey         string
	FilterValues      []string
	EnvironmentFilter []string
	InputExclude      []MatrixEntry
	InputInclude      []MatrixEntry
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

// reservedKeys are top-level keys that are never treated as dimensions.
var reservedKeys = map[string]bool{
	"global":  true,
	"exclude": true,
	"include": true,
}

// globalReservedKeys are keys inside the "global" block that are action settings,
// not config values to be merged into entries.
var globalReservedKeys = map[string]bool{
	"dimension_key": true,
	"base_dir":      true,
	"sort_by":       true,
}

// ParseOptions extracts reserved top-level keys from a raw config, returning
// the parsed options and the remaining dimensions-only config.
func ParseOptions(raw RawConfig) (OptionsConfig, RawConfig) {
	optsCfg := OptionsConfig{
		DimensionKey: "service",
	}

	dimensions := make(RawConfig)
	for k, v := range raw {
		if reservedKeys[k] {
			continue
		}
		dimensions[k] = v
	}

	// Root-level reserved keys
	if exc, ok := raw["exclude"]; ok {
		if entries, err := toMatrixEntries(exc); err == nil {
			optsCfg.Exclude = entries
		}
	}

	if inc, ok := raw["include"]; ok {
		if entries, err := toMatrixEntries(inc); err == nil {
			optsCfg.Include = entries
		}
	}

	// Global block
	globalRaw, ok := raw["global"]
	if !ok {
		return optsCfg, dimensions
	}

	globalMap, ok := globalRaw.(map[string]any)
	if !ok {
		return optsCfg, dimensions
	}

	if mk, ok := globalMap["dimension_key"].(string); ok && mk != "" {
		optsCfg.DimensionKey = mk
	}

	if bd, ok := globalMap["base_dir"].(string); ok {
		optsCfg.BaseDir = bd
	}

	if sb, ok := globalMap["sort_by"]; ok {
		if arr, ok := toSlice(sb); ok {
			sortBy := make([]string, 0, len(arr))
			for _, v := range arr {
				if s, ok := v.(string); ok {
					sortBy = append(sortBy, s)
				}
			}
			optsCfg.SortBy = sortBy
		}
	}

	// Everything else in global goes to GlobalConfig
	globalConfig := make(map[string]any)
	for k, v := range globalMap {
		if !globalReservedKeys[k] {
			globalConfig[k] = v
		}
	}
	if len(globalConfig) > 0 {
		optsCfg.GlobalConfig = globalConfig
	}

	return optsCfg, dimensions
}

// dimension represents a named list of values for cartesian product.
type dimension struct {
	key    string
	values []any
}

// FilterChanged returns the subset of knownValues that have at least one
// matching changed file. A value matches if any file starts with "{baseDir}/{value}/"
// (or "{value}/" if baseDir is empty).
func FilterChanged(changedFiles []string, baseDir string, knownValues []string) []string {
	var changed []string
	for _, val := range knownValues {
		prefix := val + "/"
		if baseDir != "" {
			prefix = baseDir + "/" + val + "/"
		}
		for _, f := range changedFiles {
			if strings.HasPrefix(strings.TrimSpace(f), prefix) {
				changed = append(changed, val)
				break
			}
		}
	}
	return changed
}

// ExtractDimensionValues returns the values for a given dimension key from a raw config.
// For array dimensions, returns the values as strings.
// For map dimensions, returns the sorted keys.
// Returns nil if the key is not defined or is not an array/map.
func ExtractDimensionValues(raw RawConfig, key string) []string {
	val, ok := raw[key]
	if !ok {
		return nil
	}
	// Array dimension
	if arr, ok := toSlice(val); ok {
		values := make([]string, 0, len(arr))
		for _, v := range arr {
			values = append(values, fmt.Sprintf("%v", v))
		}
		return values
	}
	// Map dimension
	if m, ok := val.(map[string]any); ok {
		return sortedKeys(m)
	}
	return nil
}

// Expand takes a dimensions-only config, options config, and expansion options,
// producing the expanded matrix.
func Expand(raw RawConfig, optsCfg OptionsConfig, opts Options) ([]MatrixEntry, error) {
	dimensions := extractDimensions(raw)

	var entries []MatrixEntry

	if len(dimensions) == 0 {
		// No dimensions: return single entry with all top-level scalars
		entry := make(MatrixEntry)
		for k, v := range raw {
			entry[k] = v
		}
		entries = []MatrixEntry{entry}
	} else {
		// Build cartesian product
		entries = cartesianProduct(dimensions)

		// Merge base config, global config, and per-dimension-value configs
		baseConfig := extractBaseConfig(raw)
		entries = mergeConfig(entries, baseConfig, optsCfg.GlobalConfig, raw)
	}

	// Apply options-level exclude
	if len(optsCfg.Exclude) > 0 {
		entries = applyExclude(entries, optsCfg.Exclude)
	}

	// Apply options-level include
	if len(optsCfg.Include) > 0 {
		entries = applyInclude(entries, optsCfg.Include)
	}

	// Apply input-level filters
	if len(opts.FilterValues) > 0 && opts.FilterKey != "" {
		entries = applyFilter(entries, opts.FilterKey, opts.FilterValues)
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

	// Add directory field to each entry
	addDirectoryField(entries, optsCfg)

	// Sort entries by sort_by keys (default: ["environment"])
	sortBy := optsCfg.SortBy
	if sortBy == nil {
		sortBy = []string{"environment"}
	}
	sortEntries(entries, sortBy)

	return entries, nil
}

// sortEntries sorts matrix entries by the given keys in order.
func sortEntries(entries []MatrixEntry, keys []string) {
	sort.SliceStable(entries, func(i, j int) bool {
		for _, key := range keys {
			vi := fmt.Sprintf("%v", entries[i][key])
			vj := fmt.Sprintf("%v", entries[j][key])
			if vi != vj {
				return vi < vj
			}
		}
		return false
	})
}

// addDirectoryField sets the "directory" field on each entry based on the
// dimension_key value and base_dir.
func addDirectoryField(entries []MatrixEntry, optsCfg OptionsConfig) {
	for _, entry := range entries {
		val, ok := entry[optsCfg.DimensionKey]
		if !ok {
			continue
		}
		strVal := fmt.Sprintf("%v", val)
		if optsCfg.BaseDir != "" {
			entry["directory"] = optsCfg.BaseDir + "/" + strVal
		} else {
			entry["directory"] = strVal
		}
	}
}

// extractDimensions finds all dimensions from the config.
// Arrays become dimensions directly. Maps become dimensions with sorted keys as values.
func extractDimensions(raw RawConfig) []dimension {
	var dims []dimension

	// Sort keys for determinism
	keys := sortedKeys(raw)

	for _, k := range keys {
		v := raw[k]
		if arr, ok := toSlice(v); ok {
			// Array dimension
			dims = append(dims, dimension{key: k, values: arr})
		} else if m, ok := v.(map[string]any); ok {
			// Map dimension: sorted keys become values
			mapKeys := sortedKeys(m)
			values := make([]any, len(mapKeys))
			for i, mk := range mapKeys {
				values[i] = mk
			}
			dims = append(dims, dimension{key: k, values: values})
		}
	}

	return dims
}

// extractBaseConfig returns scalar (non-array, non-map) top-level values.
func extractBaseConfig(raw RawConfig) MatrixEntry {
	base := make(MatrixEntry)
	for k, v := range raw {
		if _, isArr := toSlice(v); isArr {
			continue
		}
		if _, isMap := v.(map[string]any); isMap {
			continue
		}
		base[k] = v
	}
	return base
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

// mergeConfig merges base config, global config, and per-dimension-value configs
// into each entry.
//
// Merge order per entry:
//  1. Base config (scalar top-level values)
//  2. Global config values (from "global" minus reserved keys)
//  3. Combo dimension values (e.g. service=api, environment=dev)
//  4. Per-dimension-value configs in alphabetical dimension key order
func mergeConfig(entries []MatrixEntry, baseConfig MatrixEntry, globalConfig map[string]any, raw RawConfig) []MatrixEntry {
	result := make([]MatrixEntry, len(entries))

	for i, combo := range entries {
		entry := make(MatrixEntry)

		// 1. Base config (scalars)
		for k, v := range baseConfig {
			entry[k] = v
		}

		// 2. Global config values
		for k, v := range globalConfig {
			entry[k] = v
		}

		// 3. Combo dimension values
		for k, v := range combo {
			entry[k] = v
		}

		// 4. Per-dimension-value configs in alphabetical dimension key order
		dimKeys := sortedKeys(combo)
		for _, dimKey := range dimKeys {
			dimValue := fmt.Sprintf("%v", combo[dimKey])
			if dimMap, ok := raw[dimKey].(map[string]any); ok {
				if valConfig, ok := dimMap[dimValue].(map[string]any); ok {
					for ck, cv := range valConfig {
						entry[ck] = cv
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

// UniqueValues extracts the unique string values for a given key from matrix
// entries, preserving the order of first occurrence.
func UniqueValues(entries []MatrixEntry, key string) []string {
	seen := make(map[string]bool)
	var result []string
	for _, entry := range entries {
		if val, ok := entry[key]; ok {
			s := fmt.Sprintf("%v", val)
			if !seen[s] {
				seen[s] = true
				result = append(result, s)
			}
		}
	}
	return result
}

// ResolveTarget handles dimension selection. If dimensionKeyOverride is set (from
// dimension_key input), it overrides the config's dimension_key and removes the old
// dimension. Otherwise, if target is a single value matching a dimension name (but
// not a value of the current dimension_key), it triggers the same switch.
func ResolveTarget(raw RawConfig, optsCfg *OptionsConfig, opts *Options, dimensionKeyOverride string) {
	configDimKey := optsCfg.DimensionKey

	// Explicit dimension_key input override
	if dimensionKeyOverride != "" && dimensionKeyOverride != configDimKey {
		if isDimension(raw[dimensionKeyOverride]) {
			delete(raw, configDimKey)
			optsCfg.DimensionKey = dimensionKeyOverride
			opts.FilterKey = dimensionKeyOverride
			return
		}
	}

	// Auto-detect: single target value matching a dimension name
	if len(opts.FilterValues) != 1 {
		return
	}
	target := opts.FilterValues[0]

	if !isDimension(raw[target]) {
		return
	}

	// If it's also a value of the current dimension_key, treat as value filter
	for _, v := range ExtractDimensionValues(raw, configDimKey) {
		if v == target {
			return
		}
	}

	// Switch dimensions
	delete(raw, configDimKey)
	optsCfg.DimensionKey = target
	opts.FilterKey = target
	opts.FilterValues = nil
}

// isDimension returns true if the value is a map or slice (i.e. a dimension).
func isDimension(v any) bool {
	if _, ok := v.(map[string]any); ok {
		return true
	}
	if _, ok := toSlice(v); ok {
		return true
	}
	return false
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
