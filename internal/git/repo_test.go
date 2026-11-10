package git

import (
	"os"
	"path/filepath"
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NormalizeToHTTPS(tt.input)
			if result != tt.expected {
				t.Errorf("NormalizeToHTTPS(%q) = %q, expected %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestParseRepository(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "git-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

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

	if info.RemoteURL != "git@github.com:MathCunha16/OpenIDEA-Discord-RCP.git" {
		t.Errorf("expected RemoteURL, got %q", info.RemoteURL)
	}
	if info.WebURL != "https://github.com/MathCunha16/OpenIDEA-Discord-RCP" {
		t.Errorf("expected WebURL https://github.com/MathCunha16/OpenIDEA-Discord-RCP, got %q", info.WebURL)
	}
	if info.Branch != "feat/git-and-idle-tracker" {
		t.Errorf("expected branch feat/git-and-idle-tracker, got %q", info.Branch)
	}
}

func TestParseRepositoryNonGit(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "non-git-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	info, err := ParseRepository(tempDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info != nil {
		t.Fatalf("expected nil RepositoryInfo for non-git directory, got %+v", info)
	}
}
