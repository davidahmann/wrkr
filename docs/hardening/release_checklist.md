# Release Hardening Checklist

This checklist is mandatory for every `v*` release.

## 1. Pre-Tag Gates

- [ ] `make lint-fast test-fast sast-fast` is green.
- [ ] `make test-v1-acceptance` is green.
- [ ] `make test-hardening-acceptance` is green.
- [ ] `make install-smoke release-smoke` is green.
- [ ] `make docs-site-build` is green.

## 2. Tag and Workflow Execution

- [ ] Release tag is annotated (`git tag -a vX.Y.Z -m "vX.Y.Z"`).
- [ ] Tag is pushed from `master` only.
- [ ] `.github/workflows/release.yml` completed successfully.

## 3. Integrity Artifacts (Required)

- [ ] Release archives exist for darwin/linux/windows x amd64/arm64.
- [ ] `dist/checksums.txt` exists and validates all archives.
- [ ] `dist/wrkr.sbom.spdx.json` exists.
- [ ] `dist/wrkr.vuln.sarif` exists.
- [ ] `dist/checksums.txt.sig` and `dist/checksums.txt.pem` exist.
- [ ] Provenance attestation step succeeded.
- [ ] `scripts/verify_release_assets.sh` passed in workflow logs.

## 4. Homebrew (Mirrored Optional Path)

- [ ] `scripts/render_homebrew_formula.sh` generated `wrkr.rb` from the release checksums.
- [ ] If publishing tap: formula published to `davidahmann/homebrew-tap` (`davidahmann/tap`) on `main`.
- [ ] If token is missing, workflow explicitly logs Homebrew publish skip.

## 5. Changelog Contract

- [ ] `CHANGELOG.md` exists and is non-empty.
- [ ] Release workflow changelog validation step passed.

## 6. Release Notes and Approval Record

- [ ] GitHub release body uses `docs/launch/github_release_template.md`.
- [ ] Contract-impact section is completed (`none` is explicit if no changes).
- [ ] Known limitations and rollback instructions are included.

## 7. Single-Admin Operability

- [ ] No step required two-admin approval.
- [ ] Any skipped optional publish paths are documented in release notes.
