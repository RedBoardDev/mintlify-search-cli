package main

import "github.com/redboard/mintlify-search-cli/internal/cli"

func main() {
	cli.RunAndExit(cli.NewRootCmd())
}
