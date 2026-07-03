# Error Handling Guidelines

Reconciliation error taxonomy and status update patterns.

## ReconcileError types

Use typed errors from `pkg/controller/common/errors.go`:

| Type | Constructor | When to use |
|------|-------------|-------------|
| Irrecoverable | `NewIrrecoverableError` | RBAC denied, invalid spec, unrecoverable config |
| Retry required | `NewRetryRequiredError` | Transient API errors, operand not ready yet |
| Multiple instance | `NewMultipleInstanceError` | More than one singleton CR detected |

Check error kind with `IsIrrecoverableError`, `IsRetryRequiredError`, `IsMultipleInstanceError`.

## Kubernetes API errors

- Use `FromClientError` to classify client errors:
  - **Irrecoverable:** Unauthorized, Forbidden, Invalid, BadRequest, ServiceUnavailable
  - **Retry:** NotFound, Conflict, Timeout, other transient errors
- Use `FromError` when wrapping an existing `ReconcileError` or unknown error.

## Status condition mapping

- Route all controller-runtime reconcilers through `HandleReconcileResult` (`pkg/controller/common/reconcile_result.go`):
  - **Success:** `Degraded=False`, `Ready=True`
  - **Retry:** `Degraded=False`, `Ready=False` (ReasonInProgress), `RequeueAfter`
  - **Irrecoverable:** `Degraded=True`, `Ready=False`, no requeue
- Set both `Degraded` and `Ready` atomically before calling `updateConditionFn`.

## Error wrapping

- `wrapcheck` and `errorlint` are enabled in `.golangci.yaml`.
- Wrap errors with context: `fmt.Errorf("reconciling deployment %s: %w", name, err)`.
- Implement `Unwrap()` on custom error types (see `ReconcileError.Unwrap`).

## Nil safety

- All `New*Error` constructors return `nil` when input `err` is `nil` — safe to chain.

## Library-go controllers

- Core cert-manager controller uses library-go patterns; map errors to operator conditions via existing status helpers rather than inventing parallel error types.

## Testing errors

- Unit tests for error classification live in `pkg/controller/common/errors_test.go`.
- When adding new error paths, add table-driven tests covering both condition updates and requeue behavior.

## Logging

- Use `log.V(2).Info` for condition update details in `HandleReconcileResult`.
- Log error messages, not Secret contents or credential values.
