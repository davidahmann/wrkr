#!/usr/bin/env bash
set -euo pipefail

dist_dir="dist"
expected_tag=""
required_files=()

usage() {
  cat <<USAGE
usage: $0 [--dist <dir>] [--expected-tag <vX.Y.Z>] [--require-file <name>]...
USAGE
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --dist)
      dist_dir="$2"
      shift 2
      ;;
    --expected-tag)
      expected_tag="$2"
      shift 2
      ;;
    --require-file)
      required_files+=("$2")
      shift 2
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      echo "unknown argument: $1" >&2
      usage >&2
      exit 2
      ;;
  esac
done

if [[ ! -d "$dist_dir" ]]; then
  echo "[wrkr] release integrity check failed: missing dist directory $dist_dir" >&2
  exit 1
fi

checksums_file="$dist_dir/checksums.txt"
if [[ ! -s "$checksums_file" ]]; then
  echo "[wrkr] release integrity check failed: missing checksums.txt" >&2
  exit 1
fi

archives=("$dist_dir"/wrkr_*.tar.gz "$dist_dir"/wrkr_*.zip)
archive_count=0
for archive in "${archives[@]}"; do
  if [[ -f "$archive" ]]; then
    archive_count=$((archive_count + 1))
  fi
done
if [[ "$archive_count" -lt 6 ]]; then
  echo "[wrkr] release integrity check failed: expected at least 6 archives, found $archive_count" >&2
  exit 1
fi

if [[ -n "$expected_tag" ]]; then
  expected=(
    "wrkr_${expected_tag}_darwin_amd64.tar.gz"
    "wrkr_${expected_tag}_darwin_arm64.tar.gz"
    "wrkr_${expected_tag}_linux_amd64.tar.gz"
    "wrkr_${expected_tag}_linux_arm64.tar.gz"
    "wrkr_${expected_tag}_windows_amd64.zip"
    "wrkr_${expected_tag}_windows_arm64.zip"
  )
  for file in "${expected[@]}"; do
    if [[ ! -f "$dist_dir/$file" ]]; then
      echo "[wrkr] release integrity check failed: missing expected archive $file" >&2
      exit 1
    fi
  done
fi

while IFS= read -r line; do
  [[ -z "$line" ]] && continue
  file_name="$(awk '{print $2}' <<<"$line")"
  if [[ -z "$file_name" ]]; then
    echo "[wrkr] release integrity check failed: malformed checksum line '$line'" >&2
    exit 1
  fi
  if [[ ! -f "$dist_dir/$file_name" ]]; then
    echo "[wrkr] release integrity check failed: checksum entry references missing file $file_name" >&2
    exit 1
  fi
done < "$checksums_file"

for archive in "$dist_dir"/wrkr_*.tar.gz "$dist_dir"/wrkr_*.zip; do
  [[ -f "$archive" ]] || continue
  base="$(basename "$archive")"
  if ! awk -v f="$base" '$2==f{found=1} END{exit found?0:1}' "$checksums_file"; then
    echo "[wrkr] release integrity check failed: archive missing from checksums.txt ($base)" >&2
    exit 1
  fi
done

if (( ${#required_files[@]} > 0 )); then
  for required in "${required_files[@]}"; do
    if [[ ! -s "$dist_dir/$required" ]]; then
      echo "[wrkr] release integrity check failed: required artifact missing or empty ($required)" >&2
      exit 1
    fi
  done
fi

(
  cd "$dist_dir"
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum -c checksums.txt >/dev/null
  elif command -v shasum >/dev/null 2>&1; then
    shasum -a 256 -c checksums.txt >/dev/null
  else
    echo "[wrkr] release integrity check failed: neither sha256sum nor shasum is available" >&2
    exit 1
  fi
)

echo "[wrkr] release integrity check passed"
