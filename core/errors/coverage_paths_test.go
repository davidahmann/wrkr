package errors

import (
	stderrors "errors"
	"strings"
	"testing"
	"time"
)

func TestWrkrErrorAndEnvelopeCoveragePaths(t *testing.T) {
	t.Parallel()

	werr := WrkrError{Code: EAdapterFail, Message: "adapter failed"}
	if got := werr.Error(); got != "E_ADAPTER_FAIL: adapter failed" {
		t.Fatalf("unexpected WrkrError string: %q", got)
	}
	empty := WrkrError{Code: EStoreCorrupt}
	if got := empty.Error(); got != "E_STORE_CORRUPT" {
		t.Fatalf("unexpected empty-message WrkrError string: %q", got)
	}

	at := time.Date(2026, 2, 14, 19, 0, 0, 0, time.UTC)
	generic := ToEnvelope(stderrors.New("plain error"), "test", at)
	if generic.Code != EGenericFailure || generic.Message != "plain error" {
		t.Fatalf("unexpected generic envelope: %+v", generic)
	}

	specific := ToEnvelope(werr, "test", at)
	if specific.Code != EAdapterFail || specific.ExitCode != ExitCodeFor(EAdapterFail) {
		t.Fatalf("unexpected specific envelope: %+v", specific)
	}

	raw, err := MarshalEnvelope(werr, "test", at)
	if err != nil {
		t.Fatalf("MarshalEnvelope: %v", err)
	}
	if !strings.Contains(string(raw), `"code": "E_ADAPTER_FAIL"`) {
		t.Fatalf("unexpected marshaled envelope: %s", string(raw))
	}
}

