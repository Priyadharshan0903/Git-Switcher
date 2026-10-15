package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"ghsw/internal/account"
	"ghsw/internal/ghauth"
	"ghsw/internal/gitconfig"
	"ghsw/internal/sshconfig"
)

// cmdUse switches the active account machine-wide: the gh CLI's active
// account, git's global user.name/user.email (the default for any repo
// without a more specific local or includeIf-matched override), and which
// SSH key answers for plain git@<host> remotes on the account's GitHub host.
// Accounts on other hosts are left alone, so e.g. switching your GitHub
// Enterprise account doesn't change which key answers for github.com.
func cmdUse(home string, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("use requires an account name, e.g. ghsw use work")
	}
	name := args[0]

	reg, err := account.Load(account.DefaultPath(home))
	if err != nil {
		return fmt.Errorf("loading account registry: %w", err)
	}
	acc, ok := reg.Accounts[name]
	if !ok {
		return fmt.Errorf("no account named %q registered; run `ghsw list` to see registered accounts", name)
	}

	var failures []string

	host := acc.Host()
	out, err := ghauth.Switch(host, acc.GHUsername)
	if out != "" {
		fmt.Print(out)
	}
	if err != nil {
		failures = append(failures, fmt.Sprintf("gh auth switch failed: %v", err))
	} else {
		fmt.Printf("gh: active account on %s is now %s\n", host, acc.GHUsername)
	}

	if err := gitconfig.SetGlobalIdentity(acc.GitName, acc.GitEmail); err != nil {
		failures = append(failures, fmt.Sprintf("setting global git identity failed: %v", err))
	} else {
		fmt.Printf("git: global user.name/user.email set to %q <%s>\n", acc.GitName, acc.GitEmail)
	}

	sshConfigPath := filepath.Join(home, ".ssh", "config")
	linkPath := sshconfig.ActiveKeyLinkPath(home, host)
	addedBlock, foreignBlock, err := sshconfig.EnsureManagedDefaultHost(sshConfigPath, host, linkPath)
	if err != nil {
		failures = append(failures, fmt.Sprintf("updating %s failed: %v", sshConfigPath, err))
	} else {
		if addedBlock {
			fmt.Printf("added a managed \"Host %s\" block to %s\n", host, sshConfigPath)
		}
		if foreignBlock {
			fmt.Fprintf(os.Stderr, "warning: %s already has a \"Host %s\" block ghsw doesn't manage.\n", sshConfigPath, host)
			fmt.Fprintf(os.Stderr, "         SSH uses the first value it finds for each setting, so that block may shadow\n")
			fmt.Fprintf(os.Stderr, "         ghsw's IdentityFile. Remove or merge it, keeping IdentityFile pointed at:\n")
			fmt.Fprintf(os.Stderr, "         %s\n", linkPath)
		}
		if err := sshconfig.SetActiveKey(linkPath, acc.SSHKey); err != nil {
			failures = append(failures, fmt.Sprintf("switching active SSH key failed: %v", err))
		} else {
			fmt.Printf("ssh: git@%s now uses %s\n", host, acc.SSHKey)
		}
	}

	if len(failures) > 0 {
		return fmt.Errorf("switched to %q with errors:\n  %s", name, strings.Join(failures, "\n  "))
	}
	fmt.Printf("\nswitched to %q (%s on %s) machine-wide\n", name, acc.GHUsername, host)
	return nil
}
