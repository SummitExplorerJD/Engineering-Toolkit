package skills

import "testing"

func TestPortableIdentifiersAndResourcePaths(t *testing.T) {
	for _, id := range []string{"con", "prn", "com1", "lpt9"} {
		if validID(id) {
			t.Errorf("validID(%q) accepted a reserved Windows name", id)
		}
	}
	for _, path := range []string{
		"assets/con.txt",
		"references/name.",
		"templates/a:b.txt",
		"scripts/bad?.sh",
	} {
		if err := validateResourcePath(path); err == nil {
			t.Errorf("validateResourcePath(%q) accepted a non-portable path", path)
		}
	}
	if err := validateResourcePath("assets/portable-name.txt"); err != nil {
		t.Fatalf("portable path rejected: %v", err)
	}
}
