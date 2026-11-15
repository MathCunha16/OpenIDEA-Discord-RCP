package git

import (
	"bufio"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

// RepositoryInfo stores extracted Git repository metadata.
type RepositoryInfo struct {
	RemoteURL string // Remote URL with any userinfo (credentials) removed. Use WebURL for display.
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
	remoteURL := stripCredentials(extractOriginURL(configPath))
	branch := extractHeadBranch(filepath.Join(gitRoot, ".git", "HEAD"))

	info := &RepositoryInfo{
		RemoteURL: remoteURL,
		WebURL:    NormalizeToHTTPS(remoteURL),
		Branch:    branch,
	}

	return info, nil
}

// stripCredentials removes userinfo (a username, PAT or password) from a
// remote URL. A .git/config can embed authentication material; it must not be
// retained in RepositoryInfo or forwarded anywhere.
func stripCredentials(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	if parsed, err := url.Parse(raw); err == nil && parsed.User != nil {
		parsed.User = nil
		return parsed.String()
	}
	// SCP-style (git@host:owner/repo.git) is not parseable by net/url; drop
	// everything up to the last '@', which may include a password.
	if !strings.Contains(raw, "://") {
		if at := strings.LastIndex(raw, "@"); at != -1 {
			return raw[at+1:]
		}
	}
	return raw
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

// NormalizeToHTTPS converts SSH, Git, or HTTPS remote URLs to a clean HTTPS web
// browseable URL. HTTP(S) URLs are parsed with net/url and rebuilt from a
// validated host and path only, so userinfo, query and fragment (which may
// contain credentials) are always dropped.
//
// Examples:
//   - git@github.com:owner/repo.git           -> https://github.com/owner/repo
//   - ssh://git@gitlab.com/owner/repo.git     -> https://gitlab.com/owner/repo
//   - https://user:TOKEN@host/owner/repo.git -> https://host/owner/repo
//   - http://host/owner/repo.git?token=xyz   -> https://host/owner/repo
func NormalizeToHTTPS(rawURL string) string {
	u := strings.TrimSpace(rawURL)
	if u == "" {
		return ""
	}

	// 1. HTTP(S): parse and rebuild from host + path only. Userinfo, query and
	//    fragment are never copied into the result.
	if strings.HasPrefix(u, "http://") || strings.HasPrefix(u, "https://") {
		parsed, err := url.Parse(u)
		if err != nil || parsed.Opaque != "" || parsed.Hostname() == "" {
			return ""
		}
		return buildWebURL(parsed.Host, parsed.Path)
	}

	// 2. SSH with scheme: ssh://[user@]host[:port]/owner/repo.git
	if strings.HasPrefix(u, "ssh://") {
		rest := strings.TrimPrefix(u, "ssh://")
		if at := strings.LastIndex(rest, "@"); at != -1 {
			rest = rest[at+1:]
		}
		if parsed, err := url.Parse("ssh://" + rest); err == nil && parsed.Hostname() != "" {
			return buildWebURL(parsed.Host, parsed.Path)
		}
		// Legacy form ssh://host:owner/repo.git uses a colon where a slash is
		// expected; net/url rejects it as an invalid port, so split manually.
		if colon := strings.Index(rest, ":"); colon != -1 {
			return buildWebURL(rest[:colon], rest[colon+1:])
		}
		return ""
	}

	// Any other scheme (git://, file://, ...) is not a web URL.
	if strings.Contains(u, "://") {
		return ""
	}

	// 3. SCP-style: [userinfo@]host:owner/repo.git
	if at := strings.LastIndex(u, "@"); at != -1 {
		u = u[at+1:]
	}
	if colon := strings.Index(u, ":"); colon != -1 {
		return buildWebURL(u[:colon], u[colon+1:])
	}

	return ""
}

// buildWebURL builds an https URL from a bare host and repository path. Only
// scheme, host and path are used, so userinfo, query and fragment never
// survive. The host is validated to make sure no '@' (userinfo) is reintroduced.
func buildWebURL(host, path string) string {
	host = strings.TrimSpace(host)
	if host == "" || strings.ContainsAny(host, " /?#@\\") {
		return ""
	}
	clean := cleanRepoPath(path)
	if clean == "" {
		return ""
	}
	return (&url.URL{Scheme: "https", Host: host, Path: clean}).String()
}

// cleanRepoPath strips trailing slashes, a trailing ".git" and any
// query/fragment suffix from a repository path.
func cleanRepoPath(path string) string {
	if i := strings.IndexAny(path, "?#"); i != -1 {
		path = path[:i]
	}
	path = strings.TrimRight(path, "/")
	path = strings.TrimSuffix(path, ".git")
	path = strings.TrimRight(path, "/")
	if path == "" || path == "/" {
		return ""
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return path
}
