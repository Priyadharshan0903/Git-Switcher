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

func cmdAdd(home string, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("add requires an account name, e.g. ghsw add work --key ... --dir ... --name ... --email ... --gh-user ...")
	}
	name := args[0]

	fs := flag.NewFlagSet("add", flag.ContinueOnError)
	key := fs.String("key", "", "path to SSH private key")
	dir := fs.String("dir", "", "optional: git working directory this identity should always apply to, even when a different account is active machine-wide")
	gitName := fs.String("name", "", "git user.name")
	gitEmail := fs.String("email", "", "git user.email")
	ghUser := fs.String("gh-user", "", "GitHub username")
	if err := fs.Parse(args[1:]); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return fmt.Errorf("unexpected arguments: %v", fs.Args())
	}
	if *key == "" || *gitName == "" || *gitEmail == "" || *ghUser == "" {
		return fmt.Errorf("add requires --key, --name, --email, and --gh-user (--dir is optional)")
	}

	absKey, err := filepath.Abs(*key)
	if err != nil {
		return fmt.Errorf("resolving --key: %w", err)
	}
	var absDir string
	if *dir != "" {
		absDir, err = filepath.Abs(*dir)
		if err != nil {
			return fmt.Errorf("resolving --dir: %w", err)
		}
	}

	alias := sshHostPrefix + name
	sshConfigPath := filepath.Join(home, ".ssh", "config")
	added, err := sshconfig.AddHost(sshConfigPath, alias, defaultHostname, "git", absKey)
	if err != nil {
		return fmt.Errorf("updating %s: %w", sshConfigPath, err)
	}
	if added {
		fmt.Printf("added SSH host alias %q to %s\n", alias, sshConfigPath)
	} else {
		fmt.Printf("SSH host alias %q already exists, leaving %s as-is\n", alias, sshConfigPath)
	}

	perAccountPath := gitconfig.PerAccountPath(home, name)
	if absDir != "" {
		if err := gitconfig.WritePerAccount(perAccountPath, *gitName, *gitEmail); err != nil {
			return fmt.Errorf("writing %s: %w", perAccountPath, err)
		}
		fmt.Printf("wrote git identity file %s\n", perAccountPath)

		gitConfigPath := filepath.Join(home, ".gitconfig")
		content, err := fsutil.ReadOrEmpty(gitConfigPath)
		if err != nil {
			return fmt.Errorf("reading %s: %w", gitConfigPath, err)
		}
		if gitconfig.HasIncludeIf(content, absDir) {
			fmt.Printf("includeIf for %s already present in %s, leaving as-is\n", absDir, gitConfigPath)
		} else {
			newContent := gitconfig.AddIncludeIf(content, absDir, perAccountPath)
			if err := os.WriteFile(gitConfigPath, []byte(newContent), 0o644); err != nil {
				return fmt.Errorf("writing %s: %w", gitConfigPath, err)
			}
			fmt.Printf("added includeIf block for %s to %s\n", absDir, gitConfigPath)
		}
	}

	registryPath := account.DefaultPath(home)
	reg, err := account.Load(registryPath)
	if err != nil {
		return fmt.Errorf("loading account registry: %w", err)
	}
	reg.Accounts[name] = account.Account{
		SSHAlias:   alias,
		SSHKey:     absKey,
		GitDir:     absDir,
		GitName:    *gitName,
		GitEmail:   *gitEmail,
		GHUsername: *ghUser,
	}
	if err := reg.Save(registryPath); err != nil {
		return fmt.Errorf("saving account registry: %w", err)
	}

	fmt.Printf("\nregistered account %q\n", name)
	fmt.Println("this tool does not run `gh auth login` for you — if this identity")
	fmt.Printf("isn't authenticated yet, run:\n  gh auth login --hostname %s\n", defaultHostname)
	fmt.Printf("then switch to it machine-wide any time with:\n  ghsw use %s\n", name)
	return nil
}
