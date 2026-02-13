package errors

import (
	"encoding/json"
	"fmt"
	"time"
)

type Code string

const (
	EGenericFailure             Code = "E_GENERIC_FAILURE"
	EBudgetExceeded             Code = "E_BUDGET_EXCEEDED"
	EAdapterFail                Code = "E_ADAPTER_FAIL"
	ECheckpointApprovalRequired Code = "E_CHECKPOINT_APPROVAL_REQUIRED"
	EAcceptMissingArtifact      Code = "E_ACCEPT_MISSING_ARTIFACT"
	EAcceptTestFail             Code = "E_ACCEPT_TEST_FAIL"
	EVerifyHashMismatch         Code = "E_VERIFY_HASH_MISMATCH"
	EStoreCorrupt               Code = "E_STORE_CORRUPT"
	EEnvFingerprintMismatch     Code = "E_ENV_FINGERPRINT_MISMATCH"
	ELeaseConflict              Code = "E_LEASE_CONFLICT"
	EInvalidInputSchema         Code = "E_INVALID_INPUT_SCHEMA"
	EUnsafeOperation            Code = "E_UNSAFE_OPERATION"
)

type WrkrError struct {
	Code    Code
	Message string
	Details map[string]any
}

func (e WrkrError) Error() string {
	if e.Message == "" {
		return string(e.Code)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func New(code Code, message string, details map[string]any) error {
	return WrkrError{Code: code, Message: message, Details: details}
}

func ExitCodeFor(code Code) int {
	switch code {
	case EVerifyHashMismatch:
		return 2
	case ECheckpointApprovalRequired:
		return 4
	case EAcceptMissingArtifact, EAcceptTestFail:
		return 5
	case EInvalidInputSchema:
		return 6
	case EUnsafeOperation:
		return 8
	default:
		return 1
	}
}

type Envelope struct {
	SchemaID        string         `json:"schema_id"`
	SchemaVersion   string         `json:"schema_version"`
	CreatedAt       time.Time      `json:"created_at"`
	ProducerVersion string         `json:"producer_version"`
	Code            Code           `json:"code"`
	Message         string         `json:"message"`
	ExitCode        int            `json:"exit_code"`
	Details         map[string]any `json:"details,omitempty"`
}

func ToEnvelope(err error, producerVersion string, at time.Time) Envelope {
	werr, ok := err.(WrkrError)
	if !ok {
		return Envelope{
			SchemaID:        "wrkr.error_envelope",
			SchemaVersion:   "v1",
			CreatedAt:       at.UTC(),
			ProducerVersion: producerVersion,
			Code:            EGenericFailure,
			Message:         err.Error(),
			ExitCode:        ExitCodeFor(EGenericFailure),
		}
	}

	return Envelope{
		SchemaID:        "wrkr.error_envelope",
		SchemaVersion:   "v1",
		CreatedAt:       at.UTC(),
		ProducerVersion: producerVersion,
		Code:            werr.Code,
		Message:         werr.Message,
		ExitCode:        ExitCodeFor(werr.Code),
		Details:         werr.Details,
	}
}

func MarshalEnvelope(err error, producerVersion string, at time.Time) ([]byte, error) {
	env := ToEnvelope(err, producerVersion, at)
	out, mErr := json.MarshalIndent(env, "", "  ")
	if mErr != nil {
		return nil, fmt.Errorf("marshal error envelope: %w", mErr)
	}
	return out, nil
}
