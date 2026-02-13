package validate

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

const (
	JobspecSchemaRel                = "jobspec/jobspec.schema.json"
	EnvironmentFingerprintSchemaRel = "environment_fingerprint/environment_fingerprint.schema.json"
	LeaseRecordSchemaRel            = "lease/lease.schema.json"
	CheckpointSchemaRel             = "checkpoint/checkpoint.schema.json"
	ApprovalRecordSchemaRel         = "checkpoint/approval_record.schema.json"
	JobpackManifestSchemaRel        = "jobpack/manifest.schema.json"
	JobSchemaRel                    = "jobpack/job.schema.json"
	EventSchemaRel                  = "jobpack/event.schema.json"
	ArtifactsManifestSchemaRel      = "jobpack/artifacts_manifest.schema.json"
	AcceptResultSchemaRel           = "accept/accept_result.schema.json"
	StatusResponseSchemaRel         = "status/status_response.schema.json"
	WorkItemSchemaRel               = "bridge/work_item.schema.json"
	GitHubSummarySchemaRel          = "report/github_summary.schema.json"
	ErrorEnvelopeSchemaRel          = "serve/error_envelope.schema.json"
)

func SchemaList() []string {
	return []string{
		JobspecSchemaRel,
		EnvironmentFingerprintSchemaRel,
		LeaseRecordSchemaRel,
		CheckpointSchemaRel,
		ApprovalRecordSchemaRel,
		JobpackManifestSchemaRel,
		JobSchemaRel,
		EventSchemaRel,
		ArtifactsManifestSchemaRel,
		AcceptResultSchemaRel,
		StatusResponseSchemaRel,
		WorkItemSchemaRel,
		GitHubSummarySchemaRel,
		ErrorEnvelopeSchemaRel,
	}
}

func schemaRoot() (string, error) {
	if explicit := os.Getenv("WRKR_SCHEMA_ROOT"); explicit != "" {
		return explicit, nil
	}

	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		return "", fmt.Errorf("could not locate runtime caller for schema root")
	}

	root := filepath.Clean(filepath.Join(filepath.Dir(thisFile), "..", "..", "..", "schemas", "v1"))
	return root, nil
}

func SchemaPath(rel string) (string, error) {
	root, err := schemaRoot()
	if err != nil {
		return "", err
	}

	path := filepath.Join(root, rel)
	if _, statErr := os.Stat(path); statErr != nil {
		return "", fmt.Errorf("schema not found: %s: %w", path, statErr)
	}

	return path, nil
}
