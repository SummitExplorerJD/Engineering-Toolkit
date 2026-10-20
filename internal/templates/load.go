package templates

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/fs"
	"path"
	"regexp"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/SummitExplorerJD/Engineering-Toolkit/internal/schemautil"
)

var placeholderPattern = regexp.MustCompile(`\{\{engkit\.([a-z][a-z0-9]*(?:_[a-z0-9]+)*)\}\}`)

func Load(fsys fs.FS, definition string) (Template, error) {
	if path.Base(definition) != "template.yaml" {
		return Template{}, fmt.Errorf("template definition must be named template.yaml")
	}
	data, err := fs.ReadFile(fsys, definition)
	if err != nil {
		return Template{}, fmt.Errorf("read template definition: %w", err)
	}
	var template Template
	if err := schemautil.DecodeYAML(data, maxDefinitionBytes, definition, &template); err != nil {
		return Template{}, err
	}
	if err := Validate(fsys, path.Dir(definition), template); err != nil {
		return Template{}, err
	}
	return template, nil
}

func Validate(fsys fs.FS, root string, template Template) error {
	if template.SchemaVersion != Schema {
		return fmt.Errorf("schema_version must be %q", Schema)
	}
	if !schemautil.ValidKebab(template.ID) {
		return fmt.Errorf("template id %q is invalid", template.ID)
	}
	if !schemautil.ValidSemver(template.Version) {
		return fmt.Errorf("template version %q is invalid", template.Version)
	}
	if strings.TrimSpace(template.Description) != template.Description || template.Description == "" {
		return fmt.Errorf("template description must be normalized and non-empty")
	}
	variableNames, err := ValidateVariables(template.Variables)
	if err != nil {
		return err
	}
	if err := validateSkill(template.Skill); err != nil {
		return err
	}
	if len(template.Files) == 0 {
		return fmt.Errorf("template files must not be empty")
	}
	seenSource := make(map[string]struct{}, len(template.Files))
	seenDestination := make(map[string]struct{}, len(template.Files))
	for _, file := range template.Files {
		if err := schemautil.ValidateRelative(file.Source); err != nil || !strings.HasPrefix(file.Source, "files/") {
			return fmt.Errorf("template source %q must be a portable path below files/", file.Source)
		}
		if err := schemautil.ValidateRelative(file.Destination); err != nil {
			return fmt.Errorf("template destination: %w", err)
		}
		if _, exists := seenSource[strings.ToLower(file.Source)]; exists {
			return fmt.Errorf("duplicate or case-colliding source %q", file.Source)
		}
		if _, exists := seenDestination[strings.ToLower(file.Destination)]; exists {
			return fmt.Errorf("duplicate or case-colliding destination %q", file.Destination)
		}
		seenSource[strings.ToLower(file.Source)] = struct{}{}
		seenDestination[strings.ToLower(file.Destination)] = struct{}{}
		if !validKind(file.Destination, file.Kind) {
			return fmt.Errorf("kind %q does not match destination %q", file.Kind, file.Destination)
		}
		if file.MediaType != "text" && file.MediaType != "binary" {
			return fmt.Errorf("file %q has unsupported media_type %q", file.Source, file.MediaType)
		}
		if !schemautil.ValidDigest(file.SourceSHA256) {
			return fmt.Errorf("file %q has invalid source_sha256", file.Source)
		}
		contents, err := fs.ReadFile(fsys, path.Join(root, file.Source))
		if err != nil {
			return fmt.Errorf("read template source %q: %w", file.Source, err)
		}
		digest := sha256.Sum256(contents)
		observed := "sha256:" + hex.EncodeToString(digest[:])
		if observed != file.SourceSHA256 {
			return fmt.Errorf("template source %q digest mismatch", file.Source)
		}
		if file.MediaType == "text" {
			if !utf8.Valid(contents) {
				return fmt.Errorf("template source %q is not valid UTF-8", file.Source)
			}
			for _, match := range placeholderPattern.FindAllSubmatch(contents, -1) {
				if _, exists := variableNames[string(match[1])]; !exists {
					return fmt.Errorf("template source %q uses undeclared variable %q", file.Source, match[1])
				}
			}
			if strings.Contains(string(placeholderPattern.ReplaceAll(contents, nil)), "{{engkit.") {
				return fmt.Errorf("template source %q contains a malformed placeholder", file.Source)
			}
		} else if placeholderPattern.Match(contents) {
			return fmt.Errorf("binary template source %q contains a placeholder", file.Source)
		}
	}
	inventory, err := schemautil.RegularInventory(fsys, root)
	if err != nil {
		return fmt.Errorf("inspect template inventory: %w", err)
	}
	if len(inventory) != len(seenSource)+1 {
		return fmt.Errorf("template file inventory does not match declared sources")
	}
	for _, relative := range inventory {
		if relative == "template.yaml" {
			continue
		}
		if _, exists := seenSource[strings.ToLower(relative)]; !exists {
			return fmt.Errorf("template source %q is undeclared", relative)
		}
	}
	if _, exists := seenDestination["instructions.md"]; !exists || template.Skill.Instructions != "instructions.md" {
		return fmt.Errorf("template must generate instructions.md")
	}
	return nil
}

