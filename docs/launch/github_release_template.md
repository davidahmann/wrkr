# Wrkr Release Template

## Summary

- Version: `vX.Y.Z`
- Date: `YYYY-MM-DD`
- Scope: one-paragraph summary of what changed.

## Notable Changes

- Runtime/CLI changes:
- Contract/schema changes:
- CI/CD or hardening changes:
- Docs changes:

## Contract Impact

- Primitive contract (`dispatch/checkpoint/accept/jobpack`):
- Output layout contract (`wrkr-out`):
- Exit codes/failure taxonomy impact:
- Backward compatibility statement:

## Release Artifacts

- Multi-platform archives: darwin/linux/windows x amd64/arm64
- `checksums.txt`
- `wrkr.sbom.spdx.json`
- `wrkr.vuln.sarif`
- `checksums.txt.sig`
- `checksums.txt.pem`
- Provenance attestation

## Verification Steps

- `sha256sum -c checksums.txt`
- `wrkr --json version`
- `wrkr demo --json`
- `wrkr verify <job_id|jobpack.zip>`
- verify install matrix snippets in README:
  - source build path
  - release binary install path
  - Homebrew tap path (if published)

## Homebrew

- Formula generated from release checksums: yes/no
- Tap published: yes/no
- Tap repo/branch: `davidahmann/homebrew-tap` / `main`
- README section updated with current tap command: yes/no

## Known Issues / Follow-Ups

- List known limitations and linked issues.

## Rollback Plan

- Tag to rollback to:
- Operational rollback steps:
- Communication owner:
