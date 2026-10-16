package update

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestStatePathsForPlatforms(t *testing.T) {
	home := "/home/engineer"
	local := `C:\Users\engineer\AppData\Local`
	tests := []struct {
		goos  string
		paths StatePaths
	}{
		{"windows", mustStatePaths(t, "windows", home, local, "", "")},
		{"darwin", mustStatePaths(t, "darwin", home, local, "", "")},
		{"linux", mustStatePaths(t, "linux", home, local, "", "")},
	}
	for _, test := range tests {
		t.Run(test.goos, func(t *testing.T) {
			if test.goos == "windows" && (!filepath.IsAbs(test.paths.MetadataDir) || !filepath.IsAbs(test.paths.TargetCache)) {
				t.Fatalf("paths are not absolute: %+v", test.paths)
			}
			if strings.Contains(test.paths.MetadataDir, ".engkit") || strings.Contains(test.paths.TargetCache, ".engkit") {
				t.Fatalf("workspace state leaked into application paths: %+v", test.paths)
			}
		})
	}
}

func TestStatePathsForLinuxHonorsXDG(t *testing.T) {
	state := "/state"
	cache := "/cache"
	got, err := StatePathsFor("linux", "", "", state, cache)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(got.MetadataDir, "/state/engkit") {
		t.Fatalf("metadata path %q does not use XDG state root", got.MetadataDir)
	}
	if !strings.HasPrefix(got.TargetCache, "/cache/engkit") {
		t.Fatalf("cache path %q does not use XDG cache root", got.TargetCache)
	}
}

func mustStatePaths(t *testing.T, goos, home, local, state, cache string) StatePaths {
	t.Helper()
	paths, err := StatePathsFor(goos, home, local, state, cache)
	if err != nil {
		t.Fatal(err)
	}
	return paths
}
