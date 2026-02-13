# Wrkr Product Summary

Wrkr is a durable dispatch and supervision substrate for long-running agent jobs.

It provides four OSS primitives:

1. Dispatch: run jobs with durable state, resumability, and deterministic lifecycle controls.
2. Checkpoint: emit structured supervisory checkpoints with typed states and required actions.
3. Accept: execute deterministic acceptance checks for CI and operator review.
4. Jobpack: export a verifiable evidence bundle for offline verification and auditability.

Wrkr is vendor-neutral and offline-first for core workflows.
