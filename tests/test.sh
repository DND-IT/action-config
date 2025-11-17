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

# Test 5: JSON structure validation (array format)
echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "Test: JSON Array Structure (Legacy Format)"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
MATRIX=$(jq -c '.' "tests/valid-config.json")
if echo "$MATRIX" | jq -e 'type == "array"' > /dev/null; then
  echo -e "${GREEN}✓ PASS${NC}: Config is properly formatted as array"
  TESTS_PASSED=$((TESTS_PASSED + 1))
else
  echo -e "${RED}✗ FAIL${NC}: Config is not an array"
  TESTS_FAILED=$((TESTS_FAILED + 1))
fi

# Test 6: List-based format validation (JSON)
echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "Test: List-Based Format Expansion (JSON)"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
CONFIG=$(jq -c '.' "tests/valid-list-config.json")
EXPANDED=$(echo "$CONFIG" | jq -c '
  # Helper function to singularize common plural keys
  def singularize:
    if endswith("ies") then
      .[:-3] + "y"
    elif endswith("es") then
      .[:-2]
    elif endswith("s") then
      .[:-1]
    else
      .
    end;

  . as $root |
  to_entries |
  map(select(.value | type == "array")) as $dimensions |

  if ($dimensions | length) == 0 then
    [$root]
  else
    ($root | to_entries | map(select((.value | type != "array") and .key != "config")) | from_entries) as $base_config |

    $dimensions |
    reduce .[] as $dim (
      [{}];
      [.[] as $item |
       $dim.value[] as $val |
       $item + {($dim.key | singularize): $val}]
    ) |

    map(. as $combo |
      $combo |
      . + $base_config |
      if $root.config then
        . + (
          $combo | to_entries |
          map($root.config[.value] // {}) |
          add // {}
        )
      else . end
    )
  end
')

# Verify expansion worked correctly
ITEM_COUNT=$(echo "$EXPANDED" | jq 'length')
EXPECTED_COUNT=4  # 2 stacks × 2 environments

if [ "$ITEM_COUNT" -eq "$EXPECTED_COUNT" ]; then
  echo -e "${GREEN}✓ PASS${NC}: Expanded to $ITEM_COUNT items (expected $EXPECTED_COUNT)"
  echo "Sample output:"
  echo "$EXPANDED" | jq '.[0]'
  TESTS_PASSED=$((TESTS_PASSED + 1))
else
  echo -e "${RED}✗ FAIL${NC}: Expected $EXPECTED_COUNT items, got $ITEM_COUNT"
  TESTS_FAILED=$((TESTS_FAILED + 1))
fi

# Test 7: Verify config merging in list format
echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "Test: Config Merging in List Format"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
HAS_AWS_ID=$(echo "$EXPANDED" | jq -e '.[0] | has("aws_account_id")' > /dev/null && echo "true" || echo "false")
HAS_STACK=$(echo "$EXPANDED" | jq -e '.[0] | has("stack")' > /dev/null && echo "true" || echo "false")
HAS_ENV=$(echo "$EXPANDED" | jq -e '.[0] | has("environment")' > /dev/null && echo "true" || echo "false")

if [ "$HAS_AWS_ID" = "true" ] && [ "$HAS_STACK" = "true" ] && [ "$HAS_ENV" = "true" ]; then
  echo -e "${GREEN}✓ PASS${NC}: All required fields present after expansion"
  TESTS_PASSED=$((TESTS_PASSED + 1))
else
  echo -e "${RED}✗ FAIL${NC}: Missing required fields (aws_account_id: $HAS_AWS_ID, stack: $HAS_STACK, environment: $HAS_ENV)"
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
