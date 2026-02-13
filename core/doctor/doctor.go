package doctor

import (
	"os"
	"path/filepath"
	"time"

	"github.com/davidahmann/wrkr/core/out"
	"github.com/davidahmann/wrkr/core/schema/validate"
	"github.com/davidahmann/wrkr/core/store"
)

type CheckResult struct {
	Name    string `json:"name"`
	OK      bool   `json:"ok"`
	Details string `json:"details,omitempty"`
}

type Result struct {
	CheckedAt time.Time     `json:"checked_at"`
	OK        bool          `json:"ok"`
	Checks    []CheckResult `json:"checks"`
}

func Run(now func() time.Time) (Result, error) {
	if now == nil {
		now = time.Now
	}
	results := make([]CheckResult, 0, 4)

	s, err := store.New("")
	if err != nil {
		return Result{}, err
	}
	results = append(results, CheckResult{
		Name:    "store_root",
		OK:      true,
		Details: s.Root(),
	})

	layout := out.NewLayout("")
	if err := layout.Ensure(); err != nil {
		results = append(results, CheckResult{Name: "output_layout", OK: false, Details: err.Error()})
	} else {
		results = append(results, CheckResult{Name: "output_layout", OK: true, Details: layout.Root()})
	}

	missing := 0
	for _, rel := range validate.SchemaList() {
		if _, err := validate.SchemaPath(rel); err != nil {
			missing++
		}
	}
	results = append(results, CheckResult{
		Name:    "schemas",
		OK:      missing == 0,
		Details: "missing=" + itoa(missing),
	})

	hookPath := filepath.Clean(".githooks/pre-push")
	if info, err := os.Stat(hookPath); err == nil && !info.IsDir() {
		results = append(results, CheckResult{Name: "git_hook_pre_push", OK: true, Details: hookPath})
	} else {
		results = append(results, CheckResult{Name: "git_hook_pre_push", OK: false, Details: hookPath})
	}

	ok := true
	for _, check := range results {
		if !check.OK {
			ok = false
			break
		}
	}
	return Result{
		CheckedAt: now().UTC(),
		OK:        ok,
		Checks:    results,
	}, nil
}

func itoa(v int) string {
	if v == 0 {
		return "0"
	}
	if v < 0 {
		v = -v
	}
	digits := []byte{}
	for v > 0 {
		digits = append([]byte{byte('0' + (v % 10))}, digits...)
		v /= 10
	}
	return string(digits)
}
