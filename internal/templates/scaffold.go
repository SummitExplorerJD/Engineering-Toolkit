package templates

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/SummitExplorerJD/Engineering-Toolkit/internal/schemautil"
	"github.com/SummitExplorerJD/Engineering-Toolkit/internal/skills"
	"gopkg.in/yaml.v3"
)

func Scaffold(output, id, version string) (result ScaffoldResult, err error) {
	return scaffold(output, id, version, os.Rename)
}

func scaffold(output, id, version string, publish func(string, string) error) (result ScaffoldResult, err error) {
	if !schemautil.ValidKebab(id) {
		return result, fmt.Errorf("template id %q is invalid", id)
	}
	if !schemautil.ValidSemver(version) {
		return result, fmt.Errorf("template version %q is invalid", version)
	}
	absolute, err := filepath.Abs(output)
	if err != nil {
		return result, fmt.Errorf("resolve output: %w", err)
	}
	absolute = filepath.Clean(absolute)
	if err := rejectLinkedAncestors(filepath.Dir(absolute)); err != nil {
		return result, err
	}
	if _, err := os.Lstat(absolute); err == nil {
		return result, fmt.Errorf("output already exists: %s", absolute)
	} else if !os.IsNotExist(err) {
		return result, fmt.Errorf("inspect output: %w", err)
	}

	parent := filepath.Dir(absolute)
	stage, err := os.MkdirTemp(parent, "engkit-template-staging-")
	if err != nil {
		return result, fmt.Errorf("create scaffold staging directory: %w", err)
	}
	published := false
	defer func() {
		if !published {
			_ = os.RemoveAll(stage)
		}
	}()

	instructions := []byte("# " + id + "\n\nReplace this text with provider-neutral skill instructions.\n")
	filesRoot := filepath.Join(stage, "files")
	if err := os.Mkdir(filesRoot, 0o700); err != nil {
		return result, fmt.Errorf("create scaffold files directory: %w", err)
	}
	if err := os.WriteFile(filepath.Join(filesRoot, "instructions.md.tmpl"), instructions, 0o600); err != nil {
		return result, fmt.Errorf("write scaffold instructions: %w", err)
	}
	digest := sha256.Sum256(instructions)
	template := Template{
		SchemaVersion: Schema,
		ID:            id,
		Version:       version,
		Description:   "Replace with a description of this skill template.",
		Skill: SkillSpec{
			ID:           id,
			Version:      version,
			Description:  "Replace with a description of the generated skill.",
			Instructions: "instructions.md",
			Permissions:  &skills.Permissions{},
			Compatibility: &skills.Compatibility{
				MinimumEngkit: "0.2.1",
			},
			Ownership: &skills.Ownership{Owner: "replace-with-owner"},
		},
		Files: []File{{
			Source:       "files/instructions.md.tmpl",
			Destination:  "instructions.md",
			Kind:         "instructions",
			MediaType:    "text",
			SourceSHA256: "sha256:" + hex.EncodeToString(digest[:]),
		}},
	}
	definition, err := yaml.Marshal(template)
	if err != nil {
		return result, fmt.Errorf("encode scaffold template: %w", err)
	}
	if err := os.WriteFile(filepath.Join(stage, "template.yaml"), definition, 0o600); err != nil {
		return result, fmt.Errorf("write scaffold template: %w", err)
	}
	if err := publish(stage, absolute); err != nil {
		return result, fmt.Errorf("publish scaffold: %w", err)
	}
	published = true
	return ScaffoldResult{
		SchemaVersion: ScaffoldSchema,
		Output:        filepath.ToSlash(absolute),
		TemplateID:    id,
		Version:       version,
		Created:       []string{"files/instructions.md.tmpl", "template.yaml"},
	}, nil
}

func rejectLinkedAncestors(start string) error {
	current := filepath.Clean(start)
	for {
		info, err := os.Lstat(current)
		if err != nil {
			return fmt.Errorf("inspect output ancestor %q: %w", current, err)
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("output must not use symbolic-link or junction ancestors")
		}
		if !info.IsDir() {
			return fmt.Errorf("output ancestor %q is not a directory", current)
		}
		parent := filepath.Dir(current)
		if strings.EqualFold(parent, current) {
			return nil
		}
		current = parent
	}
}
