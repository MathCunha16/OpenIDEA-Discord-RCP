package ide

import (
	"bytes"
	"io"
	"testing"
)

func TestEventReader(t *testing.T) {
	streamData := `{"file":"main.go","language":"go","project":"OpenIDEA-Discord-Rcp","branch":"main"}
{"file":".env","privacy_mode":false}
`
	reader := NewEventReader(bytes.NewBufferString(streamData))

	ev1, err := reader.ReadNext()
	if err != nil {
		t.Fatalf("unexpected error on ev1: %v", err)
	}
	if ev1.File != "main.go" || ev1.Language != "go" || ev1.Project != "OpenIDEA-Discord-Rcp" || ev1.Branch != "main" {
		t.Fatalf("unexpected ev1 values: %+v", ev1)
	}

	ev2, err := reader.ReadNext()
	if err != nil {
		t.Fatalf("unexpected error on ev2: %v", err)
	}
	if ev2.File != ".env" || ev2.PrivacyMode != false {
		t.Fatalf("unexpected ev2 values: %+v", ev2)
	}

	_, err = reader.ReadNext()
	if err != io.EOF {
		t.Fatalf("expected io.EOF, got %v", err)
	}
}

func TestDetectLanguage(t *testing.T) {
	tests := []struct {
		name         string
		filePath     string
		reportedLang string
		expectedID   string
		expectedImg  string
	}{
		{"Go file", "/src/main.go", "", "go", "go"},
		{"Python file", "scripts/test.py", "", "python", "python"},
		{"Rust file", "crates/core/lib.rs", "", "rust", "rust"},
		{"Dockerfile", "deploy/Dockerfile", "", "dockerfile", "docker"},
		{"Makefile", "Makefile", "", "makefile", "makefile"},
		{"Cargo.toml", "Cargo.toml", "", "rust", "rust"},
		{"TypeScript React", "components/Header.tsx", "", "typescriptreact", "react"},
		{"Composite extension tar.gz", "archive.tar.gz", "", "tarball", "tarball"},
		{"Composite extension ng.html", "src/template.ng.html", "", "angular", "angular"},
		{"Composite extension coffee.erb", "views/index.coffee.erb", "", "coffeescript", "coffeescript"},
		{"Composite extension component.ts", "components/app.component.ts", "", "angular", "angular"},
		{"Reported language priority", "unknown_file", "rust", "rust", "rust"},
		{"Empty fallback", "", "", "text", "text"},
		{"Unknown extension", "archive.bin", "", "text", "text"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info := DetectLanguage(tt.filePath, tt.reportedLang)
			if info.ID != tt.expectedID {
				t.Errorf("expected ID %s, got %s", tt.expectedID, info.ID)
			}
			if info.LargeImage != tt.expectedImg {
				t.Errorf("expected LargeImage %s, got %s", tt.expectedImg, info.LargeImage)
			}
		})
	}
}

func TestPrivacySanitizer(t *testing.T) {
	tests := []struct {
		filePath    string
		privacyMode bool
		expected    string
	}{
		{"/home/user/project/main.go", false, "Editing main.go"},
		{"/home/user/project/.env", false, ProtectedConfigText},
		{"/home/user/project/.env.production", false, ProtectedConfigText},
		{"/home/user/.ssh/id_rsa", false, ProtectedConfigText},
		{"/home/user/certs/server.pem", false, ProtectedConfigText},
		{"/home/user/certs/private.key", false, ProtectedConfigText},
		{"/home/user/project/secrets.yaml", false, ProtectedConfigText},
		{"/home/user/project/main.go", true, PrivacyModeText},
		{"", false, "Idling"},
	}

	for _, tt := range tests {
		res := SanitizeFileDetails(tt.filePath, tt.privacyMode)
		if res != tt.expected {
			t.Errorf("for (%s, %v) expected %q, got %q", tt.filePath, tt.privacyMode, tt.expected, res)
		}
	}
}

func TestIsIgnoredFile(t *testing.T) {
	assertIgnored := func(path string, expected bool) {
		t.Helper()
		if got := IsIgnoredFile(path); got != expected {
			t.Errorf("IsIgnoredFile(%q) = %v; want %v", path, got, expected)
		}
	}

	assertIgnored("/path/to/.openidea/config", true)
	assertIgnored("/path/to/.openidea/settings.json", true)
	assertIgnored("/home/user/.config/openidea/settings.json", true)
	assertIgnored("/home/user/.config/openidea/settings.toml", true)
	assertIgnored("/home/user/.config/openidea/settings", true)
	assertIgnored("{temp_virtual_doc}", true)
	assertIgnored("{diff}", true)
	assertIgnored("", true)

	// User code settings files should NOT be ignored
	assertIgnored("/path/to/settings.json", false)
	assertIgnored("/app/src/settings.toml", false)

	// Must NOT ignore directories that just contain the word .openidea as substring
	assertIgnored("/home/user/.openidea-plugins/my-plugin/src/main.rs", false)
	assertIgnored("/path/to/main.rs", false)
}

// Regression: the IDE reports docker-compose.yml as "yaml". The special
// filename must win, otherwise the Docker icon never appears.
func TestDetectLanguageSpecialFilesBeatReportedLang(t *testing.T) {
	cases := []struct {
		path      string
		reported  string
		wantID    string
		wantName  string
		wantLarge string
	}{
		{"/p/docker-compose.yml", "yaml", "docker", "Docker Compose", "docker"},
		{"/p/docker-compose.yaml", "yaml", "docker", "Docker Compose", "docker"},
		{"/p/compose.yml", "yaml", "docker", "Docker Compose", "docker"},
		{"/p/compose.yaml", "yaml", "docker", "Docker Compose", "docker"},
		{"/p/docker-compose.prod.yml", "yaml", "docker", "Docker Compose", "docker"},
		{"/p/docker-compose.override.yaml", "yaml", "docker", "Docker Compose", "docker"},
		{"/p/.github/workflows/ci.yml", "yaml", "github", "GitHub Actions", "github"},
		{"/p/package.json", "json", "javascript", "Node Package", "nodejs"},
		{"/p/tsconfig.json", "json", "typescript", "TypeScript Config", "typescript"},
		{"/p/Cargo.toml", "toml", "rust", "Rust Manifest", "rust"},
	}
	for _, tc := range cases {
		got := DetectLanguage(tc.path, tc.reported)
		if got.ID != tc.wantID || got.Name != tc.wantName || got.LargeImage != tc.wantLarge {
			t.Errorf("DetectLanguage(%q, %q) = %+v, want %s/%s/%s",
				tc.path, tc.reported, got, tc.wantID, tc.wantName, tc.wantLarge)
		}
	}
}

// A normal file must still be resolved by the reported language when there is
// no special filename match.
func TestDetectLanguageReportedLangStillApplies(t *testing.T) {
	got := DetectLanguage("/p/main.go", "go")
	if got.ID != "go" {
		t.Errorf("expected go, got %+v", got)
	}
	got = DetectLanguage("/p/unknown.xyz", "python")
	if got.ID != "python" {
		t.Errorf("expected reported python for extensionless match, got %+v", got)
	}
}
