# CodeQL Triage Flow

This runbook defines deterministic handling of CodeQL findings.

## 1. Classify

- New finding in PR: treat as release-blocking until triaged.
- Existing/baseline finding: verify whether code path changed in this PR.

## 2. Reproduce

Run local wrapper (optional):

```bash
make codeql-local
```

Output SARIF is written to:

- `wrkr-out/reports/codeql-local.sarif`

## 3. Decide

- True positive: fix in PR and keep rule enabled.
- False positive: add narrow suppression and document rationale in PR.
- Accepted risk (temporary): open tracked issue with owner + due date.

## 4. Close

- CI check `codeql-scan` (workflow: `codeql`) must be green before merge.
- PR description should note triage decision for each high/critical finding.
