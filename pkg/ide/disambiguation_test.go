package ide

import "testing"

func TestDisambiguateFileName(t *testing.T) {
	cases := []struct {
		name   string
		active string
		open   []string
		want   string
	}{
		{
			name:   "no collision keeps basename",
			active: "/app/src/models.go",
			open:   []string{"/app/src/models.go", "/app/main.go"},
			want:   "models.go",
		},
		{
			name:   "single open file keeps basename",
			active: "/app/src/models.go",
			open:   []string{"/app/src/models.go"},
			want:   "models.go",
		},
		{
			name:   "empty list keeps basename",
			active: "/app/src/models.go",
			want:   "models.go",
		},
		{
			name:   "two colliding names prepend parent",
			active: "/app/src/models.go",
			open:   []string{"/app/src/models.go", "/app/rcp/models.go"},
			want:   "src/models.go",
		},
		{
			name:   "collision on the other side too",
			active: "/app/rcp/models.go",
			open:   []string{"/app/src/models.go", "/app/rcp/models.go"},
			want:   "rcp/models.go",
		},
		{
			name:   "three-way collision with equal parents",
			active: "/a/x/models.go",
			open:   []string{"/a/x/models.go", "/b/x/models.go", "/c/x/models.go"},
			want:   "a/x/models.go",
		},
		{
			name:   "different basenames are unaffected",
			active: "/app/src/index.ts",
			open:   []string{"/app/src/index.ts", "/app/rcp/app.go"},
			want:   "index.ts",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := DisambiguateFileName(tc.active, tc.open); got != tc.want {
				t.Errorf("DisambiguateFileName() = %q, want %q", got, tc.want)
			}
		})
	}
}
