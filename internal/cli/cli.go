// Package cli implements ghsw's command-line commands: add, use, list,
// status, and remove.
package cli

import (
	"fmt"
	"os"
)

const (
	sshHostPrefix   = "github-"
	defaultHostname = "github.com"
)

// Execute runs ghsw against os.Args, exiting the process on error.
func Execute() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintln(os.Stderr, "ghsw: cannot determine home directory:", err)
		os.Exit(1)
	}

	cmd := os.Args[1]
	args := os.Args[2:]

	switch cmd {
	case "add":
		err = cmdAdd(home, args)
	case "use":
		err = cmdUse(home, args)
	case "list":
		err = cmdList(home, args)
	case "status":
		err = cmdStatus(home, args)
	case "remove":
		err = cmdRemove(home, args)
	case "-h", "--help", "help":
		printUsage()
		return
	default:
		fmt.Fprintf(os.Stderr, "ghsw: unknown command %q\n\n", cmd)
		printUsage()
		os.Exit(1)
	}

	if err != nil {
		fmt.Fprintln(os.Stderr, "ghsw:", err)
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Fprint(os.Stderr, `ghsw - manage multiple GitHub identities

Usage:
  ghsw add <name> --key <ssh-key> --dir <git-folder> --name <git-name> --email <git-email> --gh-user <github-username>
  ghsw use <name>
  ghsw list
  ghsw status
  ghsw remove <name> [--purge]

See README.md for the full command reference.
`)
}
