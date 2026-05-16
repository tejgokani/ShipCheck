package main

import "github.com/tejgokani/shipcheck/cmd"

// version is set at build time via -ldflags.
var version = "dev"

func main() {
	cmd.Execute(version)
}
