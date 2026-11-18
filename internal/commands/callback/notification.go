package callback

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// Notification is the JSON document written for the app. Fields that dar did
// not supply are omitted.
type Notification struct {
	Basename     string `json:"basename,omitempty"`
	Number       *int   `json:"number,omitempty"`
	NumberPadded string `json:"number_padded,omitempty"`
	Extension    string `json:"extension,omitempty"`
	Context      string `json:"context,omitempty"`
}

// createNotification writes n to path. The complete content appears at once
// (temporary file in the same directory, then rename), so a reader never sees
// a partial document. An existing path and a missing directory are errors.
func createNotification(path string, n Notification) error {
	dir, base := filepath.Split(path)
	if dir == "" {
		dir = "."
	}
	if info, err := os.Stat(dir); err != nil || !info.IsDir() {
		return fmt.Errorf("directory %q of the notification file does not exist", filepath.Clean(dir))
	}
	if _, err := os.Lstat(path); err == nil {
		return fmt.Errorf("notification file %q already exists", path)
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("checking notification file %q: %w", path, err)
	}

	data, err := json.MarshalIndent(n, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')

	tmp, err := os.CreateTemp(dir, "."+base+".*.tmp")
	if err != nil {
		return fmt.Errorf("creating notification file %q: %w", path, err)
	}
	tmpName := tmp.Name()
	fail := func(err error) error {
		tmp.Close()
		os.Remove(tmpName)
		return fmt.Errorf("creating notification file %q: %w", path, err)
	}
	if _, err := tmp.Write(data); err != nil {
		return fail(err)
	}
	if err := tmp.Chmod(0o644); err != nil {
		return fail(err)
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpName)
		return fmt.Errorf("creating notification file %q: %w", path, err)
	}
	if err := os.Rename(tmpName, path); err != nil {
		os.Remove(tmpName)
		return fmt.Errorf("creating notification file %q: %w", path, err)
	}
	return nil
}
