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

Instead of writing 6 duplicate entries for 2 stacks x 3 environments, write this:

```json
{
  "stacks": ["api", "frontend"],
  "config": {
    "dev": {
      "aws_account_id": "111111111111",
      "aws_region": "us-east-1"
    },
    "staging": {
      "aws_account_id": "222222222222",
      "aws_region": "us-east-1"
    },
    "prod": {
      "aws_account_id": "333333333333",
      "aws_region": "us-west-2"
    }
  }
}
```

**This automatically expands to 6 matrix entries:**
```json
[
  { "stack": "api", "environment": "dev", "aws_account_id": "111111111111", "aws_region": "us-east-1" },
  { "stack": "api", "environment": "staging", "aws_account_id": "222222222222", "aws_region": "us-east-1" },
  { "stack": "api", "environment": "prod", "aws_account_id": "333333333333", "aws_region": "us-west-2" },
  { "stack": "frontend", "environment": "dev", "aws_account_id": "111111111111", "aws_region": "us-east-1" },
  { "stack": "frontend", "environment": "staging", "aws_account_id": "222222222222", "aws_region": "us-east-1" },
  { "stack": "frontend", "environment": "prod", "aws_account_id": "333333333333", "aws_region": "us-west-2" }
]
```

Environments are automatically derived from the `config` keys — no need to list them separately.

**Add a new stack?** Just add `"database"` to the `stacks` array — instantly get 3 more environments!
**Add a new environment?** Just add its config block — all stacks automatically deploy there!

**Multiple Jobs in the Same Workflow - All Reusing the Same Setup:**

<details>
<summary>Terraform Workflow</summary>

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
        uses: DND-IT/action-config@v2

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
- One `setup` job reads the config
- Multiple jobs (`plan`, `apply`) reuse the same matrix
- All Terraform logic lives in reusable workflows
- Your workflow file is just configuration mapping

Add a new environment? Update one config file, and all jobs instantly include it!

## Getting Started

### 1. Create Your Config File

Create a configuration file in your repository (`.github/matrix-config.json`):

```json
{
  "stacks": ["api"],
  "config": {
    "dev": {
      "aws_account_id": "111111111111"
    },
    "prod": {
      "aws_account_id": "222222222222"
    }
  }
}
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
        uses: DND-IT/action-config@v2
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

## Configuration Format

The configuration file must be a JSON or YAML **object**. Any top-level array becomes a matrix dimension, and the `config` object provides per-environment values.

### Basic Example

**JSON:**
```json
{
  "stacks": ["api", "frontend"],
  "config": {
    "dev": { "aws_account_id": "111111111111" },
    "staging": { "aws_account_id": "222222222222" },
    "prod": { "aws_account_id": "333333333333" }
  }
}
```

**YAML:**
```yaml
stacks:
  - api
  - frontend

config:
  dev:
    aws_account_id: "111111111111"
  staging:
    aws_account_id: "222222222222"
  prod:
    aws_account_id: "333333333333"
```

**How it works:**
- Any top-level array (like `stacks`) becomes a matrix dimension
- The `config` object's **keys** automatically become the `environment` dimension
- Each config value is merged into matching matrix entries
- Non-array, non-config top-level values are copied to every matrix entry
- Plural keys are singularized (`stacks` → `stack`, `environments` → `environment`)
- Result: 2 stacks x 3 environments = 6 matrix entries

### Exclude

Use `exclude` to remove specific combinations from the cartesian product. This follows the same pattern as [GitHub Actions matrix exclude](https://docs.github.com/en/actions/writing-workflows/choosing-what-your-workflows-do/running-variations-of-jobs-in-a-workflow#excluding-matrix-configurations).

```json
{
  "stacks": ["api", "frontend", "shared"],
  "config": {
    "dev": { "aws_account_id": "111111111111" },
    "prod": { "aws_account_id": "222222222222" }
  },
  "exclude": [
    { "stack": "shared", "environment": "dev" }
  ]
}
```

This produces 5 entries (3 stacks x 2 envs = 6, minus `shared/dev`).

Each exclude entry is a partial match — any matrix item matching **all** the key/value pairs in the pattern is removed.

### Include

Use `include` to append standalone entries that bypass the cartesian product. Useful for stacks that don't need environments at all.

```json
{
  "stacks": ["api", "frontend"],
  "config": {
    "dev": { "aws_account_id": "111111111111" },
    "prod": { "aws_account_id": "222222222222" }
  },
  "include": [
    { "stack": "shared", "aws_account_id": "333333333333" }
  ]
}
```

This produces 5 entries: the 4 from the cartesian product, plus the `shared` entry appended at the end (with no `environment` field).

### Using Exclude and Include Together

You can combine both to fully control the matrix:

```json
{
  "stacks": ["api", "shared"],
  "config": {
    "dev": { "aws_account_id": "111111111111" },
    "prod": { "aws_account_id": "222222222222" }
  },
  "exclude": [
    { "stack": "shared" }
  ],
  "include": [
    { "stack": "shared", "aws_account_id": "333333333333" }
  ]
}
```

This removes all `shared` entries from the cartesian product (both `shared/dev` and `shared/prod`), then appends a single `shared` entry without an environment.

## Examples

See the [example workflow](.github/workflows/example.yaml) and example configuration files:
- [JSON](.github/matrix-config.example.json) | [YAML](.github/matrix-config.example.yaml)

## Development

This is a composite action that uses shell scripts to read and validate configuration files. The action:

1. Reads the specified configuration file
2. Validates the JSON/YAML syntax
3. Expands the configuration into a cartesian product matrix
4. Applies exclude/include rules
5. Outputs the configuration as a JSON string for use in matrix strategies

### Testing

Run tests locally:
```bash
./tests/test.sh
```

See [tests/README.md](tests/README.md) for more details.

## Releases

This action uses semantic-release for automated versioning based on conventional commits. When you push to `main`, a new release is automatically created if there are significant changes.

### Commit Message Format

Use conventional commits to automatically determine the version bump:

**Triggers Release:**
- `feat:` - New feature (minor version bump, e.g., 1.0.0 → 1.1.0)
- `fix:` - Bug fix (patch version bump, e.g., 1.0.0 → 1.0.1)
- `perf:` - Performance improvement (patch version bump)
- `revert:` - Revert changes (patch version bump)
- `BREAKING CHANGE:` - Breaking change (major version bump, e.g., 1.0.0 → 2.0.0)

**No Release (documentation only):**
- `docs:` - Documentation changes
- `refactor:` - Code refactoring
- `style:` - Code style changes
- `chore:` - Maintenance tasks
- `test:` - Test updates
- `build:` - Build system changes
- `ci:` - CI/CD changes

### Version Aliases

The release workflow automatically updates version aliases:
- `v2` - Always points to the latest v2.x.x release
- `v2.1` - Always points to the latest v2.1.x release

This allows users to pin to major or minor versions:
```yaml
- uses: DND-IT/action-config@v2  # Always gets latest v2.x.x
- uses: DND-IT/action-config@v2.1  # Always gets latest v2.1.x
- uses: DND-IT/action-config@v2.1.0  # Pinned to specific version
```

## License

MIT

## Support

Maintained by **DAI**
