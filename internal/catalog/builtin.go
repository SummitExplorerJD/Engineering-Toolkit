package catalog

import (
	"embed"
	"io/fs"
)

//go:embed builtin
var embedded embed.FS

func Builtin() (Catalog, fs.FS, error) {
	catalogFS, err := fs.Sub(embedded, "builtin")
	if err != nil {
		return Catalog{}, nil, err
	}
	loaded, err := Load(catalogFS, "catalog.yaml")
	if err != nil {
		return Catalog{}, nil, err
	}
	return loaded, catalogFS, nil
}
