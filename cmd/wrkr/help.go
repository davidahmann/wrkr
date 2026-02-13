package main

import (
	"fmt"
	"io"
)

func runHelp(stdout io.Writer) int {
	_, _ = fmt.Fprintln(stdout, `wrkr command map:
  demo
  init
  submit
  status
  checkpoint list|show|emit
  pause
  resume
  cancel
  approve
  wrap -- <command...>
  export
  verify
  accept init|run
  report github
  bridge work-item
  serve
  job inspect|diff
  doctor [--production-readiness] [--serve-*]
  store prune`)
	return 0
}
