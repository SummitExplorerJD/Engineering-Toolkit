package skills

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"golang.org/x/mod/semver"
)

var (
	idPattern            = regexp.MustCompile(`^[a-z][a-z0-9]*(?:-[a-z0-9]+)*$`)
	tokenPattern         = regexp.MustCompile(`^[a-z][a-z0-9]*(?:[._-][a-z0-9]+)*$`)
	hashPattern          = regexp.MustCompile(`^sha256:[0-9a-f]{64}$`)
	windowsVolumePattern = regexp.MustCompile(`^[A-Za-z]:`)
)

func validID(value string) bool {
	return idPattern.MatchString(value)
}

func validateManifest(manifest Manifest, directory string) []Diagnostic {
	var findings []Diagnostic
	if manifest.SchemaVersion != ManifestSchema {
		findings = append(findings, diagnostic("schema.unsupported", "skill.yaml", fmt.Sprintf("schema_version must be %q", ManifestSchema)))
	}
	if !validID(manifest.ID) {
		findings = append(findings, diagnostic("identity.invalid", "skill.yaml", "id must be a normalized lowercase hyphenated identifier"))
	} else if manifest.ID != directory {
		findings = append(findings, diagnostic("identity.directory_mismatch", "skill.yaml", "id must exactly match the package directory name"))
	}
	if !validVersion(manifest.Version) {
		findings = append(findings, diagnostic("version.invalid", "skill.yaml", "version must be semantic versioning without a leading v"))
	}
	if strings.TrimSpace(manifest.Description) == "" || manifest.Description != strings.TrimSpace(manifest.Description) {
		findings = append(findings, diagnostic("description.invalid", "skill.yaml", "description must be non-empty and have no surrounding whitespace"))
	}
	if manifest.Instructions != "instructions.md" {
		findings = append(findings, diagnostic("instructions.invalid", "skill.yaml", "instructions must be instructions.md"))
	}
	findings = append(findings, validateTokens("capability", manifest.Capabilities)...)
	findings = append(findings, validateValueSpecs("input", manifest.Inputs, true)...)
	findings = append(findings, validateValueSpecs("output", manifest.Outputs, false)...)
	if manifest.Permissions == nil {
		findings = append(findings, diagnostic("permissions.required", "skill.yaml", "permissions is required; use an empty mapping when no capabilities are requested"))
	} else {
		findings = append(findings, validatePermissions(*manifest.Permissions)...)
	}
	if manifest.Compatibility == nil {
		findings = append(findings, diagnostic("compatibility.required", "skill.yaml", "compatibility is required"))
	} else {
		if !validVersion(manifest.Compatibility.MinimumEngkit) {
			findings = append(findings, diagnostic("compatibility.minimum_engkit", "skill.yaml", "compatibility.minimum_engkit must be semantic versioning without a leading v"))
		}
		findings = append(findings, validateTokens("provider", manifest.Compatibility.Providers)...)
	}
	findings = append(findings, validateDependencies(manifest.Dependencies, manifest.ID)...)
	if manifest.Ownership == nil {
		findings = append(findings, diagnostic("ownership.required", "skill.yaml", "ownership is required"))
	} else {
		if strings.TrimSpace(manifest.Ownership.Owner) == "" || manifest.Ownership.Owner != strings.TrimSpace(manifest.Ownership.Owner) {
			findings = append(findings, diagnostic("ownership.owner", "skill.yaml", "ownership.owner must be non-empty and have no surrounding whitespace"))
		}
		findings = append(findings, validateStrings("ownership.maintainer", manifest.Ownership.Maintainers)...)
	}
	if manifest.Provenance == nil {
		findings = append(findings, diagnostic("provenance.required", "skill.yaml", "provenance is required"))
	} else {
		if strings.TrimSpace(manifest.Provenance.Source) == "" || manifest.Provenance.Source != strings.TrimSpace(manifest.Provenance.Source) {
			findings = append(findings, diagnostic("provenance.source", "skill.yaml", "provenance.source must be non-empty and have no surrounding whitespace"))
		}
		if strings.TrimSpace(manifest.Provenance.Revision) == "" || manifest.Provenance.Revision != strings.TrimSpace(manifest.Provenance.Revision) {
			findings = append(findings, diagnostic("provenance.revision", "skill.yaml", "provenance.revision must be non-empty and have no surrounding whitespace"))
		}
	}
	return findings
}

