package v1

import "time"

// Envelope carries required producer metadata for persisted contracts.
type Envelope struct {
	SchemaID        string    `json:"schema_id"`
	SchemaVersion   string    `json:"schema_version"`
	CreatedAt       time.Time `json:"created_at"`
	ProducerVersion string    `json:"producer_version"`
}

type AdapterConfig struct {
	Name   string         `json:"name"`
	Config map[string]any `json:"config,omitempty"`
}

type BudgetSpec struct {
	MaxWallTimeSeconds int      `json:"max_wall_time_seconds"`
	MaxRetries         int      `json:"max_retries"`
	MaxStepCount       int      `json:"max_step_count"`
	MaxToolCalls       int      `json:"max_tool_calls"`
	MaxEstimatedCost   *float64 `json:"max_estimated_cost,omitempty"`
	MaxTokens          *int     `json:"max_tokens,omitempty"`
}

type CheckpointPolicy struct {
	MinIntervalSeconds int      `json:"min_interval_seconds"`
	RequiredTypes      []string `json:"required_types"`
}

type EnvironmentFingerprint struct {
	Rules []string `json:"rules"`
}

type JobSpec struct {
	Envelope
	Name                   string                 `json:"name"`
	Objective              string                 `json:"objective"`
	Inputs                 map[string]any         `json:"inputs"`
	ExpectedArtifacts      []string               `json:"expected_artifacts"`
	Adapter                AdapterConfig          `json:"adapter"`
	Budgets                BudgetSpec             `json:"budgets"`
	CheckpointPolicy       CheckpointPolicy       `json:"checkpoint_policy"`
	Acceptance             map[string]any         `json:"acceptance,omitempty"`
	EnvironmentFingerprint EnvironmentFingerprint `json:"environment_fingerprint,omitempty"`
}

type BudgetState struct {
	WallTimeSeconds int `json:"wall_time_seconds"`
	RetryCount      int `json:"retry_count"`
	StepCount       int `json:"step_count"`
	ToolCallCount   int `json:"tool_call_count"`
}

type ArtifactsDelta struct {
	Added   []string `json:"added"`
	Changed []string `json:"changed"`
	Removed []string `json:"removed"`
}

type RequiredAction struct {
	Kind         string `json:"kind,omitempty"`
	Instructions string `json:"instructions,omitempty"`
}

type Checkpoint struct {
	Envelope
	CheckpointID   string          `json:"checkpoint_id"`
	JobID          string          `json:"job_id"`
	Type           string          `json:"type"`
	Summary        string          `json:"summary"`
	Status         string          `json:"status"`
	BudgetState    BudgetState     `json:"budget_state"`
	ArtifactsDelta ArtifactsDelta  `json:"artifacts_delta"`
	RequiredAction *RequiredAction `json:"required_action,omitempty"`
	ReasonCodes    []string        `json:"reason_codes"`
}

type ApprovalRecord struct {
	Envelope
	JobID        string `json:"job_id"`
	CheckpointID string `json:"checkpoint_id"`
	Reason       string `json:"reason"`
	ApprovedBy   string `json:"approved_by"`
}

type ManifestFile struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
}

type JobpackManifest struct {
	Envelope
	JobID          string         `json:"job_id"`
	ManifestSHA256 string         `json:"manifest_sha256"`
	Files          []ManifestFile `json:"files"`
}

type JobRecord struct {
	Envelope
	JobID   string         `json:"job_id"`
	Name    string         `json:"name"`
	Status  string         `json:"status"`
	Budgets map[string]any `json:"budgets"`
}

type EventRecord struct {
	Envelope
	EventID  string         `json:"event_id"`
	JobID    string         `json:"job_id"`
	Type     string         `json:"type"`
	Executed bool           `json:"executed"`
	Payload  map[string]any `json:"payload"`
}

type ArtifactRecord struct {
	Path     string `json:"path"`
	SHA256   string `json:"sha256"`
	Redacted bool   `json:"redacted,omitempty"`
}

type ArtifactsManifest struct {
	Envelope
	JobID       string           `json:"job_id"`
	CaptureMode string           `json:"capture_mode"`
	Artifacts   []ArtifactRecord `json:"artifacts"`
}

type AcceptanceFailure struct {
	Check    string `json:"check"`
	Message  string `json:"message"`
	Artifact string `json:"artifact,omitempty"`
}

type AcceptanceResult struct {
	Envelope
	JobID        string              `json:"job_id"`
	ChecksRun    int                 `json:"checks_run"`
	ChecksPassed int                 `json:"checks_passed"`
	Failures     []AcceptanceFailure `json:"failures"`
	ReasonCodes  []string            `json:"reason_codes"`
}

type LeaseInfo struct {
	WorkerID  string    `json:"worker_id"`
	LeaseID   string    `json:"lease_id"`
	ExpiresAt time.Time `json:"expires_at"`
}

type StatusResponse struct {
	Envelope
	JobID              string     `json:"job_id"`
	Status             string     `json:"status"`
	Summary            string     `json:"summary"`
	Lease              *LeaseInfo `json:"lease,omitempty"`
	ReasonCodes        []string   `json:"reason_codes,omitempty"`
	EnvironmentHash    string     `json:"environment_hash,omitempty"`
	EnvironmentRuleSet []string   `json:"environment_rules,omitempty"`
}

type WorkItemPayload struct {
	Envelope
	JobID            string   `json:"job_id"`
	CheckpointID     string   `json:"checkpoint_id"`
	CheckpointType   string   `json:"checkpoint_type"`
	RequiredAction   string   `json:"required_action"`
	ReasonCodes      []string `json:"reason_codes"`
	ArtifactPointers []string `json:"artifact_pointers,omitempty"`
	NextCommands     []string `json:"next_commands"`
}

type GitHubSummaryAcceptance struct {
	ChecksRun    int  `json:"checks_run"`
	ChecksPassed int  `json:"checks_passed"`
	Failed       bool `json:"failed"`
}

type GitHubSummaryArtifactDelta struct {
	Added   int `json:"added"`
	Changed int `json:"changed"`
	Removed int `json:"removed"`
}

type GitHubSummary struct {
	Envelope
	JobID         string                     `json:"job_id"`
	Status        string                     `json:"status"`
	Acceptance    GitHubSummaryAcceptance    `json:"acceptance"`
	ArtifactDelta GitHubSummaryArtifactDelta `json:"artifact_delta"`
	Markdown      string                     `json:"markdown"`
}

type ErrorEnvelope struct {
	Envelope
	Code     string         `json:"code"`
	Message  string         `json:"message"`
	ExitCode int            `json:"exit_code"`
	Details  map[string]any `json:"details,omitempty"`
}
