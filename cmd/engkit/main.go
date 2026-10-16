package main

import (
	"os"

	"github.com/SummitExplorerJD/Engineering-Toolkit/internal/cli"
)

func main() {
	os.Exit(cli.Run(os.Args[1:], os.Stdout, os.Stderr))
}
