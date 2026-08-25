package cli

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"ghsw/internal/account"
	"ghsw/internal/fsutil"
	"ghsw/internal/gitconfig"
	"ghsw/internal/sshconfig"
)

func cmdRemove(home string, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("remove requires an account name, e.g. ghsw remove work")
	}
	name := args[0]

	fs := flag.NewFlagSet("remove", flag.ContinueOnError)
	purge := fs.Bool("purge", false, "also remove SSH config, gitconfig include, and the per-account gitconfig file")
	if err := fs.Parse(args[1:]); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return fmt.Errorf("unexpected arguments: %v", fs.Args())
	}

	registryPath := account.DefaultPath(home)
	reg, err := account.Load(registryPath)
	if err != nil {
		return fmt.Errorf("loading account registry: %w", err)
	}
	acc, ok := reg.Accounts[name]
	if !ok {
		return fmt.Errorf("no account named %q registered; run `ghsw list` to see registered accounts", name)
	}
	delete(reg.Accounts, name)
	if err := reg.Save(registryPath); err != nil {
		return fmt.Errorf("saving account registry: %w", err)
	}
	fmt.Printf("removed %q from the account registry\n", name)

	perAccountPath := gitconfig.PerAccountPath(home, name)
	if !*purge {
		fmt.Printf("SSH config, gitconfig include, and %s left untouched; rerun with --purge to remove them\n", perAccountPath)
		return nil
	}

	sshConfigPath := filepath.Join(home, ".ssh", "config")
	if err := sshconfig.RemoveHost(sshConfigPath, acc.SSHAlias); err != nil {
		fmt.Fprintf(os.Stderr, "warning: failed to remove SSH host block from %s: %v\n", sshConfigPath, err)
	} else {
		fmt.Printf("removed SSH host alias %q from %s\n", acc.SSHAlias, sshConfigPath)
	}

	if acc.GitDir != "" {
		gitConfigPath := filepath.Join(home, ".gitconfig")
		content, err := fsutil.ReadOrEmpty(gitConfigPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "warning: failed to read %s: %v\n", gitConfigPath, err)
		} else if newContent := gitconfig.RemoveIncludeIf(content, acc.GitDir); newContent != content {
			if err := os.WriteFile(gitConfigPath, []byte(newContent), 0o644); err != nil {
				fmt.Fprintf(os.Stderr, "warning: failed to write %s: %v\n", gitConfigPath, err)
			} else {
				fmt.Printf("removed includeIf block for %s from %s\n", acc.GitDir, gitConfigPath)
			}
		}
	}

	if err := os.Remove(perAccountPath); err != nil && !os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "warning: failed to remove %s: %v\n", perAccountPath, err)
	} else {
		fmt.Printf("removed %s\n", perAccountPath)
	}

	return nil
}
