package main

import "github.com/dpup/pls/cli"

// version is set at build time via -ldflags.
var version = "dev"

func main() {
	cli.Execute(version)
}
