package main

import (
	"bytes"
	"encoding/json"
	"testing"
	"time"
)

func TestRunServeJSONOutputRemainsParseable(t *testing.T) {
	var out bytes.Buffer
	var errBuf bytes.Buffer
	fixed := func() time.Time { return time.Date(2026, 2, 14, 3, 0, 0, 0, time.UTC) }

	code := run([]string{"serve", "--listen", "127.0.0.1:-1", "--json"}, &out, &errBuf, fixed)
	if code == 0 {
		t.Fatal("expected serve to fail for invalid listen address")
	}

	var decoded map[string]any
	if err := json.Unmarshal(out.Bytes(), &decoded); err != nil {
		t.Fatalf("expected parseable json output, got error: %v (stdout=%q)", err, out.String())
	}
	if _, ok := decoded["listen"]; !ok {
		t.Fatalf("expected listen field in json output, got %v", decoded)
	}
}
