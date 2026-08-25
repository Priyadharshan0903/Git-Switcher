// Package sshconfig manages the SSH host aliases and the managed default
// Host github.com block ghsw writes to ~/.ssh/config, plus the symlink
// indirection used to switch which key answers for plain git@github.com
// remotes.
package sshconfig

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"ghsw/internal/fsutil"
)

const (
	// DefaultHost is the plain GitHub host that unaliased remotes
	// (git@github.com:...) resolve against.
	DefaultHost = "github.com"
	// managedDefaultMarker tags the Host block ghsw owns so it can be
	// recognized (and left alone) on future runs, and distinguished from
	// any Host github.com block the user wrote by hand.
	managedDefaultMarker = "# managed by ghsw — run `ghsw use <name>` to change; do not hand-edit IdentityFile below"
)

// ActiveKeyLinkPath is the stable path the managed default Host block points
// at. `ghsw use` repoints this symlink to the active account's key, so plain
// git@github.com remotes follow whichever account is currently active
// machine-wide.
func ActiveKeyLinkPath(home string) string {
	return filepath.Join(home, ".ssh", "ghsw_current_key")
}

// hostBlock renders a Host block for ~/.ssh/config.
func hostBlock(alias, hostName, user, identityFile string) string {
	return fmt.Sprintf("\nHost %s\n  HostName %s\n  User %s\n  IdentityFile %s\n", alias, hostName, user, identityFile)
}

func hostHeaderRe(alias string) *regexp.Regexp {
	return regexp.MustCompile(`(?m)^Host\s+` + regexp.QuoteMeta(alias) + `\s*$`)
}

// hasHost reports whether content already defines the given Host alias.
func hasHost(content, alias string) bool {
	return hostHeaderRe(alias).MatchString(content)
}

// addHost appends a Host block for alias to content, unless it is already
// present, in which case content is returned unchanged.
func addHost(content, alias, hostName, user, identityFile string) string {
	if hasHost(content, alias) {
		return content
	}
	block := hostBlock(alias, hostName, user, identityFile)
	if content == "" {
		return strings.TrimPrefix(block, "\n")
	}
	if !strings.HasSuffix(content, "\n") {
		content += "\n"
	}
	return content + block
}

// removeHostBlock strips the "Host <alias>" block (header plus its indented
// body lines) from content, leaving everything else untouched.
func removeHostBlock(content, alias string) string {
	anyHostRe := regexp.MustCompile(`(?i)^Host\s+`)
	lines := strings.Split(content, "\n")
	out := make([]string, 0, len(lines))
	inBlock := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if anyHostRe.MatchString(trimmed) {
			inBlock = trimmed == "Host "+alias
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

// AddHost idempotently ensures ~/.ssh/config has a Host block for alias. It
// reports whether a new block was written.
func AddHost(path, alias, hostName, user, identityFile string) (bool, error) {
	content, err := fsutil.ReadOrEmpty(path)
	if err != nil {
		return false, err
	}
	if hasHost(content, alias) {
		return false, nil
	}
	newContent := addHost(content, alias, hostName, user, identityFile)
	if err := os.WriteFile(path, []byte(newContent), 0o600); err != nil {
		return false, err
	}
	return true, nil
}

// RemoveHost deletes the Host block for alias from ~/.ssh/config, if present.
func RemoveHost(path, alias string) error {
	content, err := fsutil.ReadOrEmpty(path)
	if err != nil {
		return err
	}
	if !hasHost(content, alias) {
		return nil
	}
	newContent := removeHostBlock(content, alias)
	return os.WriteFile(path, []byte(newContent), 0o600)
}

// hasManagedDefaultHost reports whether content already has ghsw's managed
// "Host github.com" block (identified by managedDefaultMarker on the line
// immediately above it).
func hasManagedDefaultHost(content string) bool {
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		if strings.TrimSpace(line) != managedDefaultMarker {
			continue
		}
		if i+1 < len(lines) && strings.TrimSpace(lines[i+1]) == "Host "+DefaultHost {
			return true
		}
	}
	return false
}

// hasForeignDefaultHost reports whether content has a "Host github.com"
// block that ghsw does not manage. SSH resolves each setting to the first
// matching block that defines it, so an earlier foreign block can silently
// shadow ghsw's IdentityFile.
func hasForeignDefaultHost(content string) bool {
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		if strings.TrimSpace(line) != "Host "+DefaultHost {
			continue
		}
		managed := i > 0 && strings.TrimSpace(lines[i-1]) == managedDefaultMarker
		if !managed {
			return true
		}
	}
	return false
}

// ensureManagedDefaultHost prepends ghsw's managed "Host github.com" block
// (IdentityFile pointing at symlinkPath) to content, unless it is already
// present. It is prepended rather than appended so its settings win SSH's
// first-match-per-keyword resolution ahead of any later, unrelated Host
// github.com block.
func ensureManagedDefaultHost(content, symlinkPath string) (newContent string, added bool) {
	if hasManagedDefaultHost(content) {
		return content, false
	}
	block := managedDefaultMarker + "\n" +
		"Host " + DefaultHost + "\n" +
		"  HostName " + DefaultHost + "\n" +
		"  User git\n" +
		"  IdentityFile " + symlinkPath + "\n" +
		"  IdentitiesOnly yes\n"
	if content == "" {
		return block, true
	}
	return block + "\n" + content, true
}

// EnsureManagedDefaultHost idempotently ensures ~/.ssh/config has ghsw's
// managed default Host block, and reports whether a foreign (user-authored)
// Host github.com block was also found, so the caller can warn about a
// possible conflict.
func EnsureManagedDefaultHost(path, symlinkPath string) (addedBlock, foreignBlock bool, err error) {
	content, err := fsutil.ReadOrEmpty(path)
	if err != nil {
		return false, false, err
	}
	foreignBlock = hasForeignDefaultHost(content)
	newContent, added := ensureManagedDefaultHost(content, symlinkPath)
	if !added {
		return false, foreignBlock, nil
	}
	if err := os.WriteFile(path, []byte(newContent), 0o600); err != nil {
		return false, foreignBlock, err
	}
	return true, foreignBlock, nil
}

// SetActiveKey repoints the active-key symlink at linkPath to keyPath,
// replacing whatever was there before. On platforms where creating a
// symlink isn't permitted (notably Windows without Developer Mode), it
// falls back to copying the key file's contents in place.
func SetActiveKey(linkPath, keyPath string) error {
	if err := os.RemoveAll(linkPath); err != nil {
		return err
	}
	if err := os.Symlink(keyPath, linkPath); err != nil {
		data, readErr := os.ReadFile(keyPath)
		if readErr != nil {
			return readErr
		}
		return os.WriteFile(linkPath, data, 0o600)
	}
	return nil
}
