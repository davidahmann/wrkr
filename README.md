# Dispatch and Supervise Long-Running Agent Jobs with Verifiable Jobpacks

Wrkr makes multi-hour agent work operable: durable dispatch, structured checkpoints, deterministic budgets, resumability, and offline-verifiable artifacts.

![PR Fast](https://github.com/davidahmann/wrkr/actions/workflows/pr-fast.yml/badge.svg)
![CodeQL](https://github.com/davidahmann/wrkr/actions/workflows/codeql.yml/badge.svg)
![Wrkr-Compatible Conformance](https://github.com/davidahmann/wrkr/actions/workflows/wrkr-compatible-conformance.yml/badge.svg)

Public docs: [https://davidahmann.github.io/wrkr/](https://davidahmann.github.io/wrkr/)  
Docs map: [`docs/README.md`](docs/README.md)  
Primitive contract: [`docs/contracts/primitive_contract.md`](docs/contracts/primitive_contract.md)  
Changelog: [`CHANGELOG.md`](CHANGELOG.md)

Primary CTA: `wrkr demo` (offline, <60s)  
Secondary CTA: verify with `wrkr verify <job_id|path>`

- durable execution: checkpointed state with safe resume semantics
- reviewable by default: bounded checkpoint summaries + deterministic acceptance outputs
- portable evidence: verifiable `jobpack_<job_id>.zip` artifacts for PRs/incidents

Outputs: `job_id`, `jobpack_<job_id>.zip`, and a stable ticket footer.

## Try It (Offline, <60s)

Install:

```bash
# release installer (recommended)
curl -fsSL https://raw.githubusercontent.com/davidahmann/wrkr/main/scripts/install.sh | bash

# Homebrew tap
brew tap davidahmann/tap
brew install davidahmann/tap/wrkr

# source build
git clone https://github.com/davidahmann/wrkr.git
cd wrkr
make build
./bin/wrkr --json version
```

Run demo:

```bash
wrkr --json demo
```

Verify:

```bash
wrkr --json verify <job_id>
```

Install details: [`docs/install.md`](docs/install.md), [`docs/homebrew.md`](docs/homebrew.md)

## Wrkr In 25 Seconds

![Wrkr dispatch-first terminal demo](docs/assets/wrkr_demo_25s.gif)

Regenerate assets:

```bash
bash scripts/record_wrkr_hero_demo.sh
```

## Why Wrkr

Agent intelligence is not the blocker on long-running work. Runtime durability and reviewability are.

Wrkr keeps the contract deterministic and offline-first:

- dispatch: durable state with pause/resume/cancel and deterministic stop reasons
- checkpoint: structured supervisory interrupts (`plan`, `progress`, `decision-needed`, `blocked`, `completed`)
- accept: deterministic checks for artifact presence, command execution, and CI gates
- jobpack: portable evidence bundle with manifest hashes and offline verify

If an agent changed production code or infra, attach the jobpack.

## First Win

```bash
wrkr --json demo
wrkr --json verify <job_id>
wrkr --json report github <job_id>
```

Expected outputs include:

- `job_id=...`
- jobpack under `./wrkr-out/jobpacks/`
- deterministic report artifacts under `./wrkr-out/reports/`

## Core OSS Surfaces

- `dispatch`: `init`, `submit`, `status`, `pause`, `resume`, `cancel`
- `checkpoint`: `list`, `show`, `emit`, `approve`
- `jobpack`: `export`, `verify`, `job inspect`, `job diff`, `receipt`
- `accept`: deterministic acceptance checks + optional CI/JUnit output
- `bridge`: blocked/decision checkpoint -> work-item payload
- `serve`: local API transport surface with explicit hardening controls
- `doctor`: production-readiness and configuration diagnostics

## Structured Long-Run Flow

```bash
wrkr init jobspec.yaml
wrkr --json submit jobspec.yaml --job-id job_demo
wrkr --json checkpoint list job_demo
wrkr --json approve job_demo --checkpoint <cp_id> --reason "approved"
wrkr --json resume job_demo
wrkr --json accept run job_demo --config examples/integrations/blessed_accept.yaml --ci
wrkr --json export job_demo
wrkr --json verify job_demo
```

`resume` continues from durable cursor state (`next_step_index`) for structured adapter jobs.

## Optional Local API Surface

```bash
wrkr serve --listen 127.0.0.1:9488
```

Serve contract and hardening:

- [`docs/contracts/serve_api.md`](docs/contracts/serve_api.md)
- [`docs/hardening/production_readiness.md`](docs/hardening/production_readiness.md)

## Integrations

Blessed lane artifacts and templates:

- [`examples/integrations/`](examples/integrations/)
- [`docs/ecosystem/blessed_lane.md`](docs/ecosystem/blessed_lane.md)
- [`docs/ecosystem/github_actions_kit.md`](docs/ecosystem/github_actions_kit.md)
- [`.github/workflows/adoption-regress-template.yml`](.github/workflows/adoption-regress-template.yml)

## Production Posture

Default posture is safe-by-default:

- no network listener by default
- deterministic exit codes and reason codes
- manifest/hash verify catches tampering
- non-loopback serve requires explicit hardening inputs

Readiness command:

```bash
wrkr doctor --production-readiness --json
```

## Command Surface

Most-used commands:

```text
wrkr demo
wrkr submit <jobspec.yaml>
wrkr status <job_id>
wrkr checkpoint list <job_id>
wrkr approve <job_id> --checkpoint <cp_id> --reason <text>
wrkr resume <job_id>
wrkr accept run <job_id> --config <accept.yaml>
wrkr export <job_id>
wrkr verify <job_id|path>
wrkr report github <job_id>
wrkr doctor --json
```

Major command paths support `--json`; command intent descriptions are available via `--explain`.

## Contract Commitments

- determinism: stable status/checkpoint/export/verify semantics for identical state
- offline-first: demo, export, verify, and status/checkpoint reads run locally
- fail-closed controls: unsafe serve configurations require explicit hardening flags
- schema stability: versioned artifacts and contracts under `schemas/v1/`
- stable exit codes: `0` success, `2` verify failure, `4` approval required, `5` acceptance failure, `6` invalid input/schema, `8` unsafe operation

Normative contracts: [`docs/contracts/`](docs/contracts/)

## Documentation Map

Start here:

1. [`docs/README.md`](docs/README.md)
2. [`docs/concepts/mental_model.md`](docs/concepts/mental_model.md)
3. [`docs/architecture.md`](docs/architecture.md)
4. [`docs/flows.md`](docs/flows.md)
5. [`docs/contracts/primitive_contract.md`](docs/contracts/primitive_contract.md)

## Developer Workflow

```bash
make fmt
make lint-fast
make test-fast
make test-v1-acceptance
make test-adoption
make test-hardening-acceptance
make docs-site-lint docs-site-build
```

Local hooks:

```bash
make hooks
```

Contributor guide: [`CONTRIBUTING.md`](CONTRIBUTING.md)

## Feedback

- Issues: [https://github.com/davidahmann/wrkr/issues](https://github.com/davidahmann/wrkr/issues)
- Security reporting: [`SECURITY.md`](SECURITY.md)
- Contribution guide: [`CONTRIBUTING.md`](CONTRIBUTING.md)
- Community expectations: [`CODE_OF_CONDUCT.md`](CODE_OF_CONDUCT.md)
