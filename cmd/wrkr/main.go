package main

import "fmt"

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	fmt.Printf("wrkr %s (commit=%s date=%s)\n", version, commit, date)
}
