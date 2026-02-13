# CI Required Checks (Canonical)

This file is the single source of truth for branch protection required checks.

## Required Checks for `master`

- `pr-fast`
- `ci`
- `ticket-footer-conformance`
- `wrkr-compatible-conformance`
- `codeql`

## Bootstrap / Update (single admin)

Dry run:

```bash
./scripts/bootstrap_branch_protection.sh
```

Apply:

```bash
./scripts/bootstrap_branch_protection.sh --apply
```

Set explicit repo/branch:

```bash
./scripts/bootstrap_branch_protection.sh --repo davidahmann/wrkr --branch master --apply
```
