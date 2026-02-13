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
	wd := t.TempDir()
	orig, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(wd); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(orig)
	})

	path := filepath.Join(wd, "accept.junit.xml")
	err = WriteJUnit("accept.junit.xml", []checks.CheckResult{
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