func ValidateVariables(variables []Variable) (map[string]struct{}, error) {
	names := make(map[string]struct{}, len(variables))
	previous := ""
	for _, variable := range variables {
		if !schemautil.ValidSnake(variable.Name) {
			return nil, fmt.Errorf("variable name %q is invalid", variable.Name)
		}
		if _, exists := names[variable.Name]; exists {
			return nil, fmt.Errorf("variable %q is duplicated", variable.Name)
		}
		names[variable.Name] = struct{}{}
		if previous != "" && variable.Name < previous {
			return nil, fmt.Errorf("variables must be sorted by name")
		}
		previous = variable.Name
		if variable.Type != "string" && variable.Type != "integer" && variable.Type != "boolean" {
			return nil, fmt.Errorf("variable %q has unsupported type %q", variable.Name, variable.Type)
		}
		if strings.TrimSpace(variable.Description) != variable.Description || variable.Description == "" {
			return nil, fmt.Errorf("variable %q requires a normalized description", variable.Name)
		}
		if variable.Required && variable.Default != nil {
			return nil, fmt.Errorf("required variable %q cannot define a default", variable.Name)
		}
		if variable.Default != nil && !matchesType(variable.Type, variable.Default) {
			return nil, fmt.Errorf("variable %q default does not match type %q", variable.Name, variable.Type)
		}
		for _, allowed := range variable.AllowedValues {
			if !matchesType(variable.Type, allowed) {
				return nil, fmt.Errorf("variable %q allowed value does not match type %q", variable.Name, variable.Type)
			}
		}
		if err := validateVariableConstraints(variable); err != nil {
			return nil, err
		}
	}
	return names, nil
}

func matchesType(kind string, value any) bool {
	switch kind {
	case "string":
		_, ok := value.(string)
		return ok
	case "integer":
		switch value.(type) {
		case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
			return true
		default:
			return false
		}
	case "boolean":
		_, ok := value.(bool)
		return ok
	default:
		return false
	}
}

func validateVariableConstraints(variable Variable) error {
	if variable.MinLength != nil && *variable.MinLength < 0 || variable.MaxLength != nil && *variable.MaxLength < 0 {
		return fmt.Errorf("variable %q length bounds must be non-negative", variable.Name)
	}
	if variable.MinLength != nil && variable.MaxLength != nil && *variable.MinLength > *variable.MaxLength {
		return fmt.Errorf("variable %q minimum length exceeds maximum length", variable.Name)
	}
	if variable.Minimum != nil && variable.Maximum != nil && *variable.Minimum > *variable.Maximum {
		return fmt.Errorf("variable %q minimum exceeds maximum", variable.Name)
	}
	switch variable.Type {
	case "string":
		if variable.Minimum != nil || variable.Maximum != nil {
			return fmt.Errorf("string variable %q cannot use integer bounds", variable.Name)
		}
		if variable.Pattern != "" {
			if _, err := regexp.Compile(variable.Pattern); err != nil {
				return fmt.Errorf("variable %q pattern: %w", variable.Name, err)
			}
		}
	case "integer":
		if variable.MinLength != nil || variable.MaxLength != nil || variable.Pattern != "" || len(variable.AllowedValues) != 0 {
			return fmt.Errorf("integer variable %q cannot use string constraints", variable.Name)
		}
	case "boolean":
		if variable.MinLength != nil || variable.MaxLength != nil || variable.Pattern != "" || variable.Minimum != nil || variable.Maximum != nil || len(variable.AllowedValues) != 0 {
			return fmt.Errorf("boolean variable %q cannot define value constraints", variable.Name)
		}
	}
	seenAllowed := make(map[string]struct{}, len(variable.AllowedValues))
	for _, allowed := range variable.AllowedValues {
		key := fmt.Sprintf("%T:%v", allowed, allowed)
		if _, exists := seenAllowed[key]; exists {
			return fmt.Errorf("variable %q contains duplicate allowed values", variable.Name)
		}
		seenAllowed[key] = struct{}{}
		if err := validateValue(variable, allowed, false); err != nil {
			return err
		}
	}
	if variable.Default != nil {
		if err := validateValue(variable, variable.Default, true); err != nil {
			return err
		}
	}
	return nil
}

