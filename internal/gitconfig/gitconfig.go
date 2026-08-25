// Package gitconfig manages the per-account git identity files, the
// includeIf wiring in ~/.gitconfig, and the git config shell-outs ghsw needs
// for reading and setting identity.
package gitconfig

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

// PerAccountPath returns ~/.gitconfig-<name>.
func PerAccountPath(home, name string) string {
	return filepath.Join(home, ".gitconfig-"+name)
}

// userConfigContent renders the contents of a per-account gitconfig file.
func userConfigContent(name, email string) string {
	return fmt.Sprintf("[user]\n  name = %s\n  email = %s\n", name, email)
}

// WritePerAccount writes (or overwrites) the per-account gitconfig file.
func WritePerAccount(path, name, email string) error {
	return os.WriteFile(path, []byte(userConfigContent(name, email)), 0o644)
}

// normalizeDir ensures dir ends in exactly one trailing slash, as required
// by git's includeIf "gitdir:" matching.
func normalizeDir(dir string) string {
	return strings.TrimSuffix(dir, "/") + "/"
}

func includeIfHeader(gitDir string) string {
	return fmt.Sprintf(`[includeIf "gitdir:%s"]`, normalizeDir(gitDir))
}

func includeIfBlock(gitDir, includePath string) string {
	return fmt.Sprintf("\n%s\n  path = %s\n", includeIfHeader(gitDir), includePath)
}

// HasIncludeIf reports whether content already has an includeIf section for
// gitDir.
func HasIncludeIf(content, gitDir string) bool {
	return strings.Contains(content, includeIfHeader(gitDir))
}

// AddIncludeIf appends an includeIf block for gitDir to content, unless one
// is already present, in which case content is returned unchanged.
func AddIncludeIf(content, gitDir, includePath string) string {
	if HasIncludeIf(content, gitDir) {
		return content
	}
	block := includeIfBlock(gitDir, includePath)
	if content == "" {
		return strings.TrimPrefix(block, "\n")
	}
	if !strings.HasSuffix(content, "\n") {
		content += "\n"
	}
	return content + block
}

// RemoveIncludeIf strips the includeIf section for gitDir (header plus its
// indented body lines) from content, leaving everything else untouched.
func RemoveIncludeIf(content, gitDir string) string {
	header := includeIfHeader(gitDir)
	sectionRe := regexp.MustCompile(`^\[`)
	lines := strings.Split(content, "\n")
	out := make([]string, 0, len(lines))
	inBlock := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if sectionRe.MatchString(trimmed) {
			inBlock = trimmed == header
			if inBlock {
				continue
			}
		} else if inBlock {
			if trimmed == "" || strings.HasPrefix(line, " ") || strings.HasPrefix(line, "\t") {
				continue
			}
			inBlock = false
		}
		out = append(out, line)
	}
	return strings.Join(out, "\n")
}

// SetGlobalIdentity shells out to `git config --global` to set user.name and
// user.email, so they become the default identity for any repo that doesn't
// have a more specific local or includeIf-matched override. Not
// unit-tested: it depends on a real git installation, like the rest of the
// gh/git shell-outs in this package.
func SetGlobalIdentity(name, email string) error {
	if out, err := exec.Command("git", "config", "--global", "user.name", name).CombinedOutput(); err != nil {
		return fmt.Errorf("git config --global user.name: %w (%s)", err, strings.TrimSpace(string(out)))
	}
	if out, err := exec.Command("git", "config", "--global", "user.email", email).CombinedOutput(); err != nil {
		return fmt.Errorf("git config --global user.email: %w (%s)", err, strings.TrimSpace(string(out)))
	}
	return nil
}

// EmailFor shells out to `git -C dir config user.email` and returns the
// configured email, or "" if none is set.
func EmailFor(dir string) (string, error) {
	cmd := exec.Command("git", "-C", dir, "config", "user.email")
	out, err := cmd.Output()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			// git config exits 1 when the key isn't set; not a real error.
			return "", nil
		}
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}
