package jcs

import "testing"

func TestCanonicalizeStableAcrossEquivalentJSON(t *testing.T) {
	t.Parallel()

	a := []byte(`{"b":2,"a":{"z":true,"k":[3,2,1]}}`)
	b := []byte("{\n  \"a\": {\"k\": [3,2,1], \"z\": true},\n  \"b\": 2\n}")

	ca, err := Canonicalize(a)
	if err != nil {
		t.Fatalf("canonicalize a: %v", err)
	}
	cb, err := Canonicalize(b)
	if err != nil {
		t.Fatalf("canonicalize b: %v", err)
	}

	if string(ca) != string(cb) {
		t.Fatalf("expected equal canonical output\na=%s\nb=%s", ca, cb)
	}
}

func TestCanonicalizeRejectsInvalidJSON(t *testing.T) {
	t.Parallel()

	if _, err := Canonicalize([]byte(`{"x":}`)); err == nil {
		t.Fatal("expected invalid json to fail canonicalization")
	}
}
