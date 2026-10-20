package schemautil

import (
	"fmt"
	"io/fs"
	"path"
	"sort"
	"strings"
)

// RegularInventory returns sorted regular file paths below root and rejects links and special files.
func RegularInventory(fsys fs.FS, root string) ([]string, error) {
	items := make([]string, 0)
	err := fs.WalkDir(fsys, root, func(filename string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if filename == root {
			if !entry.IsDir() {
				return fmt.Errorf("%q must be a directory", root)
			}
			return nil
		}
		if entry.Type()&fs.ModeSymlink != 0 {
			return fmt.Errorf("%q must not be a symbolic link", filename)
		}
		if entry.IsDir() {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("%q must be a regular file", filename)
		}
		relative := filename
		if root != "." {
			prefix := strings.TrimSuffix(root, "/") + "/"
			if !strings.HasPrefix(filename, prefix) {
				return fmt.Errorf("%q escapes root %q", filename, root)
			}
			relative = strings.TrimPrefix(filename, prefix)
		}
		items = append(items, path.Clean(relative))
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(items)
	return items, nil
}
