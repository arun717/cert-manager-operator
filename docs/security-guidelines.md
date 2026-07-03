# Security Guidelines

Repo-specific security rules for the OpenShift cert-manager operator.

## RBAC

- Declare RBAC with `+kubebuilder:rbac` markers on controller files (e.g. `pkg/controller/certmanager/certmanager_controller.go`). Run `make manifests` to regenerate `config/rbac/role.yaml`.
- Prefer least privilege: add verbs only for resources the controller actually reconciles.
- Operand controllers (IstioCSR, TrustManager) use controller-runtime RBAC markers on `controller.go`; core cert-manager uses library-go factory controllers with the same marker pattern.

## Managed resource labeling

- Operand-owned resources use label key `app` (`common.ManagedResourceLabelKey`) with controller-specific values.
- Controllers filter watches via cache label selectors in `pkg/operator/setup_manager.go` to limit RBAC blast radius.
- ConfigMaps are an exception: unfiltered cache with predicate filtering (documented in setup_manager.go).

## Network policies

- `DefaultNetworkPolicy` on `CertManager` CR is immutable once set to `"true"` (CEL `XValidation` in `api/operator/v1alpha1/certmanager_types.go`).
- Default policies are deny-all egress; users must supply egress rules via `NetworkPolicies` field.
- Network policy YAML lives in `bindata/networkpolicies/`; reconciliation in `pkg/controller/certmanager/cert_manager_networkpolicy.go`.

## TLS profiles

- Cluster APIServer TLS profile is read from `config.openshift.io/APIServer` and propagated to cert-manager operand args via `pkg/tlsprofile/`.
- Do not hardcode cipher suites or min TLS version — use the TLS profile hook in `pkg/controller/common/tls_profile_hook.go`.

## Cloud credentials

- AWS/GCP ambient credentials use Cloud Credential Operator via `CredentialsRequest` (`pkg/controller/certmanager/credentials_request.go`).
- Cloud credential secret name is passed at runtime via `CloudCredentialSecret` flag (see `pkg/operator/starter.go`).
- See `docs/cloud_credentials.md` for operational setup.

## Trusted CA / proxy

- Trusted CA ConfigMap name is a runtime arg (`TrustedCAConfigMapName` in `pkg/operator/starter.go`).
- Prefer patching OLM Subscription env vars over editing Deployments directly (OLM reverts direct edits).
- See `docs/proxy.md` for proxy and CA injection patterns.

## Secrets and logging

- Never log Secret `.data`, private keys, or cloud credential contents in controllers or tests.
- Safe to log: resource names, namespaces, condition statuses, certificate subjects (not PEM bodies in CI logs).

## Webhooks

- TrustManager validating webhook logic is in `pkg/controller/trustmanager/webhooks.go`.
- Webhook tests use envtest/fake clients in `webhooks_test.go`.

## Linting and scanning

- `gosec` is enabled in `.golangci.yaml` — address findings or document justified exceptions.
- Run `make govulncheck` before release; vulnerability scan is part of the verify toolchain.

## Feature gates

- TechPreview operands (TrustManager, IstioCSR) are gated via `pkg/features/features.go` and `--unsupported-addon-features` runtime flag.
- Do not bypass feature gates in production code paths.
