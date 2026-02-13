package report

import (
	"encoding/xml"
	"fmt"
	"path/filepath"

	"github.com/davidahmann/wrkr/core/accept/checks"
	"github.com/davidahmann/wrkr/core/fsx"
)

type junitTestSuite struct {
	XMLName  xml.Name        `xml:"testsuite"`
	Name     string          `xml:"name,attr"`
	Tests    int             `xml:"tests,attr"`
	Failures int             `xml:"failures,attr"`
	Cases    []junitTestCase `xml:"testcase"`
}

type junitTestCase struct {
	ClassName string        `xml:"classname,attr"`
	Name      string        `xml:"name,attr"`
	Failure   *junitFailure `xml:"failure,omitempty"`
}

type junitFailure struct {
	Message string `xml:"message,attr"`
	Body    string `xml:",chardata"`
}

func WriteJUnit(path string, checkResults []checks.CheckResult) error {
	if path == "" {
		return fmt.Errorf("empty junit path")
	}

	cases := make([]junitTestCase, 0, len(checkResults))
	failures := 0
	for _, check := range checkResults {
		tc := junitTestCase{ClassName: "wrkr.accept", Name: check.Name}
		if !check.Passed {
			failures++
			tc.Failure = &junitFailure{
				Message: check.Message,
				Body:    string(check.ReasonCode),
			}
		}
		cases = append(cases, tc)
	}

	suite := junitTestSuite{
		Name:     "wrkr-accept",
		Tests:    len(checkResults),
		Failures: failures,
		Cases:    cases,
	}

	raw, err := xml.MarshalIndent(suite, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal junit: %w", err)
	}
	content := append([]byte(xml.Header), raw...)
	content = append(content, '\n')

	if err := fsx.AtomicWriteFile(filepath.Clean(path), content, 0o600); err != nil {
		return fmt.Errorf("write junit: %w", err)
	}
	return nil
}
