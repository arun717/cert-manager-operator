# Performance Guidelines

Concurrency and resource-efficiency patterns for this operator.

## Dual controller architecture

- **Core cert-manager** uses openshift/library-go factory controllers started with a single worker: `go controller.Run(ctx, 1)` in `pkg/operator/starter.go`.
- **Optional operands** (IstioCSR, TrustManager) share one controller-runtime manager when feature gates enable them (`pkg/operator/setup_manager.go`).
- Do not add parallel workers to library-go controllers without evaluating library-go expectations.

## Informer resync

- Shared informer resync interval is `10 * time.Minute` (`resyncInterval` in `pkg/operator/starter.go`).
- Avoid lowering resync globally — it increases API server load across all watched resources.

## Cache label filtering

- Operand controllers configure controller-runtime cache with label selectors on `common.ManagedResourceLabelKey` (`app`) in `setup_manager.go`.
- When adding new watched types, prefer label-filtered caches over cluster-wide watches.
- ConfigMaps intentionally use unfiltered cache with event predicates — follow this pattern if adding similar exceptions.

## Requeue behavior

- Recoverable reconcile errors use `RequeueAfter` via `HandleReconcileResult` in `pkg/controller/common/reconcile_result.go`.
- Irrecoverable errors return without requeue — avoid tight retry loops on permanent failures.
- Pass explicit `requeueDuration` per controller; do not rely on controller-runtime default backoff for business-logic retries.

## E2E parallelism

- E2E tests run serially: Makefile uses `ginkgo -p 1` for `test-e2e`.
- Do not increase E2E parallelism without evaluating cluster resource contention and operand namespace isolation.

## Codegen and build

- Tools are built from vendor (`bin/` targets in Makefile) for reproducibility — avoid `go install` in CI recipes.
- `make generate` and `make manifests` can be expensive; run only when API or RBAC markers change.

## Bindata and manifests

- Operand manifests are embedded via bindata (`bindata/`). Large manifest changes affect binary size and startup decode time.
- Prefer Helm/Jsonnet update scripts (`hack/update-*-manifests.sh`) over hand-editing generated bindata YAML.
