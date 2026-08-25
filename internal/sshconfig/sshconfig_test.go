package sshconfig

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"ghsw/internal/fsutil"
)

func TestHasHost(t *testing.T) {
	tests := []struct {
		name    string
		content string
		alias   string
		want    bool
	}{
		{"empty content", "", "github-work", false},
		{"no match", "Host github-personal\n  HostName github.com\n", "github-work", false},
		{"exact match", "Host github-work\n  HostName github.com\n", "github-work", true},
		{"match among others", "Host a\n  HostName x\n\nHost github-work\n  HostName github.com\n\nHost b\n", "github-work", true},
		{"prefix collision does not match", "Host github-work-2\n  HostName github.com\n", "github-work", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := hasHost(tt.content, tt.alias); got != tt.want {
				t.Errorf("hasHost(%q, %q) = %v, want %v", tt.content, tt.alias, got, tt.want)
			}
		})
	}
}

func TestAddHostContent(t *testing.T) {
	tests := []struct {
		name    string
		content string
	}{
		{"empty file", ""},
		{"existing content without trailing newline", "Host other\n  HostName example.com"},
		{"existing content with trailing newline", "Host other\n  HostName example.com\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := addHost(tt.content, "github-work", "github.com", "git", "/home/user/.ssh/id_work")

			if !hasHost(result, "github-work") {
				t.Fatalf("expected host to be present after add, got:\n%s", result)
			}
			for _, want := range []string{"Host github-work", "HostName github.com", "User git", "IdentityFile /home/user/.ssh/id_work"} {
				if !strings.Contains(result, want) {
					t.Errorf("expected result to contain %q, got:\n%s", want, result)
				}
			}
			if tt.content != "" && !strings.Contains(result, tt.content) {
				t.Errorf("expected original content to be preserved, got:\n%s", result)
			}
		})
	}
}

func TestAddHostIsIdempotent(t *testing.T) {
	content := addHost("", "github-work", "github.com", "git", "/key")
	again := addHost(content, "github-work", "github.com", "git", "/key")
	if content != again {
		t.Fatalf("expected adding an existing host to be a no-op:\nfirst:\n%s\nsecond:\n%s", content, again)
	}
}

func TestRemoveHostBlock(t *testing.T) {
	content := "Host keep-before\n  HostName a.com\n" +
		"\nHost github-work\n  HostName github.com\n  User git\n  IdentityFile /key\n" +
		"\nHost keep-after\n  HostName b.com\n"

	result := removeHostBlock(content, "github-work")

	if hasHost(result, "github-work") {
		t.Fatalf("expected github-work to be removed, got:\n%s", result)
	}
	if !strings.Contains(result, "Host keep-before") || !strings.Contains(result, "Host keep-after") {
		t.Fatalf("expected unrelated hosts to be preserved, got:\n%s", result)
	}
	if strings.Contains(result, "IdentityFile /key") {
		t.Fatalf("expected removed host's body lines to be gone, got:\n%s", result)
	}
}

func TestRemoveHostBlockNoMatch(t *testing.T) {
	content := "Host keep\n  HostName a.com\n"
	result := removeHostBlock(content, "github-work")
	if result != content {
		t.Fatalf("expected content to be unchanged when alias absent:\ngot:\n%s\nwant:\n%s", result, content)
	}
}

func TestAddHostFileIO(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config")

	added, err := AddHost(path, "github-work", "github.com", "git", "/key")
	if err != nil {
		t.Fatalf("AddHost: %v", err)
	}
	if !added {
		t.Fatalf("expected added=true on first call")
	}

	added, err = AddHost(path, "github-work", "github.com", "git", "/key")
	if err != nil {
		t.Fatalf("AddHost (second call): %v", err)
	}
	if added {
		t.Fatalf("expected added=false on second call (idempotent)")
	}

	content, err := fsutil.ReadOrEmpty(path)
	if err != nil {
		t.Fatalf("ReadOrEmpty: %v", err)
	}
	if !hasHost(content, "github-work") {
		t.Fatalf("expected host to be present on disk, got:\n%s", content)
	}
}

func TestRemoveHostFileIO(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config")

	if _, err := AddHost(path, "github-work", "github.com", "git", "/key"); err != nil {
		t.Fatalf("AddHost: %v", err)
	}
	if err := RemoveHost(path, "github-work"); err != nil {
		t.Fatalf("RemoveHost: %v", err)
	}

	content, err := fsutil.ReadOrEmpty(path)
	if err != nil {
		t.Fatalf("ReadOrEmpty: %v", err)
	}
	if hasHost(content, "github-work") {
		t.Fatalf("expected host to be removed, got:\n%s", content)
	}
}

func TestHasManagedDefaultHost(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    bool
	}{
		{"empty content", "", false},
		{"no host github.com at all", "Host github-work\n  HostName github.com\n", false},
		{"foreign Host github.com without marker", "Host github.com\n  IdentityFile /key\n", false},
		{"managed block", managedDefaultMarker + "\nHost github.com\n  HostName github.com\n  User git\n  IdentityFile /link\n  IdentitiesOnly yes\n", true},
		{"marker present but not directly above the host line", managedDefaultMarker + "\n\nHost github.com\n  IdentityFile /key\n", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := hasManagedDefaultHost(tt.content); got != tt.want {
				t.Errorf("hasManagedDefaultHost(%q) = %v, want %v", tt.content, got, tt.want)
			}
		})
	}
}

