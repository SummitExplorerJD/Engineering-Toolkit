package providersyncexec

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const timeFormat = "2006-01-02T15:04:05.999999999Z07:00"

func planKey(planID string) (string, error) {
	if !strings.HasPrefix(planID, "sha256:") {
		return "", errors.New("plan ID must use sha256")
	}
	key := strings.TrimPrefix(planID, "sha256:")
	if len(key) != 64 {
		return "", errors.New("plan ID must contain 64 hexadecimal characters")
	}
	for _, value := range key {
		if !strings.ContainsRune("0123456789abcdef", value) {
			return "", errors.New("plan ID must use lowercase hexadecimal characters")
		}
	}
	return key, nil
}

func validateWorkspace(root string) (string, error) {
	absolute, err := filepath.Abs(root)
	if err != nil {
		return "", fmt.Errorf("resolve workspace root: %w", err)
	}
	absolute = filepath.Clean(absolute)
	for candidate := absolute; ; candidate = filepath.Dir(candidate) {
		info, err := os.Lstat(candidate)
		if err != nil {
			return "", fmt.Errorf("inspect workspace ancestor %s: %w", candidate, err)
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return "", errors.New("workspace root must not use symbolic-link or junction ancestors")
		}
		parent := filepath.Dir(candidate)
		if parent == candidate {
			break
		}
	}
	for _, candidate := range []string{absolute, filepath.Join(absolute, ".engkit")} {
		info, err := os.Lstat(candidate)
		if err != nil || info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
			return "", fmt.Errorf("initialized workspace path %s must be a real directory", candidate)
		}
	}
	config := filepath.Join(absolute, ".engkit", "config.yaml")
	info, err := os.Lstat(config)
	if err != nil || info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return "", errors.New("initialized workspace config must be a regular file")
	}
	return absolute, nil
}

func ensureDirectory(path string) error {
	info, err := os.Lstat(path)
	if errors.Is(err, os.ErrNotExist) {
		if err := os.Mkdir(path, 0o700); err != nil && !errors.Is(err, os.ErrExist) {
			return fmt.Errorf("create directory %s: %w", path, err)
		}
		info, err = os.Lstat(path)
	}
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return fmt.Errorf("state path %s must be a real directory", path)
	}
	return nil
}

func prepareState(root, provider, skill string) error {
	engkit := filepath.Join(root, ".engkit")
	for _, relative := range []string{
		"plans", "plans/provider-sync", "plans/provider-sync/" + provider, "plans/provider-sync/" + provider + "/" + skill,
		"journals", "journals/provider-sync", "journals/provider-sync/" + provider, "journals/provider-sync/" + provider + "/" + skill,
		"checkpoints", "checkpoints/provider-sync", "checkpoints/provider-sync/" + provider, "checkpoints/provider-sync/" + provider + "/" + skill,
		"recovery", "recovery/skills", "recovery/skills/locks", "recovery/provider-sync", "recovery/provider-sync/locks", "recovery/provider-sync/locks/" + provider,
		"recovery/provider-sync/" + provider, "recovery/provider-sync/" + provider + "/" + skill,
		"providers", "providers/" + provider,
	} {
		if err := ensureDirectory(filepath.Join(engkit, filepath.FromSlash(relative))); err != nil {
			return err
		}
	}
	return nil
}

func writeNewJSON(path string, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	return writeNewSynced(path, append(data, '\n'))
}

func writeNewSynced(path string, data []byte) error {
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return err
	}
	failed := true
	defer func() {
		_ = file.Close()
		if failed {
			_ = os.Remove(path)
		}
	}()
	if _, err := file.Write(data); err != nil {
		return err
	}
	if err := file.Sync(); err != nil {
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	failed = false
	return nil
}

func replaceJSON(path string, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	if err := ensureDirectory(filepath.Dir(path)); err != nil {
		return err
	}
	temporary, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+".tmp-")
	if err != nil {
		return err
	}
	temporaryName := temporary.Name()
	defer os.Remove(temporaryName)
	if err := temporary.Chmod(0o600); err != nil {
		_ = temporary.Close()
		return err
	}
	if _, err := temporary.Write(append(data, '\n')); err != nil {
		_ = temporary.Close()
		return err
	}
	if err := temporary.Sync(); err != nil {
		_ = temporary.Close()
		return err
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	if err := os.Rename(temporaryName, path); err != nil {
		return fmt.Errorf("replace checkpoint: %w", err)
	}
	return nil
}

