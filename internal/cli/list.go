package cli

import (
	"fmt"
	"os"
	"sort"
	"text/tabwriter"

	"ghsw/internal/account"
	"ghsw/internal/ghauth"
)

func cmdList(home string, _ []string) error {
	reg, err := account.Load(account.DefaultPath(home))
	if err != nil {
		return fmt.Errorf("loading account registry: %w", err)
	}
	if len(reg.Accounts) == 0 {
		fmt.Println("no accounts registered yet; see `ghsw add`")
		return nil
	}

	statusOut, _ := ghauth.Status()
	activeUser := ghauth.ActiveUsername(statusOut, defaultHostname)

	names := make([]string, 0, len(reg.Accounts))
	for n := range reg.Accounts {
		names = append(names, n)
	}
	sort.Strings(names)

	w := tabwriter.NewWriter(os.Stdout, 0, 2, 2, ' ', 0)
	fmt.Fprintln(w, "NAME\tGH USER\tGIT EMAIL\tGIT DIR\tACTIVE")
	for _, n := range names {
		a := reg.Accounts[n]
		active := ""
		if activeUser != "" && a.GHUsername == activeUser {
			active = "*"
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", n, a.GHUsername, a.GitEmail, a.GitDir, active)
	}
	return w.Flush()
}
