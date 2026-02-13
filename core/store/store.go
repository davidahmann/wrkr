package store

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/davidahmann/wrkr/core/fsx"
)

type Event struct {
	Seq       int64           `json:"seq"`
	CreatedAt time.Time       `json:"created_at"`
	Type      string          `json:"type"`
	Payload   json.RawMessage `json:"payload,omitempty"`
}

type Snapshot struct {
	LastSeq   int64           `json:"last_seq"`
	CreatedAt time.Time       `json:"created_at"`
	State     json.RawMessage `json:"state"`
}

type LocalStore struct {
	root string
}

var jobIDPattern = regexp.MustCompile(`^[a-zA-Z0-9._-]+$`)
var ErrCASConflict = errors.New("event append conflict")

const appendLockStaleAfter = 2 * time.Minute

func DefaultRoot() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home dir: %w", err)
	}
	return filepath.Join(home, ".wrkr"), nil
}

func New(root string) (*LocalStore, error) {
	if strings.TrimSpace(root) == "" {
		var err error
		root, err = DefaultRoot()
		if err != nil {
			return nil, err
		}
	}
	if err := os.MkdirAll(filepath.Join(root, "jobs"), 0o750); err != nil {
		return nil, fmt.Errorf("create store root: %w", err)
	}
	return &LocalStore{root: root}, nil
}

func (s *LocalStore) Root() string {
	return s.root
}

func (s *LocalStore) JobDir(jobID string) string {
	return filepath.Join(s.root, "jobs", jobID)
}

func (s *LocalStore) eventsPath(jobID string) string {
	return filepath.Join(s.JobDir(jobID), "events.jsonl")
}

func (s *LocalStore) snapshotPath(jobID string) string {
	return filepath.Join(s.JobDir(jobID), "snapshot.json")
}

func (s *LocalStore) appendLockPath(jobID string) string {
	return filepath.Join(s.JobDir(jobID), "append.lock")
}

func (s *LocalStore) EnsureJob(jobID string) error {
	if err := validateJobID(jobID); err != nil {
		return err
	}
	if err := os.MkdirAll(s.JobDir(jobID), 0o750); err != nil {
		return fmt.Errorf("create job dir: %w", err)
	}
	return nil
}

func (s *LocalStore) JobExists(jobID string) (bool, error) {
	if err := validateJobID(jobID); err != nil {
		return false, err
	}
	info, err := os.Stat(s.JobDir(jobID))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}
		return false, fmt.Errorf("stat job dir: %w", err)
	}
	return info.IsDir(), nil
}

func (s *LocalStore) AppendEvent(jobID, eventType string, payload any, now time.Time) (Event, error) {
	return s.appendEvent(jobID, eventType, payload, now, nil)
}

func (s *LocalStore) AppendEventCAS(jobID, eventType string, payload any, expectedLastSeq int64, now time.Time) (Event, error) {
	return s.appendEvent(jobID, eventType, payload, now, &expectedLastSeq)
}

func (s *LocalStore) appendEvent(jobID, eventType string, payload any, now time.Time, expectedLastSeq *int64) (Event, error) {
	if err := s.EnsureJob(jobID); err != nil {
		return Event{}, err
	}

	lock, err := fsx.AcquireLockWithOptions(
		s.appendLockPath(jobID),
		strconv.FormatInt(now.UnixNano(), 10),
		fsx.LockOptions{StaleAfter: appendLockStaleAfter},
	)
	if err != nil {
		return Event{}, err
	}
	defer func() { _ = lock.Release() }()

	events, err := s.LoadEvents(jobID)
	if err != nil {
		return Event{}, err
	}

	currentLastSeq := int64(0)
	if len(events) > 0 {
		currentLastSeq = events[len(events)-1].Seq
	}
	if expectedLastSeq != nil && currentLastSeq != *expectedLastSeq {
		return Event{}, ErrCASConflict
	}

	return s.appendEventLocked(jobID, eventType, payload, now, currentLastSeq+1)
}

