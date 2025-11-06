# Tests

This directory contains tests for the action-config GitHub Action.

## Test Files

- `valid-config.json` - Valid JSON configuration for testing
- `valid-config.yml` - Valid YAML configuration for testing
- `invalid-config.json` - Invalid JSON to test error handling
- `test.sh` - Shell script for unit testing

## Running Tests Locally

```bash
# Run unit tests
./tests/test.sh
```

## CI/CD Tests

The `.github/workflows/test.yml` workflow runs comprehensive tests on every push and pull request:

### Test Jobs

1. **unit-tests** - Runs the test.sh script to validate basic functionality
2. **test-json-config** - Tests JSON configuration file parsing
3. **test-yaml-config** - Tests YAML configuration file parsing
4. **test-invalid-config** - Verifies invalid configurations are rejected
5. **test-matrix-usage** - Generates a matrix from config
6. **verify-matrix** - Executes jobs using the generated matrix

## Test Coverage

✅ Valid JSON parsing
✅ Valid YAML parsing
✅ Invalid configuration rejection
✅ Array structure validation
✅ Matrix generation
✅ Matrix execution in workflows
✅ Error handling

## Adding New Tests

1. Add test configuration files to this directory
2. Update `test.sh` with new test cases
3. Add new test jobs to `.github/workflows/test.yml` if needed
