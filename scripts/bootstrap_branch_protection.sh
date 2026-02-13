#!/usr/bin/env bash
set -euo pipefail

repo=""
branch="master"
apply=0

usage() {
  cat <<USAGE
usage: $0 [--repo owner/name] [--branch master] [--apply]
USAGE
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --repo)
      repo="$2"
      shift 2
      ;;
    --branch)
      branch="$2"
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
      usage >&2
      exit 2
      ;;
  esac
done

if ! command -v gh >/dev/null 2>&1; then
  echo "[wrkr] gh CLI is required" >&2
  exit 1
fi

if ! gh auth status >/dev/null 2>&1; then
  echo "[wrkr] gh auth status failed; run: gh auth login" >&2
  exit 1
fi

if [[ -z "$repo" ]]; then
  repo="$(gh repo view --json nameWithOwner --jq '.nameWithOwner')"
fi

payload="$(cat <<JSON
{
  "required_status_checks": {
    "strict": true,
    "checks": [
      {"context": "pr-fast"},
      {"context": "ci"},
      {"context": "ticket-footer-conformance"},
      {"context": "wrkr-compatible-conformance"},
      {"context": "codeql-scan"}
    ]
  },
  "enforce_admins": true,
  "required_pull_request_reviews": {
    "dismiss_stale_reviews": true,
    "require_code_owner_reviews": false,
    "required_approving_review_count": 0,
    "require_last_push_approval": false
  },
  "restrictions": null,
  "allow_force_pushes": false,
  "allow_deletions": false,
  "required_linear_history": true,
  "block_creations": false,
  "required_conversation_resolution": true,
  "lock_branch": false,
  "allow_fork_syncing": false
}
JSON
)"

if [[ "$apply" -ne 1 ]]; then
  echo "[wrkr] dry-run only. computed branch protection payload for ${repo}:${branch}"
  echo "$payload"
  echo "[wrkr] apply with: $0 --repo ${repo} --branch ${branch} --apply"
  exit 0
fi

echo "[wrkr] applying branch protection for ${repo}:${branch}"
gh api \
  --method PUT \
  -H "Accept: application/vnd.github+json" \
  "/repos/${repo}/branches/${branch}/protection" \
  --input - <<<"$payload" >/dev/null

echo "[wrkr] branch protection applied"
gh api \
  "/repos/${repo}/branches/${branch}/protection" \
  --jq '{required_checks: .required_status_checks.contexts, strict: .required_status_checks.strict, required_reviews: .required_pull_request_reviews.required_approving_review_count, enforce_admins: .enforce_admins.enabled, required_conversation_resolution: .required_conversation_resolution.enabled, required_linear_history: .required_linear_history.enabled, allow_force_pushes: .allow_force_pushes.enabled, allow_deletions: .allow_deletions.enabled}'
