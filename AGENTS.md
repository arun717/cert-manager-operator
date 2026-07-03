# Cert Manager Operator — Agentic documentation

**Component**: Cert Manager Operator (OpenShift)  
**Repository**: [openshift/cert-manager-operator](https://github.com/openshift/cert-manager-operator)  
**Documentation tier**: 2 (component-specific)

> **Agent instruction**: When working in this repository, read **`README.md`** (install, upgrade, local run), **`docs/`** (proxy, metrics, cloud credentials), and the sections below. For **generic OpenShift operator patterns**, testing guidance, or security practices, use the **[Tier 1 hub](https://github.com/openshift/enhancements/tree/master/ai-docs)** under [openshift/enhancements](https://github.com/openshift/enhancements).

> **Generic platform patterns**: [openshift/enhancements `ai-docs/`](https://github.com/openshift/enhancements/tree/master/ai-docs)

---

## Why this file?

`README.md` stays focused on **human** quick starts. **`AGENTS.md`** holds **agent-oriented** detail: Make targets, test tags, controller map, and PR hygiene—so tools and contributors have one predictable entry point.

---

## What is cert-manager-operator?

An **OpenShift operator** that installs and reconciles **upstream [cert-manager](https://github.com/cert-manager/cert-manager)** (controller, webhook, cainjector) and **optional** operands (Istio CSR, trust-manager), using OpenShift `operator.openshift.io` APIs and **`library-go`** patterns. This repo **does not** implement ACME or certificate issuance logic—that behavior is **upstream cert-manager**.

---

## Core components

- **Operator process**: `library-go` `controllercmd` entry → `pkg/operator/starter.go` wires informers, static/sync controllers, and **ClusterOperator-style** status via `pkg/operator/operatorclient/`.
- **Cert-manager operand**: Deployed into **`cert-manager`**; manifests and CRDs live under **`bindata/`** (regenerated from Makefile / `hack/`).
- **Addon controllers** (feature-gated): **Istio CSR** (`pkg/controller/istiocsr/`), **trust-manager** (`pkg/controller/trustmanager/`).
- **OLM / install artifacts**: `config/`, `bundle/`, `deploy/` — Kustomize and bundle generation (`make bundle`, `make deploy`).

---

## Documentation structure

This repository does **not** use a separate `ai-docs/` tree; context is distributed as follows:

```text
README.md                 # Human quick start, install, upgrade cert-manager version
docs/
├── operand_metrics.md    # Metrics and monitoring for the operand
├── proxy.md              # Proxy-related behavior
└── cloud_credentials.md  # Ambient credentials / cloud secret wiring
api/operator/v1alpha1/    # CertManager, IstioCSR, TrustManager API + feature gates
pkg/
├── cmd/operator/         # CLI entry (start, flags)
├── operator/             # Starter, setup_manager, generated clients, OperatorClient
└── controller/
    ├── certmanager/      # Core operand: deployments, network policy, credentials, overrides
    ├── istiocsr/         # Istio CSR addon
    ├── trustmanager/     # trust-manager addon
    └── common/           # Shared constants (namespaces, trusted CA bundle names)
bindata/                  # Generated CRDs and deployment YAML (do not hand-edit long-term)
hack/                     # update-manifests, test-apis, CI helpers
test/
├── apis/                 # API / envtest suites
├── e2e/                  # Ginkgo e2e (build tag: e2e); testdata/, optional plans/
└── library/              # Shared test helpers
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
| **QE / plan notes** | `test/e2e/plans/` | Optional structured test plans (when present) |

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
| `pkg/controller/common` | Shared **labels**, **operator namespace**, **trusted CA bundle** ConfigMap name/key |
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

## Ecosystem references

- **Upstream cert-manager**: [cert-manager/cert-manager](https://github.com/cert-manager/cert-manager) — CRDs, controllers, issuance behavior.
- **OpenShift enhancements** (Istio CSR / trust-manager designs): linked from `api/operator/v1alpha1/features.go`.
- **OpenShift release / CI**: jobs often live in **[openshift/release](https://github.com/openshift/release)** (Prow); this repo may not ship `.github/workflows`.

---

## Dev environment tips

- **Work from the repository root** (directory containing `Makefile` and `go.mod`). `PROJECT_ROOT` is derived from `git rev-parse --show-toplevel` or `pwd`.
- **Go version**: Match **`go.mod`** (`go` directive).
- **Shell / Make**: `Makefile` uses `bash` with `-euo pipefail`; **`make help`** lists targets.
- **Cluster**: **`oc`** expected for `make deploy`, e2e waits, and debugging; see **`README.md`** for `make local-run` (scale in-cluster operator to 0 first).
- **After API edits**: **`make update`** or at least **`make manifests generate`** so CRDs and `pkg/operator` generated code stay consistent.
- **Do not hand-edit** **`vendor/`** or long-term **`bindata/`**; use **`make update`** / **`make update-vendor`** per `Makefile` and `hack/`.
- **Caches**: `XDG_CACHE_HOME` / `XDG_CONFIG_HOME` default under **`_output/`** when unset (CI-friendly).

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
- **Add or update tests** for code you change (unit, `test/apis`, or e2e as appropriate).

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
- **User-visible behavior**: Update **`README.md`** or **`docs/`** when needed.

---

## Quick triage

| Symptom | Likely area |
|--------|-------------|
| Certificate / issuer not Ready | Operand + cluster config; **`Makefile`** / **`bindata`** cert-manager version |
| Operator Degraded / Progressing | `pkg/operator/`, `pkg/controller/certmanager/`, `pkg/operator/operatorclient/` |
| Istio / mesh | `pkg/controller/istiocsr/` + **`features.go`** |
| Trust bundles | `pkg/controller/trustmanager/` + **`features.go`** |
| Codegen / bindata failures | **`make update`**, **`hack/`** |
