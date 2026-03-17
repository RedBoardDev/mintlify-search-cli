package main

import (
	"os"

	"github.com/redboard/mintlify-search-cli/internal/cli"
)

func main() {
	cmd := cli.NewRootCmd()
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
