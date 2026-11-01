package main

import (
	"flag"
)

// parseInterspersed parses fs over args, allowing flags before and after the
// positional arguments (the standard library stops at the first positional).
// It returns the positionals in order.
func parseInterspersed(fs *flag.FlagSet, args []string) ([]string, error) {
	var positional []string

	for {
		if err := fs.Parse(args); err != nil {
			return nil, err //nolint:wrapcheck // the caller turns it into a usage error
		}

		rest := fs.Args()
		if len(rest) == 0 {
			return positional, nil
		}

		positional = append(positional, rest[0])
		args = rest[1:]
	}
}
