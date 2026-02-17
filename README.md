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
{
  "global": {
    "dimension_key": "service",
    "base_dir": "deploy",
    "aws_region": "us-east-1"
  },
  "environment": {
    "dev": {
      "aws_account_id": "111111111111"
    },
    "staging": {
      "aws_account_id": "222222222222"
    },
    "prod": {
      "aws_account_id": "333333333333",
      "aws_region": "us-west-2"
    }
  },
  "service": {
    "api": { "port": "8080" },
    "frontend": null
  }
}
```

**This automatically expands to 6 matrix entries** (sorted by environment by default):
```json
[
  { "service": "api", "environment": "dev", "directory": "deploy/api", "aws_account_id": "111111111111", "aws_region": "us-east-1", "port": "8080" },
  { "service": "frontend", "environment": "dev", "directory": "deploy/frontend", "aws_account_id": "111111111111", "aws_region": "us-east-1" },
  { "service": "api", "environment": "prod", "directory": "deploy/api", "aws_account_id": "333333333333", "aws_region": "us-west-2", "port": "8080" },
  { "service": "frontend", "environment": "prod", "directory": "deploy/frontend", "aws_account_id": "333333333333", "aws_region": "us-west-2" },
  { "service": "api", "environment": "staging", "directory": "deploy/api", "aws_account_id": "222222222222", "aws_region": "us-east-1", "port": "8080" },
  { "service": "frontend", "environment": "staging", "directory": "deploy/frontend", "aws_account_id": "222222222222", "aws_region": "us-east-1" }
]
```

Dimensions are defined as top-level keys (maps or arrays). The `global` key holds shared settings and config values merged into every entry. Per-dimension-value configs (like `aws_account_id` per environment) are embedded directly in the dimension map. The `directory` field is automatically computed from `base_dir` and the primary dimension value.

**Add a new service?** Just add a key to the `service` map — instantly get 3 more environments!
**Add a new environment?** Just add its config block — all services automatically deploy there!

**Multiple Jobs in the Same Workflow - All Reusing the Same Setup:**

<details>
<summary>Terraform Workflow</summary>

```yaml
name: Terraform Workflow

on:
  push:
    branches: [main]

jobs:
  setup:
    runs-on: ubuntu-latest
    outputs:
      matrix: ${{ steps.set-matrix.outputs.matrix }}
    steps:
      - uses: actions/checkout@v5
      - id: set-matrix
        uses: DND-IT/action-config@v3

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
      tf_dir: ${{ matrix.directory }}
      tf_backend_config_files: ${{ matrix.directory }}/environments/${{ matrix.environment }}.s3.tfbackend
      tf_var_files: ${{ matrix.directory }}/environments/${{ matrix.environment }}.tfvars

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
      tf_dir: ${{ matrix.directory }}
      tf_backend_config_files: ${{ matrix.directory }}/environments/${{ matrix.environment }}.s3.tfbackend
      tf_var_files: ${{ matrix.directory }}/environments/${{ matrix.environment }}.tfvars
