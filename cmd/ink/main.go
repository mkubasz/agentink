// Command ink is the CLI for agents and the owner: ask a question, read the
// inbox, check the daemon. It talks to inkd over its Unix socket and knows
// nothing of pages, devices or the worker (ADR 0007).
package main

import (
	"os"
)

// Version is stamped by the justfile's ldflags.
var Version = "dev"

func main() {
	cwd, _ := os.Getwd()
	os.Exit(run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr, os.Getenv, cwd))
}
