# Blessed Lane Integration Kit

This is the canonical v1 adoption lane for Wrkr-compatible integration.

## Goal

Complete first real flow in under 15 minutes with deterministic evidence:

1. Dispatch job
2. Inspect checkpoints
3. Approve/resume
4. Export + verify jobpack
5. Run acceptance
6. Produce review summary

## Inputs

- JobSpec: `examples/integrations/blessed_jobspec.yaml`
- Acceptance config: `examples/integrations/blessed_accept.yaml`

## Command Flow

```bash
wrkr submit examples/integrations/blessed_jobspec.yaml --job-id job_blessed_lane
wrkr status job_blessed_lane --json
wrkr checkpoint list job_blessed_lane --json
wrkr bridge work-item job_blessed_lane --checkpoint <decision_checkpoint> --template github --out-dir ./wrkr-out
wrkr approve job_blessed_lane --checkpoint <decision_checkpoint> --reason "approved"
wrkr resume job_blessed_lane --json
wrkr export job_blessed_lane --out-dir ./wrkr-out --json
wrkr verify job_blessed_lane --out-dir ./wrkr-out --json
wrkr accept run job_blessed_lane --config examples/integrations/blessed_accept.yaml --ci --out-dir ./wrkr-out --json
wrkr report github job_blessed_lane --out-dir ./wrkr-out --json
```

## Expected Outputs

- `./wrkr-out/jobpacks/jobpack_<job_id>.zip`
- `./wrkr-out/reports/github_summary_<job_id>.json`
- `./wrkr-out/reports/github_summary_<job_id>.md`
- `./wrkr-out/reports/accept_<job_id>.junit.xml`
- `./wrkr-out/reports/work_item_<job_id>_<checkpoint>.json`

## Validation

Run these before rollout:

- `make test-adoption`
- `make test-conformance`
- `make test-uat-local`
