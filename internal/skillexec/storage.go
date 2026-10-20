package skillexec

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/SummitExplorerJD/Engineering-Toolkit/internal/skillplan"
	"github.com/SummitExplorerJD/Engineering-Toolkit/internal/skills"
)

const timeFormat = "2006-01-02T15:04:05.999999999Z07:00"

func planKey(planID string) (string, error) {
	const prefix = "sha256:"
	if !strings.HasPrefix(planID, prefix) {
		return "", errors.New("plan ID must use sha256")
	}
	key := strings.TrimPrefix(planID, prefix)
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
	if err := rejectLinkAncestors(absolute); err != nil {
		return "", err
	}
	for _, candidate := range []string{absolute, filepath.Join(absolute, ".engkit")} {
		info, err := os.Lstat(candidate)
		if err != nil {
			return "", fmt.Errorf("inspect initialized workspace path %s: %w", candidate, err)
		}
		if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
			return "", fmt.Errorf("initialized workspace path %s must be a real directory", candidate)
		}
	}
	config := filepath.Join(absolute, ".engkit", "config.yaml")
	info, err := os.Lstat(config)
	if err != nil {
		return "", fmt.Errorf("inspect initialized workspace config: %w", err)
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return "", errors.New("initialized workspace config must be a regular file")
	}
	return absolute, nil
}

func rejectLinkAncestors(path string) error {
	for candidate := path; ; candidate = filepath.Dir(candidate) {
		info, err := os.Lstat(candidate)
		if err != nil {
			return fmt.Errorf("inspect workspace ancestor %s: %w", candidate, err)
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return errors.New("workspace root must not use symbolic-link or junction ancestors")
		}
		parent := filepath.Dir(candidate)
		if parent == candidate {
			return nil
		}
	}
}

func prepareState(root string) error {
	engkit := filepath.Join(root, ".engkit")
	for _, relative := range []string{
		"plans", "plans/skills", "journals", "journals/skills",
		"checkpoints", "checkpoints/skills", "recovery",
		"recovery/skills", "recovery/skills/locks",
	} {
		if err := ensureDirectory(filepath.Join(engkit, filepath.FromSlash(relative))); err != nil {
			return err
		}
	}
	return ensureRecoveryIgnored(engkit)
}

func ensureDirectory(path string) error {
	info, err := os.Lstat(path)
	if errors.Is(err, os.ErrNotExist) {
		if err := os.Mkdir(path, 0o700); err != nil {
			if !errors.Is(err, os.ErrExist) {
				return fmt.Errorf("create state directory %s: %w", path, err)
			}
			info, err = os.Lstat(path)
			if err != nil {
				return fmt.Errorf("reinspect state directory %s: %w", path, err)
			}
			if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
				return fmt.Errorf("state path %s must be a real directory", path)
			}
		}
		return nil
	}
	if err != nil {
		return fmt.Errorf("inspect state directory %s: %w", path, err)
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return fmt.Errorf("state path %s must be a real directory", path)
	}
	return nil
}

func ensureRecoveryIgnored(engkit string) error {
	path := filepath.Join(engkit, ".gitignore")
	info, err := os.Lstat(path)
	if err != nil {
		return fmt.Errorf("inspect .engkit/.gitignore: %w", err)
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return errors.New(".engkit/.gitignore must be a regular file")
	}
	contents, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read .engkit/.gitignore: %w", err)
	}
	for _, line := range strings.Split(string(contents), "\n") {
		if strings.TrimSuffix(line, "\r") == "/recovery/*" {
			return nil
		}
	}
	updated := append([]byte(nil), contents...)
	if len(updated) > 0 && updated[len(updated)-1] != '\n' {
		updated = append(updated, '\n')
	}
	updated = append(updated, []byte("/recovery/*\n")...)
	return replaceSynced(path, updated, info.Mode().Perm())
}

func writeNewJSON(path string, value any) error {
	encoded, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return fmt.Errorf("encode %s: %w", path, err)
	}
	return writeNewSynced(path, append(encoded, '\n'))
}

func writeNewSynced(path string, contents []byte) error {
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return fmt.Errorf("create %s: %w", path, err)
	}
	remove := true
	defer func() {
		_ = file.Close()
		if remove {
			_ = os.Remove(path)
		}
	}()
	if _, err := file.Write(contents); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	if err := file.Sync(); err != nil {
		return fmt.Errorf("sync %s: %w", path, err)
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("close %s: %w", path, err)
	}
	remove = false
	return nil
}

func replaceSynced(path string, contents []byte, mode os.FileMode) error {
	file, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+".tmp-*")
	if err != nil {
		return fmt.Errorf("create sibling temporary file: %w", err)
	}
	temp := file.Name()
	defer os.Remove(temp)
	if err := file.Chmod(mode); err != nil {
		file.Close()
		return err
	}
	if _, err := file.Write(contents); err != nil {
		file.Close()
		return err
	}
	if err := file.Sync(); err != nil {
		file.Close()
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	if err := os.Rename(temp, path); err != nil {
		return fmt.Errorf("replace %s: %w", path, err)
	}
	return nil
}

