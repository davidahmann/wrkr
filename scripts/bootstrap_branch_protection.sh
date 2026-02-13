#!/usr/bin/env bash
set -euo pipefail

repo=""
branch="master"
apply=0

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
    *)
      echo "usage: $0 [--repo owner/name] [--branch master] [--apply]"
      exit 2
      ;;
  esac
done

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
      {"context": "codeql"}
    ]
  },
  "enforce_admins": false,
  "required_pull_request_reviews": null,
  "restrictions": null,
  "allow_force_pushes": false,
  "allow_deletions": false,
  "required_linear_history": false,
  "block_creations": false,
  "required_conversation_resolution": false,
  "lock_branch": false,
  "allow_fork_syncing": true
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
  --input - <<<"$payload"

echo "[wrkr] branch protection applied"
