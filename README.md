# action-config

A GitHub Action for reading configuration files and generating dynamic workflow matrices. This action helps you avoid hardcoding matrix configurations in your workflows by reading them from JSON or YAML files.

## Features

- 📄 Supports both JSON and YAML configuration files
- ✅ Validates configuration syntax before use
- 🔄 Generates dynamic matrices for GitHub Actions workflows
- 🎯 Perfect for managing multiple environments, stacks, or deployments

## Usage

### Basic Example

Create a configuration file in your repository (e.g., `.github/matrix-config.json`):

```json
[
  {
    "stack": "infra",
    "environment": "dev",
    "aws_account_id": "851725213542"
  },
  {
    "stack": "infra",
    "environment": "prod",
    "aws_account_id": "730335665754"
  }
]
```

Then use it in your workflow:

```yaml
name: Deploy

on:
  pull_request:

jobs:
  setup:
    runs-on: ubuntu-latest
    outputs:
      matrix: ${{ steps.set-matrix.outputs.matrix }}
    steps:
      - uses: actions/checkout@v4

      - name: Read matrix configuration
        id: set-matrix
        uses: DND-IT/action-config@v1
        with:
          config-path: '.github/matrix-config.json'

  deploy:
    needs: setup
    strategy:
      fail-fast: false
      matrix:
        include: ${{ fromJson(needs.setup.outputs.matrix) }}
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Deploy ${{ matrix.stack }} to ${{ matrix.environment }}
        run: |
          echo "Deploying to ${{ matrix.environment }}"
          echo "AWS Account: ${{ matrix.aws_account_id }}"
```

### Terraform Plan Example

Replace your hardcoded matrix with a dynamic one:

**Before:**
```yaml
jobs:
  plan:
    strategy:
      matrix:
        include:
          - stack: infra
            environment: dev
            aws_account_id: 851725213542
          - stack: infra
            environment: prod
            aws_account_id: 730335665754
    uses: DND-IT/github-workflows/.github/workflows/tf-plan.yaml@v3
    # ...
```

**After:**
```yaml
jobs:
  setup:
    runs-on: ubuntu-latest
    outputs:
      matrix: ${{ steps.set-matrix.outputs.matrix }}
    steps:
      - uses: actions/checkout@v4
      - id: set-matrix
        uses: DND-IT/action-config@v1

  plan:
    needs: setup
    strategy:
      matrix:
        include: ${{ fromJson(needs.setup.outputs.matrix) }}
    uses: DND-IT/github-workflows/.github/workflows/tf-plan.yaml@v3
    with:
      tf_dir: deploy/${{ matrix.stack }}
      tf_backend_config_files: deploy/${{ matrix.stack }}/environments/${{ matrix.environment }}.s3.tfbackend
      tf_var_files: deploy/${{ matrix.stack }}/environments/${{ matrix.environment }}.tfvars
      aws_oidc_role_arn: arn:aws:iam::${{ matrix.aws_account_id }}:role/cicd-iac
```

## Inputs

| Input | Description | Required | Default |
|-------|-------------|----------|---------|
| `config-path` | Path to the configuration file (JSON or YAML) | No | `.github/matrix-config.json` |

## Outputs

| Output | Description |
|--------|-------------|
| `matrix` | JSON string containing the matrix configuration |

## Configuration File Formats

### JSON Format

```json
[
  {
    "key1": "value1",
    "key2": "value2"
  },
  {
    "key1": "value3",
    "key2": "value4"
  }
]
```

### YAML Format

```yaml
- key1: value1
  key2: value2

- key1: value3
  key2: value4
```

## Examples

See the [example workflow](.github/workflows/example.yml) and example configuration files:
- [JSON example](.github/matrix-config.example.json)
- [YAML example](.github/matrix-config.example.yml)

## Benefits

✅ **Centralized Configuration**: Manage all your matrix configurations in one place
✅ **Version Control**: Track changes to your deployment configurations
✅ **Reusability**: Use the same configuration across multiple workflows
✅ **Flexibility**: Easy to add/remove environments without modifying workflows
✅ **Validation**: Automatic syntax validation before workflow execution

## Development

This is a composite action that uses shell scripts to read and validate configuration files. The action:

1. Reads the specified configuration file
2. Validates the JSON/YAML syntax
3. Outputs the configuration as a JSON string for use in matrix strategies

### Testing

Run tests locally:
```bash
./tests/test.sh
```

The action includes comprehensive tests:
- ✅ JSON and YAML parsing
- ✅ Invalid configuration rejection
- ✅ Matrix generation and execution
- ✅ Error handling

See [tests/README.md](tests/README.md) for more details.

## Releases

This action uses semantic-release for automated versioning based on conventional commits. When you push to `main`, a new release is automatically created if there are significant changes.

### Commit Message Format

Use conventional commits to automatically determine the version bump:

**Triggers Release:**
- `feat:` - ✨ New feature (minor version bump, e.g., 1.0.0 → 1.1.0)
- `fix:` - 🐛 Bug fix (patch version bump, e.g., 1.0.0 → 1.0.1)
- `perf:` - ⚡ Performance improvement (patch version bump)
- `revert:` - ⏪ Revert changes (patch version bump)
- `BREAKING CHANGE:` - 💥 Breaking change (major version bump, e.g., 1.0.0 → 2.0.0)

**No Release (documentation only):**
- `docs:` - 📚 Documentation changes
- `refactor:` - ♻️ Code refactoring
- `style:` - 💎 Code style changes
- `chore:` - 🔧 Maintenance tasks
- `test:` - ✅ Test updates
- `build:` - 🏗️ Build system changes
- `ci:` - 👷 CI/CD changes

The release notes are automatically generated and categorized with these emojis. A `CHANGELOG.md` file is automatically maintained in the repository.

### Manual Releases

You can also trigger a manual release via the Actions tab:

1. Go to Actions → Release
2. Click "Run workflow"
3. Enter a tag (e.g., `v1.2.3`)

### Version Aliases

The release workflow automatically updates version aliases:
- `v1` - Always points to the latest v1.x.x release
- `v1.2` - Always points to the latest v1.2.x release

This allows users to pin to major or minor versions:
```yaml
- uses: DND-IT/action-config@v1  # Always gets latest v1.x.x
- uses: DND-IT/action-config@v1.2  # Always gets latest v1.2.x
- uses: DND-IT/action-config@v1.2.3  # Pinned to specific version
```

## License

MIT

## Support

Maintained by **group:default/dai**
