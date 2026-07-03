# cert-manager-operator

OpenShift cluster operator that installs and manages [cert-manager](https://cert-manager.io/), [trust-manager](https://cert-manager.io/docs/trust/trust-manager/), and [Istio CSR](https://github.com/cert-manager/istio-csr) as platform operands.

## Components

| Operand CR | Kind | Description |
|------------|------|-------------|
| `cluster` | CertManager | Core cert-manager deployment (controller, webhook, cainjector) |
| `cluster` | TrustManager | Bundle distribution for trust anchors (TechPreview) |
| `cluster` | IstioCSR | Certificate signing for Istio workloads (TechPreview) |

All CRs use API group `operator.openshift.io/v1alpha1` and must be named `cluster`.

## Tech stack

- Go 1.26 with vendored dependencies (`go.work` spans root, `test/`, `tools/`)
- openshift/library-go for core operator pattern
- controller-runtime for optional operand controllers
- kubebuilder / operator-sdk for CRD and OLM bundle generation
- Ginkgo v2 + envtest for testing

## Project structure

```
api/operator/v1alpha1/   Operator CRD types
pkg/controller/        Reconciliation logic per operand
pkg/operator/          Operator bootstrap and manager setup
bindata/               Embedded operand manifests
config/                Kustomize overlays, CRDs, RBAC
test/e2e/              Cluster end-to-end tests
test/apis/             CRD validation integration tests
docs/                  Operational docs and agent guidelines
```

See [AGENTS.md](AGENTS.md) for AI agent conventions and the full documentation index.

## Getting started

### Prerequisites

- Go 1.26+
- `make`, `git`
- For E2E: OpenShift cluster with cert-manager operator installed via OLM
- Container engine (podman or docker) for image builds

### Build

```bash
make build           # generate, fmt, vet, compile binary
make build-operator  # compile only (skip codegen checks)
```

### Test

```bash
make test            # unit + API integration
make test-unit       # pkg/ unit tests
make test-apis       # envtest CRD/CEL validation
make test-e2e        # cluster E2E (requires OpenShift)
```

### Verify and lint

```bash
make verify          # fmt, vet, script checks
make lint            # golangci-lint
make govulncheck     # vulnerability scan
```

### Run locally

```bash
make local-run       # run operator against current kubeconfig
```

### Code generation

After API or RBAC marker changes:

```bash
make manifests generate verify
```

## Deployment

The operator is deployed via OLM:

- Operator namespace: `cert-manager-operator`
- Operand namespace: `cert-manager`

```bash
make deploy          # kustomize deploy (development)
make bundle          # generate OLM bundle
make image-build     # build operator container image
```

## Configuration

| Makefile variable | Default | Purpose |
|-------------------|---------|---------|
| `CERT_MANAGER_VERSION` | v1.20.2 | cert-manager operand version |
| `TRUST_MANAGER_VERSION` | v0.20.3 | trust-manager operand version |
| `ISTIO_CSR_VERSION` | v0.16.0 | istio-csr operand version |
| `BUNDLE_VERSION` | 1.20.0 | OLM bundle semver |
| `E2E_GINKGO_LABEL_FILTER` | Platform/credential filter | E2E test selection |

## Documentation

### Agent guidelines

- [AGENTS.md](AGENTS.md) — cross-cutting conventions and docs index
- [docs/security-guidelines.md](docs/security-guidelines.md)
- [docs/testing-guidelines.md](docs/testing-guidelines.md)
- [docs/api-contracts-guidelines.md](docs/api-contracts-guidelines.md)
- [docs/error-handling-guidelines.md](docs/error-handling-guidelines.md)
- [docs/integration-guidelines.md](docs/integration-guidelines.md)
- [docs/performance-guidelines.md](docs/performance-guidelines.md)

### Operational

- [docs/proxy.md](docs/proxy.md) — egress proxy and trusted CA configuration
- [docs/cloud_credentials.md](docs/cloud_credentials.md) — Cloud Credential Operator integration
- [docs/operand_metrics.md](docs/operand_metrics.md) — Prometheus metrics setup

## License

Apache License 2.0 — see [LICENSE](LICENSE).
