/*
Copyright © 2025
*/
package main

import "github.com/indietool/cli/cmd/indietool/cmd"

// version is set by ldflags during build
var version = "dev"

func main() {
	cmd.SetVersion(version)
	cmd.Execute()
}
