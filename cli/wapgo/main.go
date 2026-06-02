// Package main is the entrypoint for the wapgo CLI.
// Provides project scaffolding (wapgo new) and code generators (wapgo make:*).
package main

import "github.com/abdullahPrasetio/wapgo/cli/commands"

func main() {
	commands.Execute()
}
