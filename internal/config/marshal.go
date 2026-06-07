package config

import (
	"bytes"
	"errors"
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// ErrFileExists is returned by WriteConfigFile when the target already
// exists and overwrite was not requested, so the web Config Builder can
// prompt the operator before clobbering a file.
var ErrFileExists = errors.New("config: file already exists")

// ErrModifiedExternally is returned by WriteConfigFile when the on-disk
// mtime no longer matches the guard mtime the caller loaded with — the
// file changed out-of-band (a manual edit, or the daemon's settings
// PATCH) since it was read into the builder.
var ErrModifiedExternally = errors.New("config: file was modified externally since it was loaded")

// Marshal renders cfg to canonical YAML using the struct tags. Unlike the
// comment-preserving Writer.WritePatch path, this produces a fresh file
// from the struct alone (no comments) — the right behaviour for the web
// Config Builder, which builds configs from structured edits rather than
// surgically patching a hand-annotated file. Indentation matches the
// Writer (two spaces) so round-tripped files read consistently.
func Marshal(cfg Config) ([]byte, error) {
	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(cfg); err != nil {
		return nil, fmt.Errorf("config: marshal: %w", err)
	}
	if err := enc.Close(); err != nil {
		return nil, fmt.Errorf("config: marshal: %w", err)
	}
	return buf.Bytes(), nil
}

// Unmarshal parses YAML into a Config without validating it. The web
// Config Builder uses this to load a possibly-invalid file so the operator
// can fix it in the editor (Load, by contrast, rejects invalid files).
func Unmarshal(data []byte, out *Config) error {
	if err := yaml.Unmarshal(data, out); err != nil {
		return fmt.Errorf("config: parse: %w", err)
	}
	return nil
}

// WriteConfigFile validates cfg, marshals it to YAML, and atomically
// writes it to path. It is the whole-file counterpart to
// Writer.WritePatch and backs the Config Builder's save endpoint.
//
//   - cfg is validated first; an invalid config is never persisted.
//   - When overwrite is false and path already exists, ErrFileExists is
//     returned so the UI can prompt for confirmation.
//   - When guardMtime is non-zero, the on-disk mtime is compared against
//     it and ErrModifiedExternally is returned on a mismatch (optimistic
//     concurrency for an edited-then-saved file). Pass 0 for a new file.
//
// On success the new file's mtime (UnixNano) is returned so the caller
// can refresh its guard for a subsequent save.
func WriteConfigFile(path string, cfg Config, guardMtime int64, overwrite bool) (int64, error) {
	if err := cfg.Validate(); err != nil {
		return 0, fmt.Errorf("config: refusing to write invalid config: %w", err)
	}

	info, statErr := os.Stat(path)
	switch {
	case statErr == nil:
		// File exists.
		if !overwrite {
			return 0, fmt.Errorf("%w: %s", ErrFileExists, path)
		}
		if guardMtime != 0 && info.ModTime().UnixNano() != guardMtime {
			return 0, fmt.Errorf("%w: %s", ErrModifiedExternally, path)
		}
	case !os.IsNotExist(statErr):
		return 0, fmt.Errorf("config: stat %s: %w", path, statErr)
	}

	data, err := Marshal(cfg)
	if err != nil {
		return 0, err
	}
	if err := writeFileAtomic(path, data, 0o644); err != nil {
		return 0, err
	}
	newInfo, err := os.Stat(path)
	if err != nil {
		return 0, fmt.Errorf("config: stat after write %s: %w", path, err)
	}
	return newInfo.ModTime().UnixNano(), nil
}
