package sqliteindex

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

const maxIndexedContentBytes = 1 << 20

type artifact struct {
	Path       string
	Kind       string
	SizeBytes  int64
	ModifiedNS int64
	SHA256     string
	Content    *string
}

func collectArtifacts(root string) ([]artifact, string, error) {
	candidates, err := canonicalCandidates(root)
	if err != nil {
		return nil, "", err
	}

	artifacts := make([]artifact, 0, len(candidates))
	for _, relative := range candidates {
		item, err := readArtifact(root, relative)
		if err != nil {
			return nil, "", err
		}
		artifacts = append(artifacts, item)
	}

	fingerprint := sha256.New()
	for _, item := range artifacts {
		_, _ = io.WriteString(fingerprint, item.Path)
		_, _ = fingerprint.Write([]byte{0})
		_, _ = io.WriteString(fingerprint, item.Kind)
		_, _ = fingerprint.Write([]byte{0})
		_, _ = io.WriteString(fingerprint, item.SHA256)
		_, _ = fingerprint.Write([]byte{0})
		_, _ = io.WriteString(fingerprint, strconv.FormatInt(item.SizeBytes, 10))
		_, _ = fingerprint.Write([]byte{0})
		_, _ = io.WriteString(fingerprint, strconv.FormatInt(item.ModifiedNS, 10))
		_, _ = fingerprint.Write([]byte{0})
	}
	return artifacts, "sha256:" + hex.EncodeToString(fingerprint.Sum(nil)), nil
}

func canonicalCandidates(root string) ([]string, error) {
	seen := make(map[string]struct{})
	for _, relative := range []string{"AGENTS.md", "README.md", ".engkit/config.yaml"} {
		path := filepath.Join(root, filepath.FromSlash(relative))
		info, err := os.Lstat(path)
		if err == nil && info.Mode().IsRegular() {
			seen[filepath.ToSlash(relative)] = struct{}{}
			continue
		}
		if err != nil && !os.IsNotExist(err) {
			return nil, fmt.Errorf("inspect context source %s: %w", path, err)
		}
	}

	for _, relativeRoot := range []string{
		"docs",
		".agents",
		".engkit/plans",
		".engkit/journals",
		".engkit/checkpoints",
	} {
		base := filepath.Join(root, filepath.FromSlash(relativeRoot))
		if _, err := os.Lstat(base); os.IsNotExist(err) {
			continue
		} else if err != nil {
			return nil, fmt.Errorf("inspect context source root %s: %w", base, err)
		}

		err := filepath.WalkDir(base, func(path string, entry os.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if entry.Type()&os.ModeSymlink != 0 {
				if entry.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
			if entry.IsDir() {
				return nil
			}
			info, err := entry.Info()
			if err != nil {
				return err
			}
			if !info.Mode().IsRegular() {
				return nil
			}
			relative, err := filepath.Rel(root, path)
			if err != nil {
				return err
			}
			seen[filepath.ToSlash(relative)] = struct{}{}
			return nil
		})
		if err != nil {
			return nil, fmt.Errorf("walk context source root %s: %w", base, err)
		}
	}

	paths := make([]string, 0, len(seen))
	for path := range seen {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	return paths, nil
}

func readArtifact(root, relative string) (artifact, error) {
	path := filepath.Join(root, filepath.FromSlash(relative))
	before, err := os.Lstat(path)
	if err != nil {
		return artifact{}, fmt.Errorf("inspect context source %s: %w", path, err)
	}
	if !before.Mode().IsRegular() {
		return artifact{}, fmt.Errorf("context source is not a regular file: %s", path)
	}

	file, err := os.Open(path)
	if err != nil {
		return artifact{}, fmt.Errorf("open context source %s: %w", path, err)
	}
	defer file.Close()

	hash := sha256.New()
	kind := artifactKind(relative)
	var content *string
	if kind == "documentation" && before.Size() <= maxIndexedContentBytes {
		bytes, err := io.ReadAll(io.TeeReader(file, hash))
		if err != nil {
			return artifact{}, fmt.Errorf("read context source %s: %w", path, err)
		}
		text := string(bytes)
		content = &text
	} else if _, err := io.Copy(hash, file); err != nil {
		return artifact{}, fmt.Errorf("hash context source %s: %w", path, err)
	}

	after, err := file.Stat()
	if err != nil {
		return artifact{}, fmt.Errorf("reinspect context source %s: %w", path, err)
	}
	if !os.SameFile(before, after) || before.Size() != after.Size() || before.ModTime() != after.ModTime() {
		return artifact{}, fmt.Errorf("context source changed while indexing: %s", path)
	}

	return artifact{
		Path:       filepath.ToSlash(relative),
		Kind:       kind,
		SizeBytes:  before.Size(),
		ModifiedNS: before.ModTime().UnixNano(),
		SHA256:     "sha256:" + hex.EncodeToString(hash.Sum(nil)),
		Content:    content,
	}, nil
}

func artifactKind(relative string) string {
	clean := filepath.ToSlash(relative)
	switch {
	case strings.HasSuffix(strings.ToLower(clean), ".md"):
		return "documentation"
	case clean == ".engkit/config.yaml":
		return "configuration"
	case strings.HasPrefix(clean, ".engkit/plans/"):
		return "plan"
	case strings.HasPrefix(clean, ".engkit/journals/"):
		return "journal"
	case strings.HasPrefix(clean, ".engkit/checkpoints/"):
		return "checkpoint"
	default:
		return "artifact"
	}
}