func (s *LocalStore) appendEventLocked(jobID, eventType string, payload any, now time.Time, seq int64) (Event, error) {
	if seq <= 0 {
		seq = 1
	}

	var raw json.RawMessage
	if payload != nil {
		buf, err := json.Marshal(payload)
		if err != nil {
			return Event{}, fmt.Errorf("marshal event payload: %w", err)
		}
		raw = buf
	}

	event := Event{
		Seq:       seq,
		CreatedAt: now.UTC(),
		Type:      eventType,
		Payload:   raw,
	}

	buf, err := json.Marshal(event)
	if err != nil {
		return Event{}, fmt.Errorf("marshal event: %w", err)
	}

	// #nosec G304 -- eventsPath is store-scoped and job_id-validated.
	f, err := os.OpenFile(s.eventsPath(jobID), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return Event{}, fmt.Errorf("open events file: %w", err)
	}
	defer func() { _ = f.Close() }()

	if _, err := f.Write(append(buf, '\n')); err != nil {
		return Event{}, fmt.Errorf("append event: %w", err)
	}
	if err := f.Sync(); err != nil {
		return Event{}, fmt.Errorf("sync events file: %w", err)
	}

	return event, nil
}

func (s *LocalStore) LoadEvents(jobID string) ([]Event, error) {
	if err := validateJobID(jobID); err != nil {
		return nil, err
	}
	path := s.eventsPath(jobID)
	// #nosec G304 -- path is store-scoped and job_id-validated.
	f, err := os.Open(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("open events file: %w", err)
	}
	defer func() { _ = f.Close() }()

	reader := bufio.NewReader(f)
	events := make([]Event, 0, 32)

	for {
		line, err := reader.ReadBytes('\n')
		if len(line) == 0 && errors.Is(err, io.EOF) {
			break
		}

		if !bytes.HasSuffix(line, []byte("\n")) {
			// Ignore trailing partial line (e.g. process crash during append).
			break
		}

		line = bytes.TrimSpace(line)
		if len(line) == 0 {
			if errors.Is(err, io.EOF) {
				break
			}
			continue
		}

		var event Event
		if uErr := json.Unmarshal(line, &event); uErr != nil {
			return nil, fmt.Errorf("decode event line: %w", uErr)
		}
		events = append(events, event)

		if errors.Is(err, io.EOF) {
			break
		}
	}

	sort.Slice(events, func(i, j int) bool { return events[i].Seq < events[j].Seq })
	return events, nil
}

func (s *LocalStore) SaveSnapshot(jobID string, lastSeq int64, state any, now time.Time) error {
	if err := s.EnsureJob(jobID); err != nil {
		return err
	}

	buf, err := json.Marshal(state)
	if err != nil {
		return fmt.Errorf("marshal snapshot state: %w", err)
	}

	snap := Snapshot{
		LastSeq:   lastSeq,
		CreatedAt: now.UTC(),
		State:     buf,
	}

	raw, err := json.MarshalIndent(snap, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal snapshot: %w", err)
	}

	if err := fsx.AtomicWriteFile(s.snapshotPath(jobID), raw, 0o600); err != nil {
		return fmt.Errorf("write snapshot: %w", err)
	}
	return nil
}

func (s *LocalStore) LoadSnapshot(jobID string) (*Snapshot, error) {
	if err := validateJobID(jobID); err != nil {
		return nil, err
	}
	raw, err := os.ReadFile(s.snapshotPath(jobID))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("read snapshot: %w", err)
	}

	var snap Snapshot
	if err := json.Unmarshal(raw, &snap); err != nil {
		return nil, fmt.Errorf("decode snapshot: %w", err)
	}
	return &snap, nil
}

func validateJobID(jobID string) error {
	if strings.TrimSpace(jobID) == "" {
		return fmt.Errorf("empty job_id")
	}
	if !jobIDPattern.MatchString(jobID) {
		return fmt.Errorf("invalid job_id %q: only [a-zA-Z0-9._-] allowed", jobID)
	}
	return nil
}
