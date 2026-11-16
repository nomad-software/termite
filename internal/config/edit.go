package config

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// SetValues sets top-level keys in the JSON config file in one write,
// keeping every other key the user has set. The file is created from the
// defaults if it does not exist. Keys are written sorted and indented.
// A symlinked file (one kept with dotfiles, say) is written where the link
// points, and the link is kept.
func SetValues(path string, values map[string]any) error {
	path = resolveLink(path)
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		data = SampleConfig()
	} else if err != nil {
		return err
	}
	m := map[string]json.RawMessage{}
	if len(bytes.TrimSpace(data)) > 0 {
		if err := json.Unmarshal(data, &m); err != nil {
			return fmt.Errorf("%s: %w", path, describeJSONError(data, err))
		}
		if m == nil { // the file is just null
			m = map[string]json.RawMessage{}
		}
	}
	for key, value := range values {
		v, err := json.Marshal(value)
		if err != nil {
			return err
		}
		m[key] = v
	}
	out, err := marshalIndent(m)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	// Keep the file's permissions (a config kept private stays private).
	mode := os.FileMode(0o644)
	if fi, err := os.Stat(path); err == nil {
		mode = fi.Mode().Perm()
	}
	// A temporary file of its own, so two termites saving at once never
	// write into the same one.
	f, err := os.CreateTemp(filepath.Dir(path), filepath.Base(path)+".*.tmp")
	if err != nil {
		return err
	}
	tmp := f.Name()
	if _, err := f.Write(out); err != nil {
		f.Close()
		os.Remove(tmp)
		return err
	}
	if err := f.Close(); err != nil {
		os.Remove(tmp)
		return err
	}
	if err := os.Chmod(tmp, mode); err != nil { // CreateTemp makes it 0600
		os.Remove(tmp)
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		os.Remove(tmp)
		return err
	}
	return nil
}

// resolveLink follows path's symlinks to the file they name, which need not
// exist yet: a dangling link is followed too, so writing creates its target
// rather than replacing the link.
func resolveLink(path string) string {
	for range 40 { // as the kernel's limit on nested links
		target, err := os.Readlink(path)
		if err != nil {
			return path
		}
		if !filepath.IsAbs(target) {
			target = filepath.Join(filepath.Dir(path), target)
		}
		path = target
	}
	return path
}

// WriteSample writes the default configuration to path, creating its
// folder. It fails with an error matching fs.ErrExist if the file is
// already there, which it leaves alone.
func WriteSample(path string) error {
	path = resolveLink(path) // a dangling link is followed, as SetValues does
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return err
	}
	_, err = f.Write(SampleConfig())
	return errors.Join(err, f.Close())
}
