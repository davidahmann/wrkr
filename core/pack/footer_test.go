package pack

import "testing"

func TestTicketFooterRoundTrip(t *testing.T) {
	t.Parallel()

	hash := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	line := TicketFooter("job_1", hash)
	got, err := ParseTicketFooter(line)
	if err != nil {
		t.Fatalf("parse footer: %v", err)
	}
	if got.JobID != "job_1" || got.ManifestSHA256 != hash || got.VerifyTarget != "job_1" {
		t.Fatalf("unexpected parsed footer: %+v", got)
	}
}

func TestParseTicketFooterRejectsInvalid(t *testing.T) {
	t.Parallel()

	if _, err := ParseTicketFooter("not-a-footer"); err == nil {
		t.Fatal("expected invalid footer parse to fail")
	}
}
