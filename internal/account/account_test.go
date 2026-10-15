package account

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestLoadMissingFile(t *testing.T) {
	dir := t.TempDir()
	reg, err := Load(filepath.Join(dir, "does-not-exist.json"))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if reg.Accounts == nil || len(reg.Accounts) != 0 {
		t.Fatalf("expected empty non-nil Accounts map, got %#v", reg.Accounts)
	}
}

func TestSaveAndLoadRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nested", "accounts.json")

	want := &Registry{
		Accounts: map[string]Account{
			"work": {
				SSHAlias:   "github-work",
				SSHKey:     "/home/user/.ssh/id_ed25519_work",
				GitDir:     "/home/user/work",
				GitName:    "Your Name",
				GitEmail:   "you@work.com",
				GHUsername: "your-work-handle",
			},
			"personal": {
				SSHAlias:   "github-personal",
				SSHKey:     "/home/user/.ssh/id_ed25519_personal",
				GitDir:     "/home/user/personal",
				GitName:    "Your Name",
				GitEmail:   "you@personal.com",
				GHUsername: "your-personal-handle",
			},
		},
	}

	if err := want.Save(path); err != nil {
		t.Fatalf("Save: %v", err)
	}

	got, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if !reflect.DeepEqual(want.Accounts, got.Accounts) {
		t.Fatalf("round trip mismatch:\nwant %#v\ngot  %#v", want.Accounts, got.Accounts)
	}
}

func TestSaveOverwritesExisting(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "accounts.json")

	first := &Registry{Accounts: map[string]Account{"work": {GHUsername: "old"}}}
	if err := first.Save(path); err != nil {
		t.Fatalf("Save: %v", err)
	}

	second := &Registry{Accounts: map[string]Account{"work": {GHUsername: "new"}}}
	if err := second.Save(path); err != nil {
		t.Fatalf("Save: %v", err)
	}

	got, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got.Accounts["work"].GHUsername != "new" {
		t.Fatalf("expected overwrite to take effect, got %q", got.Accounts["work"].GHUsername)
	}
	if len(got.Accounts) != 1 {
		t.Fatalf("expected exactly 1 account after overwrite, got %d", len(got.Accounts))
	}
}

func TestDefaultPath(t *testing.T) {
	got := DefaultPath("/home/user")
	want := filepath.Join("/home/user", ".ghsw", "accounts.json")
	if got != want {
		t.Fatalf("DefaultPath: got %q, want %q", got, want)
	}
}

func TestHostDefaultsToGitHubCom(t *testing.T) {
	if got := (Account{}).Host(); got != DefaultHostname {
		t.Errorf("Host() with empty Hostname = %q, want %q", got, DefaultHostname)
	}
	if got := (Account{Hostname: "ghe.corp.com"}).Host(); got != "ghe.corp.com" {
		t.Errorf("Host() = %q, want %q", got, "ghe.corp.com")
	}
}

func TestLoadLegacyRegistryWithoutHostname(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "accounts.json")
	legacy := `{"accounts":{"work":{"ssh_alias":"github-work","ssh_key":"/k","git_dir":"","git_name":"N","git_email":"e@x","gh_username":"h"}}}`
	if err := os.WriteFile(path, []byte(legacy), 0o644); err != nil {
		t.Fatal(err)
	}
	reg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got := reg.Accounts["work"].Host(); got != DefaultHostname {
		t.Errorf("legacy account Host() = %q, want %q", got, DefaultHostname)
	}
}

func TestSaveOmitsHostnameForGitHubCom(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "accounts.json")
	reg := &Registry{Accounts: map[string]Account{
		"personal": {SSHAlias: "github-personal"},
		"work":     {SSHAlias: "github-work", Hostname: "ghe.corp.com"},
	}}
	if err := reg.Save(path); err != nil {
		t.Fatalf("Save: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if n := strings.Count(string(data), `"hostname"`); n != 1 {
		t.Fatalf("expected exactly one hostname field (enterprise account only), got %d:\n%s", n, data)
	}
}

func TestNormalizeHost(t *testing.T) {
	tests := []struct {
		in      string
		want    string
		wantErr bool
	}{
		{"github.com", "github.com", false},
		{"GHE.Corp.com", "ghe.corp.com", false},
		{"https://ghe.corp.com/", "ghe.corp.com", false},
		{"  http://ghe.corp.com  ", "ghe.corp.com", false},
		{"", "", true},
		{"https://", "", true},
		{"ghe.corp.com/org", "", true},
		{"ghe.corp.com:2222", "", true},
		{"git@ghe.corp.com", "", true},
		{"ghe corp.com", "", true},
	}
	for _, tt := range tests {
		got, err := NormalizeHost(tt.in)
		if (err != nil) != tt.wantErr {
			t.Errorf("NormalizeHost(%q) err = %v, wantErr %v", tt.in, err, tt.wantErr)
			continue
		}
		if got != tt.want {
			t.Errorf("NormalizeHost(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}
