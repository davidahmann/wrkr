# wrkr Python SDK

Thin wrapper around the `wrkr` CLI for scripting and integration use cases.

## Included helpers

- `run_wrkr(args)` raw subprocess invocation
- `run_wrkr_json(args)` parse `--json` output
- `status(job_id)` convenience status call
- `wrap(command, job_id=...)` wrap-mode invocation
- `accept_run(job_id, config=..., ci=...)` acceptance harness invocation
