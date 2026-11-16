package ide

import "strings"

// specialDockerCompose covers the official Compose v1/v2 filenames, including
// environment-specific variants such as docker-compose.prod.yml.
func isDockerCompose(name string) bool {
	if name == "compose.yml" || name == "compose.yaml" ||
		name == "docker-compose.yml" || name == "docker-compose.yaml" ||
		name == "docker-compose" {
		return true
	}
	for _, suffix := range []string{".yml", ".yaml"} {
		if strings.HasPrefix(name, "docker-compose.") && strings.HasSuffix(name, suffix) {
			return true
		}
	}
	return false
}

// matchSpecialFile resolves filenames whose identity is more specific than the
// language reported by the IDE. Keeping these ahead of reportedLang prevents a
// generic "yaml"/"json"/"toml" answer from shadowing a known manifest.
// Only asset keys that exist in the icon themes are used.
func matchSpecialFile(lowerFilename, lowerPath string) (LanguageInfo, bool) {
	if isDockerCompose(lowerFilename) {
		return LanguageInfo{ID: "docker", Name: "Docker Compose", LargeImage: "docker"}, true
	}
	switch lowerFilename {
	case "dockerfile", "containerfile", ".dockerfile", ".dockerignore", "docker-sync.yml":
		return LanguageInfo{ID: "dockerfile", Name: "Dockerfile", LargeImage: "docker"}, true
	}
	// GitHub Actions workflows live under .github/workflows and are reported
	// as yaml by the IDE.
	if strings.Contains(lowerPath, "/.github/workflows/") &&
		(strings.HasSuffix(lowerFilename, ".yml") || strings.HasSuffix(lowerFilename, ".yaml")) {
		return LanguageInfo{ID: "github", Name: "GitHub Actions", LargeImage: "github"}, true
	}
	return LanguageInfo{}, false
}
