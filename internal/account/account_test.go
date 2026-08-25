package account

import (
	"path/filepath"
	"reflect"
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
