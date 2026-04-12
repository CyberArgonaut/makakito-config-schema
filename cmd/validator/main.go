package main

import (
	"fmt"
	"os"

	"github.com/CyberArgonaut/makakito-config-schema/generated/go/config"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintf(os.Stderr, "usage: validator <path/to/config.json>\n")
		os.Exit(1)
	}

	path := os.Args[1]
	data, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: cannot read %s: %v\n", path, err)
		os.Exit(1)
	}

	violations, err := config.Validate(data)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: validation engine failure: %v\n", err)
		os.Exit(1)
	}

	if len(violations) > 0 {
		fmt.Fprintf(os.Stderr, "%s is invalid (schema v%s):\n", path, config.SchemaVersion)
		for _, v := range violations {
			fmt.Fprintln(os.Stderr, v)
		}
		os.Exit(1)
	}

	fmt.Printf("%s is valid (schema v%s)\n", path, config.SchemaVersion)
}
