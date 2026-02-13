# Wrkr Integration Examples

These examples define the blessed adoption lane used by `make test-adoption`.

## Files

- `blessed_jobspec.yaml`: reference adapter JobSpec that intentionally blocks on a `decision-needed` checkpoint.
- `blessed_accept.yaml`: deterministic acceptance config for demo artifacts.
- `wrap_fixture.sh`: tiny wrapped command fixture for wrap mode smoke.

## Quick Run

```bash
wrkr submit examples/integrations/blessed_jobspec.yaml --job-id job_example_lane
wrkr checkpoint list job_example_lane
wrkr approve job_example_lane --checkpoint <decision_checkpoint> --reason "approved"
wrkr resume job_example_lane
wrkr accept run job_example_lane --config examples/integrations/blessed_accept.yaml --ci
```
