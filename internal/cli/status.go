package cli

import (
	"fmt"
	"os"

	"ghsw/internal/account"
	"ghsw/internal/ghauth"
	"ghsw/internal/gitconfig"
)

func cmdStatus(home string, _ []string) error {
	out, err := ghauth.Status()
	if out != "" {
		fmt.Print(out)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "note: `gh auth status` exited non-zero (normal if you're not logged into every host)")
	}

	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getting working directory: %w", err)
	}
	email, err := gitconfig.EmailFor(cwd)
	if err != nil {
		fmt.Println("could not determine git user.email for the current directory:", err)
		return nil
	}
	if email == "" {
		fmt.Println("\nno git user.email configured for the current directory")
		return nil
	}

	reg, err := account.Load(account.DefaultPath(home))
	if err != nil {
		return fmt.Errorf("loading account registry: %w", err)
	}
	for name, a := range reg.Accounts {
		if a.GitEmail == email {
			fmt.Printf("\ncurrent directory (%s) maps to registered account %q via git user.email %s\n", cwd, name, email)
			return nil
		}
	}
	fmt.Printf("\ncurrent directory (%s) has git user.email %q, which doesn't match any registered ghsw account\n", cwd, email)
	return nil
}
