package webserver

import (
	"context"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/cryptolab"

	// Register every tool and subject so the schema is complete.
	_ "github.com/MattCheramie/GopherTrunk/internal/cryptolab/subjects/motorola"
	_ "github.com/MattCheramie/GopherTrunk/internal/cryptolab/tools"
)

// TestSchemaArgsAcceptedByEveryMode is the drift guard between the web schema
// and each mode's own flag set: it builds the CLI args the web server would
// produce for every schema parameter and runs the mode, asserting the mode's
// flag parser never rejects an arg with "flag provided but not defined". (A
// content error from dummy input is fine — only a flag-wiring mismatch fails.)
func TestSchemaArgsAcceptedByEveryMode(t *testing.T) {
	t.Parallel()
	s, err := New(Options{})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })

	for _, ms := range cryptolab.Schema() {
		ms := ms
		t.Run(ms.Tool+"/"+ms.Mode, func(t *testing.T) {
			t.Parallel()
			// Stage a real temp file for every file/optional-file param so
			// buildArgs has an upload id to resolve.
			params := map[string]string{}
			files := map[string]string{}
			for _, p := range ms.Params {
				switch p.Kind {
				case "file":
					id := randomID()
					path := filepath.Join(t.TempDir(), id+".bin")
					if err := os.WriteFile(path, []byte("\x00\x01\x02\x03"), 0o644); err != nil {
						t.Fatal(err)
					}
					s.mu.Lock()
					s.files[id] = &upload{ID: id, Path: path}
					s.mu.Unlock()
					files[p.Name] = id
				case "outfile":
					params[p.Name] = p.Default
				case "bool":
					params[p.Name] = "true"
				default:
					if p.Default != "" {
						params[p.Name] = p.Default
					} else {
						params[p.Name] = "1"
					}
				}
			}
			jobDir := t.TempDir()
			args, err := s.buildArgs(ms, runRequest{Tool: ms.Tool, Mode: ms.Mode, Params: params, Files: files}, jobDir)
			if err != nil {
				t.Fatalf("buildArgs: %v", err)
			}
			_, mode, err := cryptolab.LookupMode(ms.Tool, ms.Mode)
			if err != nil {
				t.Fatal(err)
			}
			env := cryptolab.Env{Logger: slog.New(slog.NewTextHandler(io.Discard, nil)), OutDir: jobDir, Stdout: io.Discard}
			_, runErr := mode.Run(context.Background(), args, env)
			if runErr != nil {
				low := strings.ToLower(runErr.Error())
				if strings.Contains(low, "not defined") || strings.Contains(low, "flag provided") {
					t.Fatalf("schema/flag drift for %s/%s: %v (args=%v)", ms.Tool, ms.Mode, runErr, args)
				}
				// Any other error is an expected content/validation failure on
				// the dummy input, not a flag-wiring problem.
			}
		})
	}
}
