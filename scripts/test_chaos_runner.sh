#!/usr/bin/env bash
set -euo pipefail

echo "[wrkr] chaos/runner: lease, heartbeat, env gating, override audit"
go test ./core/runner -run 'TestLeaseConflictAndExpiryClaim|TestConcurrentLeaseAcquireReturnsSingleWinner|TestResumeBlocksOnEnvMismatchUnlessOverridden|TestResumeBlocksOnEnvMismatchFromBlockedDecision' -count=1
go test ./internal/integration -run 'TestLeaseHeartbeatExpiryAndSafeReclaim|TestResumeEnvMismatchOverrideWritesAuditTrail|TestTraceAndArtifactUniqueness' -count=1
