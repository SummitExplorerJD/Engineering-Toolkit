package compositionexec

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
	"strings"

	"github.com/SummitExplorerJD/Engineering-Toolkit/internal/composition"
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
	if _, err := hex.DecodeString(key); err != nil || strings.ToLower(key) != key {
		return "", errors.New("plan ID must use lowercase hexadecimal characters")
	}
	return key, nil
}

func writeNewJSON(path string, value any) error {
	encoded, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
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
		return err
	}
	if err := file.Sync(); err != nil {
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	remove = false
	return nil
}

func replaceJSON(path string, value any) error {
	encoded, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	encoded = append(encoded, '\n')
	file, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+".tmp-*")
	if err != nil {
		return err
	}
	temporary := file.Name()
	defer os.Remove(temporary)
	if err := file.Chmod(0o600); err != nil {
		file.Close()
		return err
	}
	if _, err := file.Write(encoded); err != nil {
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
	if err := os.Rename(temporary, path); err != nil {
		return fmt.Errorf("replace checkpoint: %w", err)
	}
	return nil
}

func appendEvent(tx *transaction, eventType string, payload map[string]any) error {
	sequence := tx.checkpoint.JournalSequence + 1
	event := journalEvent{SchemaVersion: JournalSchema, EventID: fmt.Sprintf("%s:%06d", tx.checkpoint.PlanID, sequence), PlanID: tx.checkpoint.PlanID, Sequence: sequence, Timestamp: tx.now().UTC().Format(timeFormat), Type: eventType, Payload: payload}
	file, err := os.OpenFile(tx.journalPath, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0o600)
	if err != nil {
		return fmt.Errorf("open composition journal: %w", err)
	}
	if err := json.NewEncoder(file).Encode(event); err != nil {
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
	tx.checkpoint.JournalSequence = sequence
	return nil
}

func writeCheckpoint(tx *transaction, status string) error {
	tx.checkpoint.Status = status
	tx.checkpoint.UpdatedAt = tx.now().UTC().Format(timeFormat)
	if _, err := os.Lstat(tx.checkpointPath); errors.Is(err, os.ErrNotExist) {
		return writeNewJSON(tx.checkpointPath, tx.checkpoint)
	}
	return replaceJSON(tx.checkpointPath, tx.checkpoint)
}

func validateJournal(path, planID string) (int, error) {
	contents, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}
	if len(contents) == 0 || contents[len(contents)-1] != '\n' {
		return 0, errors.New("composition journal has a truncated final line")
	}
	scanner := bufio.NewScanner(bytes.NewReader(contents))
	sequence := 0
	for scanner.Scan() {
		sequence++
		var event journalEvent
		if err := decodeStrict(scanner.Bytes(), &event); err != nil {
			return 0, fmt.Errorf("decode composition journal line %d: %w", sequence, err)
		}
		if event.SchemaVersion != JournalSchema || event.PlanID != planID || event.Sequence != sequence || event.EventID != fmt.Sprintf("%s:%06d", planID, sequence) {
			return 0, fmt.Errorf("composition journal line %d has inconsistent identity", sequence)
		}
	}
	return sequence, scanner.Err()
}

func readPlan(path string) (composition.Plan, error) {
	contents, err := os.ReadFile(path)
	if err != nil {
		return composition.Plan{}, err
	}
	var value composition.Plan
	if err := decodeStrict(contents, &value); err != nil {
		return composition.Plan{}, err
	}
	if err := composition.ValidateIdentity(value); err != nil {
		return composition.Plan{}, err
	}
	return value, nil
}

func readCheckpoint(path string) (checkpoint, error) {
	contents, err := os.ReadFile(path)
	if err != nil {
		return checkpoint{}, err
	}
	var value checkpoint
	if err := decodeStrict(contents, &value); err != nil {
		return checkpoint{}, err
	}
	if value.SchemaVersion != CheckpointSchema {
		return checkpoint{}, errors.New("unsupported composition checkpoint schema")
	}
	return value, nil
}

func decodeStrict(contents []byte, value any) error {
	decoder := json.NewDecoder(bytes.NewReader(contents))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(value); err != nil {
		return err
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

func fileDigest(path string) (string, error) {
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

func ensureRealDirectory(path string) error {
	info, err := os.Lstat(path)
	if errors.Is(err, os.ErrNotExist) {
		return os.Mkdir(path, 0o700)
	}
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return fmt.Errorf("path %s must be a real directory", path)
	}
	return nil
}

func slashRelative(root, target string) string {
	value, err := filepath.Rel(root, target)
	if err != nil {
		return filepath.ToSlash(target)
	}
	return filepath.ToSlash(value)
}
