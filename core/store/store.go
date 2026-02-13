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
const appendLockRetryAttempts = 128

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
	resolvedRoot, err := fsx.NormalizeAbsolutePath(root)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Join(resolvedRoot, "jobs"), 0o750); err != nil {
		return nil, fmt.Errorf("create store root: %w", err)
	}
	return &LocalStore{root: resolvedRoot}, nil
}

func (s *LocalStore) Root() string {
	return s.root
}

func (s *LocalStore) JobDir(jobID string) string {
	return filepath.Join(s.root, "jobs", jobID)
}

func (s *LocalStore) safeJobPath(jobID, leaf string) (string, error) {
	if err := validateJobID(jobID); err != nil {
		return "", err
	}
	if strings.TrimSpace(leaf) == "" {
		return "", fmt.Errorf("empty job leaf path")
	}
	return fsx.ResolveWithinBase(s.JobDir(jobID), leaf)
}

func (s *LocalStore) EnsureJob(jobID string) error {
	if err := validateJobID(jobID); err != nil {
		return err
	}
	jobDir := s.JobDir(jobID)
	jobsRoot := filepath.Join(s.root, "jobs")
	rel, err := filepath.Rel(jobsRoot, jobDir)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		return fmt.Errorf("job directory escapes store root")
	}
	if err := os.MkdirAll(jobDir, 0o750); err != nil {
		return fmt.Errorf("create job dir: %w", err)
	}
	return nil
}

func (s *LocalStore) JobExists(jobID string) (bool, error) {
	if err := validateJobID(jobID); err != nil {
		return false, err
	}
	jobDir := s.JobDir(jobID)
	jobsRoot := filepath.Join(s.root, "jobs")
	rel, err := filepath.Rel(jobsRoot, jobDir)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		return false, fmt.Errorf("job directory escapes store root")
	}
	info, err := os.Stat(jobDir)
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
	lockPath, err := s.safeJobPath(jobID, "append.lock")
	if err != nil {
		return Event{}, err
	}
	jobDir := s.JobDir(jobID)
	lockRel, err := filepath.Rel(jobDir, lockPath)
	if err != nil || lockRel == ".." || strings.HasPrefix(lockRel, ".."+string(os.PathSeparator)) {
		return Event{}, fmt.Errorf("append lock escapes job directory")
	}

	var (
		lock    *fsx.FileLock
		lockErr error
	)
	for attempt := 0; attempt < appendLockRetryAttempts; attempt++ {
		lock, lockErr = fsx.AcquireLockWithOptions(
			lockPath,
			fmt.Sprintf("pid=%d;ts=%d", os.Getpid(), now.UnixNano()),
			fsx.LockOptions{StaleAfter: appendLockStaleAfter},
		)
		if lockErr == nil {
			break
		}
		if errors.Is(lockErr, fsx.ErrLockBusy) {
			time.Sleep(1 * time.Millisecond)
			continue
		}
		return Event{}, lockErr
	}
	if lockErr != nil {
		return Event{}, lockErr
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
	if err := validateJobID(jobID); err != nil {
		return Event{}, err
	}
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

	eventsPath, err := s.safeJobPath(jobID, "events.jsonl")
	if err != nil {
		return Event{}, err
	}
	jobDir := s.JobDir(jobID)
	eventsRel, err := filepath.Rel(jobDir, eventsPath)
	if err != nil || eventsRel == ".." || strings.HasPrefix(eventsRel, ".."+string(os.PathSeparator)) {
		return Event{}, fmt.Errorf("events file escapes job directory")
	}
	jobRoot, err := os.OpenRoot(jobDir)
	if err != nil {
		return Event{}, fmt.Errorf("open job root: %w", err)
	}
	defer func() { _ = jobRoot.Close() }()
	f, err := jobRoot.OpenFile("events.jsonl", os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
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
	path, err := s.safeJobPath(jobID, "events.jsonl")
	if err != nil {
		return nil, err
	}
	jobDir := s.JobDir(jobID)
	eventsRel, err := filepath.Rel(jobDir, path)
	if err != nil || eventsRel == ".." || strings.HasPrefix(eventsRel, ".."+string(os.PathSeparator)) {
		return nil, fmt.Errorf("events file escapes job directory")
	}
	jobRoot, err := os.OpenRoot(jobDir)
	if err != nil {
		return nil, fmt.Errorf("open job root: %w", err)
	}
	defer func() { _ = jobRoot.Close() }()
	f, err := jobRoot.Open("events.jsonl")
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

	snapshotPath, err := s.safeJobPath(jobID, "snapshot.json")
	if err != nil {
		return err
	}
	if err := fsx.AtomicWriteFile(snapshotPath, raw, 0o600); err != nil {
		return fmt.Errorf("write snapshot: %w", err)
	}
	return nil
}

func (s *LocalStore) LoadSnapshot(jobID string) (*Snapshot, error) {
	if err := validateJobID(jobID); err != nil {
		return nil, err
	}
	snapshotPath, err := s.safeJobPath(jobID, "snapshot.json")
	if err != nil {
		return nil, err
	}
	jobDir := s.JobDir(jobID)
	snapshotRel, err := filepath.Rel(jobDir, snapshotPath)
	if err != nil || snapshotRel == ".." || strings.HasPrefix(snapshotRel, ".."+string(os.PathSeparator)) {
		return nil, fmt.Errorf("snapshot file escapes job directory")
	}
	jobRoot, err := os.OpenRoot(jobDir)
	if err != nil {
		return nil, fmt.Errorf("open job root: %w", err)
	}
	defer func() { _ = jobRoot.Close() }()
	raw, err := jobRoot.ReadFile("snapshot.json")
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
	jobID = strings.TrimSpace(jobID)
	if jobID == "" {
		return fmt.Errorf("empty job_id")
	}
	if strings.Contains(jobID, "..") || strings.Contains(jobID, "/") || strings.Contains(jobID, "\\") {
		return fmt.Errorf("invalid job_id %q: path separators and '..' are not allowed", jobID)
	}
	if filepath.Clean(jobID) != jobID {
		return fmt.Errorf("invalid job_id %q: must be a single path component", jobID)
	}
	if !jobIDPattern.MatchString(jobID) {
		return fmt.Errorf("invalid job_id %q: only [a-zA-Z0-9._-] allowed", jobID)
	}
	return nil
}