func TestHasForeignDefaultHost(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    bool
	}{
		{"empty content", "", false},
		{"no github.com host", "Host github-work\n  HostName github.com\n", false},
		{"foreign block", "Host github.com\n  IdentityFile /key\n", true},
		{"managed block only", managedDefaultMarker + "\nHost github.com\n  IdentityFile /link\n", false},
		{"foreign block alongside a managed one", "Host github.com\n  IdentityFile /old\n\n" + managedDefaultMarker + "\nHost github.com\n  IdentityFile /link\n", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := hasForeignDefaultHost(tt.content); got != tt.want {
				t.Errorf("hasForeignDefaultHost(%q) = %v, want %v", tt.content, got, tt.want)
			}
		})
	}
}

func TestEnsureManagedDefaultHostContent(t *testing.T) {
	tests := []struct {
		name    string
		content string
	}{
		{"empty file", ""},
		{"existing content without trailing newline", "Host other\n  HostName example.com"},
		{"existing content with trailing newline", "Host other\n  HostName example.com\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, added := ensureManagedDefaultHost(tt.content, "/home/user/.ssh/ghsw_current_key")
			if !added {
				t.Fatalf("expected added=true when no managed block exists yet")
			}
			if !hasManagedDefaultHost(result) {
				t.Fatalf("expected managed default host to be present after ensure, got:\n%s", result)
			}
			for _, want := range []string{"Host github.com", "IdentityFile /home/user/.ssh/ghsw_current_key", "IdentitiesOnly yes"} {
				if !strings.Contains(result, want) {
					t.Errorf("expected result to contain %q, got:\n%s", want, result)
				}
			}
			if tt.content != "" && !strings.Contains(result, tt.content) {
				t.Errorf("expected original content to be preserved, got:\n%s", result)
			}
			if !strings.HasPrefix(result, managedDefaultMarker) {
				t.Errorf("expected managed block to be prepended so it wins SSH's first-match resolution, got:\n%s", result)
			}
		})
	}
}

func TestEnsureManagedDefaultHostContentIsIdempotent(t *testing.T) {
	content, added := ensureManagedDefaultHost("", "/link")
	if !added {
		t.Fatalf("expected added=true on first call")
	}
	again, added := ensureManagedDefaultHost(content, "/link")
	if added {
		t.Fatalf("expected added=false once a managed block already exists")
	}
	if content != again {
		t.Fatalf("expected content unchanged on second call:\nfirst:\n%s\nsecond:\n%s", content, again)
	}
}

func TestEnsureManagedDefaultHostFileIO(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config")

	added, foreign, err := EnsureManagedDefaultHost(path, "/link")
	if err != nil {
		t.Fatalf("EnsureManagedDefaultHost: %v", err)
	}
	if !added || foreign {
		t.Fatalf("expected added=true, foreign=false on a fresh file, got added=%v foreign=%v", added, foreign)
	}

	added, foreign, err = EnsureManagedDefaultHost(path, "/link")
	if err != nil {
		t.Fatalf("EnsureManagedDefaultHost (second call): %v", err)
	}
	if added || foreign {
		t.Fatalf("expected added=false, foreign=false on second call, got added=%v foreign=%v", added, foreign)
	}
}

func TestEnsureManagedDefaultHostDetectsForeignBlock(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config")
	if err := os.WriteFile(path, []byte("Host github.com\n  IdentityFile /manual/key\n"), 0o600); err != nil {
		t.Fatalf("seeding config: %v", err)
	}

	added, foreign, err := EnsureManagedDefaultHost(path, "/link")
	if err != nil {
		t.Fatalf("EnsureManagedDefaultHost: %v", err)
	}
	if !added {
		t.Fatalf("expected ghsw's managed block to still be added")
	}
	if !foreign {
		t.Fatalf("expected the pre-existing manual block to be reported as foreign")
	}
}

func TestSetActiveKey(t *testing.T) {
	dir := t.TempDir()
	linkPath := filepath.Join(dir, "ghsw_current_key")
	keyA := filepath.Join(dir, "key_a")
	keyB := filepath.Join(dir, "key_b")
	if err := os.WriteFile(keyA, []byte("key-a-contents"), 0o600); err != nil {
		t.Fatalf("writing keyA: %v", err)
	}
	if err := os.WriteFile(keyB, []byte("key-b-contents"), 0o600); err != nil {
		t.Fatalf("writing keyB: %v", err)
	}

	if err := SetActiveKey(linkPath, keyA); err != nil {
		t.Fatalf("SetActiveKey(keyA): %v", err)
	}
	data, err := os.ReadFile(linkPath)
	if err != nil {
		t.Fatalf("reading linkPath after first switch: %v", err)
	}
	if string(data) != "key-a-contents" {
		t.Fatalf("expected linkPath to resolve to keyA's contents, got %q", data)
	}

	if err := SetActiveKey(linkPath, keyB); err != nil {
		t.Fatalf("SetActiveKey(keyB): %v", err)
	}
	data, err = os.ReadFile(linkPath)
	if err != nil {
		t.Fatalf("reading linkPath after second switch: %v", err)
	}
	if string(data) != "key-b-contents" {
		t.Fatalf("expected linkPath to resolve to keyB's contents after re-switching, got %q", data)
	}
}
