# Testing Guidelines

Unit, API integration, and E2E test patterns.

## Test tiers

| Tier | Location | Command | Build tag |
|------|----------|---------|-----------|
| Unit | `pkg/**/*_test.go` | `make test-unit` | none |
| API integration | `test/apis/` | `make test-apis` | none (separate module) |
| E2E | `test/e2e/` | `make test-e2e` | `//go:build e2e` |

Full CI path: `make test` (manifests, generate, vet, test-apis, test-unit).

## Go workspace

- Three modules in `go.work`: root, `test/`, `tools/`.
- E2E and API tests import the operator module; run tests from repo root via Makefile targets.

## Unit tests

- Place tests alongside code in `pkg/`.
- Use table-driven tests for validation and error classification.
- Counterfeiter fakes: run `make generate-fakes` when adding interfaces under `pkg/controller/`.
- Exclude e2e/apis from unit run: `-skip '(/apis|/e2e|/utils)'` in Makefile `test-unit` target.

## API integration (envtest)

- Suite bootstraps envtest with CRDs from `config/crd/bases/` (`test/apis/suite_test.go`).
- Uses Ginkgo + Gomega + komega matchers.
- Requires envtest assets: `make test-apis` downloads via `setup-envtest` (`ENVTEST_K8S_VERSION` default `1.32.0`).
- See `test/apis/README.md` for openshift/api test pattern reference.

## E2E tests

- Entry: `test/e2e/suite_test.go` with Ginkgo bootstrap.
- Shared helpers: `test/library/` (`kube_client.go`, `dynamic_resources.go`, `cert_utils.go`, `istiocsr.go`).
- Namespaces: operator `cert-manager-operator`, operand `cert-manager`.
- Operator deployment: `cert-manager-operator-controller-manager`.

### Ginkgo labels

Filter tests with `E2E_GINKGO_LABEL_FILTER` (Makefile default excludes ServiceMesh):

| Label | Purpose |
|-------|---------|
| `Platform:Generic` | Runs on any supported platform |
| `Platform:AWS` | AWS-specific (DNS-01 credentials) |
| `Feature:TrustManager` | TrustManager operand |
| `Feature:IstioCSR` | Istio CSR operand |
| `Feature:IstioCSR-ServiceMesh` | OpenShift Service Mesh integration |
| `Feature:TLSProfile` | Cluster TLS profile propagation |
| `TechPreview` | Requires TechPreview feature gate |
| `TechPreview:Inverted` | Negative gate tests |
| `CredentialsMode:Mint` | CCO mint mode |

- Individual tests use traceability labels like `ISTIOCSR-001`.
- Use `Ordered` contexts when tests depend on prior setup within a Describe block.

### E2E conventions

- Use `Eventually`/`Consistently` with explicit timeouts.
- Register cleanup via Ginkgo DeferCleanup or explicit teardown in Ordered suites.
- Testdata YAML lives under `test/e2e/testdata/`.
- Default timeout: `E2E_TIMEOUT` (2h in Makefile).

## Coverage

- Unit coverage: `make test-unit` with `-coverprofile cover.out`.
- E2E coverage image: `make image-build-coverage`, script `hack/e2e-coverage.sh`.

## Lint before commit

- `make lint` runs golangci-lint v2 (config in `.golangci.yaml`).
- `make verify` runs script checks, dep verification, fmt, vet.

## Adding new E2E tests

1. Check existing specs for reusable helpers in `test/library/`.
2. Add to an existing `Describe` with matching labels if behavior fits.
3. Use `//go:build e2e` at top of file.
4. Never log Secret data or private keys in test output.
