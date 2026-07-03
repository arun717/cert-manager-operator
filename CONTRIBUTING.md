# Contributing to Cert Manager Operator for Red Hat OpenShift

Thank you for your interest in contributing! This guide covers the PR process, review expectations, coding conventions, and testing requirements.

## Getting Started

1. Fork the repository and clone your fork.
2. Create a feature branch from `master`.
3. Set up your environment:
   - Go version matching `go.mod` (currently Go 1.26)
   - `oc` CLI with access to an OpenShift cluster (required for `make deploy`, E2E tests, and local-run workflows)
   - `kubectl` is useful for debugging but most Make targets and docs assume `oc`

## Development Workflow

### Build and Verify

```sh
make build            # Full build: codegen + fmt + vet + compile
make build-operator   # Fast rebuild (binary only, no codegen)
```

After **any** code change, regenerate and verify before committing:

```sh
make update && make verify
```

This regenerates deepcopy/clientgen code, upstream operand manifests, and bindata, then runs verification checks (bindata freshness, deepcopy, clientgen, bundle, deps, fmt, vet). CI will reject PRs with stale generated files.

To regenerate CRDs and RBAC after API changes, `make test` also runs `manifests` and `generate`; commit those outputs along with `make update` results.

### Linting

```sh
make lint             # Run golangci-lint with all configured linters
make lint-fix         # Auto-fix linting issues
```

### Testing

Run the full local test suite (no cluster required):

```sh
make test             # manifests + generate + vet + API + unit tests
```

Individual test targets:

| Target | Description |
|--------|-------------|
| `make test-unit` | Unit tests (excludes `test/e2e`, `test/apis`, `test/library`) |
| `make test-apis` | API validation tests via envtest + Ginkgo |
| `make test-e2e` | End-to-end tests against a live OpenShift cluster |

For E2E tests, the operator and stable operands must already be running on the cluster. `make test-e2e` waits for cert-manager deployments to become Available before running tests.

Narrow E2E execution:

```sh
# Ginkgo label filter (default excludes Service Mesh and non-Mint credential modes)
make test-e2e E2E_GINKGO_LABEL_FILTER="Platform:Generic"

# Go test -run regex
make test-e2e TEST="Overrides"
```

Common Ginkgo labels in this repo: `Platform:Generic`, `Platform:AWS`, `Platform:GCP`, `Platform:Azure`, `Feature:IstioCSR`, `CredentialsMode:Mint`, `CredentialsMode:Manual`.

See `README.md` for cluster setup (`make deploy`, `make local-run`) and `AGENTS.md` for the full E2E harness description.

### Pre-Submission Checklist

Before opening a PR, always run:

```sh
make verify
make lint
make test
```

If you changed APIs, bindata sources, or codegen inputs, also run `make update` and commit the generated outputs before the steps above.

## Coding Conventions

### Import Order

Imports must follow the order enforced by `gci` in [`.golangci.yaml`](.golangci.yaml):

1. Standard library
2. Third-party packages
3. `github.com/openshift/cert-manager-operator` (project-local)
4. Blank imports, dot imports, aliases, local module

### File Headers

All `.go` files must include the Apache 2.0 license header from [`hack/boilerplate.go.txt`](hack/boilerplate.go.txt).

### Naming Conventions

- **Go packages**: Controller packages use lowercase names (`certmanager`, `istiocsr`, `trustmanager`, `common`).
- **Controller names**: kebab-case in log/event strings (e.g. `cert-manager-istio-csr-controller`, `cert-manager-controller-deployment`).
- **Finalizers** (addon controllers): `<crd-plural>.<api-group>/<controller-name>` — e.g. `istiocsr.openshift.operator.io/cert-manager-istio-csr-controller`.
- **Managed resource label**: `app=<component>` via `common.ManagedResourceLabelKey`.
- **Bindata YAML**: Upstream operand manifests live under `bindata/cert-manager-deployment/`, `bindata/istio-csr/`, `bindata/trust-manager/resources/`, and `bindata/networkpolicies/`. trust-manager resources follow `<kind-lowercase>_<resource-name>.yml`; core cert-manager files use descriptive kebab-case names (e.g. `cert-manager-deployment.yaml`).

### Linter Rules

The repo uses golangci-lint v2 (see [`.golangci.yaml`](.golangci.yaml)). Key rules to be aware of:

- **golines** enforces a max line length of 200 characters.
- **gofmt** rewrites `interface{}` to `any`.
- **wrapcheck** is configured with project-specific ignore patterns for Kubernetes client calls and controller-runtime builders.
- **depguard** is currently disabled due to golangci-lint v2 configuration issues — still avoid introducing `github.com/pkg/errors` or `github.com/sirupsen/logrus`; use standard `errors`/`fmt` and `klog`/`logr` instead.

### Error Handling

**Addon controllers** (Istio CSR, trust-manager):

- Wrap API call errors with `common.FromClientError`.
- Use `common.NewIrrecoverableError` for configuration validation failures.
- Route results through `common.HandleReconcileResult` to set `Ready`/`Degraded` conditions.
- Never return both `RequeueAfter` and a non-nil error from `Reconcile`.

**Core cert-manager controllers** (library-go):

