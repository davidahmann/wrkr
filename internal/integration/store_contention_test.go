package integration

import (
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/davidahmann/wrkr/core/fsx"
	"github.com/davidahmann/wrkr/core/store"
)

func TestStoreConcurrentAppendNoCorruption(t *testing.T) {
	t.Parallel()

	s, err := store.New(t.TempDir())
	if err != nil {
		t.Fatalf("store.New: %v", err)
	}

	const (
		workers       = 8
		eventsPerWork = 40
	)

	var wg sync.WaitGroup
	errs := make(chan error, workers)

	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func(worker int) {
			defer wg.Done()
			deadline := time.Now().Add(8 * time.Second)
			for i := 0; i < eventsPerWork; i++ {
				ok := false
				for time.Now().Before(deadline) {
					_, err := s.AppendEvent("job_lock", "progress", map[string]any{
						"worker": worker,
						"idx":    i,
					}, time.Now().UTC())
					if err == nil {
						ok = true
						break
					}
					if !errors.Is(err, fsx.ErrLockBusy) {
						errs <- err
						return
					}
					time.Sleep(1 * time.Millisecond)
				}
				if !ok {
					errs <- errors.New("lock starvation exceeded threshold")
					return
				}
			}
		}(w)
	}

	wg.Wait()
	close(errs)

	for err := range errs {
		if err != nil {
			t.Fatalf("append failure: %v", err)
		}
	}

	events, err := s.LoadEvents("job_lock")
	if err != nil {
		t.Fatalf("load events: %v", err)
	}

	expected := workers * eventsPerWork
	if len(events) != expected {
		t.Fatalf("expected %d events, got %d", expected, len(events))
	}

	for i, event := range events {
		expectedSeq := int64(i + 1)
		if event.Seq != expectedSeq {
			t.Fatalf("expected seq %d, got %d", expectedSeq, event.Seq)
		}
	}
}
