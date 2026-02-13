#!/usr/bin/env bash
set -euo pipefail

echo "[wrkr] ticket footer + verify conformance"
go test ./core/pack -run 'TestTicketFooterRoundTrip|TestParseTicketFooterRejectsInvalid|TestVerifyDetectsTampering|TestVerifyRejectsUndeclaredArchiveEntries|TestVerifyRejectsOversizedInvalidJSONLRecord' -count=1
go test ./cmd/wrkr -run 'TestExportVerifyAndReceipt' -count=1
