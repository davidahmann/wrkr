package errors

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestExitCodeMapping(t *testing.T) {
	t.Parallel()

	cases := map[Code]int{
		EVerifyHashMismatch:         2,
		ECheckpointApprovalRequired: 4,
		EAcceptMissingArtifact:      5,
		EAcceptTestFail:             5,
		EInvalidInputSchema:         6,
		EUnsafeOperation:            8,
		EAdapterFail:                1,
		EInvalidStateTransition:     1,
	}

	for code, expected := range cases {
		if got := ExitCodeFor(code); got != expected {
			t.Fatalf("code %s: expected %d, got %d", code, expected, got)
		}
	}
}

func TestErrorEnvelopeGolden(t *testing.T) {
	fixed := time.Date(2026, 2, 13, 0, 0, 0, 0, time.UTC)

	goldens := []struct {
		name string
		err  error
		file string
	}{
		{
			name: "invalid-input",
			err:  New(EInvalidInputSchema, "invalid jobspec", map[string]any{"field": "name"}),
			file: "invalid_input.json",
		},
		{
			name: "verify-hash-mismatch",
			err:  New(EVerifyHashMismatch, "manifest hash mismatch", map[string]any{"path": "manifest.json"}),
			file: "verify_hash_mismatch.json",
		},
	}

	for _, tc := range goldens {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			actual, err := MarshalEnvelope(tc.err, "dev", fixed)
			if err != nil {
				t.Fatalf("marshal envelope: %v", err)
			}

			expected, err := os.ReadFile(filepath.Join("testdata", "golden", tc.file))
			if err != nil {
				t.Fatalf("read golden %s: %v", tc.file, err)
			}

			if strings.TrimSpace(string(actual)) != strings.TrimSpace(string(expected)) {
				t.Fatalf("golden mismatch for %s\nexpected:\n%s\nactual:\n%s", tc.name, expected, actual)
			}
		})
	}
}
