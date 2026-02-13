# Security Policy

Wrkr is OSS v1 and maintained by a single administrator.

## Reporting a Vulnerability

- Use GitHub Security Advisories for private disclosure.
- Do not open public issues for unpatched vulnerabilities.

## Scope

Security-sensitive surfaces in scope:
- durable store and recovery integrity
- jobpack hash verification and schema validation
- CLI/serve unsafe-operation guardrails
- release artifact integrity (checksums, signatures, provenance)

## Response Posture (OSS v1)

- Initial triage target: within 5 business days.
- Confirmed critical/high issues are prioritized for the next patch release.
- Best-effort fixes for moderate/low issues based on risk and exploitability.

## Coordination

- Reports may be acknowledged with mitigation guidance before a patch is cut.
- Credits are included in release notes unless anonymous disclosure is requested.
