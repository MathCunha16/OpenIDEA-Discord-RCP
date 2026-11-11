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
