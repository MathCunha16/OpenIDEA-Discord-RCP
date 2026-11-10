package ide

import (
	"path/filepath"
	"strings"
)

// ProtectedConfigText is the replacement details string when a sensitive file is open.
const ProtectedConfigText = "Editing protected configuration"

// PrivacyModeText is the replacement details string when PrivacyMode is enabled.
const PrivacyModeText = "Writing code"

// exactSensitiveFiles lists files that must never be exposed over RPC.
var exactSensitiveFiles = map[string]struct{}{
	".env":               {},
	".env.local":         {},
	".env.production":    {},
	".env.development":   {},
	".env.staging":       {},
	".env.test":          {},
	"id_rsa":             {},
	"id_rsa.pub":         {},
	"id_ed25519":         {},
	"id_ed25519.pub":     {},
	"id_ecdsa":           {},
	"id_dsa":             {},
	"known_hosts":        {},
	"authorized_keys":    {},
	"credentials":        {},
	"credentials.json":   {},
	"service-account":    {},
	"secrets.toml":       {},
	"secrets.yaml":       {},
	"secrets.json":       {},
	"master.key":         {},
}

// sensitiveExtensions lists extensions that often represent keys or certificates.
var sensitiveExtensions = map[string]struct{}{
	".pem":   {},
	".key":   {},
	".pfx":   {},
	".p12":   {},
	".keystore": {},
	".jks":   {},
	".pkcs12": {},
	".ovpn":  {},
}

// IsSensitiveFile checks if the given file path represents a credential, secret, or private key.
func IsSensitiveFile(filePath string) bool {
	if filePath == "" {
		return false
	}

	base := strings.ToLower(filepath.Base(filePath))

	// Check exact matches
	if _, ok := exactSensitiveFiles[base]; ok {
		return true
	}

	// Check prefix matches (.env.*)
	if strings.HasPrefix(base, ".env.") {
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
