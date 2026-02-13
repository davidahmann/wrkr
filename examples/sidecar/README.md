# Wrkr Sidecar Example

This example demonstrates a transport-only sidecar lane for non-Python stacks.

## Run Offline Fixture

```bash
python3 ./examples/sidecar/sidecar.py \
  --request ./examples/sidecar/request_fixture.json \
  --dry-run
```

Outputs are written deterministically under:

- `./wrkr-out/integrations/<lane>/request.json`
- `./wrkr-out/integrations/<lane>/result.json`
- `./wrkr-out/integrations/<lane>/sidecar.log`

The fixture includes a fixed `timestamp_utc` to guarantee deterministic dry-run output.
