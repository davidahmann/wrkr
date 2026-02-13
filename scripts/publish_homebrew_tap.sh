#!/usr/bin/env bash
set -euo pipefail

formula_file=""
tap_repo="davidahmann/homebrew-wrkr"
tap_branch="master"
apply=0

usage() {
  cat <<USAGE
usage: $0 --formula <path> [--tap-repo <owner/repo>] [--tap-branch <branch>] [--apply]
USAGE
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --formula)
      formula_file="$2"
      shift 2
      ;;
    --tap-repo)
      tap_repo="$2"
      shift 2
      ;;
    --tap-branch)
      tap_branch="$2"
      shift 2
      ;;
    --apply)
      apply=1
      shift
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

if [[ -z "$formula_file" ]]; then
  usage >&2
  exit 2
fi

if [[ ! -f "$formula_file" ]]; then
  echo "formula not found: $formula_file" >&2
  exit 1
fi

if ! command -v gh >/dev/null 2>&1; then
  echo "gh CLI is required" >&2
  exit 1
fi

if [[ -z "${GH_TOKEN:-}" && -n "${GITHUB_TOKEN:-}" ]]; then
  export GH_TOKEN="$GITHUB_TOKEN"
fi

if [[ "$apply" -eq 1 && -z "${GH_TOKEN:-}" ]]; then
  echo "GH_TOKEN (or GITHUB_TOKEN) is required with --apply" >&2
  exit 1
fi

tmp_dir="$(mktemp -d)"
trap 'rm -rf "$tmp_dir"' EXIT

repo_dir="$tmp_dir/tap"

echo "[wrkr] cloning ${tap_repo}@${tap_branch}"
gh repo clone "$tap_repo" "$repo_dir" -- --branch "$tap_branch" --single-branch >/dev/null

mkdir -p "$repo_dir/Formula"
cp "$formula_file" "$repo_dir/Formula/wrkr.rb"

pushd "$repo_dir" >/dev/null

if git diff --quiet -- Formula/wrkr.rb; then
  echo "[wrkr] no Homebrew formula changes to publish"
  exit 0
fi

if [[ "$apply" -ne 1 ]]; then
  echo "[wrkr] dry-run: formula change detected for ${tap_repo}:${tap_branch}"
  git --no-pager diff -- Formula/wrkr.rb || true
  echo "[wrkr] apply with: $0 --formula $formula_file --tap-repo $tap_repo --tap-branch $tap_branch --apply"
  exit 0
fi

git config user.name "wrkr-release-bot"
git config user.email "wrkr-release-bot@users.noreply.github.com"

git add Formula/wrkr.rb
git commit -m "chore: update wrkr formula"
git push origin "$tap_branch"

echo "[wrkr] published formula to ${tap_repo}:${tap_branch}"

popd >/dev/null