func validateValue(variable Variable, value any, checkAllowed bool) error {
	switch variable.Type {
	case "string":
		text := value.(string)
		length := utf8.RuneCountInString(text)
		if variable.MinLength != nil && length < *variable.MinLength || variable.MaxLength != nil && length > *variable.MaxLength {
			return fmt.Errorf("variable %q value violates length bounds", variable.Name)
		}
		if variable.Pattern != "" && !regexp.MustCompile(variable.Pattern).MatchString(text) {
			return fmt.Errorf("variable %q value does not match its pattern", variable.Name)
		}
	case "integer":
		integer, ok := integerValue(value)
		if !ok {
			return fmt.Errorf("variable %q value is outside the supported integer range", variable.Name)
		}
		if variable.Minimum != nil && integer < *variable.Minimum || variable.Maximum != nil && integer > *variable.Maximum {
			return fmt.Errorf("variable %q value violates integer bounds", variable.Name)
		}
	}
	if checkAllowed && len(variable.AllowedValues) != 0 {
		key := fmt.Sprintf("%T:%v", value, value)
		for _, allowed := range variable.AllowedValues {
			if fmt.Sprintf("%T:%v", allowed, allowed) == key {
				return nil
			}
		}
		return fmt.Errorf("variable %q default is not an allowed value", variable.Name)
	}
	return nil
}

func integerValue(value any) (int64, bool) {
	switch typed := value.(type) {
	case int:
		return int64(typed), true
	case int8:
		return int64(typed), true
	case int16:
		return int64(typed), true
	case int32:
		return int64(typed), true
	case int64:
		return typed, true
	case uint:
		if uint64(typed) <= uint64(^uint64(0)>>1) {
			return int64(typed), true
		}
	case uint8:
		return int64(typed), true
	case uint16:
		return int64(typed), true
	case uint32:
		return int64(typed), true
	case uint64:
		if typed <= uint64(^uint64(0)>>1) {
			return int64(typed), true
		}
	}
	return 0, false
}

func validateSkill(skill SkillSpec) error {
	if !schemautil.ValidKebab(skill.ID) || !schemautil.ValidSemver(skill.Version) {
		return fmt.Errorf("generated skill identity or version is invalid")
	}
	if strings.TrimSpace(skill.Description) != skill.Description || skill.Description == "" {
		return fmt.Errorf("generated skill description must be normalized and non-empty")
	}
	if skill.Instructions != "instructions.md" || skill.Permissions == nil || skill.Compatibility == nil || skill.Ownership == nil {
		return fmt.Errorf("generated skill requires instructions.md, permissions, compatibility, and ownership")
	}
	if !schemautil.ValidSemver(skill.Compatibility.MinimumEngkit) {
		return fmt.Errorf("generated skill minimum_engkit is invalid")
	}
	if strings.TrimSpace(skill.Ownership.Owner) != skill.Ownership.Owner || skill.Ownership.Owner == "" {
		return fmt.Errorf("generated skill owner must be normalized and non-empty")
	}
	return nil
}

func validKind(destination, kind string) bool {
	switch kind {
	case "instructions":
		return destination == "instructions.md"
	case "reference":
		return strings.HasPrefix(destination, "references/")
	case "script":
		return strings.HasPrefix(destination, "scripts/")
	case "template":
		return strings.HasPrefix(destination, "templates/")
	case "asset":
		return strings.HasPrefix(destination, "assets/")
	default:
		return false
	}
}

func SortedReferences(items []Reference) []Reference {
	result := append([]Reference(nil), items...)
	sort.Slice(result, func(i, j int) bool {
		if result[i].ID != result[j].ID {
			return result[i].ID < result[j].ID
		}
		return result[i].Version < result[j].Version
	})
	return result
}
