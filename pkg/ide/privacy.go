package ide

import (
	"path/filepath"
	"strings"
)

// ProtectedConfigText is the replacement details string when a sensitive file is open.
const ProtectedConfigText = "Editing a private file"

// PrivacyModeText is the replacement details string when PrivacyMode is enabled.
const PrivacyModeText = "Writing code"

// exactSensitiveFiles lists files that must never be exposed over RPC.
var exactSensitiveFiles = map[string]struct{}{
	".env":                     {},
	".envrc":                   {},
	".netrc":                   {},
	"_netrc":                   {},
	".npmrc":                   {},
	".pypirc":                  {},
	".pgpass":                  {},
	".git-credentials":         {},
	".htpasswd":                {},
	"credentials":              {},
	"credentials.json":         {},
	"id_rsa":                   {},
	"id_rsa.pub":               {},
	"id_ed25519":               {},
	"id_ed25519.pub":           {},
	"id_ecdsa":                 {},
	"id_dsa":                   {},
	"known_hosts":              {},
	"authorized_keys":          {},
	"kubeconfig":               {},
	"terraform.tfstate":        {},
	"terraform.tfstate.backup": {},
	"service-account":          {},
	"service-account.json":     {},
	"google-services.json":     {},
	"googleservice-info.plist": {},
	"local.properties":         {},
	"master.key":               {},
	"secrets.toml":             {},
	"secrets.yaml":             {},
	"secrets.json":             {},
}

// sensitiveExtensions lists extensions that often represent keys or certificates.
var sensitiveExtensions = map[string]struct{}{
	".pem":      {},
	".key":      {},
	".pfx":      {},
	".p12":      {},
	".keystore": {},
	".jks":      {},
	".pkcs12":   {},
	".ovpn":     {},
	".crt":      {},
	".cer":      {},
	".der":      {},
	".csr":      {},
	".p8":       {},
	".ppk":      {},
	".asc":      {},
	".gpg":      {},
	".pgp":      {},
	".kdbx":     {},
	".tfstate":  {},
	".tfvars":   {},
	".age":      {},
	".secret":   {},
}

var sensitiveDirs = []string{
	"/.ssh/",
	"/.gnupg/",
	"/.aws/",
	"/.kube/",
	"/.docker/",
}

// IsSensitiveFile checks if the given file path represents a credential, secret, or private key.
func IsSensitiveFile(filePath string) bool {
	if filePath == "" {
		return false
	}

	normPath := "/" + strings.Trim(strings.ToLower(filepath.ToSlash(filePath)), "/") + "/"
	for _, dir := range sensitiveDirs {
		if strings.Contains(normPath, dir) {
			return true
		}
	}

	base := strings.ToLower(filepath.Base(filePath))

	// `.env.example`, `.env.sample`, `.env.template`, etc. are explicitly sharable.
	// Only apply these exemptions to .env template files, not unrelated credentials or secrets.
	if strings.Contains(base, ".env") {
		for _, safeSuffix := range []string{"example", "sample", "template", "dist", "defaults"} {
			if strings.HasSuffix(base, safeSuffix) {
				return false
			}
		}
	}

	// Check exact matches
	if _, ok := exactSensitiveFiles[base]; ok {
		return true
	}

	// Check prefix matches (.env.* or ends with .env)
	if strings.HasPrefix(base, ".env.") || strings.HasSuffix(base, ".env") {
		return true
	}

	// Check sensitive extensions
	ext := strings.ToLower(filepath.Ext(base))
	if _, ok := sensitiveExtensions[ext]; ok {
		return true
	}

	// Check secret-related prefixes
	if strings.HasPrefix(base, "secret") || strings.HasPrefix(base, "credential") {
		return true
	}

	words := []string{"secret", "secrets", "credential", "credentials", "passwd", "passwords"}
	parts := strings.FieldsFunc(base, func(r rune) bool {
		return r == '.' || r == '-' || r == '_'
	})
	for _, p := range parts {
		for _, w := range words {
			if p == w {
				return true
			}
		}
	}

	return false
}

// SanitizeFileDetails produces safe presence details according to privacy preferences and file sensitivity.
func SanitizeFileDetails(filePath string, privacyMode bool) string {
	if privacyMode {
		return PrivacyModeText
	}

	if filePath == "" {
		return "Idling"
	}

	if IsSensitiveFile(filePath) {
		return ProtectedConfigText
	}

	return "Editing " + filepath.Base(filePath)
}

// IsIgnoredFile reports whether a path should be excluded from presence tracking,
// such as OpenIDEA configuration files or virtual documents.
func IsIgnoredFile(path string) bool {
	if path == "" {
		return true
	}

	norm := filepath.ToSlash(path)

	// Virtual documents emitted by OpenIDEA (diffs, terminal views, git commits)
	if strings.HasPrefix(norm, "{") && strings.HasSuffix(norm, "}") {
		return true
	}

	// Config / state folders
	if strings.Contains(norm, "/.openidea/") || strings.HasPrefix(norm, ".openidea/") {
		return true
	}

	// Host-managed settings documents
	base := strings.ToLower(filepath.Base(norm))
	if base == "settings.json" || base == "settings.toml" || base == "settings" {
		if strings.Contains(norm, "/openidea/") || strings.HasPrefix(norm, "openidea/") ||
			strings.Contains(norm, "/.openidea/") || strings.HasPrefix(norm, ".openidea/") {
			return true
		}
	}

	return false
}
