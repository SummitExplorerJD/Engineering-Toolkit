package skills

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// FileSnapshot binds a package-relative path to its observed content.
type FileSnapshot struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
}

// Snapshot is an immutable, read-only view used to construct mutation plans.
type Snapshot struct {
	PackageRoot   string         `json:"package_root"`
	Package       Package        `json:"package"`
	Directories   []string       `json:"directories"`
	Files         []FileSnapshot `json:"files"`
	PackageSHA256 string         `json:"package_sha256"`
}

type snapshotDigestBody struct {
	Directories []string       `json:"directories"`
	Files       []FileSnapshot `json:"files"`
}

// SnapshotSource validates and snapshots a package directory supplied as
// explicit local input. It never writes to that directory.
func SnapshotSource(source string) (Snapshot, error) {
	if strings.TrimSpace(source) == "" {
		return Snapshot{}, errors.New("source package path is required")
	}
	absSource, err := filepath.Abs(source)
	if err != nil {
		return Snapshot{}, fmt.Errorf("resolve source package: %w", err)
	}
	absSource = filepath.Clean(absSource)
	info, err := os.Lstat(absSource)
	if err != nil {
		return Snapshot{}, fmt.Errorf("inspect source package: %w", err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return Snapshot{}, errors.New("source package must not be a symbolic link")
	}
	if !info.IsDir() {
		return Snapshot{}, errors.New("source package is not a directory")
	}
	resolved, err := filepath.EvalSymlinks(absSource)
	if err != nil {
		return Snapshot{}, fmt.Errorf("resolve source package links: %w", err)
	}
	resolved = filepath.Clean(resolved)
	info, err = os.Lstat(resolved)
	if err != nil {
		return Snapshot{}, fmt.Errorf("reinspect source package: %w", err)
	}
	packageState := inspectPackage(filepath.Dir(resolved), filepath.Base(resolved), info)
	if packageState.Classification != ClassificationCanonical {
		return Snapshot{}, packageStateError("source", packageState)
	}
	return stableSnapshot(resolved, packageState)
}

// SnapshotCanonical snapshots one existing canonical workspace package.
func SnapshotCanonical(root, id string) (Snapshot, error) {
	inspection, err := Inspect(root, id)
	if err != nil {
		return Snapshot{}, err
	}
	if inspection.Package.Classification != ClassificationCanonical {
		return Snapshot{}, packageStateError("target", inspection.Package)
	}
	packageRoot := filepath.Join(filepath.FromSlash(inspection.Root), ".agents", "skills", id)
	return stableSnapshot(packageRoot, inspection.Package)
}

func stableSnapshot(packageRoot string, initial Package) (Snapshot, error) {
	first, err := inventory(packageRoot, initial)
	if err != nil {
		return Snapshot{}, err
	}
	info, err := os.Lstat(packageRoot)
	if err != nil {
		return Snapshot{}, fmt.Errorf("reinspect skill package: %w", err)
	}
	secondState := inspectPackage(filepath.Dir(packageRoot), filepath.Base(packageRoot), info)
	if secondState.Classification != ClassificationCanonical {
		return Snapshot{}, errors.New("skill package changed while it was being inspected")
	}
	second, err := inventory(packageRoot, secondState)
	if err != nil {
		return Snapshot{}, err
	}
	if first.PackageSHA256 != second.PackageSHA256 {
		return Snapshot{}, errors.New("skill package changed while it was being inspected")
	}
	return second, nil
}

func inventory(packageRoot string, packageState Package) (Snapshot, error) {
	directories := []string{}
	files := []FileSnapshot{}
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
			return fmt.Errorf("snapshot path %s is a symbolic link", relative)
		}
		if entry.IsDir() {
			directories = append(directories, relative)
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("snapshot path %s is not a regular file", relative)
		}
		// The root SKILL.md is a provider-owned Codex sidecar and must not
		// influence the immutable canonical package snapshot.
		if relative == "SKILL.md" {
			return nil
		}
		digest, err := fileDigest(filename)
		if err != nil {
			return fmt.Errorf("hash snapshot path %s: %w", relative, err)
		}
		files = append(files, FileSnapshot{Path: relative, SHA256: digest})
		return nil
	})
	if err != nil {
		return Snapshot{}, fmt.Errorf("snapshot skill package: %w", err)
	}
	sort.Strings(directories)
	sort.Slice(files, func(i, j int) bool { return files[i].Path < files[j].Path })
	body := snapshotDigestBody{Directories: directories, Files: files}
	encoded, err := json.Marshal(body)
	if err != nil {
		return Snapshot{}, fmt.Errorf("encode skill snapshot: %w", err)
	}
	digest := sha256.Sum256(encoded)
	return Snapshot{
		PackageRoot:   filepath.ToSlash(packageRoot),
		Package:       packageState,
		Directories:   directories,
		Files:         files,
		PackageSHA256: "sha256:" + hex.EncodeToString(digest[:]),
	}, nil
}

func packageStateError(role string, packageState Package) error {
	codes := make([]string, 0, len(packageState.Diagnostics))
	for _, finding := range packageState.Diagnostics {
		codes = append(codes, finding.Code)
	}
	if len(codes) == 0 {
		return fmt.Errorf("%s skill %q is %s, not canonical", role, packageState.Directory, packageState.Classification)
	}
	return fmt.Errorf("%s skill %q is %s: %s", role, packageState.Directory, packageState.Classification, strings.Join(codes, ", "))
}
