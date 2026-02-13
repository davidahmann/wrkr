# Project Defaults

## Runtime

- Local store root: `~/.wrkr`
- Output root: `./wrkr-out`
- Default serve bind: `127.0.0.1:9488`

## Capture

- Default artifact mode: reference-only
- Deterministic JSON outputs for major machine workflows
- `--explain` provides a bounded command-intent summary without executing work

## Reliability

- Append-only events + snapshots
- Resume and recovery are first-class paths

## Platform Support (OSS v1)

- Core CLI/state/artifact commands are cross-platform (macOS/Linux/Windows binaries).
- Command-executing adapter paths (`reference`, `wrap`) invoke `sh -lc` and are documented as Unix-oriented in OSS v1.
