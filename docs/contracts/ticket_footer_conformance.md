# Ticket Footer Conformance (v1)

Stable footer format:

`WRKR job_id=<job_id> manifest=sha256:<64-hex> verify="wrkr verify <job_id>"`

Contract rules:
- Prefix is `WRKR` (uppercase).
- `job_id` uses the same identifier emitted by submit/export.
- `manifest` is the `manifest_sha256` value from `manifest.json`.
- `verify` command must be copy-paste runnable for local verification.
- Format is additive-only within `v1.x`.

Reference parser/generator:
- `core/pack/footer.go`
- `cmd/wrkr/receipt.go`
