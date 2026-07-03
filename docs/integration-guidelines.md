# Integration Guidelines

OpenShift platform integration, OLM deployment, and operand lifecycle.

## OLM deployment context

| Property | Value |
|----------|-------|
| Operator namespace | `cert-manager-operator` |
| Operand namespace | `cert-manager` |
| Operator deployment | `cert-manager-operator-controller-manager` |
| CSV pattern | Bundle version from `BUNDLE_VERSION` (Makefile) |

- Prefer patching Subscription for operator env/config — OLM reverts direct Deployment edits.
- See `docs/proxy.md` for subscription patch examples.

## Dual runtime architecture

1. **library-go path** (`pkg/operator/starter.go`): cert-manager core reconciliation, status controller, config observers.
2. **controller-runtime path** (`pkg/operator/setup_manager.go`): IstioCSR and TrustManager when feature gates allow.

When adding platform integrations, place them in the controller path that owns the affected operand.

## OpenShift API dependencies

| API | Package | Usage |
|-----|---------|-------|
| FeatureGate | `pkg/features/` | TechPreview operand enablement |
| APIServer | `pkg/tlsprofile/` | TLS security profile |
| Infrastructure | cert-manager controller | Platform detection |
| Cloud Credential Operator | `credentials_request.go` | DNS-01 ambient creds |

## Operand manifest pipeline

- Embedded manifests: `bindata/` (cert-manager, trust-manager, istio-csr, network policies).
- Update via scripts: `hack/update-cert-manager-manifests.sh`, `hack/update-trust-manager-manifests.sh`, `hack/update-istio-csr-manifests.sh`.
- Run `make update-manifests` after bumping operand versions (`CERT_MANAGER_VERSION`, etc. in Makefile).

## Upstream forks

- cert-manager: `github.com/openshift/jetstack-cert-manager` (version in Makefile `CERT_MANAGER_VERSION`).
- trust-manager and istio-csr versions pinned in Makefile operand version section.

## Feature gates

- Runtime flag: `--unsupported-addon-features` → `UnsupportedAddonFeatures` in starter.go.
- Discovery: `pkg/features/features.go` reads cluster FeatureGate status.
- E2E labels `TechPreview` / `TechPreview:Inverted` align with gate behavior.

## Metrics

- Operand metrics setup documented in `docs/operand_metrics.md`.
- Operator metrics via library-go status/metrics patterns.

## Bundle and catalog

```bash
make bundle          # Generate OLM bundle
make bundle-build    # Build bundle image
make catalog-build   # Build catalog/index image
```

- Bundle inputs: `config/manifests/`, CRDs, RBAC from `make manifests`.

## Local development

```bash
make local-run       # Run operator locally against current kubeconfig
make deploy          # Kustomize deploy to cluster
make undeploy        # Remove deployment
```

## Jsonnet

- Jsonnet configs in `jsonnet/` for manifest templating.
- Tool built from vendor: `$(BIN_DIR)/jsonnet`.

## CI

- OpenShift CI uses `.ci-operator.yaml` and `images/ci/` Dockerfiles.
- E2E detects CI via `OPENSHIFT_CI=true` in `test/e2e/suite_test.go` for path resolution.

## Service Mesh (Istio CSR)

- Service Mesh E2E uses testdata under `test/e2e/testdata/servicemesh/`.
- Version pins: `E2E_OSM_ISTIO_VERSION`, `E2E_OSM_OPERATOR_VERSION` in Makefile — keep in sync with `servicemesh_helpers_test.go` constants.
