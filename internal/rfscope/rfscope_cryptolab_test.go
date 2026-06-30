//go:build cryptolab

package rfscope

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestRunCryptolabToolClassify(t *testing.T) {
	t.Parallel()
	if !CryptolabLinked {
		t.Fatal("CryptolabLinked must be true under -tags cryptolab")
	}
	// Write a random payload and run `classify auto` over it via the registry.
	path := filepath.Join(t.TempDir(), "payload.bin")
	if err := os.WriteFile(path, randomBytes(4096, 5), 0o644); err != nil {
		t.Fatalf("write payload: %v", err)
	}
	res, err := RunCryptolabTool(context.Background(), "classify", "auto", []string{"-in", path})
	if err != nil {
		t.Fatalf("RunCryptolabTool: %v", err)
	}
	if res == nil || res.Tool != "classify" || res.Mode != "auto" {
		t.Fatalf("unexpected result: %+v", res)
	}
}

func TestRunCryptolabToolUnknownTool(t *testing.T) {
	t.Parallel()
	if _, err := RunCryptolabTool(context.Background(), "nope", "x", nil); err == nil {
		t.Fatal("want error for unknown tool")
	}
}
