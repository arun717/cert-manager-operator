# API Contracts Guidelines

CRD types, validation, and codegen conventions.

## API group and versions

- Group: `operator.openshift.io`
- Version: `v1alpha1`
- Package: `api/operator/v1alpha1/`
- Operand CRs: `CertManager`, `TrustManager`, `IstioCSR`

## Singleton CR naming

- Operand CRs must be named `cluster` (enforced by CEL validation on types).
- E2E tests assert rejection of non-`cluster` names (e.g. `test/e2e/istio_csr_operand_test.go`).

## Kubebuilder markers

- Use `+kubebuilder:object:root=true` on top-level types.
- Embed `github.com/openshift/api/operator/v1.OperatorSpec` and `OperatorStatus` for OpenShift operator consistency.
- Mark optional fields with `+optional` and `+kubebuilder:validation:Optional`.

## CEL validation

- Use `+kubebuilder:validation:XValidation` for immutability and cross-field rules (see `DefaultNetworkPolicy` in `certmanager_types.go`).
- API integration tests in `test/apis/` require Kubernetes ≥1.25 for CEL support — suite checks server version in `suite_test.go`.

## Generated artifacts

After API changes, run:

```bash
make manifests   # CRDs → config/crd/bases/
make generate    # deepcopy, client-gen, applyconfiguration
make verify      # ensure no drift
```

- CRD YAML: `config/crd/bases/operator.openshift.io_*.yaml`
- Generated clients: `pkg/operator/clientset/`, `pkg/operator/informers/`, `pkg/operator/applyconfigurations/`
- Client-gen script: `hack/update-clientgen.sh`

## DeploymentConfig pattern

- `DeploymentConfig` embeds `OverrideArgs`, `OverrideEnvs`, and resource overrides for operand Deployments.
- Validation logic for overrides is in `pkg/controller/common/` and tested in `deployment_overrides_validation_test.go`.

## Status conditions

- Custom conditions use types in `api/operator/v1alpha1/` (`Degraded`, `Ready`, reason constants).
- `ConditionalStatus` helpers set conditions idempotently before status patch.

## API test generation

- `test/apis/generator.go` loads test specs from `api/` tree via `LoadTestSuiteSpecs`.
- Add new validation cases alongside type definitions; generator picks them up automatically.

## Breaking changes

- Prefer CEL immutability rules over admission webhooks for field-level constraints.
- Document new fields in kubebuilder comments — they appear in CRD OpenAPI schema.

## OLM bundle

- Bundle version uses semver (`BUNDLE_VERSION` in Makefile, default `1.20.0`).
- Run `make bundle` after CRD or CSV changes.
