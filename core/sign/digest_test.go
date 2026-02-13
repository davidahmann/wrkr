package sign

import "testing"

func TestCanonicalJSONSHA256HexStable(t *testing.T) {
	t.Parallel()

	a := []byte(`{"b":2,"a":1}`)
	b := []byte(`{"a":1,"b":2}`)

	da, err := CanonicalJSONSHA256Hex(a)
	if err != nil {
		t.Fatalf("digest a: %v", err)
	}
	db, err := CanonicalJSONSHA256Hex(b)
	if err != nil {
		t.Fatalf("digest b: %v", err)
	}

	if da != db {
		t.Fatalf("expected equal digests, got %s != %s", da, db)
	}
}
