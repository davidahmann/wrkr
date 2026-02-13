#!/usr/bin/env bash
set -euo pipefail

echo "[wrkr] release contracts"

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
tmp_root="$(mktemp -d)"
trap 'rm -rf "$tmp_root"' EXIT

dist_dir="$tmp_root/dist"
mkdir -p "$dist_dir"

tag="v1.2.3"
archive_version="${tag#v}"
for file in \
  "wrkr_${archive_version}_darwin_amd64.tar.gz" \
  "wrkr_${archive_version}_darwin_arm64.tar.gz" \
  "wrkr_${archive_version}_linux_amd64.tar.gz" \
  "wrkr_${archive_version}_linux_arm64.tar.gz" \
  "wrkr_${archive_version}_windows_amd64.zip" \
  "wrkr_${archive_version}_windows_arm64.zip"; do
  printf 'fixture-%s\n' "$file" > "$dist_dir/$file"
done

(
  cd "$dist_dir"
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum wrkr_* > checksums.txt
  else
    shasum -a 256 wrkr_* > checksums.txt
  fi
)

"$repo_root/scripts/verify_release_assets.sh" --dist "$dist_dir" --expected-tag "$tag"

formula="$tmp_root/wrkr.rb"
"$repo_root/scripts/render_homebrew_formula.sh" \
  --version "$tag" \
  --checksums "$dist_dir/checksums.txt" \
  --repo-owner davidahmann \
  --repo-name wrkr \
  --output "$formula"

if ! grep -q "wrkr_1.2.3_darwin_arm64.tar.gz" "$formula"; then
  echo "[wrkr][release contracts] formula missing non-prefixed archive name"
  cat "$formula"
  exit 1
fi
if ! grep -q "releases/download/v1.2.3" "$formula"; then
  echo "[wrkr][release contracts] formula missing v-tag release URL"
  cat "$formula"
  exit 1
fi

workflow="$repo_root/.github/workflows/release.yml"
if ! grep -q "acceptance-gate" "$workflow"; then
  echo "[wrkr][release contracts] missing acceptance-gate job in release workflow"
  exit 1
fi
if ! grep -q "Checkout requested tag" "$workflow"; then
  echo "[wrkr][release contracts] missing tag checkout step"
  exit 1
fi
if ! grep -Fq 'git checkout "refs/tags/${TAG_NAME}"' "$workflow"; then
  echo "[wrkr][release contracts] missing explicit refs/tags checkout command"
  exit 1
fi

echo "[wrkr] release contracts ok"
