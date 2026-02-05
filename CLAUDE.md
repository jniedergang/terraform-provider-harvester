# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Git Repositories

**Working repositories:**
- https://github.com/jniedergang/terraform-provider-harvester
- https://gitea.home.zypp.fr/jniedergang/terraform-provider-harvester

**Reference (upstream):**
- https://github.com/harvester/terraform-provider-harvester

## Build Commands

```bash
# Build provider binaries (amd64 and arm64)
make build

# Run tests with coverage (generates coverage.html)
make test

# Run linters and formatters (golangci-lint, go fmt, go generate)
make validate

# Generate documentation (terraform-plugin-docs)
make generate

# Interactive dev helper
./dev-env.sh
```

### Acceptance Tests

Acceptance tests require a `kubeconfig_test.yaml` file in the repository root:
```bash
TF_ACC=1 go test ./internal/tests/... -v
```

### Test Environment

Le fichier kubeconfig pour gérer le cluster Harvester de test : `/root/workspace/TERRAFORM/TESTBACKUP/rke2.yaml`

Pour lancer les tests avec ce cluster :
```bash
cp /root/workspace/TERRAFORM/TESTBACKUP/rke2.yaml ./kubeconfig_test.yaml
TF_ACC=1 go test ./internal/tests/... -v
```

Le Terraform dans `/root/workspace/TERRAFORM/TESTBACKUP` permet de gérer la VM `test-vm2` qui peut être utilisée pour les tests manuels.

## Architecture

### Provider Overview

Terraform provider for Harvester HCI (Hyper-Converged Infrastructure). Uses HashiCorp Terraform Plugin SDK v2 with Kubernetes client-go for API interactions.

- Minimum Terraform: >= 0.13.x
- Minimum Harvester: v1.1.0 (v1.0.x not supported)

### Project Structure

```
internal/
├── config/          # Provider configuration and K8s client initialization
├── provider/        # Provider definition and all resources/data sources
│   └── <resource>/  # Each resource has its own directory
├── tests/           # Acceptance tests (resource_*_test.go)
└── util/            # Constructor pattern, schema utilities, state management

pkg/
├── client/          # Multi-client aggregator (KubeVirt, Harvester CRDs, etc.)
├── constants/       # Field and resource type constants
├── helper/          # ID/naming utilities (namespace/name format)
└── importer/        # Resource importers
```

### Resource Pattern

Each resource follows this structure in `internal/provider/<resource>/`:

| File | Purpose |
|------|---------|
| `schema_<name>.go` | Schema definition with field types and validation |
| `resource_<name>.go` | CRUD operations (Create, Read, Update, Delete) |
| `resource_<name>_constructor.go` | Builds Kubernetes objects from Terraform state |
| `datasource_<name>.go` | Read-only data source implementation |
| `schema_<name>_*.go` | Nested schema definitions for complex types |

### Key Patterns

**Constructor Pattern** (`internal/util/constructor.go`):
- `Constructor` interface with `Setup()`, `Result()`, `Validate()` methods
- `Processor` pattern for field parsing with required/optional validation

**Schema Wrapping** (`internal/util/schema.go`):
- `NamespacedSchemaWrap()` - adds common fields (name, namespace, tags, labels, description, state, message)
- `NonNamespacedSchemaWrap()` - for cluster-level resources
- `DataSourceSchemaWrap()` - converts resource schema to read-only

**ID Format** (`pkg/helper/id.go`):
- Namespaced resources: `"namespace/name"`
- Non-namespaced: `"name"`
- Use `helper.BuildID()` and `helper.IDParts()`

**Constants** (`pkg/constants/`):
- Resource types: `constants.ResourceTypeVirtualMachine`
- Field names: `constants.FieldCommonName`, `constants.FieldVirtualMachineCPU`
- Define new constants in `constants_<resource>.go`

### Client Architecture

`pkg/client/client.go` aggregates multiple clients:
- `KubeClient` - standard Kubernetes
- `HarvesterClient` - Harvester CRDs
- `HarvesterNetworkClient` - network controller CRDs
- `HarvesterLoadbalancerClient` - load balancer CRDs
- `KubeVirtSubresourceClient` - KubeVirt subresources

### Testing

Tests use HashiCorp's testing framework in `internal/tests/`:
- `VMResourceBuilder` - fluent builder for VM test configs
- `testAccPreCheck()` - validates kubeconfig before tests
- Use `getStateChangeConf()` for waiting on resource deletion

## Adding a New Resource

1. Create directory `internal/provider/<resource>/`
2. Add constants in `pkg/constants/constants_<resource>.go`
3. Implement schema, resource operations, and constructor
4. Register in `internal/provider/provider.go` (ResourcesMap and DataSourcesMap)
5. Add acceptance tests in `internal/tests/resource_<name>_test.go`
6. Run `go generate` to update documentation

## Harvester Upstream Guidelines

Reference documentation from the main Harvester project:
- [CONTRIBUTING.md](https://github.com/harvester/harvester/blob/master/CONTRIBUTING.md)
- [DEVELOPER_GUIDE.md](https://github.com/harvester/harvester/blob/master/DEVELOPER_GUIDE.md)

### Contribution Process

1. **Find or create an issue first** - Every PR should link to at least one issue
2. **Run validation before submitting**: `make validate` (golangci-lint)
3. **Include test plan** in PR description
4. **Sign off commits** with `Signed-off-by` line (use `git commit -s`)

### Commit Message Format

```
<type>: <subject>

<body>

Signed-off-by: Your Name <your.email@example.com>
```

Types: `feat`, `fix`, `docs`, `chore`, `refactor`, `test`

### PR Requirements

- Target `master` or `main` branch (check repo conventions)
- Link to related issues
- Provide clear description of changes
- Include test coverage for new features

### Harvester Architecture Reference

The main Harvester project components (useful for understanding CRDs):
- **Custom Resources**: `pkg/apis/` - Harvester Kubernetes API definitions
- **Controllers**: `pkg/controller/master/` - implemented with rancher/wrangler
- **Webhooks**: `pkg/webhook/resources/` - validation and mutation rules
- **API Server**: `pkg/api` and `pkg/server` - built with rancher/apiserver

### Support Channels

- GitHub Issues: https://github.com/harvester/harvester/issues
- Slack: `#harvester` channel on Rancher Users Slack
