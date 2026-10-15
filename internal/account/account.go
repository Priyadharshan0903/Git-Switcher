// Package account manages ghsw's on-disk registry of GitHub identities.
package account

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// DefaultHostname is the GitHub host an account belongs to when it doesn't
// name one. Registries written before hostnames existed load as accounts on
// this host.
const DefaultHostname = "github.com"

// Account is a single registered GitHub identity.
type Account struct {
	SSHAlias   string `json:"ssh_alias"`
	SSHKey     string `json:"ssh_key"`
	GitDir     string `json:"git_dir"`
	GitName    string `json:"git_name"`
	GitEmail   string `json:"git_email"`
	GHUsername string `json:"gh_username"`
	// Hostname is the GitHub host (e.g. a GitHub Enterprise Server
	// instance) this identity lives on. Empty means DefaultHostname, and is
	// left empty for github.com accounts so their JSON stays unchanged.
	Hostname string `json:"hostname,omitempty"`
}

// Host returns the GitHub host this account lives on.
func (a Account) Host() string {
	if a.Hostname == "" {
		return DefaultHostname
	}
	return a.Hostname
}

// NormalizeHost cleans up a user-supplied hostname: it lowercases it and
// strips a leading scheme and trailing slash, so "https://GHE.corp.com/"
// becomes "ghe.corp.com". It rejects values that still contain a path,
// port, or whitespace, since those can't be used as an SSH or gh hostname.
func NormalizeHost(raw string) (string, error) {
	h := strings.ToLower(strings.TrimSpace(raw))
	h = strings.TrimPrefix(h, "https://")
	h = strings.TrimPrefix(h, "http://")
	h = strings.TrimSuffix(h, "/")
	if h == "" {
		return "", fmt.Errorf("hostname is empty")
	}
	if strings.ContainsAny(h, "/: \t@") {
		return "", fmt.Errorf("invalid hostname %q: expected a bare host like ghe.example.com", raw)
	}
	return h, nil
}

// Registry is the on-disk accounts.json document.
type Registry struct {
	Accounts map[string]Account `json:"accounts"`
}

// DefaultPath returns ~/.ghsw/accounts.json.
func DefaultPath(home string) string {
	return filepath.Join(home, ".ghsw", "accounts.json")
}

// Load reads the registry at path, returning an empty registry if the file
// does not exist yet.
func Load(path string) (*Registry, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return &Registry{Accounts: map[string]Account{}}, nil
	}
	if err != nil {
		return nil, err
	}
	var r Registry
	if err := json.Unmarshal(data, &r); err != nil {
		return nil, err
	}
	if r.Accounts == nil {
		r.Accounts = map[string]Account{}
	}
	return &r, nil
}

// Save writes the registry to path as indented JSON, creating parent
// directories as needed.
func (r *Registry) Save(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0o644)
}
