# CSM Flow Service — working notes

Go event consumer that replaces ServiceNow flow automation for WSO2 CSM, as a
**pragmatic hand-port** (see `docs/cutover-porting-plan.md`). ServiceNow is
being decommissioned; the live flows become Go handlers, not config.

## Architecture

```
entity-service ──emit case.* / entity.changed──▶ Event Hub (cs-events)
                                                     │
                     ┌───────────────────────────────┴──────────────┐
                     ▼                                               ▼
        csm-notification-service                        csm-flow-service (this)
        (unchanged — sends email/chat/twilio)           cmd/consumer: event → flow
                     ▲                                   internal/flows: one Go type/flow
                     └──────── notification request ─────┘
```

One running process: `cmd/consumer`. It joins the bus, and for each record the
`flows.Registry` runs every `Flow` whose `Match` returns true. A flow reacts by
patching a native entity via `internal/entity` and/or publishing a
notification-request event that `csm-notification-service` sends — this service
does not send email/chat/twilio itself.

## Hard rules

- **Do not edit `csm-notification-service` or `entity-service`.** `eventbus`,
  `events`, `entity`, `apierror`, `middleware` here are COPIES (separate Go
  modules). Keep them in sync by hand; the originals are canonical.
- **Double-fire guard.** Registering a flow (`internal/flows/register.go`) is a
  paired change with disabling its ServiceNow (and any hardcoded Go) counterpart
  in the SAME commit. Three possible senders — ServiceNow, the notification
  service's hardcoded dispatcher, a new flow — must never overlap for one event.
- **Flows must be idempotent.** A retry re-runs the whole record (see
  `Registry.Handle`); the per-(event, flow) dispatch log that would make retries
  exactly-once is Phase 2 (`docs/architecture.md` §9). Until then, prefer
  notify-only or naturally-idempotent actions.
- **Entity migration gates writes.** Cases and case comments are native in
  Postgres — flows may write them. `change_request` and `incident` are still
  ServiceNow-backed; a mutating action against them returns 401. Notify-only
  flows on any entity work today (`docs/architecture.md` §20).
- **No customer PII in logs.** Log entity ids, never email addresses or case
  text (`internal/entity` already omits upstream bodies for this reason).
- **Config is flat single-line strings** (Choreo constraint); event bus is
  `mustEnv`, entity-service and DLQ are optional and fail on first use.
- **Correlation prefix is `cfs-`.**

## Conventions inherited from the repo

Strict `Handler → Service/Flow → Client` layering, no DI framework, wiring
explicit in `cmd/consumer/main.go` · `internal/apierror` for upstream errors ·
pure Go, no cgo · migrations (when they arrive) applied by hand with a working
`.down.sql` · CI: `go vet`, `go test -race`, `gosec` at 0, `govulncheck`.

## Reuse, don't reinvent

- `internal/eval` — pure condition evaluator; a flow's `Match` may use it.
- `internal/porting/snquery` — translates a ServiceNow encoded query to a
  `spec.Condition`; a migration aid, deleted after cutover.
- `docs/flow-porting-specs.md` — every flow's real trigger/condition/actions.
- `docs/servicenow-discovery/` — the read-only scripts that produce the above.

## Ported flows

| Flow | Key | Registered? |
|---|---|---|
| CR Approval notifications (+ its two subflows) | `cr_approval_notice` | **No** — see below |

`cr_approval_notice` is written and tested but deliberately **not** in `All()`. Two things
have to land first, and a test (`TestCRApprovalNotice_NotRegistered`) fails if it is
registered before they do:

1. **The double-fire guard.** ServiceNow still sends these notifications. Registering is a
   paired change with disabling `CR Approval notifications` there, in the same commit.
2. **A consumer.** It publishes `change_request.approval_requested`, which
   `csm-notification-service` does not handle yet — this service is forbidden from sending
   email itself. That handler is a change in that repo, not this one.

It also needs `change_request.state` in Postgres, which
`wso2-enterprise/digiops-cs` PR #3058 adds, and an `entity.changed` publisher for
`change_request` — nothing emits one today.

## Adding a flow

Copy `internal/flows/example_template.go`, implement `Key`/`Match`/`Run` from
the flow's spec row, add table-driven tests with its real conditions, register
it, disable its SN counterpart in the same commit.
