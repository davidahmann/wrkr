# Wrkr-Compatible Lane Contract

A lane can claim "Wrkr-compatible" if it satisfies the core contract surfaces.

## Required

- Uses Wrkr checkpoint types and schema-compatible records.
- Produces deterministic output layout under `./wrkr-out/`.
- Supports jobpack export and offline verification.
- Supports deterministic acceptance execution.
- Preserves stable reason codes and exit codes.

## Conformance References

- `scripts/test_contracts.sh`
- `scripts/test_wrkr_compatible_conformance.sh`
- `scripts/test_github_summary_golden.sh`
- `scripts/test_serve_hardening.sh`
- `docs/contracts/ticket_footer_conformance.md`
- `docs/contracts/github_summary_conformance.md`
- `docs/contracts/work_item_bridge_contract.md`
