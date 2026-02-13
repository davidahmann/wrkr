#!/usr/bin/env bash
set -euo pipefail

echo "[wrkr] chaos/store: append corruption + lock contention"
go test ./core/store -run 'TestAppendEventReclaimsStaleLock|TestCrashPartialLineDoesNotCorruptCommittedEvents|TestAppendEventCASConflict' -count=1
go test ./internal/integration -run 'TestStoreConcurrentAppendNoCorruption' -count=1
