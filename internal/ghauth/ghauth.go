// Package ghauth wraps the parts of the gh CLI ghsw needs: switching the
// active authenticated account and reading which one is currently active.
package ghauth

import (
	"os/exec"
	"regexp"
	"strings"
)

// Switch shells out to `gh auth switch`. Not unit-tested: it depends on a
// real, authenticated gh CLI installation.
func Switch(hostname, username string) (string, error) {
	cmd := exec.Command("gh", "auth", "switch", "--hostname", hostname, "--user", username)
	out, err := cmd.CombinedOutput()
	return string(out), err
}

// Status shells out to `gh auth status`. Not unit-tested: it depends on a
// real, authenticated gh CLI installation. `gh auth status` exits non-zero
// whenever any configured host isn't logged in, so callers should still use
// its output even when err != nil.
func Status() (string, error) {
	cmd := exec.Command("gh", "auth", "status")
	out, err := cmd.CombinedOutput()
	return string(out), err
}

var loggedInRe = regexp.MustCompile(`Logged in to (\S+) account (\S+)`)

// ActiveUsername scans `gh auth status` output for the account marked active
// on hostname, returning its username, or "" if none is found. This is pure
// string processing and is unit-tested against captured sample output.
func ActiveUsername(output, hostname string) string {
	currentHost := ""
	currentUser := ""
	for _, line := range strings.Split(output, "\n") {
		trimmed := strings.TrimSpace(line)
		if m := loggedInRe.FindStringSubmatch(trimmed); m != nil {
			currentHost = m[1]
			currentUser = m[2]
			continue
		}
		if currentHost == hostname && strings.Contains(trimmed, "Active account: true") {
			return currentUser
		}
	}
	return ""
}
