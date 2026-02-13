#!/usr/bin/env bash
set -euo pipefail

version=""
checksums_file=""
repo_owner=""
repo_name="wrkr"
output_file="./wrkr-out/release/wrkr.rb"

usage() {
  cat <<USAGE
usage: $0 --version <vX.Y.Z> --checksums <path> [--repo-owner <owner>] [--repo-name <name>] [--output <path>]
USAGE
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --version)
      version="$2"
      shift 2
      ;;
    --checksums)
      checksums_file="$2"
      shift 2
      ;;
    --repo-owner)
      repo_owner="$2"
      shift 2
      ;;
    --repo-name)
      repo_name="$2"
      shift 2
      ;;
    --output)
      output_file="$2"
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

if [[ -z "$version" || -z "$checksums_file" || -z "$repo_owner" ]]; then
  usage >&2
  exit 2
fi

if [[ ! -f "$checksums_file" ]]; then
  echo "checksums file not found: $checksums_file" >&2
  exit 1
fi

checksum_for() {
  local file_name="$1"
  local value
  value="$(awk -v f="$file_name" '$2==f{print $1}' "$checksums_file")"
  if [[ -z "$value" ]]; then
    echo "missing checksum for $file_name" >&2
    exit 1
  fi
  if [[ "$(wc -l <<<"$value" | tr -d ' ')" -ne 1 ]]; then
    echo "multiple checksums for $file_name" >&2
    exit 1
  fi
  printf "%s" "$value"
}

archive_version="${version#v}"
archive_darwin_amd64="wrkr_${archive_version}_darwin_amd64.tar.gz"
archive_darwin_arm64="wrkr_${archive_version}_darwin_arm64.tar.gz"
archive_linux_amd64="wrkr_${archive_version}_linux_amd64.tar.gz"
archive_linux_arm64="wrkr_${archive_version}_linux_arm64.tar.gz"

sha_darwin_amd64="$(checksum_for "$archive_darwin_amd64")"
sha_darwin_arm64="$(checksum_for "$archive_darwin_arm64")"
sha_linux_amd64="$(checksum_for "$archive_linux_amd64")"
sha_linux_arm64="$(checksum_for "$archive_linux_arm64")"

mkdir -p "$(dirname "$output_file")"

release_url_base="https://github.com/${repo_owner}/${repo_name}/releases/download/${version}"

cat > "$output_file" <<FORMULA
class Wrkr < Formula
  desc "Dispatch and supervision for long-running agent jobs"
  homepage "https://github.com/${repo_owner}/${repo_name}"
  version "${version#v}"

  on_macos do
    if Hardware::CPU.arm?
      url "${release_url_base}/${archive_darwin_arm64}"
      sha256 "${sha_darwin_arm64}"
    else
      url "${release_url_base}/${archive_darwin_amd64}"
      sha256 "${sha_darwin_amd64}"
    end
  end

  on_linux do
    if Hardware::CPU.arm?
      url "${release_url_base}/${archive_linux_arm64}"
      sha256 "${sha_linux_arm64}"
    else
      url "${release_url_base}/${archive_linux_amd64}"
      sha256 "${sha_linux_amd64}"
    end
  end

  def install
    bin.install "wrkr"
  end

  test do
    output = shell_output("#{bin}/wrkr --json version")
    assert_match '"version"', output
  end
end
FORMULA

echo "[wrkr] rendered Homebrew formula: $output_file"
