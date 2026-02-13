package expander

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"testing"
)

func TestParseConfigFile_InvalidJSON(t *testing.T) {
	path := filepath.Join("..", "..", "testdata", "invalid-config.json")
	_, err := ParseConfigFile(path)
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestParseConfigFile_MissingFile(t *testing.T) {
	_, err := ParseConfigFile("nonexistent.json")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestParseConfigFile_UnsupportedExtension(t *testing.T) {
	tmp, err := os.CreateTemp("", "config-*.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmp.Name())
	_, _ = tmp.WriteString(`{"key": "value"}`)
	_ = tmp.Close()

	_, err = ParseConfigFile(tmp.Name())
	if err == nil {
		t.Fatal("expected error for unsupported extension")
	}
}

func TestParseConfigFile_NonObjectConfig(t *testing.T) {
	tmp, err := os.CreateTemp("", "config-*.json")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmp.Name())
	_, _ = tmp.WriteString(`["not", "an", "object"]`)
	_ = tmp.Close()

	_, err = ParseConfigFile(tmp.Name())
	if err == nil {
		t.Fatal("expected error for non-object config")
	}
}

func TestParseConfigFile_ValidJSON(t *testing.T) {
	path := filepath.Join("..", "..", "testdata", "valid-list-config.json")
	raw, err := ParseConfigFile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if raw == nil {
		t.Fatal("expected non-nil config")
	}
}

func TestParseConfigFile_ValidYAML(t *testing.T) {
	path := filepath.Join("..", "..", "testdata", "valid-list-config.yml")
	raw, err := ParseConfigFile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if raw == nil {
		t.Fatal("expected non-nil config")
	}
}

func TestExpand_BasicExpansion(t *testing.T) {
	raw := RawConfig{
		"stack": []any{"api", "frontend"},
		"config": map[string]any{
			"dev":  map[string]any{"aws_account_id": "111111111111"},
			"prod": map[string]any{"aws_account_id": "222222222222"},
		},
	}

	entries, err := Expand(raw, Options{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// 2 stacks × 2 environments = 4 entries
	if len(entries) != 4 {
		t.Fatalf("expected 4 entries, got %d", len(entries))
	}

	for _, entry := range entries {
		if _, ok := entry["stack"]; !ok {
			t.Error("entry missing 'stack' field")
		}
		if _, ok := entry["environment"]; !ok {
			t.Error("entry missing 'environment' field")
		}
		if _, ok := entry["aws_account_id"]; !ok {
			t.Error("entry missing 'aws_account_id' field")
		}
	}
}

func TestExpand_ConfigMerging(t *testing.T) {
	raw := RawConfig{
		"stack": []any{"api", "frontend"},
		"config": map[string]any{
			"dev":  map[string]any{"aws_account_id": "111111111111"},
			"prod": map[string]any{"aws_account_id": "222222222222"},
		},
	}

	entries, err := Expand(raw, Options{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for _, entry := range entries {
		if entry["stack"] == "api" && entry["environment"] == "dev" {
			if entry["aws_account_id"] != "111111111111" {
				t.Errorf("expected aws_account_id '111111111111', got %v", entry["aws_account_id"])
			}
			return
		}
	}
	t.Error("api/dev entry not found")
}

func TestExpand_JSONFixture(t *testing.T) {
	path := filepath.Join("..", "..", "testdata", "valid-list-config.json")
	raw, err := ParseConfigFile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	entries, err := Expand(raw, Options{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(entries) != 4 {
		t.Fatalf("expected 4 entries, got %d", len(entries))
	}
}

func TestExpand_YAMLFixture(t *testing.T) {
	path := filepath.Join("..", "..", "testdata", "valid-list-config.yml")
	raw, err := ParseConfigFile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	entries, err := Expand(raw, Options{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(entries) != 4 {
		t.Fatalf("expected 4 entries, got %d", len(entries))
	}
}

func TestExpand_JSONAndYAMLProduceSameResult(t *testing.T) {
	jsonPath := filepath.Join("..", "..", "testdata", "valid-list-config.json")
	yamlPath := filepath.Join("..", "..", "testdata", "valid-list-config.yml")

	jsonRaw, err := ParseConfigFile(jsonPath)
	if err != nil {
		t.Fatalf("unexpected error parsing JSON: %v", err)
	}
	yamlRaw, err := ParseConfigFile(yamlPath)
	if err != nil {
		t.Fatalf("unexpected error parsing YAML: %v", err)
	}

	jsonEntries, err := Expand(jsonRaw, Options{})
	if err != nil {
		t.Fatalf("unexpected error expanding JSON: %v", err)
	}
	yamlEntries, err := Expand(yamlRaw, Options{})
	if err != nil {
		t.Fatalf("unexpected error expanding YAML: %v", err)
	}

	if len(jsonEntries) != len(yamlEntries) {
		t.Fatalf("JSON entries (%d) != YAML entries (%d)", len(jsonEntries), len(yamlEntries))
	}

	sortEntries := func(entries []MatrixEntry) {
		sort.Slice(entries, func(i, j int) bool {
			si := entries[i]["stack"].(string) + entries[i]["environment"].(string)
			sj := entries[j]["stack"].(string) + entries[j]["environment"].(string)
			return si < sj
		})
	}
	sortEntries(jsonEntries)
	sortEntries(yamlEntries)

	jsonJSON, _ := json.Marshal(jsonEntries)
	yamlJSON, _ := json.Marshal(yamlEntries)

	if string(jsonJSON) != string(yamlJSON) {
		t.Errorf("JSON and YAML results differ:\nJSON: %s\nYAML: %s", jsonJSON, yamlJSON)
	}
}

func TestExpand_ConfigLevelExclude(t *testing.T) {
	raw := RawConfig{
		"stack": []any{"api", "shared"},
		"config": map[string]any{
			"dev":  map[string]any{"aws_account_id": "111111111111"},
			"prod": map[string]any{"aws_account_id": "222222222222"},
		},
		"exclude": []any{
			map[string]any{"stack": "shared", "environment": "dev"},
		},
	}

	entries, err := Expand(raw, Options{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// 2×2=4 minus 1 excluded = 3
	if len(entries) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(entries))
	}

	for _, entry := range entries {
		if entry["stack"] == "shared" && entry["environment"] == "dev" {
			t.Error("shared/dev should have been excluded")
		}
	}
}

func TestExpand_ConfigLevelInclude(t *testing.T) {
	raw := RawConfig{
		"stack": []any{"api"},
		"config": map[string]any{
			"dev": map[string]any{"aws_account_id": "111111111111"},
		},
		"include": []any{
			map[string]any{"stack": "shared", "environment": "all"},
		},
	}

	entries, err := Expand(raw, Options{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// 1×1=1 plus 1 included = 2
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}

	found := false
	for _, entry := range entries {
		if entry["stack"] == "shared" && entry["environment"] == "all" {
			found = true
		}
	}
	if !found {
		t.Error("shared/all should have been included")
	}
}

func TestExpand_InputStackFilter(t *testing.T) {
	raw := RawConfig{
		"stack": []any{"api", "frontend", "backend"},
		"config": map[string]any{
			"dev": map[string]any{"aws_account_id": "111111111111"},
		},
	}

	entries, err := Expand(raw, Options{
		StackFilter: []string{"api", "frontend"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}

	for _, entry := range entries {
		stack := entry["stack"].(string)
		if stack != "api" && stack != "frontend" {
			t.Errorf("unexpected stack %q", stack)
		}
	}
}

func TestExpand_InputEnvironmentFilter(t *testing.T) {
	raw := RawConfig{
		"stack": []any{"api"},
		"config": map[string]any{
			"dev":     map[string]any{"aws_account_id": "111111111111"},
			"staging": map[string]any{"aws_account_id": "222222222222"},
			"prod":    map[string]any{"aws_account_id": "333333333333"},
		},
	}

	entries, err := Expand(raw, Options{
		EnvironmentFilter: []string{"dev", "prod"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}

	for _, entry := range entries {
		env := entry["environment"].(string)
		if env != "dev" && env != "prod" {
			t.Errorf("unexpected environment %q", env)
		}
	}
}

func TestExpand_InputExclude(t *testing.T) {
	raw := RawConfig{
		"stack": []any{"api", "frontend"},
		"config": map[string]any{
			"dev":  map[string]any{"aws_account_id": "111111111111"},
			"prod": map[string]any{"aws_account_id": "222222222222"},
		},
	}

	entries, err := Expand(raw, Options{
		InputExclude: []MatrixEntry{
			{"stack": "api", "environment": "dev"},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(entries) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(entries))
	}
}

func TestExpand_InputInclude(t *testing.T) {
	raw := RawConfig{
		"stack": []any{"api"},
		"config": map[string]any{
			"dev": map[string]any{"aws_account_id": "111111111111"},
		},
	}

	entries, err := Expand(raw, Options{
		InputInclude: []MatrixEntry{
			{"stack": "monitoring", "environment": "global"},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
}

func TestExpand_NoDimensions(t *testing.T) {
	raw := RawConfig{
		"app_name": "myapp",
		"version":  "1.0",
	}

	entries, err := Expand(raw, Options{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}

	if entries[0]["app_name"] != "myapp" {
		t.Errorf("expected app_name 'myapp', got %v", entries[0]["app_name"])
	}
	if entries[0]["version"] != "1.0" {
		t.Errorf("expected version '1.0', got %v", entries[0]["version"])
	}
}

func TestExpand_MultipleCustomDimensions(t *testing.T) {
	raw := RawConfig{
		"stack":  []any{"api", "frontend"},
		"region": []any{"us-east-1", "eu-west-1"},
	}

	entries, err := Expand(raw, Options{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// 2 stacks × 2 regions = 4 entries
	if len(entries) != 4 {
		t.Fatalf("expected 4 entries, got %d", len(entries))
	}

	for _, entry := range entries {
		if _, ok := entry["stack"]; !ok {
			t.Error("entry missing 'stack' field")
		}
		if _, ok := entry["region"]; !ok {
			t.Error("entry missing 'region' field")
		}
	}
}

func TestExpand_BaseConfigPropagated(t *testing.T) {
	raw := RawConfig{
		"stack":    []any{"api"},
		"app_name": "myapp",
		"config": map[string]any{
			"dev": map[string]any{"aws_account_id": "111111111111"},
		},
	}

	entries, err := Expand(raw, Options{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}

	if entries[0]["app_name"] != "myapp" {
		t.Errorf("expected app_name 'myapp', got %v", entries[0]["app_name"])
	}
	if entries[0]["stack"] != "api" {
		t.Errorf("expected stack 'api', got %v", entries[0]["stack"])
	}
	if entries[0]["environment"] != "dev" {
		t.Errorf("expected environment 'dev', got %v", entries[0]["environment"])
	}
}

func TestExpand_ExplicitEnvironmentOverridesConfig(t *testing.T) {
	// If "environment" array is explicitly provided, don't derive from config keys
	raw := RawConfig{
		"stack":       []any{"api"},
		"environment": []any{"staging"},
		"config": map[string]any{
			"dev":     map[string]any{"aws_account_id": "111111111111"},
			"staging": map[string]any{"aws_account_id": "222222222222"},
			"prod":    map[string]any{"aws_account_id": "333333333333"},
		},
	}

	entries, err := Expand(raw, Options{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Only 1 stack × 1 explicit environment = 1 entry
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}

	if entries[0]["environment"] != "staging" {
		t.Errorf("expected environment 'staging', got %v", entries[0]["environment"])
	}
	if entries[0]["aws_account_id"] != "222222222222" {
		t.Errorf("expected aws_account_id '222222222222', got %v", entries[0]["aws_account_id"])
	}
}

func TestExpand_ConfigMergeAllComboValues(t *testing.T) {
	// Config lookup should match ANY combo value, not just environment.
	// Here "api" is both a stack value and a config key.
	// Config keys "dev" and "api" derive environments: ["api", "dev"] (sorted).
	// So stack=["api"] × environment=["api","dev"] = 2 entries.
	raw := RawConfig{
		"stack": []any{"api"},
		"config": map[string]any{
			"dev": map[string]any{"aws_account_id": "111111111111"},
			"api": map[string]any{"port": "8080"},
		},
	}

	entries, err := Expand(raw, Options{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}

	for _, e := range entries {
		if e["stack"] == "api" && e["environment"] == "api" {
			if e["port"] != "8080" {
				t.Errorf("expected port '8080', got %v", e["port"])
			}
		}
		if e["stack"] == "api" && e["environment"] == "dev" {
			if e["aws_account_id"] != "111111111111" {
				t.Errorf("expected aws_account_id '111111111111', got %v", e["aws_account_id"])
			}
			// "api" stack should also get config["api"] merged
			if e["port"] != "8080" {
				t.Errorf("expected port '8080' from config[api] merge, got %v", e["port"])
			}
		}
	}
}

func TestExpand_ExcludePartialMatch(t *testing.T) {
	raw := RawConfig{
		"stack": []any{"api", "frontend"},
		"config": map[string]any{
			"dev":  map[string]any{"aws_account_id": "111111111111"},
			"prod": map[string]any{"aws_account_id": "222222222222"},
		},
		"exclude": []any{
			map[string]any{"environment": "dev"},
		},
	}

	entries, err := Expand(raw, Options{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should exclude all dev entries: 4-2=2
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}

	for _, entry := range entries {
		if entry["environment"] == "dev" {
			t.Error("dev entries should have been excluded")
		}
	}
}

func TestExpand_EmptyDimensionArray(t *testing.T) {
	raw := RawConfig{
		"stack": []any{},
		"config": map[string]any{
			"dev": map[string]any{"aws_account_id": "111111111111"},
		},
	}

	entries, err := Expand(raw, Options{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// 0 stacks × 1 env = 0 entries
	if len(entries) != 0 {
		t.Fatalf("expected 0 entries, got %d", len(entries))
	}
}

func TestExpand_DimensionKeysUsedAsIs(t *testing.T) {
	// Verify that dimension keys are NOT singularized
	raw := RawConfig{
		"services": []any{"web", "worker"},
	}

	entries, err := Expand(raw, Options{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}

	// Key should be "services", not "service"
	for _, entry := range entries {
		if _, ok := entry["services"]; !ok {
			t.Error("entry should use key 'services' as-is, not singularized")
		}
	}
}

func TestFilterChangedStacks_WithBaseDir(t *testing.T) {
	files := []string{
		"deploy/infra/waf.tf",
		"deploy/infra/variables.tf",
		"deploy/networking/vpc.tf",
		"README.md",
	}
	changed := FilterChangedStacks(files, "deploy", []string{"infra", "networking", "frontend"})
	if len(changed) != 2 {
		t.Fatalf("expected 2 stacks, got %d: %v", len(changed), changed)
	}
	if changed[0] != "infra" || changed[1] != "networking" {
		t.Errorf("expected [infra networking], got %v", changed)
	}
}

func TestFilterChangedStacks_WithoutBaseDir(t *testing.T) {
	files := []string{
		"infra/waf.tf",
		"networking/vpc.tf",
	}
	changed := FilterChangedStacks(files, "", []string{"infra", "networking"})
	if len(changed) != 2 {
		t.Fatalf("expected 2 stacks, got %d: %v", len(changed), changed)
	}
	if changed[0] != "infra" || changed[1] != "networking" {
		t.Errorf("expected [infra networking], got %v", changed)
	}
}

func TestFilterChangedStacks_NoMatchingFiles(t *testing.T) {
	files := []string{
		"README.md",
		".github/workflows/ci.yaml",
	}
	changed := FilterChangedStacks(files, "deploy", []string{"infra", "frontend"})
	if len(changed) != 0 {
		t.Fatalf("expected 0 stacks, got %d: %v", len(changed), changed)
	}
}

func TestFilterChangedStacks_EmptyFiles(t *testing.T) {
	changed := FilterChangedStacks([]string{}, "deploy", []string{"infra"})
	if len(changed) != 0 {
		t.Fatalf("expected 0 stacks, got %d: %v", len(changed), changed)
	}
}

func TestFilterChangedStacks_OnlyMatchesKnownStacks(t *testing.T) {
	files := []string{
		"deploy/infra/waf.tf",
		"deploy/unknown/file.tf",
	}
	// "unknown" is not in knownStacks, so it should not appear
	changed := FilterChangedStacks(files, "deploy", []string{"infra", "frontend"})
	if len(changed) != 1 {
		t.Fatalf("expected 1 stack, got %d: %v", len(changed), changed)
	}
	if changed[0] != "infra" {
		t.Errorf("expected [infra], got %v", changed)
	}
}

func TestFilterChangedStacks_MultipleFilesInSameStack(t *testing.T) {
	files := []string{
		"deploy/infra/waf.tf",
		"deploy/infra/outputs.tf",
		"deploy/infra/variables.tf",
	}
	changed := FilterChangedStacks(files, "deploy", []string{"infra", "frontend"})
	if len(changed) != 1 {
		t.Fatalf("expected 1 stack, got %d: %v", len(changed), changed)
	}
	if changed[0] != "infra" {
		t.Errorf("expected [infra], got %v", changed)
	}
}

func TestExtractStacks_Present(t *testing.T) {
	raw := RawConfig{
		"stack": []any{"api", "infra"},
	}
	stacks := ExtractStacks(raw)
	if len(stacks) != 2 {
		t.Fatalf("expected 2 stacks, got %d", len(stacks))
	}
	if stacks[0] != "api" || stacks[1] != "infra" {
		t.Errorf("expected [api infra], got %v", stacks)
	}
}

func TestExtractStacks_NotPresent(t *testing.T) {
	raw := RawConfig{
		"app_name": "myapp",
	}
	stacks := ExtractStacks(raw)
	if stacks != nil {
		t.Fatalf("expected nil, got %v", stacks)
	}
}

