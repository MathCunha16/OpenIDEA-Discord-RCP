package ide

import (
	"path/filepath"
	"strings"
)

// DisambiguateFileName returns the shortest display name for activePath that
// distinguishes it from the other open files. With no name collision it is just
// the basename; when two open files share a basename, the smallest number of
// parent directories is prepended until the names differ.
//
// Examples:
//
//	only /app/src/models.go                       -> models.go
//	/app/src/models.go + /app/rcp/models.go      -> src/models.go, rcp/models.go
func DisambiguateFileName(activePath string, openPaths []string) string {
	base := filepath.Base(activePath)
	if base == "" || base == "." || base == string(filepath.Separator) {
		return base
	}

	var colliding []string
	for _, p := range openPaths {
		if p == "" || p == activePath {
			continue
		}
		if filepath.Base(p) == base {
			colliding = append(colliding, p)
		}
	}
	if len(colliding) == 0 {
		return base
	}

	// Build the candidate set (active first) and grow the number of trailing
	// path components until all candidates are distinct.
	candidates := append([]string{activePath}, colliding...)
	for depth := 2; ; depth++ {
		parts := make([]string, 0, len(candidates))
		unique := true
		seen := make(map[string]bool, len(candidates))
		for _, p := range candidates {
			name := trailingPath(p, depth)
			if seen[name] {
				unique = false
				break
			}
			seen[name] = true
			parts = append(parts, name)
		}
		if unique {
			return parts[0]
		}
		// Absolute fallback: once we run out of components, use full paths.
		if depth > maxDepth(candidates) {
			return filepath.ToSlash(activePath)
		}
	}
}

// trailingPath returns the last depth components of a path, slash-separated.
func trailingPath(p string, depth int) string {
	clean := filepath.ToSlash(filepath.Clean(p))
	parts := strings.Split(clean, "/")
	if len(parts) <= depth {
		return strings.Join(parts, "/")
	}
	return strings.Join(parts[len(parts)-depth:], "/")
}

func maxDepth(paths []string) int {
	max := 2
	for _, p := range paths {
		n := len(strings.Split(filepath.ToSlash(filepath.Clean(p)), "/"))
		if n > max {
			max = n
		}
	}
	return max + 1
}
