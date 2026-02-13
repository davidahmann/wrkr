# Homebrew Publishing (Tap-First)

This document defines the Homebrew strategy for Wrkr.

## Position

- GitHub Releases are the release source of truth.
- Homebrew is a distribution adapter, not the release system.
- Publish to custom tap first.
- Tap repo: `davidahmann/homebrew-tap` (tap alias `davidahmann/tap`).

## Install from Tap

```bash
brew tap davidahmann/tap
brew install davidahmann/tap/wrkr
wrkr --json version
```

## Verify Tap Formula

```bash
brew update
brew reinstall davidahmann/tap/wrkr
brew test davidahmann/tap/wrkr
wrkr --json demo
```

## Tap Update Workflow

1. Cut release in `davidahmann/wrkr` (`vX.Y.Z`).
2. Render formula from release checksums:

```bash
bash scripts/render_homebrew_formula.sh \
  --version vX.Y.Z \
  --checksums dist/checksums.txt \
  --repo-owner davidahmann \
  --repo-name wrkr \
  --output Formula/wrkr.rb
```

3. Commit formula in tap repo (`davidahmann/homebrew-tap`, `Formula/wrkr.rb`).
4. Merge and verify install/test locally.

## Release Automation

`release.yml` publishes formula on tag pushes (or manual dispatch when `publish_homebrew_tap=true`).

Required secret in `davidahmann/wrkr`:

- `HOMEBREW_TAP_TOKEN`: token with `contents: write` on `davidahmann/homebrew-tap`

Behavior:

- generates `dist/wrkr.rb` from release checksums
- publishes only when formula content changes
- skips publish cleanly when token is missing

Manual fallback remains supported via `scripts/publish_homebrew_tap.sh`.
