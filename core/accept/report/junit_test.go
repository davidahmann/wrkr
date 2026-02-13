package report

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/davidahmann/wrkr/core/accept/checks"
	wrkrerrors "github.com/davidahmann/wrkr/core/errors"
)

func TestWriteJUnit(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "accept.junit.xml")
	err := WriteJUnit(path, []checks.CheckResult{
		{Name: "schema_validity", Passed: true, Message: "ok"},
		{Name: "test_command", Passed: false, Message: "failed", ReasonCode: wrkrerrors.EAcceptTestFail},
	})
	if err != nil {
		t.Fatalf("write junit: %v", err)
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read junit: %v", err)
	}
	text := string(raw)
	if !strings.Contains(text, `<testsuite name="wrkr-accept" tests="2" failures="1">`) {
		t.Fatalf("unexpected testsuite header: %s", text)
	}
	if !strings.Contains(text, `<testcase classname="wrkr.accept" name="test_command">`) {
		t.Fatalf("missing testcase: %s", text)
	}
}
