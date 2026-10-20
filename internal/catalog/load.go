package catalog

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/fs"
	"path"
	"sort"
	"strings"

	"github.com/SummitExplorerJD/Engineering-Toolkit/internal/schemautil"
)

func Load(fsys fs.FS, filename string) (Catalog, error) {
	data, err := fs.ReadFile(fsys, filename)
	if err != nil {
		return Catalog{}, fmt.Errorf("read catalog: %w", err)
	}
	var catalog Catalog
	if err := schemautil.DecodeYAML(data, maxDefinitionBytes, filename, &catalog); err != nil {
		return Catalog{}, err
	}
	if catalog.SchemaVersion != Schema {
		return Catalog{}, fmt.Errorf("schema_version must be %q", Schema)
	}
	if catalog.Templates == nil || catalog.Profiles == nil {
		return Catalog{}, fmt.Errorf("templates and profiles inventories are required")
	}
	if !schemautil.ValidKebab(catalog.ID) {
		return Catalog{}, fmt.Errorf("catalog id %q is invalid", catalog.ID)
	}
	if !schemautil.ValidSemver(catalog.Version) || !schemautil.ValidSemver(catalog.MinimumEngkit) {
		return Catalog{}, fmt.Errorf("catalog version or minimum_engkit is invalid")
	}
	base := path.Dir(filename)
	if err := validateEntries(fsys, base, "templates", catalog.Templates); err != nil {
		return Catalog{}, err
	}
	if err := validateEntries(fsys, base, "profiles", catalog.Profiles); err != nil {
		return Catalog{}, err
	}
	digest := sha256.Sum256(data)
	catalog.SHA256 = "sha256:" + hex.EncodeToString(digest[:])
	return catalog, nil
}

func validateEntries(fsys fs.FS, base, kind string, entries []Entry) error {
	seen := make(map[string]struct{}, len(entries))
	previous := ""
	for _, entry := range entries {
		if !schemautil.ValidKebab(entry.ID) || !schemautil.ValidSemver(entry.Version) {
			return fmt.Errorf("%s entry identity or version is invalid", kind)
		}
		key := entry.ID + "@" + entry.Version
		if _, exists := seen[key]; exists {
			return fmt.Errorf("duplicate %s entry %q", kind, key)
		}
		if previous != "" && key < previous {
			return fmt.Errorf("%s entries must be sorted by id and version", kind)
		}
		seen[key] = struct{}{}
		previous = key
		if err := schemautil.ValidateRelative(entry.Path); err != nil || !strings.HasPrefix(entry.Path, kind+"/") {
			return fmt.Errorf("%s entry %q has invalid path %q", kind, key, entry.Path)
		}
		if entry.Path != path.Join(kind, entry.ID, entry.Version) {
			return fmt.Errorf("%s entry %q path must match its identity and version", kind, key)
		}
		if !schemautil.ValidDigest(entry.SHA256) {
			return fmt.Errorf("%s entry %q has invalid digest", kind, key)
		}
		observed, err := DirectoryDigest(fsys, path.Join(base, entry.Path))
		if err != nil {
			return fmt.Errorf("digest %s entry %q: %w", kind, key, err)
		}
		if observed != entry.SHA256 {
			return fmt.Errorf("%s entry %q digest mismatch", kind, key)
		}
	}
	return nil
}

func DirectoryDigest(fsys fs.FS, root string) (string, error) {
	type item struct{ path, digest string }
	items := make([]item, 0)
	err := fs.WalkDir(fsys, root, func(filename string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if filename == root {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if info.Mode()&fs.ModeSymlink != 0 || (!entry.IsDir() && !info.Mode().IsRegular()) {
			return fmt.Errorf("catalog entry contains unsupported file %q", filename)
		}
		if entry.IsDir() {
			return nil
		}
		contents, err := fs.ReadFile(fsys, filename)
		if err != nil {
			return err
		}
		digest := sha256.Sum256(contents)
		relative := strings.TrimPrefix(filename, strings.TrimSuffix(root, "/")+"/")
		if relative == filename || relative == "" {
			return fmt.Errorf("catalog file %q escapes entry root %q", filename, root)
		}
		items = append(items, item{path: relative, digest: hex.EncodeToString(digest[:])})
		return nil
	})
	if err != nil {
		return "", err
	}
	if len(items) == 0 {
		return "", fmt.Errorf("catalog entry directory is empty")
	}
	sort.Slice(items, func(i, j int) bool { return items[i].path < items[j].path })
	hash := sha256.New()
	for _, item := range items {
		_, _ = hash.Write([]byte(item.path))
		_, _ = hash.Write([]byte{0})
		_, _ = hash.Write([]byte(item.digest))
		_, _ = hash.Write([]byte{0})
	}
	return "sha256:" + hex.EncodeToString(hash.Sum(nil)), nil
}

func Find(entries []Entry, id, version string) (Entry, bool) {
	index := sort.Search(len(entries), func(index int) bool {
		key := entries[index].ID + "@" + entries[index].Version
		return key >= id+"@"+version
	})
	if index < len(entries) && entries[index].ID == id && entries[index].Version == version {
		return entries[index], true
	}
	return Entry{}, false
}
