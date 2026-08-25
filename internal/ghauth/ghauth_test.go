package ghauth

import "testing"

// Switch and Status themselves just shell out to the gh CLI and aren't
// unit-tested here: doing so would require a real, authenticated gh
// installation, which isn't available in CI. ActiveUsername is pure string
// processing over gh's output and is tested against captured samples.

func TestActiveUsername(t *testing.T) {
	tests := []struct {
		name     string
		output   string
		hostname string
		want     string
	}{
		{
			name:     "empty output",
			output:   "",
			hostname: "github.com",
			want:     "",
		},
		{
			name: "single account active",
			output: `github.com
  ✓ Logged in to github.com account work-handle (keyring)
  - Active account: true
  - Git operations protocol: https
  - Token: gho_****
`,
			hostname: "github.com",
			want:     "work-handle",
		},
		{
			name: "multiple accounts, second is active",
			output: `github.com
  ✓ Logged in to github.com account work-handle (keyring)
  - Active account: false
  - Git operations protocol: https

  ✓ Logged in to github.com account personal-handle (keyring)
  - Active account: true
  - Git operations protocol: ssh
`,
			hostname: "github.com",
			want:     "personal-handle",
		},
		{
			name: "no host match",
			output: `github.com
  ✓ Logged in to github.com account work-handle (keyring)
  - Active account: true
`,
			hostname: "github.enterprise.example.com",
			want:     "",
		},
		{
			name: "no active account marked",
			output: `github.com
  ✓ Logged in to github.com account work-handle (keyring)
  - Active account: false
`,
			hostname: "github.com",
			want:     "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ActiveUsername(tt.output, tt.hostname); got != tt.want {
				t.Errorf("ActiveUsername(...) = %q, want %q", got, tt.want)
			}
		})
	}
}
