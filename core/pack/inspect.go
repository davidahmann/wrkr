package pack

import (
	"fmt"
	"sort"
	"time"
)

type TimelineItem struct {
	At      time.Time `json:"at"`
	Kind    string    `json:"kind"`
	ID      string    `json:"id"`
	Summary string    `json:"summary"`
}

type InspectResult struct {
	JobID           string         `json:"job_id"`
	Status          string         `json:"status"`
	EventCount      int            `json:"event_count"`
	CheckpointCount int            `json:"checkpoint_count"`
	ApprovalCount   int            `json:"approval_count"`
	Timeline        []TimelineItem `json:"timeline"`
}

func InspectJobpack(path string) (InspectResult, error) {
	archive, err := LoadArchive(path)
	if err != nil {
		return InspectResult{}, err
	}

	job, err := DecodeJobRecord(archive.Files)
	if err != nil {
		return InspectResult{}, err
	}
	events, err := DecodeEvents(archive.Files)
	if err != nil {
		return InspectResult{}, err
	}
	checkpoints, err := DecodeCheckpoints(archive.Files)
	if err != nil {
		return InspectResult{}, err
	}
	approvals, err := DecodeApprovals(archive.Files)
	if err != nil {
		return InspectResult{}, err
	}

	timeline := make([]TimelineItem, 0, len(events)+len(checkpoints)+len(approvals))
	for _, evt := range events {
		timeline = append(timeline, TimelineItem{
			At:      evt.CreatedAt.UTC(),
			Kind:    "event",
			ID:      evt.EventID,
			Summary: evt.Type,
		})
	}
	for _, cp := range checkpoints {
		timeline = append(timeline, TimelineItem{
			At:      cp.CreatedAt.UTC(),
			Kind:    "checkpoint",
			ID:      cp.CheckpointID,
			Summary: fmt.Sprintf("%s:%s", cp.Type, cp.Status),
		})
	}
	for i, approval := range approvals {
		timeline = append(timeline, TimelineItem{
			At:      approval.CreatedAt.UTC(),
			Kind:    "approval",
			ID:      fmt.Sprintf("approval_%d", i+1),
			Summary: approval.CheckpointID,
		})
	}

	sort.Slice(timeline, func(i, j int) bool {
		if timeline[i].At.Equal(timeline[j].At) {
			if timeline[i].Kind == timeline[j].Kind {
				return timeline[i].ID < timeline[j].ID
			}
			return timeline[i].Kind < timeline[j].Kind
		}
		return timeline[i].At.Before(timeline[j].At)
	})

	return InspectResult{
		JobID:           job.JobID,
		Status:          job.Status,
		EventCount:      len(events),
		CheckpointCount: len(checkpoints),
		ApprovalCount:   len(approvals),
		Timeline:        timeline,
	}, nil
}
