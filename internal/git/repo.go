package git

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

// RepositoryInfo stores extracted Git repository metadata.
type RepositoryInfo struct {
	RemoteURL string // Original remote URL (e.g. git@github.com:user/repo.git or https://github.com/user/repo)
	WebURL    string // Sanitized HTTPS web browseable URL (e.g. https://github.com/user/repo)
	Branch    string // Active branch name
}

// FindGitRoot traverses upwards starting from dir looking for a .git directory.
func FindGitRoot(startDir string) string {
	curr := filepath.Clean(startDir)
	for {
		gitPath := filepath.Join(curr, ".git")
		if fi, err := os.Stat(gitPath); err == nil && fi.IsDir() {
			return curr
		}

		parent := filepath.Dir(curr)
		if parent == curr {
			break
		}
		curr = parent
	}
	return ""
}

// ParseRepository resolves Git repository information for a project root directory.
func ParseRepository(projectRoot string) (*RepositoryInfo, error) {
	if projectRoot == "" {
		return nil, nil
	}

	gitRoot := FindGitRoot(projectRoot)
	if gitRoot == "" {
		return nil, nil
	}

	configPath := filepath.Join(gitRoot, ".git", "config")
	remoteURL := extractOriginURL(configPath)
	branch := extractHeadBranch(filepath.Join(gitRoot, ".git", "HEAD"))

	info := &RepositoryInfo{
		RemoteURL: remoteURL,
		WebURL:    NormalizeToHTTPS(remoteURL),
		Branch:    branch,
	}

	return info, nil
}

// extractOriginURL parses the .git/config file to retrieve the remote "origin" URL.
func extractOriginURL(configPath string) string {
	file, err := os.Open(configPath)
	if err != nil {
		return ""
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	inOriginSection := false

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if strings.HasPrefix(line, "[") {
			inOriginSection = line == `[remote "origin"]`
			continue
		}

		if inOriginSection && strings.HasPrefix(line, "url") {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				return strings.TrimSpace(parts[1])
			}
		}
	}

	return ""
}

// extractHeadBranch parses the active branch from .git/HEAD (e.g. "ref: refs/heads/main").
func extractHeadBranch(headPath string) string {
	data, err := os.ReadFile(headPath)
	if err != nil {
		return ""
	}

	content := strings.TrimSpace(string(data))
	if strings.HasPrefix(content, "ref: refs/heads/") {
		return strings.TrimPrefix(content, "ref: refs/heads/")
	}

	// Detached HEAD: return short commit hash if available
	if len(content) >= 7 {
		return content[:7]
	}

	return content
}

// NormalizeToHTTPS converts SSH, Git, or HTTPS remote URLs to a clean HTTPS web browseable URL.
// Examples:
// - git@github.com:owner/repo.git -> https://github.com/owner/repo
// - ssh://git@gitlab.com/owner/repo.git -> https://gitlab.com/owner/repo
// - https://github.com/owner/repo.git -> https://github.com/owner/repo
func NormalizeToHTTPS(rawURL string) string {
	u := strings.TrimSpace(rawURL)
	if u == "" {
		return ""
	}

	// 1. Strip trailing .git
	u = strings.TrimSuffix(u, ".git")

	// 2. Handle SSH protocol with prefix (ssh://git@host/path)
	if strings.HasPrefix(u, "ssh://") {
		u = strings.TrimPrefix(u, "ssh://")
		if atIdx := strings.Index(u, "@"); atIdx != -1 {
			u = u[atIdx+1:]
		}
		return "https://" + strings.Replace(u, ":", "/", 1)
	}

	// 3. Handle standard SCP-style SSH (git@host:owner/repo)
	if strings.Contains(u, "@") && strings.Contains(u, ":") && !strings.HasPrefix(u, "http://") && !strings.HasPrefix(u, "https://") {
		atIdx := strings.Index(u, "@")
		colonIdx := strings.Index(u, ":")
		if atIdx < colonIdx {
			host := u[atIdx+1 : colonIdx]
			path := u[colonIdx+1:]
			path = strings.TrimPrefix(path, "/")
			return "https://" + host + "/" + path
		}
	}

	// 4. Handle HTTP -> HTTPS upgrade
	if strings.HasPrefix(u, "http://") {
		return "https://" + strings.TrimPrefix(u, "http://")
	}

	// 5. Already clean HTTPS
	if strings.HasPrefix(u, "https://") {
		return u
	}

	return ""
}