func validateValueSpecs(label string, values []ValueSpec, allowRequired bool) []Diagnostic {
	var findings []Diagnostic
	seen := make(map[string]struct{})
	allowedTypes := map[string]bool{"string": true, "number": true, "boolean": true, "object": true, "array": true}
	for _, value := range values {
		if !tokenPattern.MatchString(value.Name) {
			findings = append(findings, diagnostic(label+".name", "skill.yaml", fmt.Sprintf("%s name %q is invalid", label, value.Name)))
		}
		if _, exists := seen[value.Name]; exists {
			findings = append(findings, diagnostic(label+".duplicate", "skill.yaml", fmt.Sprintf("%s %q is declared more than once", label, value.Name)))
		}
		seen[value.Name] = struct{}{}
		if !allowedTypes[value.Type] {
			findings = append(findings, diagnostic(label+".type", "skill.yaml", fmt.Sprintf("%s %q has unsupported type %q", label, value.Name, value.Type)))
		}
		if strings.TrimSpace(value.Description) == "" || value.Description != strings.TrimSpace(value.Description) {
			findings = append(findings, diagnostic(label+".description", "skill.yaml", fmt.Sprintf("%s %q requires a normalized description", label, value.Name)))
		}
		if !allowRequired && value.Required {
			findings = append(findings, diagnostic(label+".required", "skill.yaml", "outputs cannot be marked required"))
		}
	}
	return findings
}

func validatePermissions(permissions Permissions) []Diagnostic {
	var findings []Diagnostic
	findings = append(findings, validateStrings("permission.filesystem_read", permissions.FilesystemRead)...)
	findings = append(findings, validateStrings("permission.filesystem_write", permissions.FilesystemWrite)...)
	findings = append(findings, validateStrings("permission.network", permissions.Network)...)
	findings = append(findings, validateStrings("permission.process", permissions.Process)...)
	return findings
}

func validateStrings(code string, values []string) []Diagnostic {
	var findings []Diagnostic
	seen := make(map[string]struct{})
	for _, value := range values {
		if strings.TrimSpace(value) == "" || value != strings.TrimSpace(value) {
			findings = append(findings, diagnostic(code+".invalid", "skill.yaml", fmt.Sprintf("%s value %q is empty or has surrounding whitespace", code, value)))
		}
		if _, exists := seen[value]; exists {
			findings = append(findings, diagnostic(code+".duplicate", "skill.yaml", fmt.Sprintf("%s value %q is duplicated", code, value)))
		}
		seen[value] = struct{}{}
	}
	return findings
}

func validateTokens(label string, values []string) []Diagnostic {
	findings := validateStrings(label, values)
	for _, value := range values {
		if !tokenPattern.MatchString(value) {
			findings = append(findings, diagnostic(label+".invalid", "skill.yaml", fmt.Sprintf("%s %q is not a normalized token", label, value)))
		}
	}
	return findings
}

func validateDependencies(dependencies []Dependency, ownID string) []Diagnostic {
	var findings []Diagnostic
	seen := make(map[string]struct{})
	for _, dependency := range dependencies {
		if !validID(dependency.ID) {
			findings = append(findings, diagnostic("dependency.id", "skill.yaml", fmt.Sprintf("dependency id %q is invalid", dependency.ID)))
		}
		if dependency.ID == ownID {
			findings = append(findings, diagnostic("dependency.self", "skill.yaml", "a skill cannot depend on itself"))
		}
		if _, exists := seen[dependency.ID]; exists {
			findings = append(findings, diagnostic("dependency.duplicate", "skill.yaml", fmt.Sprintf("dependency %q is declared more than once", dependency.ID)))
		}
		seen[dependency.ID] = struct{}{}
		if !validVersion(dependency.Version) {
			findings = append(findings, diagnostic("dependency.version", "skill.yaml", fmt.Sprintf("dependency %q must use an exact semantic version without a leading v", dependency.ID)))
		}
	}
	return findings
}

func validVersion(value string) bool {
	return value != "" && !strings.HasPrefix(value, "v") && semver.IsValid("v"+value)
}

