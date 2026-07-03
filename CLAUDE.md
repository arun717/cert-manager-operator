@AGENTS.md

## Build and test commands

Run from the repository root:

```bash
make verify          # fmt, vet, script checks, dep verification
make test            # generate, vet, test-apis, test-unit
make lint            # golangci-lint
make build           # generate, fmt, vet, build binary
```

Before proposing API or RBAC changes:

```bash
make manifests generate verify
```

E2E (requires live OpenShift cluster):

```bash
make test-e2e
# Filter: E2E_GINKGO_LABEL_FILTER='Platform:Generic && Feature:IstioCSR'
```

## Codegen workflow

When modifying `api/operator/v1alpha1/` types or `+kubebuilder:rbac` markers:

1. `make manifests` — regenerate CRDs and RBAC
2. `make generate` — deepcopy, client-gen, fakes
3. `make verify` — confirm no drift

Operand version bumps:

```bash
make update-manifests   # after changing CERT_MANAGER_VERSION etc. in Makefile
```

## Pre-commit expectations

- Run `make verify` and `make test` before suggesting a PR is ready.
- Run `make lint` when changing Go code; use `make lint-fix` for auto-fixable issues.
- Do not commit `_output/`, `bin/` tool binaries, or coverage files.

## Local operator run

```bash
make local-run       # against current kubeconfig
make deploy          # kustomize deploy
make undeploy        # remove
```

## Vulnerability scan

```bash
make govulncheck
```

Run before release-related changes.
