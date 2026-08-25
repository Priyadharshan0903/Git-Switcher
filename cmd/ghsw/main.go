// Command ghsw (GitHub Switch) manages multiple GitHub identities on one
// machine by orchestrating SSH config aliases, git's includeIf conditional
// config, and the gh CLI's native auth switch — see README.md.
package main

import "ghsw/internal/cli"

func main() {
	cli.Execute()
}
