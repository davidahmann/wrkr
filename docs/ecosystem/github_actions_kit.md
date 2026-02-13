# GitHub Actions Kit

## CI Job Sequence

1. Checkout repository
2. Run `wrkr accept run <job_id> --ci --json`
3. Run `wrkr export <job_id> --json`
4. Run `wrkr verify <job_id> --json`
5. Upload `./wrkr-out/` artifacts

## Recommended Published Artifacts

- Jobpack zip
- Acceptance JUnit (if generated)
- GitHub summary JSON/Markdown
- Bridge work-item payloads (if produced)
