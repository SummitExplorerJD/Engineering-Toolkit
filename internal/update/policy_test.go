package update

import (
	"strings"
	"testing"
)

func TestTargetPathAndPolicy(t *testing.T) {
	platform := Platform{OS: "windows", Arch: "amd64"}
	got, err := TargetPath(StableChannel, "v1.2.3", platform)
	if err != nil {
		t.Fatal(err)
	}
	want := "releases/stable/v1.2.3/windows-amd64/engkit.exe"
	if got != want {
		t.Fatalf("TargetPath() = %q, want %q", got, want)
	}
	target := Target{
		Version:  "v1.2.3",
		Channel:  StableChannel,
		Platform: platform,
		Path:     want,
		Length:   42,
		SHA256:   strings.Repeat("a", 64),
		Format:   ExecutableFormatRaw,
	}
	if err := ValidateTarget(target, StableChannel, "v1.2.3", platform); err != nil {
		t.Fatal(err)
	}
}

func TestPolicyRejectsUnsafeValues(t *testing.T) {
	tests := []struct {
		name string
		fn   func() error
	}{
		{"prerelease", func() error { return ValidateStableVersion("v1.2.3-rc.1") }},
		{"noncanonical", func() error { return ValidateStableVersion("v1.2") }},
		{"downgrade", func() error { return CompareCandidate("v2.0.0", "v1.9.9") }},
		{"channel", func() error { _, err := TargetPath("beta", "v1.2.3", Platform{OS: "linux", Arch: "amd64"}); return err }},
		{"platform", func() error { return ValidatePlatform(Platform{OS: "plan9", Arch: "amd64"}) }},
		{"http", func() error {
			return ValidateRepository(Repository{MetadataURL: "http://example.com/metadata", TargetsURL: "https://example.com/targets", Root: []byte("root")})
		}},
		{"userinfo", func() error {
			return ValidateRepository(Repository{MetadataURL: "https://user@example.com/metadata", TargetsURL: "https://example.com/targets", Root: []byte("root")})
		}},
		{"traversal", func() error {
			return ValidateRepository(Repository{MetadataURL: "https://example.com/%2e%2e/metadata", TargetsURL: "https://example.com/targets", Root: []byte("root")})
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if err := test.fn(); err == nil {
				t.Fatal("expected error")
			}
		})
	}
}

func TestValidateTargetRejectsMetadataMismatch(t *testing.T) {
	platform := Platform{OS: "linux", Arch: "arm64"}
	path, err := TargetPath(StableChannel, "v1.0.0", platform)
	if err != nil {
		t.Fatal(err)
	}
	valid := Target{
		Version: "v1.0.0", Channel: StableChannel, Platform: platform,
		Path: path, Length: 1, SHA256: strings.Repeat("0", 64), Format: ExecutableFormatRaw,
	}
	mutations := []func(*Target){
		func(v *Target) { v.Path = "../engkit" },
		func(v *Target) { v.Channel = "beta" },
		func(v *Target) { v.Length = 0 },
		func(v *Target) { v.SHA256 = "abcd" },
		func(v *Target) { v.Format = "zip" },
	}
	for i, mutate := range mutations {
		candidate := valid
		mutate(&candidate)
		if err := ValidateTarget(candidate, StableChannel, "v1.0.0", platform); err == nil {
			t.Fatalf("mutation %d was accepted", i)
		}
	}
}
