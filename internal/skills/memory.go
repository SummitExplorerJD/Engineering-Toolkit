package skills

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"path"
	"sort"
	"strings"
)

// MemoryFile is one rendered canonical-package file held entirely in memory.
type MemoryFile struct {
	Path string
	Data []byte
}

// ValidateMemory validates a rendered manifest and its resources without
// creating a temporary package on disk.
func ValidateMemory(manifest Manifest, files []MemoryFile) error {
	findings := validateManifest(manifest, manifest.ID)
	actual := make(map[string][]byte, len(files))
	for _, file := range files {
		if file.Path == "skill.yaml" {
			findings = append(findings, diagnostic("resource.reserved", file.Path, "skill.yaml is generated from the manifest and cannot be a resource"))
			continue
		}
		if err := validateResourcePath(file.Path); err != nil {
			findings = append(findings, diagnostic("resource.path", file.Path, err.Error()))
			continue
		}
		if _, exists := actual[file.Path]; exists {
			findings = append(findings, diagnostic("resource.duplicate", file.Path, "rendered resource path is duplicated"))
			continue
		}
		actual[file.Path] = file.Data
	}
	for relative := range actual {
		for parent := path.Dir(relative); parent != "."; parent = path.Dir(parent) {
			if _, exists := actual[parent]; exists {
				findings = append(findings, diagnostic("resource.collision", relative, "rendered resources have a file/directory collision"))
				break
			}
		}
	}

	declared := make(map[string]Resource, len(manifest.Resources))
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
	for relative := range declared {
		for parent := path.Dir(relative); parent != "."; parent = path.Dir(parent) {
			if _, exists := declared[parent]; exists {
				findings = append(findings, diagnostic("resource.collision", relative, "declared resources have a file/directory collision"))
				break
			}
		}
	}
	if _, exists := declared["instructions.md"]; !exists {
		findings = append(findings, diagnostic("resource.instructions", "instructions.md", "instructions.md must be declared as an instructions resource"))
	}
	for relative, data := range actual {
		resource, exists := declared[relative]
		if !exists {
			findings = append(findings, diagnostic("resource.undeclared", relative, "rendered package file is not declared in skill.yaml"))
			continue
		}
		digest := sha256.Sum256(data)
		observed := "sha256:" + hex.EncodeToString(digest[:])
		if resource.SHA256 != observed {
			findings = append(findings, diagnostic("resource.integrity", relative, "rendered resource content does not match its declared sha256"))
		}
	}
	for relative := range declared {
		if _, exists := actual[relative]; !exists {
			findings = append(findings, diagnostic("resource.missing", relative, "declared rendered resource is missing"))
		}
	}
	if len(findings) == 0 {
		return nil
	}
	sort.Slice(findings, func(i, j int) bool {
		if findings[i].Path != findings[j].Path {
			return findings[i].Path < findings[j].Path
		}
		return findings[i].Code < findings[j].Code
	})
	messages := make([]string, 0, len(findings))
	for _, finding := range findings {
		messages = append(messages, fmt.Sprintf("%s: %s", finding.Code, finding.Message))
	}
	return fmt.Errorf("rendered canonical skill is invalid: %s", strings.Join(messages, "; "))
}