- Use deployment hook validators in `deployment_overrides_validation.go` for user-facing override errors.
- Management state and operand sync are handled by `staticresourcecontroller` and `deploymentcontroller` from `openshift/library-go` — follow existing hook patterns rather than adding a new controller-runtime reconciler for the core operand.

### Generated Files

Never edit these files by hand — always use `make update` (and `make manifests` when CRD/RBAC markers change):

- `api/**/zz_generated.deepcopy.go`
- `config/crd/bases/*.yaml`
- `config/rbac/role.yaml`
- `pkg/operator/assets/bindata.go`
- `pkg/operator/clientset/`, `informers/`, `applyconfigurations/`
- `bundle/` (regenerate with `make bundle` when OLM metadata changes)

## Testing Requirements

### Unit Tests

- Add unit tests for new reconciliation logic using table-driven tests.
- Addon controller tests use counterfeiter-generated fakes from `pkg/controller/common/fakes/`.
- Regenerate fakes after changing the `CtrlClient` interface:

  ```sh
  make generate-fakes
  # or: go generate ./pkg/controller/common/...
  ```

- Core cert-manager deployment override and validation logic has extensive unit tests under `pkg/controller/certmanager/` — extend those when changing hooks or allowed override values.

### API Tests

- Add test cases in `api/operator/v1alpha1/tests/<crd-domain>/` for any new CRD field or CEL validation rule.
- Suites use `.testsuite.yaml` files and run via envtest + Ginkgo (`make test-apis`).

### E2E Tests

- Add E2E test cases with appropriate Ginkgo labels (`Platform:Generic` for portable tests; platform-specific labels for cloud credential or Service Mesh scenarios).
- E2E tests require a live OpenShift cluster with the operator deployed and operands in a stable state.
- Use the `e2e` build tag — run via `make test-e2e`, not plain `go test ./...`.
- Tech Preview addon tests (e.g. trust-manager) may require enabling the feature gate on the operator (`--unsupported-addon-features=TrustManager=true`) and an appropriate `E2E_GINKGO_LABEL_FILTER`.

## Pull Request Process

### Creating a PR

1. Keep diffs small and focused. One logical change per PR.
2. Write a clear, descriptive PR title.
3. Reference the relevant Jira ticket in your commit messages (e.g. `OCPBUGS-12345: description` or your team's key if required).
4. Describe your changes and the motivation behind them in the PR body.

### What to Include

- **Code changes** with corresponding tests (unit, API, or E2E as appropriate).
- **Generated file updates** — run `make update` (and `make manifests` / `make bundle` when applicable) and commit the results. CI verifies generated artifacts via `make verify`.
- **Documentation updates** — update `README.md` or files under `docs/` when behavior changes are user-visible.

### Review Process

- PRs are reviewed and approved by maintainers listed in the [OWNERS](OWNERS) file.
- All CI checks must pass before merge.
- Reviewers will check for adherence to the architectural patterns documented in [ARCHITECTURE.md](ARCHITECTURE.md) and [AGENTS.md](AGENTS.md), including:
  - Core operand changes use library-go static/deployment controllers and bindata — not ad-hoc controller-runtime reconcilers.
  - Addon changes use controller-runtime with `common.CtrlClient` and the unified manager in `pkg/operator/setup_manager.go`.
  - Proper use of deployment override validation for user-facing `CertManager` spec fields.
  - Feature gates respected for Istio CSR and trust-manager.

### CI Expectations

CI runs checks equivalent to the local pre-merge loop:

- `make verify` — bindata, deepcopy, clientgen, bundle, dependency verification, fmt, vet.
- `make lint` — full golangci-lint suite.
- `make test` — unit and API integration tests.

Ensure all of these pass locally before pushing.

## Go Workspace

The repo uses `go.work` with three modules: `.`, `./test`, `./tools`. Key implications:

- Do **not** use `-mod=vendor` for `go test`, `go fmt`, or E2E targets — the Makefile clears `GOFLAGS` for those targets to avoid conflicting with `go.work`.
- To add a dependency: `make update-dep PKG=pkg@version`, then `make update-vendor`.
- All build-time tools are vendored and built from source into `bin/`. Do not install tools globally.

## Container Builds

The default container engine is `podman`. Override with:

```sh
CONTAINER_ENGINE=docker make image-build
```

See `README.md` for image registry, bundle, and catalog build/push workflows (`make image-build`, `make bundle`, `make catalog-build`).

## Additional Resources

- [ARCHITECTURE.md](ARCHITECTURE.md) — High-level architecture, dual-runtime model, reconciliation flows, and invariants
- [AGENTS.md](AGENTS.md) — Make targets, controller map, test tags, and PR hygiene
- [README.md](README.md) — Install, upgrade, local run, and cluster deployment
- [docs/operand_metrics.md](docs/operand_metrics.md) — Operand metrics and monitoring
- [docs/proxy.md](docs/proxy.md) — Proxy-related behavior
- [docs/cloud_credentials.md](docs/cloud_credentials.md) — Ambient credentials / cloud secret wiring
- [OpenShift Tier 1 ai-docs](https://github.com/openshift/enhancements/tree/master/ai-docs) — Generic operator, testing, and security practices

## License

By contributing, you agree that your contributions will be licensed under the [Apache License 2.0](LICENSE).
