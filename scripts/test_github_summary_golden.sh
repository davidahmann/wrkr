#!/usr/bin/env bash
set -euo pipefail

echo "[wrkr] github summary conformance"
go test ./core/report -run 'TestBuildAndWriteGitHubSummary|TestLatestCheckpointSummaryNumericTieBreak' -count=1
go test ./cmd/wrkr -run 'TestAcceptRunCIAndReport' -count=1
