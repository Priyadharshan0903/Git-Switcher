// Package account manages ghsw's on-disk registry of GitHub identities.
package account

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Account is a single registered GitHub identity.
type Account struct {
	SSHAlias   string `json:"ssh_alias"`
	SSHKey     string `json:"ssh_key"`
	GitDir     string `json:"git_dir"`
	GitName    string `json:"git_name"`
	GitEmail   string `json:"git_email"`
	GHUsername string `json:"gh_username"`
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
