package main

import "github.com/lazypower/clorch/internal/cli"

var version = "dev"

func main() {
	cli.SetVersion(version)
	cli.Execute()
}
