# Wrkr

Wrkr is a durable dispatch and supervision substrate for long-running agent jobs.

Wrkr is not an agent framework. It provides deterministic job lifecycle control and portable evidence (`jobpack`) across agent tools.

## Install

### From source (works today)

```bash
git clone https://github.com/davidahmann/wrkr.git
cd wrkr
make build
./bin/wrkr --json version
```

### From GitHub Releases (single binary)

1. Download the latest archive for your OS/arch from Releases.
2. Verify checksums.
3. Place `wrkr` on your `PATH`.

```bash
wrkr --json version
```

### Homebrew (when tap is published)

```bash
brew tap davidahmann/wrkr
brew install wrkr
wrkr --json version
```

## 60-second first win

```bash
wrkr --json demo
wrkr --json verify <job_id>
```

Expected outputs:
- a deterministic `job_id`
- `./wrkr-out/jobpacks/jobpack_<job_id>.zip`
- successful offline verify result

## Core flow (structured dispatch)

```bash
wrkr init jobspec.yaml
wrkr --json submit jobspec.yaml --job-id job_demo
wrkr --json checkpoint list job_demo
wrkr --json approve job_demo --checkpoint <cp_id> --reason "approved"
wrkr --json resume job_demo
wrkr --json export job_demo
wrkr --json verify job_demo
wrkr --json accept run job_demo --config examples/integrations/blessed_accept.yaml --ci
wrkr --json report github job_demo
```

For structured jobs, `resume` continues remaining steps from the durable cursor and can complete the run.

## Platform support

- Binary: macOS, Linux, Windows.
- Command-executing adapter paths (`reference`, `wrap`) run commands via `sh -lc` in OSS v1 and are currently Unix-oriented.
- Non-command paths (store/status/export/verify/accept/report) are cross-platform.

## Troubleshooting

- `wrkr: command not found`: add the binary location to `PATH` or run `./bin/wrkr`.
- Store path: default runtime state is under `~/.wrkr`.
- Output path: deterministic artifacts are written under `./wrkr-out/` unless overridden.

## Product docs

- PRD: `product/PRD.md`
- Plan: `product/PLAN_v1.md`
- Gaps and closure plan: `product/GAPS.md`
- Docs map: `docs/README.md`