```
</details>

## Getting Started

### 1. Create Your Config File

Create a configuration file in your repository (`.github/matrix-config.json`):

```json
{
  "environment": {
    "dev": { "aws_account_id": "111111111111" },
    "prod": { "aws_account_id": "222222222222" }
  },
  "service": {
    "api": null,
    "frontend": null
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
        uses: DND-IT/action-config@v3
        with:
          config_path: '.github/matrix-config.json'  # optional, this is the default

  deploy:
    needs: setup
    strategy:
      matrix:
        include: ${{ fromJson(needs.setup.outputs.matrix) }}
    runs-on: ubuntu-latest
    steps:
      - name: Deploy to ${{ matrix.environment }}
        run: echo "Deploying ${{ matrix.service }} from ${{ matrix.directory }}"
```

## Inputs

| Input | Description | Required | Default |
|-------|-------------|----------|---------|
| `config_path` | Path to the configuration file (JSON or YAML) | No | `.github/matrix-config.json` |
| `target` | Filter by dimension_key value(s). Comma-separated for multiple (e.g. `api,frontend`). | No | |
| `environment` | Filter environments. Comma-separated for multiple (e.g. `dev,prod`) | No | |
| `exclude` | JSON array of patterns to exclude (e.g. `[{"service":"shared","environment":"dev"}]`) | No | |
| `include` | JSON array of entries to append (e.g. `[{"service":"shared"}]`) | No | |
| `change_detection` | Filter matrix to only entries with file changes. Uses `base_dir` from global to map file paths. Requires `actions/checkout` with `fetch-depth: 0`. | No | `false` |

The `target` and `environment` inputs are convenience filters applied **after** the config file is expanded. The `exclude` and `include` inputs work the same way as their config file counterparts but are applied after them, allowing workflow-level overrides.

## Outputs

| Output | Description |
|--------|-------------|
| `matrix` | JSON string containing the matrix configuration |
| `changes_detected` | Whether any entries have changes (`true`/`false`). Only meaningful when `change_detection` is `true`. |

## Configuration Format

The configuration file must be a JSON or YAML **object**. There are three reserved top-level keys (`global`, `exclude`, `include`). Everything else is a dimension.

### Reserved Top-Level Keys

| Key | Description |
|-----|-------------|
| `global` | Settings and shared config values. Contains action settings (`dimension_key`, `base_dir`, `sort_by`) plus any other keys which are merged as config values into every entry. |
| `exclude` | Array of patterns to exclude from the cartesian product. |
| `include` | Array of entries to append to the matrix. |

### Dimensions

Any non-reserved top-level key is a dimension. Dimensions can be:

- **Map dimensions** — keys become dimension values, objects hold per-value config:
  ```yaml
  environment:
    dev:
      aws_account_id: "111111111111"
    prod:
      aws_account_id: "222222222222"
  ```
- **Array dimensions** — simple list of values:
  ```yaml
  region:
    - us-east-1
    - eu-west-1
  ```

Both formats are supported and can be mixed in the same config.

### Global Settings

The `global` key holds action settings and shared config values:

| Key | Description | Default |
|-----|-------------|---------|
| `dimension_key` | Name of the primary dimension key (used for filtering via `services` input and change detection) | `"service"` |
| `base_dir` | Base directory for building the `directory` output field and mapping file paths for change detection | (empty) |
| `sort_by` | Array of keys to sort the matrix entries by | `["environment"]` |

Any other key in `global` is a config value merged into every matrix entry. For example, `aws_region: us-east-1` in `global` gives every entry that value unless overridden by a per-dimension-value config.

### Merge Order

For each matrix entry, values are merged in this order (later overrides earlier):

1. **Scalar top-level values** — non-dimension, non-reserved top-level values
2. **Global config values** — `global` minus reserved keys (`dimension_key`, `base_dir`, `sort_by`)
3. **Dimension values** — e.g. `environment: dev`, `service: api`
4. **Per-dimension-value configs** — in alphabetical dimension key order (e.g. `environment` before `service`)

### Basic Example

**JSON:**
```json
{
  "environment": {
    "dev": { "aws_account_id": "111111111111" },
    "staging": { "aws_account_id": "222222222222" },
    "prod": { "aws_account_id": "333333333333" }
  },
  "service": {
    "api": null,
    "frontend": null
  }
}
```

**YAML:**
```yaml
environment:
  dev:
    aws_account_id: "111111111111"
  staging:
    aws_account_id: "222222222222"
  prod:
    aws_account_id: "333333333333"

service:
  api:
  frontend:
```

**How it works:**
- Map dimensions (like `environment` and `service`) have their keys become dimension values
- Array dimensions have their items become dimension values
- Per-value config objects in map dimensions are merged into matching entries
- Non-array, non-map, non-reserved top-level values are copied to every matrix entry
- A `directory` field is automatically added: `base_dir/value` (or just `value` if `base_dir` is not set)
- Result: 2 services x 3 environments = 6 matrix entries

### Shared Config Values

Use the `global` key to define values shared across all entries. Per-dimension-value configs override global values:

```json
{
  "global": {
    "aws_region": "us-east-1",
    "timeout": "30"
  },
  "environment": {
    "dev": { "aws_account_id": "111111111111" },
    "prod": {
      "aws_account_id": "222222222222",
      "aws_region": "us-west-2"
    }
  },
  "service": { "api": null }
}
```

This produces:
- `api/dev`: `aws_region: "us-east-1"` (from global), `timeout: "30"`, `aws_account_id: "111111111111"`
- `api/prod`: `aws_region: "us-west-2"` (overrides global), `timeout: "30"`, `aws_account_id: "222222222222"`

### Per-Dimension-Value Config

Map dimensions can embed config per value. These are merged in alphabetical dimension key order:

```json
{
  "environment": {
    "dev": { "aws_account_id": "111111111111" },
    "prod": { "aws_account_id": "222222222222" }
  },
  "service": {
    "api": { "port": "8080" },
    "frontend": { "port": "3000" }
  }
}
```

For the `api/dev` entry: first `environment:dev` config is applied (`aws_account_id`), then `service:api` config is applied (`port`). If both dimensions set the same key, the later one alphabetically wins.

### Sorting

Matrix entries are sorted by `["environment"]` by default, which groups entries by environment. Override with `sort_by` in global:

```json
{
  "global": {
    "sort_by": ["service", "environment"]
  },
  "environment": {
    "dev": { "aws_account_id": "111111111111" },
    "prod": { "aws_account_id": "222222222222" }
  },
  "service": { "frontend": null, "api": null }
}
```

With the default sort (`["environment"]`), entries are grouped as: all `dev` entries, then all `prod` entries.
With `["service", "environment"]`, entries are grouped as: `api/dev`, `api/prod`, `frontend/dev`, `frontend/prod`.

### Custom Primary Dimension

The dimension name is fully configurable. You can use `service`, `app`, `component`, or any name that fits your project:

```json
{
  "global": {
    "dimension_key": "app",
    "base_dir": "apps"
  },
  "environment": {
    "dev": { "cluster": "dev-cluster" },
    "prod": { "cluster": "prod-cluster" }
  },
  "app": { "web": null, "worker": null, "cron": null }
}
```

This produces entries like `{"app": "web", "environment": "dev", "directory": "apps/web", "cluster": "dev-cluster"}`.

### Exclude

Use `exclude` to remove specific combinations from the cartesian product:

```json
{
  "exclude": [
    { "service": "shared", "environment": "dev" }
  ],
  "environment": {
    "dev": { "aws_account_id": "111111111111" },
    "prod": { "aws_account_id": "222222222222" }
  },
  "service": { "api": null, "frontend": null, "shared": null }
}
```

This produces 5 entries (3 services x 2 envs = 6, minus `shared/dev`).

Each exclude entry is a partial match — any matrix item matching **all** the key/value pairs in the pattern is removed.

### Include

Use `include` to append standalone entries that bypass the cartesian product:

```json
{
  "include": [
    { "service": "shared", "aws_account_id": "333333333333" }
  ],
  "environment": {
    "dev": { "aws_account_id": "111111111111" },
    "prod": { "aws_account_id": "222222222222" }
  },
  "service": { "api": null, "frontend": null }
}
```

This produces 5 entries: the 4 from the cartesian product, plus the `shared` entry appended at the end (with no `environment` field).

### Using Exclude and Include Together

You can combine both to fully control the matrix:

```json
{
  "exclude": [
    { "service": "shared" }
  ],
  "include": [
    { "service": "shared", "aws_account_id": "333333333333" }
  ],
  "environment": {
    "dev": { "aws_account_id": "111111111111" },
    "prod": { "aws_account_id": "222222222222" }
  },
  "service": { "api": null, "shared": null }
}
```

This removes all `shared` entries from the cartesian product (both `shared/dev` and `shared/prod`), then appends a single `shared` entry without an environment.

### Filtering via Action Inputs

The `target`, `environment`, `exclude`, and `include` inputs let you filter at the workflow level without changing the config file. This is especially useful with `workflow_dispatch`:

```yaml
on:
  workflow_dispatch:
    inputs:
      environment:
        description: 'Target environment'
        type: choice
        options:
          - ''
          - dev
          - staging
          - prod

jobs:
  setup:
    runs-on: ubuntu-latest
    outputs:
      matrix: ${{ steps.set-matrix.outputs.matrix }}
    steps:
      - uses: actions/checkout@v4
      - id: set-matrix
        uses: DND-IT/action-config@v3
        with:
          environment: ${{ inputs.environment }}

  deploy:
    needs: setup
    strategy:
      matrix:
        include: ${{ fromJson(needs.setup.outputs.matrix) }}
    runs-on: ubuntu-latest
    steps:
      - run: echo "Deploying ${{ matrix.service }} to ${{ matrix.environment }}"
```

When triggered manually with `environment: prod`, only prod entries are included. When left empty, all environments are included.

### Running Jobs Sequentially

By default, matrix jobs run in parallel. To run them one at a time, set `max-parallel: 1` in the strategy:

```yaml
jobs:
  setup:
    runs-on: ubuntu-latest
    outputs:
      matrix: ${{ steps.set-matrix.outputs.matrix }}
    steps:
      - uses: actions/checkout@v4
      - id: set-matrix
        uses: DND-IT/action-config@v3

  deploy:
    needs: setup
    strategy:
      max-parallel: 1
      matrix:
        include: ${{ fromJson(needs.setup.outputs.matrix) }}
    runs-on: ubuntu-latest
    steps:
      - run: echo "Deploying ${{ matrix.service }} to ${{ matrix.environment }}"
```

> **Note:** GitHub Actions does not guarantee the execution order of matrix entries when using `max-parallel: 1`. If you need a strict order (e.g., deploy to `dev` before `prod`), split them into separate jobs with `needs` dependencies.

## Examples

See the [example workflow](.github/workflows/example.yaml) and example configuration files:
- [JSON](.github/matrix-config.example.json) | [YAML](.github/matrix-config.example.yaml)

## Development

This action is written in Go and runs as a Docker container. It:

1. Reads the specified configuration file
2. Parses the `global` block and dimension maps/arrays
3. Expands the configuration into a cartesian product matrix
4. Applies exclude/include rules and filters
5. Adds the `directory` field to each entry
6. Outputs the configuration as a JSON string for use in matrix strategies

### Testing

Run tests locally:
```bash
go test -v -race ./...
```

### Building

```bash
go build -o action-config ./cmd/action-config
```

## Releases

This action uses semantic-release for automated versioning based on conventional commits. When you push to `main`, a new release is automatically created if there are significant changes.

### Commit Message Format

Use conventional commits to automatically determine the version bump:

**Triggers Release:**
- `feat:` - New feature (minor version bump, e.g., 1.0.0 -> 1.1.0)
- `fix:` - Bug fix (patch version bump, e.g., 1.0.0 -> 1.0.1)
- `perf:` - Performance improvement (patch version bump)
- `revert:` - Revert changes (patch version bump)
- `BREAKING CHANGE:` - Breaking change (major version bump, e.g., 1.0.0 -> 2.0.0)

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
- `v3` - Always points to the latest v3.x.x release
- `v3.1` - Always points to the latest v3.1.x release

This allows users to pin to major or minor versions:
```yaml
- uses: DND-IT/action-config@v3        # Always gets latest v3.x.x
- uses: DND-IT/action-config@v3.1      # Always gets latest v3.1.x
- uses: DND-IT/action-config@v3.1.0    # Pinned to specific version
```

## License

MIT

## Support

Maintained by **DAI**
