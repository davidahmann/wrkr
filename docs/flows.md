# Runtime Flows

## Flow A: Structured Submit

1. `wrkr init` creates a JobSpec template.
2. `wrkr submit jobspec.yaml` initializes job state and runs adapter steps.
3. Runner emits `plan`, `progress`, `decision-needed`, `blocked`, `completed` checkpoints.
4. On decision checkpoint, manager approves with `wrkr approve`.
5. Job resumes with `wrkr resume`.

## Flow B: Wrap Adoption

1. `wrkr wrap -- <agent command...>` runs existing agent CLI without rewrite.
2. Wrap adapter emits bounded checkpoints and artifacts references.
3. `wrkr export` creates jobpack and ticket footer.
4. `wrkr verify` confirms integrity.

## Flow C: Deterministic Acceptance

1. `wrkr accept init` creates acceptance config.
2. `wrkr accept run <job_id>` executes deterministic checks.
3. Optional CI mode emits JUnit and GitHub summary artifacts.
