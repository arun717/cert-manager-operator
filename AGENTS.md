# Cert Manager Operator — Agentic documentation

**Component**: Cert Manager Operator (OpenShift)  
**Repository**: [openshift/cert-manager-operator](https://github.com/openshift/cert-manager-operator)  
**Documentation tier**: 2 (component-specific)

> **Agent instruction**: When working in this repository, read **`README.md`** (install, upgrade, local run), **`docs/`** (guidelines and operational docs below), and the sections in this file. For **generic OpenShift operator patterns**, testing guidance, or security practices, use the **[Tier 1 hub](https://github.com/openshift/enhancements/tree/master/ai-docs)** under [openshift/enhancements](https://github.com/openshift/enhancements).

> **Generic platform patterns**: [openshift/enhancements `ai-docs/`](https://github.com/openshift/enhancements/tree/master/ai-docs)

---

## Why this file?

`README.md` stays focused on **human** quick starts. **`AGENTS.md`** holds **agent-oriented** detail: Make targets, test tags, controller map, conventions, and PR hygiene—so tools and contributors have one predictable entry point.

---

## What is cert-manager-operator?

An **OpenShift operator** that installs and reconciles **upstream [cert-manager](https://github.com/cert-manager/cert-manager)** (controller, webhook, cainjector) and **optional** operands (Istio CSR, trust-manager), using OpenShift `operator.openshift.io` APIs and **`library-go`** patterns. This repo **does not** implement ACME or certificate issuance logic—that behavior is **upstream cert-manager**.

- **Module:** `github.com/openshift/cert-manager-operator`
- **Go workspace:** root + `test/` + `tools/` (`go.work`)
- **Frameworks:** openshift/library-go (core), controller-runtime v0.23 (operand controllers)

---

## Core components

- **Operator process**: `library-go` `controllercmd` entry → `pkg/operator/starter.go` wires informers, static/sync controllers, and **ClusterOperator-style** status via `pkg/operator/operatorclient/`.
- **Cert-manager operand**: Deployed into **`cert-manager`**; manifests and CRDs live under **`bindata/`** (regenerated from Makefile / `hack/`).
- **Addon controllers** (feature-gated): **Istio CSR** (`pkg/controller/istiocsr/`), **trust-manager** (`pkg/controller/trustmanager/`).
- **Supporting packages**: `pkg/tlsprofile/` (APIServer TLS profile → cert-manager args), `pkg/features/` (feature gate discovery).
- **OLM / install artifacts**: `config/`, `bundle/`, `deploy/` — Kustomize and bundle generation (`make bundle`, `make deploy`).

---

## Repository layout

```text
README.md                      # Human quick start, install, upgrade cert-manager version
AGENTS.md                      # This file — agent onboarding and conventions
docs/
├── *-guidelines.md            # Agent-oriented deep guidelines (see index below)
├── proxy.md                   # Proxy-related behavior (when present)
├── cloud_credentials.md       # Ambient credentials / cloud secret wiring (when present)
└── operand_metrics.md         # Metrics and monitoring for the operand (when present)
api/operator/v1alpha1/         # CertManager, IstioCSR, TrustManager API + feature gates
pkg/
├── cmd/operator/              # CLI entry (start, flags)
├── operator/                  # Starter, setup_manager, generated clients, operatorclient
├── controller/
│   ├── certmanager/           # Core operand: deployments, network policy, credentials, overrides
│   ├── istiocsr/              # Istio CSR addon
│   ├── trustmanager/          # trust-manager addon
│   └── common/                # Shared errors, validation, TLS hooks, HandleReconcileResult
├── tlsprofile/                # APIServer TLS profile → cert-manager args
└── features/                  # Feature gate discovery
bindata/                       # Embedded operand YAML (do not hand-edit long-term)
config/                        # Kustomize, CRDs, RBAC, OLM manifests
hack/                          # update-manifests, test-apis, CI helpers
test/
├── apis/                      # API / envtest suites
├── e2e/                       # Ginkgo e2e (build tag: e2e); testdata/
└── library/                   # Shared E2E helpers
```

---

## Tier 1 links (ecosystem)

| Topic | Location |
|-------|----------|
| Operator practices | [ai-docs/practices/operator-patterns.md](https://github.com/openshift/enhancements/blob/master/ai-docs/practices/operator-patterns.md) |
| Testing practices | [ai-docs/practices/testing.md](https://github.com/openshift/enhancements/blob/master/ai-docs/practices/testing.md) |
| Security practices | [ai-docs/practices/security.md](https://github.com/openshift/enhancements/blob/master/ai-docs/practices/security.md) |

---

## Quick navigation

| Topic | Location | Description |
|-------|----------|-------------|
| **CertManager API** | `api/operator/v1alpha1/certmanager_types.go` | Singleton `cluster`; deployment overrides, network policies, `OperatorSpec` inline |
| **IstioCSR API** | `api/operator/v1alpha1/istiocsr_types.go` | Addon CR for Istio + cert-manager |
| **TrustManager API** | `api/operator/v1alpha1/trustmanager_types.go` | Addon CR for trust-manager |
| **Feature gates** | `api/operator/v1alpha1/features.go` | `IstioCSR`, `TrustManager` + enhancement links |
| **Operator startup** | `pkg/operator/starter.go`, `pkg/operator/setup_manager.go` | Controller registration, informers |
| **Status / spec apply** | `pkg/operator/operatorclient/` | `TargetNamespace` = `cert-manager`; singleton `cluster` |
| **Core reconciliation** | `pkg/controller/certmanager/` | Controller, webhook, cainjector, network policy, credentials |
| **Operand versions** | `Makefile` | `CERT_MANAGER_VERSION`, `ISTIO_CSR_VERSION`, `TRUST_MANAGER_VERSION` |
| **E2E harness** | `test/e2e/suite_test.go` | Namespaces, clients, build tag `e2e` |

---

## Management state (`CertManager` / OpenShift `OperatorSpec`)

`CertManager` embeds **`github.com/openshift/api/operator/v1`.OperatorSpec** (`managementState`, `unsupportedConfigOverrides`, etc.). Typical values:

| State | Behavior (high level) |
|-------|------------------------|
| **Managed** | Operator owns install and upgrades of the operand for this component. |
| **Unmanaged** | Operator does not reconcile the operand; user owns lifecycle. |
| **Removed** | Operator tears down managed resources (see OpenShift docs for semantics). |

Exact semantics follow OpenShift **operator API** conventions—when in doubt, cross-check [operator API](https://github.com/openshift/api) and Tier 1 operator docs above.

---

## Key controller packages

| Package / area | Purpose |
|----------------|---------|
| `pkg/controller/certmanager` | Main operand: **controller**, **webhook**, **cainjector** deployments, related images, network policies, **CredentialsRequest** integration, unsupported overrides validation |
| `pkg/controller/istiocsr` | **Istio CSR** deployment, RBAC, services, certificates (feature-gated) |
| `pkg/controller/trustmanager` | **trust-manager** install, webhooks, bundles (feature-gated) |
| `pkg/controller/common` | Shared **labels**, **operator namespace**, **trusted CA bundle** ConfigMap name/key; `HandleReconcileResult` for status conditions |
| `pkg/operator/starter.go` | Composes **kube** / **operator** informers, **config** informers, registers workload loops |

---

## Feature gates (addons)

| Feature | API / controller | Notes |
|---------|------------------|--------|
| **IstioCSR** | `IstioCSR` CR, `pkg/controller/istiocsr` | Defaults and release level in `api/operator/v1alpha1/features.go` |
| **TrustManager** | `TrustManager` CR, `pkg/controller/trustmanager` | Defaults and release level in `features.go` |

Always read **`features.go`** for current defaults and links to **OpenShift enhancements**.

---

## Knowledge graph

```text
CertManager (CR, name: cluster)
  ├─> pkg/operator/operatorclient (spec/status, TargetNamespace = cert-manager)
  ├─> pkg/controller/certmanager
  │     ├─> cert-manager-controller / webhook / cainjector deployments
  │     ├─> bindata manifests + RELATED_IMAGE_* / operand versions (Makefile)
  │     ├─> optional default network policies + egress overrides
  │     └─> cloud / platform integration (e.g. CredentialsRequest, trusted CA)
  └─> ClusterOperator-style conditions (via library-go + operator API)

IstioCSR (CR) ──> pkg/controller/istiocsr (feature gate)
TrustManager (CR) ──> pkg/controller/trustmanager (feature gate)

Upstream cert-manager
  └─> Certificate / Issuer / ACME behavior (not implemented in this repo)
```

---

## Cross-cutting conventions

### Naming

- Operand CR name is always `cluster` (CEL-enforced).
- Managed operand resources use label key `app` (`common.ManagedResourceLabelKey`).
- Package imports: prefer full paths; no dot imports except Ginkgo/Gomega in tests.

### Code generation

After changing API types or RBAC markers:

```bash
make manifests generate verify
```

Never hand-edit generated files under `pkg/operator/clientset/`, `config/crd/bases/`, or deepcopy output.

Operand version bumps:

```bash
make update-manifests   # after changing CERT_MANAGER_VERSION etc. in Makefile
```

### Controller patterns

- controller-runtime reconcilers: use `common.HandleReconcileResult` for status conditions.
- library-go controllers: follow existing cert-manager controller patterns in `pkg/controller/certmanager/`.
- Add RBAC markers before implementing new API calls; regenerate with `make manifests`.

### OLM safety

- Patch Subscription for operator env changes — do not edit Deployments directly.
- Operand namespace: `cert-manager`; operator namespace: `cert-manager-operator`.

### Linting

- golangci-lint v2 (`.golangci.yaml`): gosec, wrapcheck, errorlint, contextcheck, musttag enabled.
- Run `make lint` and `make verify` before proposing changes.

### Vendor

- Dependencies are vendored. After module changes: `make update-vendor`.
- Build tools from vendor via Makefile `bin/` targets.
- Do not hand-edit `vendor/` or long-term `bindata/`; use `make update` / `make update-vendor` per `Makefile` and `hack/`.

---

## Documentation index

### Agent guidelines (in-repo)

| Guideline | Scope |
|-----------|-------|
| [docs/security-guidelines.md](docs/security-guidelines.md) | RBAC, network policies, TLS, credentials, secrets |
| [docs/performance-guidelines.md](docs/performance-guidelines.md) | Workers, cache filtering, requeue, E2E parallelism |
| [docs/error-handling-guidelines.md](docs/error-handling-guidelines.md) | ReconcileError taxonomy, condition mapping |
| [docs/api-contracts-guidelines.md](docs/api-contracts-guidelines.md) | CRDs, CEL validation, codegen |
| [docs/testing-guidelines.md](docs/testing-guidelines.md) | Unit, envtest, E2E patterns and labels |
| [docs/integration-guidelines.md](docs/integration-guidelines.md) | OLM, OpenShift APIs, operand manifests |

### Operational docs

| Doc | Topic |
|-----|-------|
| [docs/proxy.md](docs/proxy.md) | Egress proxy and trusted CA |
| [docs/cloud_credentials.md](docs/cloud_credentials.md) | CCO ambient credentials |
| [docs/operand_metrics.md](docs/operand_metrics.md) | Prometheus metrics for operands |

---

## Dev environment tips

- **Work from the repository root** (directory containing `Makefile` and `go.mod`). `PROJECT_ROOT` is derived from `git rev-parse --show-toplevel` or `pwd`.
- **Go version**: Match **`go.mod`** (`go` directive).
- **Shell / Make**: `Makefile` uses `bash` with `-euo pipefail`; **`make help`** lists targets.
- **Cluster**: **`oc`** expected for `make deploy`, e2e waits, and debugging; see **`README.md`** for `make local-run` (scale in-cluster operator to 0 first).
- **After API edits**: **`make update`** or at least **`make manifests generate`** so CRDs and `pkg/operator` generated code stay consistent.
- **Caches**: `XDG_CACHE_HOME` / `XDG_CONFIG_HOME` default under **`_output/`** when unset (CI-friendly).

Local operator run:

```bash
make local-run       # against current kubeconfig
make deploy          # kustomize deploy
make undeploy        # remove
```

---

## Testing instructions

- **CI**: Prefer matching **Prow** / **openshift/release** jobs locally with **`make verify`**, **`make lint`**, **`make test`** before merge.
- **Pre-merge loop**:

  ```sh
  make verify
  make lint
  make test
  ```

- **`make test`**: `manifests`, `generate`, `vet`, **`test-apis`** (`hack/test-apis.sh`, envtest + Ginkgo), **`test-unit`**.
- **`make test-unit`**: Excludes `test/e2e`, `test/apis`, `test/utils` (see `Makefile`).
- **E2E** (`test/e2e/`, tag **`e2e`**): cluster must already run the operator and stable operands; **`make test-e2e`** (uses **`make test-e2e-wait-for-stable-state`**). Tech Preview CI uses **`make test-e2e-tech-preview`** (`E2E_GINKGO_LABEL_FILTER_TECH_PREVIEW` in the Makefile). Narrow with **`TEST=...`** (`go test -run` regex) and **`E2E_GINKGO_LABEL_FILTER`** (`-ginkgo.label-filter`). Use the Make target—plain **`go test ./...`** does not cover e2e.
- **After refactors**: **`make verify`** and **`make lint`** (`.golangci.yaml`).
- **Add or update tests** for code you change (unit, `test/apis`, or e2e as appropriate). See [docs/testing-guidelines.md](docs/testing-guidelines.md) for patterns.

Vulnerability scan before release-related changes:

```bash
make govulncheck
```

---

## PR instructions

- **Title**: Clear, descriptive; follow repo / org template (**Jira** `OCPBUGS-…` or team key if required).
- **Before PR**:

  ```sh
  make verify
  make lint
  make test
  ```

- **API / bindata / manifests**: Commit outputs from **`make update`** (or minimal `make manifests generate`) so verify passes.
- **Scope**: Small diffs; follow existing **`library-go`** patterns.
- **API changes** include regenerated CRDs and clients.
- **New reconcile paths** include unit tests; user-facing behavior includes E2E when feasible.
- **RBAC changes** reflected in markers and `config/rbac/role.yaml`.
- **User-visible behavior**: Update **`README.md`** or **`docs/`** when needed.
- **No secrets** or credential values in logs or test output.
- Do not commit `_output/`, `bin/` tool binaries, or coverage files.

---

## Common pitfalls

- Assuming direct Deployment control on OLM-managed operators — use Subscription patches.
- Adding cluster-wide watches without label filtering — follow `setup_manager.go` patterns.
- Skipping `make generate` after API changes — CI `make verify` will fail.
- Creating operand CRs with names other than `cluster` — rejected by CEL.
- Enabling `DefaultNetworkPolicy` without egress rules — operands get deny-all egress.

---

## Quick triage

| Symptom | Likely area |
|--------|-------------|
| Certificate / issuer not Ready | Operand + cluster config; **`Makefile`** / **`bindata`** cert-manager version |
| Operator Degraded / Progressing | `pkg/operator/`, `pkg/controller/certmanager/`, `pkg/operator/operatorclient/` |
| Istio / mesh | `pkg/controller/istiocsr/` + **`features.go`** |
| Trust bundles | `pkg/controller/trustmanager/` + **`features.go`** |
| Codegen / bindata failures | **`make update`**, **`hack/`** |

---

## Ecosystem references

- **Upstream cert-manager**: [cert-manager/cert-manager](https://github.com/cert-manager/cert-manager) — CRDs, controllers, issuance behavior.
- **OpenShift enhancements** (Istio CSR / trust-manager designs): linked from `api/operator/v1alpha1/features.go`.
- **OpenShift release / CI**: jobs often live in **[openshift/release](https://github.com/openshift/release)** (Prow); this repo may not ship `.github/workflows`.
