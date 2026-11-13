package git

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNormalizeToHTTPS(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "SSH SCP-style GitHub",
			input:    "git@github.com:MathCunha16/OpenIDEA-Discord-RCP.git",
			expected: "https://github.com/MathCunha16/OpenIDEA-Discord-RCP",
		},
		{
			name:     "SSH protocol GitLab",
			input:    "ssh://git@gitlab.com/company/project.git",
			expected: "https://gitlab.com/company/project",
		},
		{
			name:     "SSH protocol with port",
			input:    "ssh://git@git.example.com:2222/team/project.git",
			expected: "https://git.example.com:2222/team/project",
		},
		{
			name:     "SSH legacy colon form",
			input:    "ssh://git@gitlab.com:company/project.git",
			expected: "https://gitlab.com/company/project",
		},
		{
			name:     "HTTPS with .git suffix",
			input:    "https://github.com/user/repository.git",
			expected: "https://github.com/user/repository",
		},
		{
			name:     "HTTP upgraded to HTTPS",
			input:    "http://github.com/user/repository",
			expected: "https://github.com/user/repository",
		},
		{
			name:     "Empty input",
			input:    "",
			expected: "",
		},
		// Regression: credentials and extra URL parts must never reach the Discord button.
		{
			name:     "HTTPS strips userinfo token",
			input:    "https://alice:EXAMPLE_NOT_A_REAL_TOKEN@github.com/acme/private.git",
			expected: "https://github.com/acme/private",
		},
		{
			name:     "HTTP strips userinfo password and upgrades",
			input:    "http://alice:EXAMPLE_NOT_A_REAL_TOKEN@github.com/acme/private.git",
			expected: "https://github.com/acme/private",
		},
		{
			name:     "HTTPS strips query and fragment",
			input:    "https://token@github.com/acme/private.git?token=secret#section",
			expected: "https://github.com/acme/private",
		},
		{
			name:     "HTTPS with trailing slash and .git",
			input:    "https://user:pw@github.com/acme/private.git/",
			expected: "https://github.com/acme/private",
		},
		{
			name:     "SCP-style strips query and fragment",
			input:    "git@github.com:acme/private.git?token=secret#section",
			expected: "https://github.com/acme/private",
		},
		{
			name:     "HTTPS host without path",
			input:    "https://github.com",
			expected: "",
		},
		{
			name:     "HTTPS empty path",
			input:    "https://github.com/",
			expected: "",
		},
		{
			name:     "Unsupported scheme",
			input:    "git://github.com/acme/private.git",
			expected: "",
		},
		{
			name:     "Garbage input",
			input:    "not a url",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NormalizeToHTTPS(tt.input)
			if result != tt.expected {
				t.Errorf("NormalizeToHTTPS(%q) = %q, expected %q", tt.input, result, tt.expected)
			}
			// Defense in depth: a web URL must never contain userinfo, query or fragment.
			if strings.ContainsAny(result, "?#") || strings.Contains(result, "@") {
				t.Errorf("NormalizeToHTTPS(%q) leaked URL parts: %q", tt.input, result)
			}
		})
	}
}

func TestStripCredentials(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "HTTPS userinfo",
			input:    "https://alice:TOKEN@github.com/acme/private.git",
			expected: "https://github.com/acme/private.git",
		},
		{
			name:     "SSH scheme user",
			input:    "ssh://git@gitlab.com/company/project.git",
			expected: "ssh://gitlab.com/company/project.git",
		},
		{
			name:     "SCP-style user",
			input:    "git@github.com:MathCunha16/OpenIDEA-Discord-RCP.git",
			expected: "github.com:MathCunha16/OpenIDEA-Discord-RCP.git",
		},
		{
			name:     "SCP-style user with password",
			input:    "user:pass@github.com:acme/private.git",
			expected: "github.com:acme/private.git",
		},
		{
			name:     "Clean URL unchanged",
			input:    "https://github.com/user/repository.git",
			expected: "https://github.com/user/repository.git",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := stripCredentials(tt.input); got != tt.expected {
				t.Errorf("stripCredentials(%q) = %q, expected %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestParseRepositoryRedactsCredentials(t *testing.T) {
	tempDir := t.TempDir()
	gitDir := filepath.Join(tempDir, ".git")
	if err := os.MkdirAll(gitDir, 0755); err != nil {
		t.Fatal(err)
	}

	config := `[core]
	repositoryformatversion = 0
[remote "origin"]
	url = https://alice:EXAMPLE_NOT_A_REAL_TOKEN@github.com/acme/private.git
	fetch = +refs/heads/*:refs/remotes/origin/*
`
	if err := os.WriteFile(filepath.Join(gitDir, "config"), []byte(config), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(gitDir, "HEAD"), []byte("ref: refs/heads/main\n"), 0644); err != nil {
		t.Fatal(err)
	}

	info, err := ParseRepository(tempDir)
	if err != nil {
		t.Fatal(err)
	}
	if info == nil {
		t.Fatal("expected non-nil RepositoryInfo")
	}
	if strings.Contains(info.RemoteURL, "EXAMPLE_NOT_A_REAL_TOKEN") || strings.Contains(info.RemoteURL, "alice") {
		t.Errorf("RemoteURL still carries credentials: %q", info.RemoteURL)
	}
	if info.WebURL != "https://github.com/acme/private" {
		t.Errorf("expected sanitized WebURL, got %q", info.WebURL)
	}
	if strings.Contains(info.WebURL, "@") || strings.Contains(info.WebURL, "?") {
		t.Errorf("WebURL leaked URL parts: %q", info.WebURL)
	}
}

func TestParseRepository(t *testing.T) {
	tempDir := t.TempDir()

	gitDir := filepath.Join(tempDir, ".git")
	if err := os.MkdirAll(gitDir, 0755); err != nil {
		t.Fatalf("failed to create .git dir: %v", err)
	}

	configContent := `[core]
	repositoryformatversion = 0
	filemode = true
[remote "origin"]
	url = git@github.com:MathCunha16/OpenIDEA-Discord-RCP.git
	fetch = +refs/heads/*:refs/remotes/origin/*
`
	if err := os.WriteFile(filepath.Join(gitDir, "config"), []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	if err := os.WriteFile(filepath.Join(gitDir, "HEAD"), []byte("ref: refs/heads/feat/git-and-idle-tracker\n"), 0644); err != nil {
		t.Fatalf("failed to write HEAD file: %v", err)
	}

	// Test nested subdirectory discovery
	subDir := filepath.Join(tempDir, "src", "cmd")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatalf("failed to create sub dir: %v", err)
	}

	info, err := ParseRepository(subDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info == nil {
		t.Fatalf("expected non-nil RepositoryInfo")
	}

	// ParseRepository strips SSH userinfo from the stored remote URL.
	if info.RemoteURL != "github.com:MathCunha16/OpenIDEA-Discord-RCP.git" {
		t.Errorf("expected redacted RemoteURL, got %q", info.RemoteURL)
	}
	if info.WebURL != "https://github.com/MathCunha16/OpenIDEA-Discord-RCP" {
		t.Errorf("expected WebURL https://github.com/MathCunha16/OpenIDEA-Discord-RCP, got %q", info.WebURL)
	}
	if info.Branch != "feat/git-and-idle-tracker" {
		t.Errorf("expected branch feat/git-and-idle-tracker, got %q", info.Branch)
	}
}

func TestParseRepositoryNonGit(t *testing.T) {
	tempDir := t.TempDir()

	info, err := ParseRepository(tempDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info != nil {
		t.Fatalf("expected nil RepositoryInfo for non-git directory, got %+v", info)
	}
}