func validateResources(packageRoot string, manifest Manifest) []Diagnostic {
	var findings []Diagnostic
	declared := make(map[string]Resource)
	for _, resource := range manifest.Resources {
		if err := validateResourcePath(resource.Path); err != nil {
			findings = append(findings, diagnostic("resource.path", resource.Path, err.Error()))
			continue
		}
		if _, exists := declared[resource.Path]; exists {
			findings = append(findings, diagnostic("resource.duplicate", resource.Path, "resource path is declared more than once"))
			continue
		}
		declared[resource.Path] = resource
		if !validResourceKind(resource.Path, resource.Kind) {
			findings = append(findings, diagnostic("resource.kind", resource.Path, "resource kind does not match its canonical location"))
		}
		if !hashPattern.MatchString(resource.SHA256) {
			findings = append(findings, diagnostic("resource.sha256", resource.Path, "sha256 must use sha256:<64 lowercase hexadecimal characters>"))
		}
	}
	if _, exists := declared["instructions.md"]; !exists {
		findings = append(findings, diagnostic("resource.instructions", "instructions.md", "instructions.md must be declared as an instructions resource"))
	}

	actual := make(map[string]string)
	err := filepath.WalkDir(packageRoot, func(filename string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if filename == packageRoot {
			return nil
		}
		relative, err := filepath.Rel(packageRoot, filename)
		if err != nil {
			return err
		}
		relative = filepath.ToSlash(relative)
		if entry.Type()&os.ModeSymlink != 0 {
			findings = append(findings, diagnostic("resource.symlink", relative, "symbolic links are not allowed in canonical skill packages"))
			if entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if entry.IsDir() {
			if !validResourceDirectory(relative) {
				findings = append(findings, diagnostic("resource.directory", relative, "package directories must be under references, scripts, templates, or assets"))
			}
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			findings = append(findings, diagnostic("resource.type", relative, "package resources must be regular files"))
			return nil
		}
		if relative == "skill.yaml" {
			return nil
		}
		digest, err := fileDigest(filename)
		if err != nil {
			return err
		}
		actual[relative] = digest
		return nil
	})
	if err != nil {
		findings = append(findings, diagnostic("resource.walk", filepath.ToSlash(packageRoot), err.Error()))
		return findings
	}

	for relative, digest := range actual {
		resource, exists := declared[relative]
		if !exists {
			findings = append(findings, diagnostic("resource.undeclared", relative, "managed package file is not declared in skill.yaml"))
			continue
		}
		if hashPattern.MatchString(resource.SHA256) && resource.SHA256 != digest {
			findings = append(findings, diagnostic("resource.integrity", relative, "resource content does not match its declared sha256"))
		}
	}
	for relative := range declared {
		if _, exists := actual[relative]; !exists {
			findings = append(findings, diagnostic("resource.missing", relative, "declared resource does not exist as a regular file"))
		}
	}
	return findings
}

func validateResourcePath(value string) error {
	native := filepath.FromSlash(value)
	if value == "" || strings.Contains(value, `\`) || windowsVolumePattern.MatchString(value) || path.IsAbs(value) || filepath.IsAbs(native) || filepath.VolumeName(native) != "" || path.Clean(value) != value || value == "." || value == ".." || strings.HasPrefix(value, "../") {
		return fmt.Errorf("resource path %q must be a normalized relative slash path", value)
	}
	return nil
}

func validResourceDirectory(value string) bool {
	for _, root := range []string{"references", "scripts", "templates", "assets"} {
		if value == root || strings.HasPrefix(value, root+"/") {
			return true
		}
	}
	return false
}

func validResourceKind(resourcePath, kind string) bool {
	switch kind {
	case "instructions":
		return resourcePath == "instructions.md"
	case "reference":
		return strings.HasPrefix(resourcePath, "references/")
	case "script":
		return strings.HasPrefix(resourcePath, "scripts/")
	case "template":
		return strings.HasPrefix(resourcePath, "templates/")
	case "asset":
		return strings.HasPrefix(resourcePath, "assets/")
	default:
		return false
	}
}

func fileDigest(filename string) (string, error) {
	file, err := os.Open(filename)
	if err != nil {
		return "", err
	}
	defer file.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}
	return "sha256:" + hex.EncodeToString(hash.Sum(nil)), nil
}
