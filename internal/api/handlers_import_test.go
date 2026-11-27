package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"sort"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/events"
)

// fakeImporter records calls. Parse returns a canned DTO; Commit
// returns either the configured result or the configured error.
type fakeImporter struct {
	parses     atomic.Int32
	commits    atomic.Int32
	commitErr  error
	commitResp ImportCommitResult
	lastSrcs   atomic.Int32
}

func (f *fakeImporter) Parse(s ImportSource) (ParsedSystemDTO, error) {
	f.parses.Add(1)
	return ParsedSystemDTO{
		Name:        "FakeSys",
		Protocol:    "p25",
		SiteCount:   1,
		TalkgroupCt: 2,
		SourcePath:  s.Filename,
	}, nil
}

func (f *fakeImporter) Commit(sources []ImportSource, force bool) (ImportCommitResult, error) {
	f.commits.Add(1)
	f.lastSrcs.Store(int32(len(sources)))
	if f.commitErr != nil {
		return ImportCommitResult{}, f.commitErr
	}
	return f.commitResp, nil
}

// uploadMultipart builds the multipart body the handler expects.
func uploadMultipart(t *testing.T, files map[string][]byte) (*bytes.Buffer, string) {
	t.Helper()
	buf := new(bytes.Buffer)
	w := multipart.NewWriter(buf)
	for name, body := range files {
		fw, err := w.CreateFormFile("files", name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := io.Copy(fw, bytes.NewReader(body)); err != nil {
			t.Fatal(err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	return buf, w.FormDataContentType()
}

func TestImportUpload_503WhenNoImporter(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	base, teardown := mkServer(t, ServerOptions{Bus: bus, AllowMutations: true})
	defer teardown()

	body, ct := uploadMultipart(t, map[string][]byte{"x.csv": []byte("ignored")})
	resp, err := http.Post(base+"/api/v1/import", ct, body)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("status=%d want 503", resp.StatusCode)
	}
}

func TestImportUpload_BadExtension400(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	fi := &fakeImporter{}
	base, teardown := mkServer(t, ServerOptions{
		Bus:            bus,
		AllowMutations: true,
		Importer:       fi,
	})
	defer teardown()

	body, ct := uploadMultipart(t, map[string][]byte{"x.txt": []byte("nope")})
	resp, err := http.Post(base+"/api/v1/import", ct, body)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status=%d want 400", resp.StatusCode)
	}
}

func TestImportUpload_PreviewAndCommit(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	fi := &fakeImporter{
		commitResp: ImportCommitResult{
			SystemsAdded: []string{"FakeSys"},
			ConfigPath:   "/tmp/cfg.yaml",
		},
	}
	base, teardown := mkServer(t, ServerOptions{
		Bus:            bus,
		AllowMutations: true,
		Importer:       fi,
	})
	defer teardown()

	body, ct := uploadMultipart(t, map[string][]byte{"sys.csv": []byte("# csv")})
	resp, err := http.Post(base+"/api/v1/import", ct, body)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		buf := new(bytes.Buffer)
		_, _ = buf.ReadFrom(resp.Body)
		t.Fatalf("upload status=%d body=%s", resp.StatusCode, buf.String())
	}
	var preview ImportPreviewResponse
	if err := json.NewDecoder(resp.Body).Decode(&preview); err != nil {
		t.Fatal(err)
	}
	if preview.ID == "" {
		t.Fatal("expected non-empty staging id")
	}
	if len(preview.Systems) != 1 {
		t.Fatalf("expected 1 system in preview, got %d", len(preview.Systems))
	}

	// Commit by id.
	commitURL := fmt.Sprintf("%s/api/v1/import/%s/commit", base, preview.ID)
	commitResp, err := http.Post(commitURL, "application/json", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer commitResp.Body.Close()
	if commitResp.StatusCode != 200 {
		buf := new(bytes.Buffer)
		_, _ = buf.ReadFrom(commitResp.Body)
		t.Fatalf("commit status=%d body=%s", commitResp.StatusCode, buf.String())
	}
	var out ImportCommitResult
	if err := json.NewDecoder(commitResp.Body).Decode(&out); err != nil {
		t.Fatal(err)
	}
	if len(out.SystemsAdded) != 1 || out.SystemsAdded[0] != "FakeSys" {
		t.Errorf("commit result systems_added=%v want [FakeSys]", out.SystemsAdded)
	}
	if fi.commits.Load() != 1 {
		t.Errorf("commit calls=%d want 1", fi.commits.Load())
	}
}

func TestImportCommit_NotFound(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	fi := &fakeImporter{}
	base, teardown := mkServer(t, ServerOptions{
		Bus:            bus,
		AllowMutations: true,
		Importer:       fi,
	})
	defer teardown()

	resp, err := http.Post(base+"/api/v1/import/bogus-id/commit", "application/json", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status=%d want 404", resp.StatusCode)
	}
}

func TestImportDiscard(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	fi := &fakeImporter{}
	base, teardown := mkServer(t, ServerOptions{
		Bus:            bus,
		AllowMutations: true,
		Importer:       fi,
	})
	defer teardown()

	body, ct := uploadMultipart(t, map[string][]byte{"sys.csv": []byte("# csv")})
	resp, err := http.Post(base+"/api/v1/import", ct, body)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var preview ImportPreviewResponse
	if err := json.NewDecoder(resp.Body).Decode(&preview); err != nil {
		t.Fatal(err)
	}

	req, err := http.NewRequest(http.MethodDelete,
		fmt.Sprintf("%s/api/v1/import/%s", base, preview.ID), nil)
	if err != nil {
		t.Fatal(err)
	}
	dresp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer dresp.Body.Close()
	if dresp.StatusCode != http.StatusNoContent {
		t.Fatalf("status=%d want 204", dresp.StatusCode)
	}

	// Commit by the same id should now 404.
	cresp, err := http.Post(
		fmt.Sprintf("%s/api/v1/import/%s/commit", base, preview.ID),
		"application/json", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer cresp.Body.Close()
	if cresp.StatusCode != http.StatusNotFound {
		t.Fatalf("commit after discard status=%d want 404", cresp.StatusCode)
	}
}

func TestImportCommit_ErrorPreservesStaging(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	fi := &fakeImporter{commitErr: errors.New("boom")}
	base, teardown := mkServer(t, ServerOptions{
		Bus:            bus,
		AllowMutations: true,
		Importer:       fi,
	})
	defer teardown()

	body, ct := uploadMultipart(t, map[string][]byte{"sys.csv": []byte("# csv")})
	resp, err := http.Post(base+"/api/v1/import", ct, body)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var preview ImportPreviewResponse
	if err := json.NewDecoder(resp.Body).Decode(&preview); err != nil {
		t.Fatal(err)
	}

	first, err := http.Post(
		fmt.Sprintf("%s/api/v1/import/%s/commit", base, preview.ID),
		"application/json", nil)
	if err != nil {
		t.Fatal(err)
	}
	first.Body.Close()
	if first.StatusCode != http.StatusBadRequest {
		t.Fatalf("first commit status=%d want 400", first.StatusCode)
	}

	// Retry should reach the importer again — staging entry must
	// still be present.
	second, err := http.Post(
		fmt.Sprintf("%s/api/v1/import/%s/commit", base, preview.ID),
		"application/json", nil)
	if err != nil {
		t.Fatal(err)
	}
	second.Body.Close()
	if second.StatusCode != http.StatusBadRequest {
		t.Fatalf("second commit status=%d want 400 (same error)", second.StatusCode)
	}
	if fi.commits.Load() != 2 {
		t.Errorf("commit calls=%d want 2 (retry)", fi.commits.Load())
	}
}

// fakeBatchImporter also implements BatchImporter: it previews a whole
// upload as ONE system, the way the daemon joins a RadioReference talkgroup
// CSV with its sites CSV (#1403).
type fakeBatchImporter struct {
	fakeImporter
	batches atomic.Int32
}

func (f *fakeBatchImporter) ParseAll(sources []ImportSource) ([]ParsedSystemDTO, error) {
	f.batches.Add(1)
	names := make([]string, 0, len(sources))
	for _, s := range sources {
		names = append(names, s.Filename)
	}
	sort.Strings(names)
	return []ParsedSystemDTO{{Name: "Joined", Protocol: "p25", SiteCount: 2, TalkgroupCt: 3,
		SourcePath: strings.Join(names, " + ")}}, nil
}

// Both upload endpoints must hand the whole upload to a BatchImporter, so a
// pair of files can preview as one system. Parsing file by file is what
// split a RadioReference talkgroup CSV and its sites CSV into two systems.
func TestUploadsPreviewThroughBatchImporter(t *testing.T) {
	for _, path := range []string{"/api/v1/import", "/api/v1/config/parse"} {
		bus := events.NewBus(8)
		fi := &fakeBatchImporter{}
		base, teardown := mkServer(t, ServerOptions{Bus: bus, AllowMutations: true, Importer: fi,
			ConfigBuilder: ConfigBuilderOptions{Enabled: true, ConfigDir: t.TempDir()}})

		body, ct := uploadMultipart(t, map[string][]byte{
			"trs_tg_7715.csv":    []byte("a"),
			"trs_sites_7715.csv": []byte("b"),
		})
		resp, err := http.Post(base+path, ct, body)
		if err != nil {
			t.Fatal(err)
		}
		var out struct {
			Systems []ParsedSystemDTO `json:"systems"`
		}
		if resp.StatusCode != http.StatusOK {
			raw, _ := io.ReadAll(resp.Body)
			t.Fatalf("%s: status=%d: %s", path, resp.StatusCode, raw)
		}
		if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		if len(out.Systems) != 1 || out.Systems[0].SourcePath != "trs_sites_7715.csv + trs_tg_7715.csv" {
			t.Errorf("%s: systems = %+v, want the one joined preview", path, out.Systems)
		}
		if fi.batches.Load() != 1 || fi.parses.Load() != 0 {
			t.Errorf("%s: %d batch parses and %d per-file parses, want 1 and 0", path, fi.batches.Load(), fi.parses.Load())
		}
		teardown()
		bus.Close()
	}
}
