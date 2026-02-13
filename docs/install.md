# Install Wrkr

This document is the canonical installation guide for Wrkr OSS v1.

## Supported Platforms

- macOS
- Linux
- Windows (manual release asset path)

Wrkr ships as a single static Go binary.

## Install Paths (v1)

1. Source build (`go build`) 
2. GitHub release installer (`scripts/install.sh`) 
3. Homebrew tap (`davidahmann/tap/wrkr`)

Local UAT assumes these three paths unless explicitly skipped.

## Option 1: Source Build

```bash
git clone https://github.com/davidahmann/wrkr.git
cd wrkr
make build
./bin/wrkr --json version
```

## Option 2: Release Installer (Recommended)

```bash
curl -fsSL https://raw.githubusercontent.com/davidahmann/wrkr/main/scripts/install.sh | bash
```

This path requires published GitHub release assets.

Pin a specific tag:

```bash
bash scripts/install.sh --version vX.Y.Z --install-dir ~/.local/bin
```

The installer:

- resolves release tag (`latest` by default)
- downloads `checksums.txt` and the matching archive
- verifies SHA-256 checksum
- installs `wrkr` into `~/.local/bin` by default

## Option 3: Homebrew Tap

```bash
brew tap davidahmann/tap
brew install davidahmann/tap/wrkr
wrkr --json version
```

Validate brew formula:

```bash
brew test davidahmann/tap/wrkr
```

See `docs/homebrew.md` for tap publishing details.

## Verify Installation

```bash
wrkr --json demo
wrkr --json verify <job_id>
```

Expected:

- deterministic `job_id`
- `./wrkr-out/jobpacks/jobpack_<job_id>.zip`
- successful offline verify output

## Windows Manual Path

1. Download the matching Windows archive from GitHub Releases.
2. Verify against `checksums.txt`.
3. Extract `wrkr.exe` and place it on `PATH`.
4. Validate with `wrkr --json version`.

## Troubleshooting

- `wrkr: command not found`: verify install directory is on `PATH`.
- Source-build path uses `./bin/wrkr` unless copied to `PATH`.
- Runtime store default: `~/.wrkr`.
- Deterministic output root: `./wrkr-out/`.
