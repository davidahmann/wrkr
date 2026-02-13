# GitHub Actions Kit

Use this kit to make Wrkr-compatible behavior a CI-enforced claim.

## Reusable Workflow Template

- Template path: `.github/workflows/adoption-regress-template.yml`
- Example consumer usage: `examples/ci/github-actions-adoption.yml`

## Template Behavior

The template runs:

1. `make test-adoption`
2. `make test-conformance`

and uploads `wrkr-out/**` artifacts for review.

## Recommended CI Job Sequence for Agent Jobs

1. `wrkr accept run <job_id> --ci --json`
2. `wrkr export <job_id> --json`
3. `wrkr verify <job_id> --json`
4. `wrkr report github <job_id> --json`
5. Upload `./wrkr-out/` artifacts

## Required Published Artifacts

- Jobpack zip
- Acceptance JUnit (if generated)
- GitHub summary JSON + markdown
- Work-item payload files when decision checkpoints occur