func appendEvent(tx *transaction, eventType string, payload map[string]any) error {
	tx.checkpoint.JournalSequence++
	sequence := tx.checkpoint.JournalSequence
	event := journalEvent{SchemaVersion: JournalSchema, EventID: eventID(tx.checkpoint.PlanID, sequence), PlanID: tx.checkpoint.PlanID, Sequence: sequence, Timestamp: tx.now().UTC().Format(timeFormat), Type: eventType, Payload: payload}
	data, err := json.Marshal(event)
	if err != nil {
		return err
	}
	file, err := os.OpenFile(tx.journalPath, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0o600)
	if err != nil {
		return err
	}
	if _, err := file.Write(append(data, '\n')); err != nil {
		_ = file.Close()
		return err
	}
	if err := file.Sync(); err != nil {
		_ = file.Close()
		return err
	}
	return file.Close()
}

func eventID(planID string, sequence int) string {
	sum := sha256.Sum256([]byte(fmt.Sprintf("%s:%d", planID, sequence)))
	return "sha256:" + hex.EncodeToString(sum[:])
}

func writeCheckpoint(tx *transaction, status string) error {
	tx.checkpoint.Status = status
	tx.checkpoint.UpdatedAt = tx.now().UTC().Format(timeFormat)
	return replaceJSON(tx.checkpointPath, tx.checkpoint)
}

func strictJSON(path string, value any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(value); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return errors.New("JSON contains trailing content")
	}
	return nil
}

func validateJournal(path, planID string, expected int) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	sequence := 0
	for scanner.Scan() {
		sequence++
		var event journalEvent
		decoder := json.NewDecoder(bytes.NewReader(scanner.Bytes()))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&event); err != nil || decoder.Decode(&struct{}{}) != io.EOF || event.SchemaVersion != JournalSchema || event.PlanID != planID || event.Sequence != sequence || event.EventID != eventID(planID, sequence) {
			return errors.New("provider journal is malformed or discontinuous")
		}
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	if sequence != expected {
		return errors.New("provider journal sequence does not match checkpoint")
	}
	return nil
}

func digestFile(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}
	return "sha256:" + hex.EncodeToString(hash.Sum(nil)), nil
}

func digestBytes(data []byte) string {
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func relative(root, path string) string {
	value, err := filepath.Rel(root, path)
	if err != nil {
		return filepath.ToSlash(path)
	}
	return filepath.ToSlash(value)
}

func targetPath(root, relativePath string) (string, error) {
	if relativePath == "" || strings.Contains(relativePath, "\\") || filepath.IsAbs(relativePath) {
		return "", errors.New("provider path must be portable and relative")
	}
	target := filepath.Clean(filepath.Join(root, filepath.FromSlash(relativePath)))
	relative, err := filepath.Rel(root, target)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return "", errors.New("provider path escapes workspace root")
	}
	return target, nil
}

func ensureTargetParents(root, target string, created *[]string) error {
	var missing []string
	for current := filepath.Dir(target); current != root; current = filepath.Dir(current) {
		info, err := os.Lstat(current)
		if errors.Is(err, os.ErrNotExist) {
			missing = append(missing, current)
			continue
		}
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
			return fmt.Errorf("provider output ancestor %s must be a real directory", current)
		}
		break
	}
	for index := len(missing) - 1; index >= 0; index-- {
		if err := os.Mkdir(missing[index], 0o700); err != nil {
			return err
		}
		*created = append(*created, relative(root, missing[index]))
	}
	sort.Strings(*created)
	return nil
}

func probeNoReplace(base string, link func(string, string) error) error {
	source := filepath.Join(base, ".no-replace-source")
	target := filepath.Join(base, ".no-replace-target")
	if err := writeNewSynced(source, []byte("probe")); err != nil {
		return err
	}
	defer os.Remove(source)
	defer os.Remove(target)
	if err := link(source, target); err != nil {
		return fmt.Errorf("%w: %v", ErrNoReplaceUnsupported, err)
	}
	return nil
}
