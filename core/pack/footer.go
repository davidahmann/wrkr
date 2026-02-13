package pack

import (
	"fmt"
	"regexp"

	wrkrerrors "github.com/davidahmann/wrkr/core/errors"
)

var footerPattern = regexp.MustCompile(`^WRKR job_id=([a-zA-Z0-9._-]+) manifest=sha256:([a-f0-9]{64}) verify="wrkr verify ([^"]+)"$`)

func TicketFooter(jobID, manifestSHA256 string) string {
	return fmt.Sprintf(`WRKR job_id=%s manifest=sha256:%s verify="wrkr verify %s"`, jobID, manifestSHA256, jobID)
}

type Footer struct {
	JobID          string
	ManifestSHA256 string
	VerifyTarget   string
}

func ParseTicketFooter(line string) (Footer, error) {
	m := footerPattern.FindStringSubmatch(line)
	if len(m) != 4 {
		return Footer{}, wrkrerrors.New(wrkrerrors.EInvalidInputSchema, "invalid ticket footer format", nil)
	}
	return Footer{
		JobID:          m[1],
		ManifestSHA256: m[2],
		VerifyTarget:   m[3],
	}, nil
}
