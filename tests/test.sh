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

# jq expansion logic (mirrors action.yaml)
expand_config() {
  jq -c '
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

    def matches(pattern):
      . as $item |
      all(pattern | to_entries[]; .key as $k | .value as $v | $item[$k] == $v);

    (if $root.config then
      [{key: "environments", value: ($root.config | keys)}]
    else
      []
    end) as $config_dimensions |

    (to_entries | map(select(
      (.value | type == "array") and
      (.key != "exclude") and
      (.key != "include")
    ))) as $explicit_dimensions |

    ($explicit_dimensions | map(.key)) as $explicit_keys |
    ($explicit_dimensions + ($config_dimensions | map(select(.key as $k | $explicit_keys | index($k) | not)))) as $dimensions |

    ($root | to_entries | map(select(
      (.value | type != "array") and
      (.key != "config") and
      (.key != "exclude") and
      (.key != "include")
    )) | from_entries) as $base_config |

    (if ($dimensions | length) == 0 then
      [$root | del(.config) | del(.exclude) | del(.include)]
    else
      $dimensions |
      reduce .[] as $dim (
        [{}];
        [.[] as $item |
         $dim.value[] as $val |
         $item + {($dim.key | singularize): $val}]
      ) |

      map(. as $combo |
        $base_config |
        . + $combo |
        if $root.config then
          . + (
            $combo | to_entries |
            map($root.config[.value] // {}) |
            add // {}
          )
        else . end
      )
    end) |

    if $root.exclude then
      map(select(. as $item |
        all($root.exclude[]; . as $pattern | ($item | matches($pattern)) | not)
      ))
    else . end |

    if $root.include then . + $root.include else . end
  '
}

# Test 1: Invalid JSON configuration
echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "Test: Invalid JSON Configuration"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
if MATRIX=$(jq -c '.' "tests/invalid-config.json" 2>&1); then
  echo -e "${RED}✗ FAIL${NC}: Expected validation to fail but it succeeded"
  TESTS_FAILED=$((TESTS_FAILED + 1))
else
  echo -e "${GREEN}✓ PASS${NC}: Invalid JSON rejected as expected"
  TESTS_PASSED=$((TESTS_PASSED + 1))
fi

# Test 2: Non-existent file
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

# Test 3: Config expansion (JSON)
echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "Test: Config Expansion (JSON)"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
CONFIG=$(jq -c '.' "tests/valid-list-config.json")
EXPANDED=$(echo "$CONFIG" | expand_config)

ITEM_COUNT=$(echo "$EXPANDED" | jq 'length')
EXPECTED_COUNT=4  # 2 stacks × 2 environments (derived from config keys)

if [ "$ITEM_COUNT" -eq "$EXPECTED_COUNT" ]; then
  echo -e "${GREEN}✓ PASS${NC}: Expanded to $ITEM_COUNT items (expected $EXPECTED_COUNT)"
  echo "Sample output:"
  echo "$EXPANDED" | jq '.[0]'
  TESTS_PASSED=$((TESTS_PASSED + 1))
else
  echo -e "${RED}✗ FAIL${NC}: Expected $EXPECTED_COUNT items, got $ITEM_COUNT"
  echo "$EXPANDED" | jq '.'
  TESTS_FAILED=$((TESTS_FAILED + 1))
fi

# Test 4: Verify config merging
echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "Test: Config Merging"
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

# Test 5: YAML config expansion
echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "Test: Config Expansion (YAML)"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
if command -v yq &> /dev/null; then
  CONFIG=$(yq -o=json -I=0 '.' "tests/valid-list-config.yml")
  EXPANDED=$(echo "$CONFIG" | expand_config)

  ITEM_COUNT=$(echo "$EXPANDED" | jq 'length')
  EXPECTED_COUNT=4

  if [ "$ITEM_COUNT" -eq "$EXPECTED_COUNT" ]; then
    echo -e "${GREEN}✓ PASS${NC}: YAML expanded to $ITEM_COUNT items (expected $EXPECTED_COUNT)"
    TESTS_PASSED=$((TESTS_PASSED + 1))
  else
    echo -e "${RED}✗ FAIL${NC}: Expected $EXPECTED_COUNT items, got $ITEM_COUNT"
    TESTS_FAILED=$((TESTS_FAILED + 1))
  fi
else
  echo -e "${YELLOW}⊘ SKIP${NC}: yq not installed, skipping YAML test"
fi

# Test 6: Exclude support
echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "Test: Exclude"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
EXCLUDE_CONFIG='{"stacks":["api","shared"],"config":{"dev":{"aws_account_id":"111"},"prod":{"aws_account_id":"222"}},"exclude":[{"stack":"shared","environment":"dev"}]}'
EXPANDED=$(echo "$EXCLUDE_CONFIG" | expand_config)
ITEM_COUNT=$(echo "$EXPANDED" | jq 'length')
# 2 stacks × 2 envs = 4, minus 1 excluded = 3
HAS_EXCLUDED=$(echo "$EXPANDED" | jq '[.[] | select(.stack == "shared" and .environment == "dev")] | length')

if [ "$ITEM_COUNT" -eq 3 ] && [ "$HAS_EXCLUDED" -eq 0 ]; then
  echo -e "${GREEN}✓ PASS${NC}: Exclude removed shared/dev (got $ITEM_COUNT items)"
  TESTS_PASSED=$((TESTS_PASSED + 1))
else
  echo -e "${RED}✗ FAIL${NC}: Expected 3 items without shared/dev, got $ITEM_COUNT (excluded present: $HAS_EXCLUDED)"
  echo "$EXPANDED" | jq '.'
  TESTS_FAILED=$((TESTS_FAILED + 1))
fi

# Test 7: Include support
echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "Test: Include"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
INCLUDE_CONFIG='{"stacks":["api"],"config":{"dev":{"aws_account_id":"111"}},"include":[{"stack":"shared","aws_account_id":"333"}]}'
EXPANDED=$(echo "$INCLUDE_CONFIG" | expand_config)
ITEM_COUNT=$(echo "$EXPANDED" | jq 'length')
HAS_SHARED=$(echo "$EXPANDED" | jq '[.[] | select(.stack == "shared")] | length')

if [ "$ITEM_COUNT" -eq 2 ] && [ "$HAS_SHARED" -eq 1 ]; then
  echo -e "${GREEN}✓ PASS${NC}: Include appended shared entry (got $ITEM_COUNT items)"
  TESTS_PASSED=$((TESTS_PASSED + 1))
else
  echo -e "${RED}✗ FAIL${NC}: Expected 2 items with shared, got $ITEM_COUNT (shared count: $HAS_SHARED)"
  echo "$EXPANDED" | jq '.'
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