func copyPackage(snapshot skills.Snapshot, destination string) error {
	if err := os.Mkdir(destination, 0o700); err != nil {
		return fmt.Errorf("create staged package: %w", err)
	}
	for _, relative := range snapshot.Directories {
		if err := os.Mkdir(filepath.Join(destination, filepath.FromSlash(relative)), 0o700); err != nil {
			return fmt.Errorf("create staged directory %s: %w", relative, err)
		}
	}
	for _, file := range snapshot.Files {
		source := filepath.Join(filepath.FromSlash(snapshot.PackageRoot), filepath.FromSlash(file.Path))
		target := filepath.Join(destination, filepath.FromSlash(file.Path))
		if err := copyRegularFile(source, target); err != nil {
			return fmt.Errorf("copy staged file %s: %w", file.Path, err)
		}
	}
	return nil
}

func copyRegularFile(source, target string) error {
	info, err := os.Lstat(source)
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return errors.New("source changed to a non-regular file")
	}
	input, err := os.Open(source)
	if err != nil {
		return err
	}
	defer input.Close()
	output, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return err
	}
	remove := true
	defer func() {
		_ = output.Close()
		if remove {
			_ = os.Remove(target)
		}
	}()
	if _, err := io.Copy(output, input); err != nil {
		return err
	}
	if err := output.Sync(); err != nil {
		return err
	}
	if err := output.Close(); err != nil {
		return err
	}
	remove = false
	return nil
}

func appendEvent(tx *transaction, eventType string, payload map[string]any) error {
	sequence := tx.checkpoint.JournalSequence + 1
	event := journalEvent{
		SchemaVersion: JournalSchema,
		EventID:       fmt.Sprintf("%s:%06d", tx.checkpoint.PlanID, sequence),
		PlanID:        tx.checkpoint.PlanID, Sequence: sequence,
		Timestamp: tx.now().UTC().Format(timeFormat), Type: eventType, Payload: payload,
	}
	file, err := os.OpenFile(tx.journalPath, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0o600)
	if err != nil {
		return fmt.Errorf("open skill journal: %w", err)
	}
	encoder := json.NewEncoder(file)
	if err := encoder.Encode(event); err != nil {
		file.Close()
		return fmt.Errorf("append skill journal: %w", err)
	}
	if err := file.Sync(); err != nil {
		file.Close()
		return fmt.Errorf("sync skill journal: %w", err)
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("close skill journal: %w", err)
	}
	tx.checkpoint.JournalSequence = sequence
	return nil
}

func writeCheckpoint(tx *transaction, status string) error {
	tx.checkpoint.Status = status
	tx.checkpoint.UpdatedAt = tx.now().UTC().Format(timeFormat)
	encoded, err := json.MarshalIndent(tx.checkpoint, "", "  ")
	if err != nil {
		return err
	}
	encoded = append(encoded, '\n')
	if _, err := os.Stat(tx.checkpointPath); errors.Is(err, os.ErrNotExist) {
		return writeNewSynced(tx.checkpointPath, encoded)
	}
	return replaceSynced(tx.checkpointPath, encoded, 0o600)
}

func validateJournal(path, planID string) (int, error) {
	contents, err := os.ReadFile(path)
	if err != nil {
		return 0, fmt.Errorf("read skill journal: %w", err)
	}
	if len(contents) == 0 || contents[len(contents)-1] != '\n' {
		return 0, errors.New("skill journal has a truncated final line")
	}
	scanner := bufio.NewScanner(bytes.NewReader(contents))
	sequence := 0
	for scanner.Scan() {
		sequence++
		var event journalEvent
		if err := decodeStrict(scanner.Bytes(), &event); err != nil {
			return 0, fmt.Errorf("decode skill journal line %d: %w", sequence, err)
		}
		if event.SchemaVersion != JournalSchema || event.PlanID != planID || event.Sequence != sequence ||
			event.EventID != fmt.Sprintf("%s:%06d", planID, sequence) {
			return 0, fmt.Errorf("skill journal line %d has inconsistent identity", sequence)
		}
	}
	if err := scanner.Err(); err != nil {
		return 0, err
	}
	return sequence, nil
}

func readPlan(path string) (skillplan.Plan, error) {
	contents, err := os.ReadFile(path)
	if err != nil {
		return skillplan.Plan{}, err
	}
	var preview skillplan.Plan
	if err := decodeStrict(contents, &preview); err != nil {
		return skillplan.Plan{}, fmt.Errorf("decode persisted skill plan: %w", err)
	}
	if err := skillplan.ValidateIdentity(preview); err != nil {
		return skillplan.Plan{}, err
	}
	return preview, nil
}

func readCheckpoint(path string) (checkpoint, error) {
	contents, err := os.ReadFile(path)
	if err != nil {
		return checkpoint{}, err
	}
	var value checkpoint
	if err := decodeStrict(contents, &value); err != nil {
		return checkpoint{}, fmt.Errorf("decode skill checkpoint: %w", err)
	}
	if value.SchemaVersion != CheckpointSchema {
		return checkpoint{}, errors.New("unsupported skill checkpoint schema")
	}
	return value, nil
}

func decodeStrict(contents []byte, value any) error {
	decoder := json.NewDecoder(bytes.NewReader(contents))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(value); err != nil {
		return err
	}
	if decoder.More() {
		return errors.New("unexpected trailing JSON value")
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		if err == nil {
			return errors.New("unexpected trailing JSON value")
		}
		return err
	}
	return nil
}
