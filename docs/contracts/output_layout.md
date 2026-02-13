# Output Layout Contract

Default output root: `./wrkr-out/`

## Paths

- Jobpacks: `./wrkr-out/jobpacks/jobpack_<job_id>.zip`
- Reports: `./wrkr-out/reports/*`
- Integrations: `./wrkr-out/integrations/<lane>/*`

## Rules

- Paths are deterministic from job ID + artifact type.
- Report outputs are stable and machine readable.
- Integration outputs must be lane-scoped and reproducible.
