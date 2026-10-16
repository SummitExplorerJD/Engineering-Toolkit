package update

import (
	"strings"
	"testing"
)

func TestStatePathsForPlatforms(t *testing.T) {
	tests := []struct {
		goos         string
		home         string
		localAppData string
		wantMetadata string
		wantCache    string
	}{
		{
			goos:         "windows",
			localAppData: `C:\Users\engineer\AppData\Local`,
			wantMetadata: `C:\Users\engineer\AppData\Local\EngineeringToolkit\update\metadata`,
			wantCache:    `C:\Users\engineer\AppData\Local\EngineeringToolkit\update\targets`,
		},
		{
			goos:         "darwin",
			home:         "/Users/engineer",
			wantMetadata: "/Users/engineer/Library/Application Support/EngineeringToolkit/update/metadata",
			wantCache:    "/Users/engineer/Library/Caches/EngineeringToolkit/update/targets",
		},
		{
			goos:         "linux",
			home:         "/home/engineer",
			wantMetadata: "/home/engineer/.local/state/engkit/update/metadata",
			wantCache:    "/home/engineer/.cache/engkit/update/targets",
		},
	}
	for _, test := range tests {
		t.Run(test.goos, func(t *testing.T) {
			paths, err := StatePathsFor(test.goos, test.home, test.localAppData, "", "")
			if err != nil {
				t.Fatal(err)
			}
			if paths.MetadataDir != test.wantMetadata {
				t.Fatalf("metadata path = %q, want %q", paths.MetadataDir, test.wantMetadata)
			}
			if paths.TargetCache != test.wantCache {
				t.Fatalf("target cache = %q, want %q", paths.TargetCache, test.wantCache)
			}
			if strings.Contains(paths.MetadataDir, ".engkit") || strings.Contains(paths.TargetCache, ".engkit") {
				t.Fatalf("workspace state leaked into application paths: %+v", paths)
			}
		})
	}
}

func TestStatePathsForWindowsAcceptsDriveAndUNCPaths(t *testing.T) {
	tests := []struct {
		name         string
		localAppData string
		wantPrefix   string
	}{
		{name: "drive", localAppData: `D:/Profiles/engineer/AppData/Local`, wantPrefix: `D:\Profiles\engineer\AppData\Local\EngineeringToolkit`},
		{name: "UNC", localAppData: `\\server\profiles\engineer`, wantPrefix: `\\server\profiles\engineer\EngineeringToolkit`},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			paths, err := StatePathsFor("windows", "", test.localAppData, "", "")
			if err != nil {
				t.Fatal(err)
			}
			if !strings.HasPrefix(paths.MetadataDir, test.wantPrefix) || !strings.HasPrefix(paths.TargetCache, test.wantPrefix) {
				t.Fatalf("paths do not use normalized Windows root %q: %+v", test.wantPrefix, paths)
			}
		})
	}
}

func TestStatePathsForRejectsRelativeRoots(t *testing.T) {
	tests := []struct {
		name         string
		goos         string
		home         string
		localAppData string
		xdgState     string
		xdgCache     string
	}{
		{name: "windows", goos: "windows", localAppData: `Users\engineer\AppData\Local`},
		{name: "darwin", goos: "darwin", home: "Users/engineer"},
		{name: "linux", goos: "linux", xdgState: "state", xdgCache: "cache"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := StatePathsFor(test.goos, test.home, test.localAppData, test.xdgState, test.xdgCache); err == nil {
				t.Fatal("expected relative roots to be rejected")
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
