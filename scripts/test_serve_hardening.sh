#!/usr/bin/env bash
set -euo pipefail

echo "[wrkr] serve hardening conformance"
go test ./core/serve -run 'TestValidateConfigNonLoopbackGuardrails|TestAuthAndBodyLimit|TestPathTraversalRejected|TestJobIDTraversalRejectedOnMutationEndpoints' -count=1
go test ./cmd/wrkr -run 'TestRunServeJSONOutputRemainsParseable' -count=1
