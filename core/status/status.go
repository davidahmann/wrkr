package status

import (
	"time"

	"github.com/davidahmann/wrkr/core/runner"
	v1 "github.com/davidahmann/wrkr/core/schema/v1"
)

func FromRunnerState(state *runner.State, producerVersion string, now time.Time) v1.StatusResponse {
	resp := v1.StatusResponse{
		Envelope: v1.Envelope{
			SchemaID:        "wrkr.status_response",
			SchemaVersion:   "v1",
			CreatedAt:       now.UTC(),
			ProducerVersion: producerVersion,
		},
		JobID:   state.JobID,
		Status:  string(state.Status),
		Summary: "status materialized from durable store",
	}

	if state.Lease != nil {
		resp.Lease = &v1.LeaseInfo{
			WorkerID:  state.Lease.WorkerID,
			LeaseID:   state.Lease.LeaseID,
			ExpiresAt: state.Lease.ExpiresAt,
		}
	}

	return resp
}
