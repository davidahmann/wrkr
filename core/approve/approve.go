package approve

import (
	"os"
	"strings"

	wrkrerrors "github.com/davidahmann/wrkr/core/errors"
)

func ResolveApprovedBy(explicit string) string {
	v := strings.TrimSpace(explicit)
	if v != "" {
		return v
	}
	if env := strings.TrimSpace(os.Getenv("WRKR_APPROVED_BY")); env != "" {
		return env
	}
	if user := strings.TrimSpace(os.Getenv("USER")); user != "" {
		return user
	}
	return "unknown"
}

func ValidateReason(reason string) error {
	if strings.TrimSpace(reason) == "" {
		return wrkrerrors.New(wrkrerrors.EInvalidInputSchema, "approval reason is required", nil)
	}
	return nil
}
