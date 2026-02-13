# Wrkr Serve API Contract (v1)

`wrkr serve` exposes a local HTTP API using Wrkr schemas and error envelopes.

## Safety Defaults

- Default bind: `127.0.0.1:9488`
- No non-loopback bind unless:
  - `--allow-non-loopback`
  - `--auth-token <token>`
  - `--max-body-bytes <n>`
- Request bodies are bounded by `max_body_bytes`.
- Path traversal (`..`) in request path fields is rejected.

## Endpoints

- `POST /v1/jobs:submit`
  - Body: `{ "jobspec_path": "...", "job_id": "..." }`
- `GET /v1/jobs/{job_id}:status`
- `GET /v1/jobs/{job_id}/checkpoints`
- `GET /v1/jobs/{job_id}/checkpoints/{checkpoint_id}`
- `POST /v1/jobs/{job_id}:approve`
  - Body: `{ "checkpoint_id": "...", "reason": "...", "approved_by": "..." }`
- `POST /v1/jobs/{job_id}:export`
  - Body: `{ "out_dir": "..." }` (optional)
- `POST /v1/jobs/{job_id}:verify`
  - Body: `{ "out_dir": "..." }` (optional)
- `POST /v1/jobs/{job_id}:accept`
  - Body: `{ "config_path": "..." }`
- `POST /v1/jobs/{job_id}:report-github`
  - Body: `{ "out_dir": "..." }` (optional)

## Error Contract

Errors return `wrkr.error_envelope` with stable `code` and `exit_code`.
Schema: `schemas/v1/serve/error_envelope.schema.json`

OpenAPI baseline: `schemas/v1/serve/api.openapi.json`
