package providersync

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"sort"

	"github.com/SummitExplorerJD/Engineering-Toolkit/internal/skills"
	"gopkg.in/yaml.v3"
)

type skillFrontmatter struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
}

func renderSnapshot(snapshot skills.Snapshot, adapter AdapterReference) ([]renderedFile, Mapping, SourceReference, error) {
	manifest := snapshot.Package.Manifest
	if manifest == nil {
		return nil, Mapping{}, SourceReference{}, errors.New("canonical skill snapshot has no manifest")
	}
	files, err := readSnapshotFiles(snapshot)
	if err != nil {
		return nil, Mapping{}, SourceReference{}, err
	}
	manifestDigest, ok := snapshotFileDigest(snapshot, "skill.yaml")
	if !ok {
		return nil, Mapping{}, SourceReference{}, errors.New("canonical skill snapshot has no skill.yaml digest")
	}
	source := SourceReference{ID: manifest.ID, Version: manifest.Version, PackageSHA256: snapshot.PackageSHA256, ManifestSHA256: manifestDigest}
	mapping := Mapping{SkillID: manifest.ID, Quality: "exact", Findings: []Finding{}}

	if len(manifest.Compatibility.Providers) > 0 && !contains(manifest.Compatibility.Providers, adapter.Provider) {
		mapping.Findings = append(mapping.Findings, Finding{Severity: "blocking", Code: "mapping.provider_incompatible", Path: "skill.yaml", Message: "canonical compatibility.providers does not include " + adapter.Provider})
		mapping.Quality = "unsupported"
	}
	if !permissionsEmpty(*manifest.Permissions) {
		mapping.Findings = append(mapping.Findings, Finding{Severity: "blocking", Code: "mapping.permissions_unmapped", Path: "skill.yaml", Message: "provider runtime policy cannot represent canonical permission requests"})
		mapping.Quality = worseQuality(mapping.Quality, "lossy")
	}
	if len(manifest.Dependencies) > 0 {
		mapping.Findings = append(mapping.Findings, Finding{Severity: "blocking", Code: "mapping.dependency_unresolved", Path: "skill.yaml", Message: "provider synchronization does not resolve canonical dependencies"})
		mapping.Quality = "unsupported"
	}
	if len(manifest.Capabilities)+len(manifest.Inputs)+len(manifest.Outputs) > 0 {
		mapping.Findings = append(mapping.Findings, Finding{Severity: "info", Code: "mapping.canonical_metadata_only", Path: "skill.yaml", Message: "structured capabilities, inputs, and outputs remain canonical planning metadata"})
	}

	instructions, ok := files[manifest.Instructions]
	if !ok {
		return nil, Mapping{}, SourceReference{}, errors.New("canonical instructions are absent from the stable snapshot")
	}
	frontmatter, err := yaml.Marshal(skillFrontmatter{Name: manifest.ID, Description: manifest.Description})
	if err != nil {
		return nil, Mapping{}, SourceReference{}, fmt.Errorf("encode provider skill frontmatter: %w", err)
	}
	body := normalizeMarkdown(instructions)
	skillData := make([]byte, 0, len(frontmatter)+len(body)+9)
	skillData = append(skillData, []byte("---\n")...)
	skillData = append(skillData, frontmatter...)
	skillData = append(skillData, []byte("---\n\n")...)
	skillData = append(skillData, body...)
	outputs := []renderedFile{{path: path.Join(adapter.TargetRoot, manifest.ID, "SKILL.md"), sourcePath: manifest.Instructions, data: skillData}}

	resources := append([]skills.Resource(nil), manifest.Resources...)
	sort.Slice(resources, func(i, j int) bool { return resources[i].Path < resources[j].Path })
	for _, resource := range resources {
		if resource.Path == manifest.Instructions {
			continue
		}
		if resource.Kind == "script" {
			mapping.Findings = append(mapping.Findings, Finding{Severity: "blocking", Code: "mapping.script_unsupported", Path: resource.Path, Message: "executable resources are outside the initial synchronization scope"})
			mapping.Quality = "unsupported"
			continue
		}
		if adapter.Provider == "codex" {
			continue
		}
		data, ok := files[resource.Path]
		if !ok {
			return nil, Mapping{}, SourceReference{}, fmt.Errorf("canonical resource %s is absent from the stable snapshot", resource.Path)
		}
		outputs = append(outputs, renderedFile{path: path.Join(adapter.TargetRoot, manifest.ID, resource.Path), sourcePath: resource.Path, data: data})
	}
	sort.Slice(mapping.Findings, func(i, j int) bool {
		if mapping.Findings[i].Code != mapping.Findings[j].Code {
			return mapping.Findings[i].Code < mapping.Findings[j].Code
		}
		return mapping.Findings[i].Path < mapping.Findings[j].Path
	})
	return outputs, mapping, source, nil
}

func readSnapshotFiles(snapshot skills.Snapshot) (map[string][]byte, error) {
	files := make(map[string][]byte, len(snapshot.Files))
	for _, item := range snapshot.Files {
		data, err := os.ReadFile(pathFromSnapshot(snapshot.PackageRoot, item.Path))
		if err != nil {
			return nil, fmt.Errorf("read canonical skill path %s: %w", item.Path, err)
		}
		if digestBytes(data) != item.SHA256 {
			return nil, fmt.Errorf("canonical skill path %s changed while it was being read", item.Path)
		}
		files[item.Path] = data
	}
	return files, nil
}

func pathFromSnapshot(root, relative string) string {
	return filepath.Join(filepath.FromSlash(root), filepath.FromSlash(relative))
}

func snapshotFileDigest(snapshot skills.Snapshot, relative string) (string, bool) {
	for _, file := range snapshot.Files {
		if file.Path == relative {
			return file.SHA256, true
		}
	}
	return "", false
}

func normalizeMarkdown(data []byte) []byte {
	data = bytes.ReplaceAll(data, []byte("\r\n"), []byte("\n"))
	data = bytes.ReplaceAll(data, []byte("\r"), []byte("\n"))
	return append(bytes.TrimRight(data, "\n"), '\n')
}

func permissionsEmpty(value skills.Permissions) bool {
	return len(value.FilesystemRead) == 0 && len(value.FilesystemWrite) == 0 && len(value.Network) == 0 && len(value.Process) == 0
}

func contains(values []string, wanted string) bool {
	for _, value := range values {
		if value == wanted {
			return true
		}
	}
	return false
}

func worseQuality(current, candidate string) string {
	rank := map[string]int{"exact": 0, "lossy": 1, "unsupported": 2}
	if rank[candidate] > rank[current] {
		return candidate
	}
	return current
}

func digestBytes(data []byte) string {
	digest := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(digest[:])
}
