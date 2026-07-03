# Architecture

This document describes the high-level architecture of the Cert Manager Operator for Red Hat OpenShift.
If you want to familiarize yourself with the codebase, you are in the right place!

For detailed guidelines on specific areas, see the files listed in `AGENTS.md`.
For install, upgrade, and local development, see `README.md`.

## Bird's Eye View

On the highest level, this is a Kubernetes operator that installs and manages the upstream [cert-manager](https://github.com/cert-manager/cert-manager) application on OpenShift clusters. The upstream project provides the actual certificate-issuance logic (Certificate, Issuer, ACME, etc.). This operator does **not** embed or fork that code. Instead, it manages upstream resources as static YAML manifests compiled into the binary (`bindata/`), decoded at runtime, mutated with operator-controlled configuration, and applied to the cluster.

Optional **addon operands** — **Istio CSR** and **trust-manager** — follow the same bindata pattern but are reconciled by separate controller-runtime reconcilers behind a unified manager.

```text
                  ┌─────────────────────────────────────────────┐
                  │         CertManager CR (cluster)             │
                  │   singleton "cluster", user-facing API       │
                  │   embeds OpenShift OperatorSpec            │
                  └──────────────────┬──────────────────────────┘
                                     │
         ┌───────────────────────────┼───────────────────────────┐
         │                           │                           │
┌────────▼─────────┐     ┌───────────▼──────────┐    ┌──────────▼──────────┐
│ library-go       │     │ DefaultCertManager   │    │ controller-runtime  │
│ controllers (8)  │     │ Controller           │    │ manager (optional)  │
│                  │     │                      │    │                     │
│ staticresource + │     │ Auto-creates         │    │ IstioCSR reconciler │
│ deployment       │     │ CertManager/cluster  │    │ (feature-gated)     │
│ controllers for  │     │ with ManagementState │    │                     │
│ controller,      │     │ = Managed if missing │    │ TrustManager        │
│ webhook,         │     │                      │    │ reconciler          │
│ cainjector,      │     │                      │    │ (feature-gated)     │
│ network policies │     │                      │    │                     │
└────────┬─────────┘     └──────────────────────┘    └──────────┬──────────┘
         │                                                        │
         ▼                                                        ▼
┌────────────────────┐                              ┌─────────────────────────┐
│ Upstream operand   │                              │ Addon operands          │
│ (controller,       │                              │ IstioCSR (per-namespace │
│  webhook,          │                              │  "default" CR)          │
│  cainjector in     │                              │ trust-manager           │
│  "cert-manager"    │                              │ (cluster "cluster" CR)  │
│  namespace)        │                              │                         │
└────────────────────┘                              └─────────────────────────┘
```

Three operator CRs drive the system:

- **CertManager** (name: `cluster`, cluster-scoped): the primary user-facing CR. Controls operand installation, deployment overrides (args, env, resources, scheduling), default and user-defined network policies, and OpenShift `OperatorSpec` fields (`managementState`, `logLevel`, `unsupportedConfigOverrides`).
- **IstioCSR** (name: `default`, namespaced): addon CR for the Istio CSR agent. One instance per namespace; enables Istio workloads to obtain certificates via cert-manager.
- **TrustManager** (name: `cluster`, cluster-scoped): addon CR for trust-manager. Manages trust bundles and CA package distribution.

The operator process runs in the **`cert-manager-operator`** namespace. Core and addon operands deploy into **`cert-manager`** (and trust-manager may reference a user-specified trust namespace).

**Important:** this operator uses **two reconciliation frameworks** in one process — `openshift/library-go` factory controllers for the core cert-manager operand, and `controller-runtime` for feature-gated addons. Do not assume every reconciler follows the same pattern.

## Code Map

This section describes important directories and data structures.
Pay attention to the **Architecture Invariant** sections.

### `api/operator/v1alpha1/`

CRD type definitions. This is where `CertManager`, `IstioCSR`, and `TrustManager` structs live, along with shared types (`DeploymentConfig`, `ConditionalStatus`, `Mode`), condition constants, feature gate declarations, and deepcopy code.

Validation is primarily **CEL-based** via `+kubebuilder:validation:XValidation` markers on the types. Runtime validation also occurs in deployment hooks (core operand) and controller-local `validate*Config()` functions (addons). There are no admission webhooks for the operator's own CRDs.

`api/operator/v1alpha1/tests/` contains declarative YAML test suites (`.testsuite.yaml` files) exercised by the API integration test generator.

**Architecture Invariant:** `TrustManager` and `IstioCSR` singleton names are CEL-enforced (`cluster` and `default` respectively). `CertManager` is cluster-scoped and conventionally named `cluster`; the `DefaultCertManagerController` auto-creates it if missing.

**Architecture Invariant:** several fields are immutable once set, enforced by CEL — for example `defaultNetworkPolicy` cannot revert from `"true"` to `"false"`, `IstioCSR.spec.issuerRef` is immutable, and `TrustManager.spec.trustNamespace` is immutable once set. This prevents configuration drift that would leave the cluster in an inconsistent state.

### `bindata/`

Static YAML manifests for operands. These are the Deployments, Services, RBAC, NetworkPolicies, Certificates, and WebhookConfigurations that the operator creates in the cluster.

| Subdirectory | Contents |
|--------------|----------|
| `cert-manager-deployment/` | Core cert-manager controller, webhook, and cainjector manifests |
| `istio-csr/` | Istio CSR operand manifests |
| `trust-manager/resources/` | trust-manager operand manifests |
| `networkpolicies/` | Default deny-all and allow policies for cert-manager and Istio CSR |

These files are compiled into Go via go-bindata and live as `pkg/operator/assets/bindata.go` at build time. Sources are refreshed from upstream releases by `hack/update-*-manifests.sh`.

**Architecture Invariant:** bindata manifests are templates, not final resources. Controllers decode them, mutate them (namespace, labels, annotations, images, env vars, scheduling, TLS profile), and then create or update them. Never deploy bindata YAML directly to a cluster.

**Architecture Invariant:** do not hand-edit `pkg/operator/assets/bindata.go`. Run `make update` (or `make update-bindata`) to regenerate it from the YAML sources.

### `main.go` and `pkg/cmd/operator/`

The operator binary entrypoint. Uses `openshift/library-go` `controllercmd` — not a standalone controller-runtime `main`:

```text
main.go → pkg/cmd/operator/cmd.go → controllercmd.NewControllerCommandConfig(...).NewCommandWithContext()
                                    → pkg/operator/starter.go (RunOperator)
```

Runtime flags:

- `--trusted-ca-configmap` — trusted CA bundle for operand containers
- `--cloud-credentials-secret` — ambient cloud credentials secret name
- `--unsupported-addon-features` — e.g. `IstioCSR=true`, `TrustManager=true`

### `pkg/operator/`

Manager setup, informer wiring, and controller registration.

| File / package | Role |
|----------------|------|
| `starter.go` | Process entry: clients, informers, library-go controller goroutines, feature gate setup, optional controller-runtime manager |
| `setup_manager.go` | Unified controller-runtime manager for addon controllers; label-filtered cache |
| `operatorclient/` | `OperatorClient` implementing `v1helpers.OperatorClient` for `CertManager/cluster` |
| `assets/bindata.go` | Generated embed of `bindata/` |
| `clientset/`, `informers/`, `applyconfigurations/` | Codegen from `hack/update-clientgen.sh` |

**`OperatorClient` responsibilities:**

- `GetOperatorState()` — reads singleton `cluster` from informer lister
- `ApplyOperatorStatus()` — status updates with OpenShift condition canonicalization
- `EnsureFinalizer` / `RemoveFinalizer`
- `GetUnsupportedConfigOverrides()` — parses break-glass JSON from `OperatorSpec`

Constants in `operatorclient/interfaces.go`:

- `TargetNamespace` = `cert-manager`
- `OperatorNamespace` = `cert-manager-operator`

### `pkg/controller/`

The heart of the operator. Three controller packages plus shared infrastructure.

#### `pkg/controller/certmanager/`

The main reconciliation path for the core cert-manager operand. Uses **library-go** `staticresourcecontroller` and `deploymentcontroller` — not a single controller-runtime `Reconcile` loop.

**`CertManagerControllerSet`** (`cert_manager_controller_set.go`) starts **eight controllers concurrently**:

```text
1. controller static resources (RBAC, SA, namespace, etc.)
2. controller deployment
3. webhook static resources
4. webhook deployment
5. cainjector static resources
6. cainjector deployment
7. network policy static resources (conditional on defaultNetworkPolicy="true")
8. network policy user-defined (from CertManager.spec.networkPolicies)
```

Each static resource controller decodes bindata YAML and applies it via `resourceapply`. Each deployment controller reads a bindata Deployment template and runs a chain of **deployment hooks** (`generic_deployment_controller.go`):

- Operand image override from `RELATED_IMAGE_*` env vars
- Log level from `OperatorSpec`
- Pod label, container arg, env, replica, resource, and scheduling overrides from `CertManager.spec`
- Proxy env injection (`operator-lib/proxy`)
- Trusted CA ConfigMap mount
- Service-account-bound token configuration
- Cloud credential secret mounting (when Infrastructure CR indicates AWS/GCP)
- Cluster TLS profile from APIServer CR (`tls_profile_hook.go`)

Override values are validated at apply time by hooks in `deployment_overrides_validation.go` (allowed args, env keys, resource limits, scheduling fields).

**Watches (library-go pattern):**

- `operatorClient.Informer()` — `CertManager/cluster`
- Target namespace informers: Deployments, ConfigMaps, Secrets
- Optional: `Infrastructure`, `APIServer` informers (cloud credentials, TLS profile)

**Drift correction:** Built into library-go `staticresourcecontroller` and `deploymentcontroller` via `resourceapply` / `resourcemerge`. Manual edits to managed ClusterRoles, Deployments, or NetworkPolicies are reverted on the next sync.

**Note:** `certmanager_controller.go` is a **placeholder** kubebuilder RBAC stub with an empty `Reconcile()`. It exists for RBAC codegen only and is not the real reconciler.

#### `pkg/controller/istiocsr/`

Controller-runtime reconciler for the Istio CSR addon (feature-gated).

**Primary watch:** `IstioCSR` (generation changes)

**Secondary watches** (enqueue the namespaced `default` IstioCSR):

- Managed resources (`app=cert-manager-istio-csr`): Deployment, Certificate, RBAC, Service, ServiceAccount, NetworkPolicy
- Watched (not managed): ConfigMaps with `istiocsr.openshift.operator.io/watched-by` label, Secret metadata, Issuer, ClusterIssuer

**Reconcile order** (`install_istiocsr.go`):

```text
validate config → network policies → services → service accounts → RBAC
→ certificates → deployments → processed annotation
```

**Singleton enforcement:** `disallowMultipleIstioCSRInstances()` — only one `IstioCSR` per namespace is supported.

#### `pkg/controller/trustmanager/`

Controller-runtime reconciler for the trust-manager addon (feature-gated, Tech Preview by default).

**Primary watch:** `TrustManager/cluster` (generation changes)

**Secondary watches** (all enqueue singleton `cluster`):

- Managed resources (`app=cert-manager-trust-manager`): Deployment, SA, Service, ConfigMap, RBAC, Certificate, ValidatingWebhookConfiguration
- Special: CNO-injected `cert-manager-operator-trusted-ca-bundle` ConfigMap in the operator namespace (no managed label)

**Reconcile order** (`install_trustmanager.go`):

```text
validate config → validate trust namespace exists → default CA package ConfigMap
→ service accounts → RBAC → services → issuer → certificate → deployment
→ validating webhook → status observed state
```

#### `pkg/controller/common/`

Shared utilities used by addon controllers (and some core hooks):

| File | Purpose |
|------|---------|
| `constants.go` | `ManagedResourceLabelKey=app`, operator namespace, trusted CA ConfigMap name/key |
| `client.go` | `CtrlClient` wrapper over `mgr.GetClient()` with `UpdateWithRetry` |
| `errors.go` | `ReconcileError` with `IrrecoverableError`, `RetryRequiredError`, `MultipleInstanceError` |
| `reconcile_result.go` | `HandleReconcileResult()` — Ready/Degraded condition updates for addon CRs |
| `validation.go` | Kubernetes-native validation helpers for scheduling and metadata |
| `tls_profile_hook.go` | Cluster TLS profile from APIServer CR |
| `container_args.go` | Container arg merge utilities |

**Architecture Invariant:** addon resource updates go through `UpdateWithRetry`, which wraps `retry.RetryOnConflict` with a read-modify-write pattern.

### `config/`

Kustomize manifests for CRDs, RBAC, manager deployment, samples, and OLM bundle inputs.

| Path | Purpose |
|------|---------|
| `config/crd/bases/` | Generated CRDs (operator APIs + upstream cert-manager CRDs) |
| `config/rbac/` | Operator RBAC (`role.yaml` is generated) |
| `config/manager/` | Operator Deployment; trusted CA ConfigMap with `config.openshift.io/inject-trusted-cabundle` |
| `config/manifests/` | OLM CSV base for bundle generation |
| `config/samples/` | Sample CRs |
| `config/console/` | Console quick starts / YAML samples |

CRDs in `config/crd/bases/` and RBAC in `config/rbac/role.yaml` are generated — do not hand-edit them.

### `bundle/`

OLM bundle artifacts generated by `make bundle`:

- `bundle/manifests/cert-manager-operator.clusterserviceversion.yaml` — OLM CSV
- Operand CRDs, upstream cert-manager CRDs, trust-manager CRDs, RBAC, metrics Service, trusted CA ConfigMap
- `bundle/metadata/annotations.yaml`

### `test/`

A separate Go module (`./test` in `go.work`) containing:

- `test/apis/` — API integration tests using Ginkgo + envtest. A generator auto-creates Ginkgo specs from the YAML test suites in `api/operator/v1alpha1/tests/`.
- `test/e2e/` — End-to-end tests using Ginkgo against a live cluster (build tag `e2e`). Covers deployment overrides, issuers (ACME, Vault, self-signed), Istio CSR, trust-manager, TLS profile, and observability.
- `test/library/` — Shared test helpers (kube client, cert utilities, dynamic resources, Istio CSR helpers).

**Run:** `make test-e2e` (waits for stable operand state first). Tech Preview CI uses `make test-e2e-tech-preview`. Narrow with `TEST=...` and `E2E_GINKGO_LABEL_FILTER`.

### `hack/`

Shell scripts for codegen, verification, and CI. Notable scripts:

| Script | Purpose |
|--------|---------|
| `update-cert-manager-manifests.sh` | Download upstream cert-manager release YAML → jsonnet patch → `bindata/` + CRDs |
| `update-istio-csr-manifests.sh` | Istio CSR operand manifests |
| `update-trust-manager-manifests.sh` | trust-manager operand manifests |
| `update-clientgen.sh` | Generate clientset, informers, apply configurations |
| `test-apis.sh` | Run API integration suite |
| `verify-*.sh` | deepcopy, clientgen, bundle, CRDs, deps, types |
| `e2e-coverage.sh` | E2E coverage image setup and collection |
| `local-run-config.yaml` | Config for `make local-run` |

### `tools/`

A separate Go module for build-time tool dependencies (controller-gen, golangci-lint, ginkgo, etc.). Tools are vendored and built from source.

## Cross-Cutting Concerns

### Dual Runtime Model

The operator process runs two concurrent reconciliation models:

| Path | Framework | CRs | Client pattern |
|------|-----------|-----|----------------|
| Core cert-manager | library-go `factory.Controller` | `CertManager/cluster` | Kubernetes client + informer listers |
| Addon operands | controller-runtime `manager` | `IstioCSR/default`, `TrustManager/cluster` | Cached `mgr.GetClient()` via `common.CtrlClient` |

Both share the same process, logging (`klog` via `ctrl.SetLogger`), and startup context. Addon controllers start only when their feature gates are enabled.

**Architecture Invariant:** do not add addon reconciliation logic to the library-go controller set, or core operand logic to the controller-runtime manager, without understanding this split.

### Management State (`Managed` / `Unmanaged` / `Removed`)

`CertManager` embeds `github.com/openshift/api/operator/v1.OperatorSpec`. Management state is enforced by library-go `staticresourcecontroller` and `deploymentcontroller`, which read `operatorClient.GetOperatorState()` on each sync.

| State | Behavior (high level) |
|-------|------------------------|
| **Managed** | Operator owns install and upgrades of the core operand. |
| **Unmanaged** | Operator does not reconcile the operand; user owns lifecycle. |
| **Removed** | Operator tears down managed resources. |

The `DefaultCertManagerController` auto-creates `CertManager/cluster` with `ManagementState: Managed` if the CR is missing.

IstioCSR and TrustManager CRs do **not** embed `OperatorSpec`; they are feature-gated addons with their own lifecycle and `Ready`/`Degraded` conditions.

### Feature Gates

Declared in `api/operator/v1alpha1/features.go` and wired in `pkg/features/`:

| Feature | Default | Release level |
|---------|---------|---------------|
| `IstioCSR` | `true` | GA |
| `TrustManager` | `false` | Tech Preview |

Enabled via `--unsupported-addon-features` flag (e.g. `TrustManager=true`). At startup, `setupFeatureGates()` also reads `featuregates/cluster` when available. If neither IstioCSR nor TrustManager is enabled, the controller-runtime manager is not started.

Always read `features.go` for current defaults and links to OpenShift enhancements.

### Drift Detection

| Area | Mechanism |
|------|-----------|
| Core operand | library-go `resourceapply` + `resourcemerge` in static/deployment controllers |
| Addon operands | Label/annotation drift checks, spec drift on Deployments and Certificates, secondary watches on managed resources with label predicates |
| User edits to managed resources | Re-enqueue via `Watches` + generation or label/annotation change predicates |

**Architecture Invariant:** if someone manually modifies a managed ClusterRole, Deployment, or NetworkPolicy, the operator will detect the change and revert it. This is a critical security property for the core operand path.

### Error Classification (Addon Controllers)

Reconciliation errors in IstioCSR and TrustManager flow through `common.ReconcileError` and `HandleReconcileResult()`:

| Reason | Requeue? | Status |
|--------|----------|--------|
| `IrrecoverableError` | No | Degraded=True, Ready=False |
| `RetryRequiredError` | Yes (30s) | Ready=False |
| `MultipleInstanceError` | No | Event recorded; reconcile skipped |

`FromClientError()` auto-classifies Kubernetes API errors: `Unauthorized`, `Forbidden`, `Invalid`, and `BadRequest` become irrecoverable; most others become retry-required.

Core library-go controllers use the event recorder and deployment hook errors directly; status flows through `CertManager.status` (OpenShift `OperatorStatus`: Available, Degraded, Progressing, Upgradeable).

### Cache Strategy (Addon Manager)

`setup_manager.go` configures a label-filtered controller-runtime cache for managed resources per addon controller. **ConfigMaps are intentionally excluded** from the label filter because:

1. TrustManager watches both managed ConfigMaps and the CNO-injected `cert-manager-operator-trusted-ca-bundle` ConfigMap (no managed label).
2. IstioCSR watches managed ConfigMaps and user-created ConfigMaps with `istiocsr.openshift.operator.io/watched-by` (a different label key).

ConfigMaps therefore use the default unfiltered informer; each controller applies predicate-level filtering.

**Issuer** and **ClusterIssuer** are also unfiltered — IstioCSR reconciles user-created Issuers referenced from the spec.

### TLS, Proxy, and Cloud Integration

- **TLS profile:** Cluster minimum TLS version and cipher suites from the APIServer CR are applied to operand deployments via `tls_profile_hook.go`.
- **Proxy:** HTTP/HTTPS/NO_PROXY env vars injected via `operator-lib/proxy` based on cluster proxy configuration.
- **Cloud credentials:** On AWS/GCP platforms, the operator mounts a cloud credential secret (from `--cloud-credentials-secret`) into the cert-manager controller for ambient issuer authentication.
- **Trusted CA:** Operand containers mount the trusted CA bundle from the ConfigMap named by `--trusted-ca-configmap` (injected by OLM/CNO in production).

### Code Generation

Several artifacts are generated and must be committed:

- `zz_generated.deepcopy.go` — from `make generate`
- `config/crd/bases/*.yaml` — from `make manifests`
- `pkg/operator/assets/bindata.go` — from `make update-bindata`
- `config/rbac/role.yaml` — from `make manifests`
- `pkg/operator/clientset/`, `informers/`, `applyconfigurations/` — from `hack/update-clientgen.sh`

`make update` runs the full pipeline (`generate`, `update-manifests`, `update-bindata`). `make verify` checks that generated files are fresh. CI will reject PRs with stale generated files.

**Architecture Invariant:** never edit generated files by hand. Always use `make update`.

### Network Policy Architecture

When `CertManager.spec.defaultNetworkPolicy` is `"true"`, the operator deploys a **deny-all** base NetworkPolicy, then layers specific allow-policies for API server egress, webhook ingress, metrics ingress, and DNS egress. User-defined policies from `CertManager.spec.networkPolicies` are reconciled separately (egress-focused). Istio CSR deploys its own network policies from `bindata/networkpolicies/`.

**Architecture Invariant:** `defaultNetworkPolicy` cannot be changed from `"true"` back to `"false"` once enabled (CEL-enforced).

### Operand Versions and Images

Upstream operand versions are controlled by Makefile variables and refreshed via `hack/update-*-manifests.sh`:

```
CERT_MANAGER_VERSION ?= v1.20.2
ISTIO_CSR_VERSION    ?= v0.16.0
TRUST_MANAGER_VERSION ?= v0.20.3
```

At runtime, operand container images are pinned by OLM through `RELATED_IMAGE_*` environment variables:

- `RELATED_IMAGE_CERT_MANAGER_CONTROLLER`
- `RELATED_IMAGE_CERT_MANAGER_WEBHOOK`
- `RELATED_IMAGE_CERT_MANAGER_CA_INJECTOR`
- `RELATED_IMAGE_CERT_MANAGER_ACMESOLVER`
- `RELATED_IMAGE_CERT_MANAGER_ISTIOCSR`
- `RELATED_IMAGE_CERT_MANAGER_TRUST_MANAGER`

The operator has no compile-time dependency on upstream cert-manager Go code.

### Go Workspace

The repo uses `go.work` with three modules: `.`, `./test`, `./tools`. This means:

- `GOFLAGS` is cleared for test and fmt targets to avoid `-mod=vendor` conflicting with `go.work`.
- Vendoring is workspace-level (`go work vendor`).
- All build-time tools are vendored and built from source.

### Relationship to Upstream

This operator manages — but does not contain — the upstream [cert-manager](https://github.com/cert-manager/cert-manager) project. The upstream project defines the CRDs that end users interact with (`Certificate`, `Issuer`, `ClusterIssuer`, `CertificateRequest`, etc.) and the controllers that perform issuance. This operator's job is to deploy, configure, and lifecycle-manage those upstream components on OpenShift, providing an opinionated, security-hardened, and OLM-integrated installation.

Similarly, [Istio CSR](https://github.com/cert-manager/istio-csr) and [trust-manager](https://github.com/cert-manager/trust-manager) are separate upstream projects whose manifests are vendored into `bindata/` and managed by addon controllers.

## Startup Sequence

```text
1. controllercmd starts → RunOperator(ctx)
2. Build kube, operator, and apiextensions clients
3. Start informers (operator CRs, kube namespaces "", kube-system, cert-manager)
4. Optionally start Infrastructure informer (cloud credentials)
5. Start 8 library-go cert-manager controllers + DefaultCertManagerController (concurrent goroutines)
6. setupFeatureGates() — parse --unsupported-addon-features, read featuregates/cluster
7. If IstioCSR and/or TrustManager enabled → NewControllerManager() → manager.Start() in goroutine
8. Block on <-ctx.Done()
```

## Differences from a Typical Controller-Runtime Operator

| Typical controller-runtime | This operator |
|-----------------------------|---------------|
| Single `Manager` + one reconciler per CR | Split: library-go factory controllers + separate ctrl manager for addons |
| Reconcile loop reads CR, applies manifests | Core: bindata + static/deployment controllers with hooks; addons: imperative per-resource reconcilers |
| Uniform client pattern | Informers/listers (core) vs cached ctrl client (addons) |
| Feature gates in manager setup | library-go startup + optional second manager |
| OpenShift `OperatorSpec` / management state | Yes — core path follows OpenShift operator conventions |
| Operand versions in code | Makefile + `hack/update-*-manifests.sh` from upstream releases |

## Quick Triage

| Symptom | Likely area |
|---------|-------------|
| Certificate / issuer not Ready | Operand + cluster config; `Makefile` / `bindata` cert-manager version |
| Operator Degraded / Progressing | `pkg/operator/`, `pkg/controller/certmanager/`, `pkg/operator/operatorclient/` |
| Istio / mesh | `pkg/controller/istiocsr/` + `features.go` |
| Trust bundles | `pkg/controller/trustmanager/` + `features.go` |
| Codegen / bindata failures | `make update`, `hack/` |
| Override rejected | `deployment_overrides_validation.go`, `CertManager.spec` fields |
| Addon not starting | `--unsupported-addon-features`, feature gate defaults in `features.go` |
