package zipx

import "testing"

func TestBuildDeterministicStableAcrossInputOrder(t *testing.T) {
	t.Parallel()

	a, err := BuildDeterministic([]Entry{
		{Name: "b.txt", Data: []byte("second")},
		{Name: "a.txt", Data: []byte("first")},
	})
	if err != nil {
		t.Fatalf("build zip a: %v", err)
	}
	b, err := BuildDeterministic([]Entry{
		{Name: "a.txt", Data: []byte("first")},
		{Name: "b.txt", Data: []byte("second")},
	})
	if err != nil {
		t.Fatalf("build zip b: %v", err)
	}

	if string(a) != string(b) {
		t.Fatal("expected deterministic zip bytes to match")
	}
}
