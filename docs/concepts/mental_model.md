# Mental Model

Wrkr is dispatch and supervision infrastructure for long-running agent jobs.

## Core Primitive Chain

1. `JobSpec` defines objective, adapter, budgets, and checkpoints policy.
2. Runner executes durable steps and emits typed checkpoints.
3. Approvals and budget stops gate forward progress deterministically.
4. Jobpack bundles evidence for offline verification and review.
5. Acceptance harness provides deterministic pass/fail signals.

## What Wrkr Is Not

- Not an agent planning framework.
- Not a model host.
- Not enterprise secret management.

## Operator View

- Dispatch work with `wrkr submit` or `wrkr wrap`.
- Supervise by checkpoint summaries, not token streams.
- Approve only at decision-needed checkpoints.
- Accept with deterministic checks before merge/deploy.
