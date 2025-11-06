#!/bin/bash
set -e

echo "🧪 Running action-config tests..."

# Color codes
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Test counter
TESTS_PASSED=0
TESTS_FAILED=0

# Helper function to run tests
run_test() {
  local test_name="$1"
  local config_file="$2"
  local should_succeed="$3"

  echo ""
  echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
  echo "Test: $test_name"
  echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

  # Run the action logic
  if [[ "$config_file" == *.json ]]; then
    if MATRIX=$(jq -c '.' "$config_file" 2>&1); then
      if [ "$should_succeed" = true ]; then
        echo -e "${GREEN}✓ PASS${NC}: Valid JSON parsed successfully"
        echo "Output: $MATRIX"
        TESTS_PASSED=$((TESTS_PASSED + 1))
      else
        echo -e "${RED}✗ FAIL${NC}: Expected validation to fail but it succeeded"
        TESTS_FAILED=$((TESTS_FAILED + 1))
        return 1
      fi
    else
      if [ "$should_succeed" = false ]; then
        echo -e "${GREEN}✓ PASS${NC}: Invalid JSON rejected as expected"
        echo "Error: $MATRIX"
        TESTS_PASSED=$((TESTS_PASSED + 1))
      else
        echo -e "${RED}✗ FAIL${NC}: Valid JSON was rejected"
        echo "Error: $MATRIX"
        TESTS_FAILED=$((TESTS_FAILED + 1))
        return 1
      fi
    fi
  elif [[ "$config_file" == *.yml ]] || [[ "$config_file" == *.yaml ]]; then
    if command -v yq &> /dev/null; then
      if MATRIX=$(yq -o=json -I=0 '.' "$config_file" 2>&1); then
        if [ "$should_succeed" = true ]; then
          echo -e "${GREEN}✓ PASS${NC}: Valid YAML parsed successfully"
          echo "Output: $MATRIX"
          TESTS_PASSED=$((TESTS_PASSED + 1))
        else
          echo -e "${RED}✗ FAIL${NC}: Expected validation to fail but it succeeded"
          TESTS_FAILED=$((TESTS_FAILED + 1))
          return 1
        fi
      else
        if [ "$should_succeed" = false ]; then
          echo -e "${GREEN}✓ PASS${NC}: Invalid YAML rejected as expected"
          echo "Error: $MATRIX"
          TESTS_PASSED=$((TESTS_PASSED + 1))
        else
          echo -e "${RED}✗ FAIL${NC}: Valid YAML was rejected"
          echo "Error: $MATRIX"
          TESTS_FAILED=$((TESTS_FAILED + 1))
          return 1
        fi
      fi
    else
      echo -e "${YELLOW}⊘ SKIP${NC}: yq not installed, skipping YAML test"
    fi
  fi
}

# Test 1: Valid JSON configuration
run_test "Valid JSON Configuration" "tests/valid-config.json" true

# Test 2: Valid YAML configuration
run_test "Valid YAML Configuration" "tests/valid-config.yml" true

# Test 3: Invalid JSON configuration
run_test "Invalid JSON Configuration" "tests/invalid-config.json" false

# Test 4: Non-existent file
echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "Test: Non-existent File"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
if [ ! -f "tests/non-existent.json" ]; then
  echo -e "${GREEN}✓ PASS${NC}: Non-existent file correctly not found"
  TESTS_PASSED=$((TESTS_PASSED + 1))
else
  echo -e "${RED}✗ FAIL${NC}: File check failed"
  TESTS_FAILED=$((TESTS_FAILED + 1))
fi

# Test 5: JSON structure validation
echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "Test: JSON Array Structure"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
MATRIX=$(jq -c '.' "tests/valid-config.json")
if echo "$MATRIX" | jq -e 'type == "array"' > /dev/null; then
  echo -e "${GREEN}✓ PASS${NC}: Config is properly formatted as array"
  TESTS_PASSED=$((TESTS_PASSED + 1))
else
  echo -e "${RED}✗ FAIL${NC}: Config is not an array"
  TESTS_FAILED=$((TESTS_FAILED + 1))
fi

# Summary
echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "📊 Test Summary"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo -e "${GREEN}Passed: $TESTS_PASSED${NC}"
echo -e "${RED}Failed: $TESTS_FAILED${NC}"
echo "Total:  $((TESTS_PASSED + TESTS_FAILED))"

if [ $TESTS_FAILED -eq 0 ]; then
  echo ""
  echo -e "${GREEN}✓ All tests passed!${NC}"
  exit 0
else
  echo ""
  echo -e "${RED}✗ Some tests failed${NC}"
  exit 1
fi
