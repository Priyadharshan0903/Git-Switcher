package gitconfig

import (
	"strings"
	"testing"
)

func TestNormalizeDir(t *testing.T) {
	tests := []struct{ in, want string }{
		{"/home/user/work", "/home/user/work/"},
		{"/home/user/work/", "/home/user/work/"},
		{"/", "/"},
	}
	for _, tt := range tests {
		if got := normalizeDir(tt.in); got != tt.want {
			t.Errorf("normalizeDir(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestUserConfigContent(t *testing.T) {
	got := userConfigContent("Your Name", "you@work.com")
	want := "[user]\n  name = Your Name\n  email = you@work.com\n"
	if got != want {
		t.Errorf("userConfigContent = %q, want %q", got, want)
	}
}

func TestHasIncludeIf(t *testing.T) {
	tests := []struct {
		name    string
		content string
		gitDir  string
		want    bool
	}{
		{"empty", "", "/home/user/work", false},
		{"no match", `[includeIf "gitdir:/home/user/personal/"]` + "\n  path = x\n", "/home/user/work", false},
		{"match with trailing slash in query", "[includeIf \"gitdir:/home/user/work/\"]\n  path = x\n", "/home/user/work/", true},
		{"match without trailing slash in query", "[includeIf \"gitdir:/home/user/work/\"]\n  path = x\n", "/home/user/work", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := HasIncludeIf(tt.content, tt.gitDir); got != tt.want {
				t.Errorf("HasIncludeIf = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAddIncludeIf(t *testing.T) {
	tests := []struct {
		name    string
		content string
	}{
		{"empty file", ""},
		{"existing content without trailing newline", "[user]\n  name = x"},
		{"existing content with trailing newline", "[user]\n  name = x\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := AddIncludeIf(tt.content, "/home/user/work", "/home/user/.gitconfig-work")

			if !HasIncludeIf(result, "/home/user/work") {
				t.Fatalf("expected includeIf to be present, got:\n%s", result)
			}
			if !strings.Contains(result, "path = /home/user/.gitconfig-work") {
				t.Errorf("expected path line, got:\n%s", result)
			}
			if tt.content != "" && !strings.Contains(result, tt.content) {
				t.Errorf("expected original content preserved, got:\n%s", result)
			}
		})
	}
}

func TestAddIncludeIfIsIdempotent(t *testing.T) {
	content := AddIncludeIf("", "/home/user/work", "/home/user/.gitconfig-work")
	again := AddIncludeIf(content, "/home/user/work", "/home/user/.gitconfig-work")
	if content != again {
		t.Fatalf("expected adding an existing includeIf to be a no-op:\nfirst:\n%s\nsecond:\n%s", content, again)
	}
}

func TestRemoveIncludeIf(t *testing.T) {
	content := "[user]\n  name = Keep\n" +
		"\n[includeIf \"gitdir:/home/user/work/\"]\n  path = /home/user/.gitconfig-work\n" +
		"\n[alias]\n  co = checkout\n"

	result := RemoveIncludeIf(content, "/home/user/work")

	if HasIncludeIf(result, "/home/user/work") {
		t.Fatalf("expected includeIf to be removed, got:\n%s", result)
	}
	if !strings.Contains(result, "[user]") || !strings.Contains(result, "[alias]") {
		t.Fatalf("expected unrelated sections preserved, got:\n%s", result)
	}
	if strings.Contains(result, ".gitconfig-work") {
		t.Fatalf("expected removed section's body lines gone, got:\n%s", result)
	}
}

func TestRemoveIncludeIfNoMatch(t *testing.T) {
	content := "[user]\n  name = Keep\n"
	result := RemoveIncludeIf(content, "/home/user/work")
	if result != content {
		t.Fatalf("expected content unchanged when includeIf absent:\ngot:\n%s\nwant:\n%s", result, content)
	}
}

func TestPerAccountPath(t *testing.T) {
	got := PerAccountPath("/home/user", "work")
	want := "/home/user/.gitconfig-work"
	if got != want {
		t.Errorf("PerAccountPath = %q, want %q", got, want)
	}
}
