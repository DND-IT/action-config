# action-config

A GitHub Action for centralized configuration management that enables you to define your workflow matrices once and reuse them everywhere. Stop duplicating configuration across multiple workflows!

## Why Use This?

**The Problem:** You need to deploy to multiple environments (dev, staging, prod) and run different workflows (plan, apply, test, lint). Without this action, you'd have to:
- Duplicate the same matrix configuration in every workflow file
- Update 5+ files when adding a new environment
- Risk inconsistencies between workflows
- Maintain hundreds of lines of repetitive YAML

**The Solution:** Define your matrix once in a config file and reference it in all workflows. Add a new environment? Just update one file.

## Quick Example

**One Config File** (`.github/matrix-config.json`):
```json
[
  {
    "stack": "api",
    "environment": "dev",
    "aws_account_id": "111111111111",
    "aws_region": "us-east-1"
  },
  {
    "stack": "api",
    "environment": "staging",
    "aws_account_id": "222222222222",
    "aws_region": "us-east-1"
  },
  {
    "stack": "api",
    "environment": "prod",
    "aws_account_id": "333333333333",
    "aws_region": "us-west-2"
  },
  {
    "stack": "frontend",
    "environment": "dev",
    "aws_account_id": "111111111111",
    "aws_region": "us-east-1"
  },
  {
    "stack": "frontend",
    "environment": "staging",
    "aws_account_id": "222222222222",
    "aws_region": "us-east-1"
  },
  {
    "stack": "frontend",
    "environment": "prod",
    "aws_account_id": "333333333333",
    "aws_region": "us-west-2"
  }
]
```

**Multiple Jobs in the Same Workflow - All Reusing the Same Setup:**

<details>
<summary>🚀 Terraform Workflow</summary>

```yaml
name: Terraform Workflow

on:
  push:
    branches: [main]

jobs:
  # Setup once - both plan and apply reuse this
  setup:
    runs-on: ubuntu-latest
    outputs:
      matrix: ${{ steps.set-matrix.outputs.matrix }}
    steps:
      - uses: actions/checkout@v5
      - id: set-matrix
        uses: DND-IT/action-config@v1

  # Plan before apply for all environments
  plan:
    needs: setup
    strategy:
      fail-fast: false
      matrix:
        include: ${{ fromJson(needs.setup.outputs.matrix) }}
    uses: DND-IT/github-workflows/.github/workflows/tf-plan.yaml@v3
    with:
      environment: ${{ matrix.environment }}
      aws_account_id: ${{ matrix.aws_account_id }}
      aws_region: ${{ matrix.aws_region }}
      aws_oidc_role_arn: arn:aws:iam::${{ matrix.aws_account_id }}:role/cicd-iac
      tf_dir: deploy/${{ matrix.stack }}
      tf_backend_config_files: deploy/${{ matrix.stack }}/environments/${{ matrix.environment }}.s3.tfbackend
      tf_var_files: deploy/${{ matrix.stack }}/environments/${{ matrix.environment }}.tfvars

  # Apply using the reusable workflow
  apply:
    needs: [setup, plan]
    strategy:
      fail-fast: false
      matrix:
        include: ${{ fromJson(needs.setup.outputs.matrix) }}
    uses: DND-IT/github-workflows/.github/workflows/tf-apply.yaml@v3
    with:
      environment: ${{ matrix.environment }}
      aws_account_id: ${{ matrix.aws_account_id }}
      aws_region: ${{ matrix.aws_region }}
      aws_oidc_role_arn: arn:aws:iam::${{ matrix.aws_account_id }}:role/cicd-iac
      tf_dir: deploy/${{ matrix.stack }}
      tf_backend_config_files: deploy/${{ matrix.stack }}/environments/${{ matrix.environment }}.s3.tfbackend
      tf_var_files: deploy/${{ matrix.stack }}/environments/${{ matrix.environment }}.tfvars
```
</details>

**Result:**
- **Plan workflow**: 1 setup + 6 plan jobs (one per environment) = 7 total jobs
- **Apply workflow**: 1 setup + 6 plan + 6 apply jobs = 13 total jobs
- **Total**: 20 jobs across both workflows, all using the same config file

**The Power of This Pattern:**
- ✅ One `setup` job reads the config
- ✅ Multiple jobs (`plan`, `apply`) reuse the same matrix
- ✅ All Terraform logic lives in reusable workflows
- ✅ Your workflow file is just configuration mapping

Add a new environment? Update one config file, and all jobs instantly include it!

## Benefits

| Without action-config | With action-config |
|----------------------|-------------------|
| ❌ 150+ lines of duplicated YAML across 3 workflows | ✅ 30 lines in one config file |
| ❌ Update 3+ workflows to add an environment | ✅ Update 1 config file |
| ❌ Risk of drift between workflows | ✅ Single source of truth |
| ❌ Hard to maintain and error-prone | ✅ Easy to maintain and validate |

### Key Benefits

- **🎯 Single Source of Truth**: Define your environments once, use everywhere
- **♻️ Reusability**: Share the same matrix across plan, apply, test, lint, and custom workflows
- **🛡️ Consistency**: No more config drift between workflows
- **⚡ Faster Development**: Add environments in seconds, not hours
- **✅ Automatic Validation**: Catches JSON/YAML syntax errors before workflow runs
- **📦 Version Control**: Track all config changes in one place with proper git history

## Getting Started

### 1. Create Your Config File

Create a configuration file in your repository (`.github/matrix-config.json`):

```json
[
  {
    "environment": "dev",
    "aws_account_id": "111111111111"
  },
  {
    "environment": "prod",
    "aws_account_id": "222222222222"
  }
]
```

### 2. Use in Your Workflow

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
        with:
          config-path: '.github/matrix-config.json'  # optional, this is the default

  deploy:
    needs: setup
    strategy:
      matrix:
        include: ${{ fromJson(needs.setup.outputs.matrix) }}
    runs-on: ubuntu-latest
    steps:
      - name: Deploy to ${{ matrix.environment }}
        run: echo "Deploying..."
```

That's it! Your workflow now uses the centralized config.

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

See the [example workflow](.github/workflows/example.yaml) and [example configuration file](.github/matrix-config.example.json) for a complete working example.

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

Maintained by **DAI**
