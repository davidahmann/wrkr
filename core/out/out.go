package out

import (
	"fmt"
	"os"
	"path/filepath"
)

const defaultRoot = "./wrkr-out"

type Layout struct {
	root string
}

func NewLayout(explicit string) Layout {
	root := explicit
	if root == "" {
		root = defaultRoot
	}
	return Layout{root: filepath.Clean(root)}
}

func (l Layout) Root() string {
	return l.root
}

func (l Layout) JobpackPath(jobID string) string {
	return filepath.Join(l.root, "jobpacks", fmt.Sprintf("jobpack_%s.zip", jobID))
}

func (l Layout) IntegrationPath(lane, name string) string {
	return filepath.Join(l.root, "integrations", lane, name)
}

func (l Layout) ReportPath(name string) string {
	return filepath.Join(l.root, "reports", name)
}

func (l Layout) Ensure() error {
	for _, dir := range []string{
		filepath.Join(l.root, "jobpacks"),
		filepath.Join(l.root, "integrations"),
		filepath.Join(l.root, "reports"),
	} {
		if err := os.MkdirAll(dir, 0o750); err != nil {
			return fmt.Errorf("create output dir %s: %w", dir, err)
		}
	}
	return nil
}
