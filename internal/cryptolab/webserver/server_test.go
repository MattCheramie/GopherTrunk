package webserver

import (
	"bytes"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	// Register the generic tools so the schema and runs work.
	_ "github.com/MattCheramie/GopherTrunk/internal/cryptolab/tools"
)

func newTestServer(t *testing.T) *httptest.Server {
	t.Helper()
	s, err := New(Options{}) // own temp dir, no assets
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })
	ts := httptest.NewServer(s.Handler())
	t.Cleanup(ts.Close)
	return ts
}

func TestToolsEndpoint(t *testing.T) {
	t.Parallel()
	ts := newTestServer(t)
	resp, err := http.Get(ts.URL + "/api/v1/cryptolab/tools")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var schema []map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&schema); err != nil {
		t.Fatal(err)
	}
	if len(schema) == 0 {
		t.Fatal("tools endpoint returned no modes")
	}
}

func uploadFile(t *testing.T, baseURL, name string, data []byte) string {
	t.Helper()
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	fw, err := mw.CreateFormFile("file", name)
	if err != nil {
		t.Fatal(err)
	}
	fw.Write(data)
	mw.Close()
	resp, err := http.Post(baseURL+"/api/v1/cryptolab/files", mw.FormDataContentType(), &buf)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("upload status %d: %s", resp.StatusCode, b)
	}
	var out struct {
		ID string `json:"id"`
	}
	json.NewDecoder(resp.Body).Decode(&out)
	if out.ID == "" {
		t.Fatal("upload returned no id")
	}
	return out.ID
}

func TestUploadRunPoll(t *testing.T) {
	t.Parallel()
	ts := newTestServer(t)
	// A clear single-byte-XOR ciphertext so brute/xor has something to find.
	pt := []byte("CALL UNIT 12 TO DISPATCH ON CHANNEL SEVEN RIGHT NOW PLEASE OKAY")
	ct := make([]byte, len(pt))
	for i := range pt {
		ct[i] = pt[i] ^ 0x5B
	}
	id := uploadFile(t, ts.URL, "cipher.bin", ct)

	runBody, _ := json.Marshal(map[string]any{
		"tool":   "brute",
		"mode":   "xor",
		"params": map[string]string{"crib": "UNIT"},
		"files":  map[string]string{"in": id},
	})
	resp, err := http.Post(ts.URL+"/api/v1/cryptolab/run", "application/json", bytes.NewReader(runBody))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusAccepted {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("run status %d: %s", resp.StatusCode, b)
	}
	var jr struct {
		JobID string `json:"job_id"`
	}
	json.NewDecoder(resp.Body).Decode(&jr)

	// Poll the job until it finishes.
	deadline := time.Now().Add(5 * time.Second)
	for {
		r, err := http.Get(ts.URL + "/api/v1/cryptolab/jobs/" + jr.JobID)
		if err != nil {
			t.Fatal(err)
		}
		var dto struct {
			State  string `json:"state"`
			Result *struct {
				Tool string `json:"tool"`
				Mode string `json:"mode"`
			} `json:"result"`
		}
		json.NewDecoder(r.Body).Decode(&dto)
		r.Body.Close()
		if dto.State == "done" {
			if dto.Result == nil || dto.Result.Tool != "brute" {
				t.Fatalf("unexpected result %+v", dto.Result)
			}
			return
		}
		if dto.State == "error" {
			t.Fatal("job errored")
		}
		if time.Now().After(deadline) {
			t.Fatal("job did not finish in time")
		}
		time.Sleep(20 * time.Millisecond)
	}
}

func TestRunUnknownToolRejected(t *testing.T) {
	t.Parallel()
	ts := newTestServer(t)
	body := strings.NewReader(`{"tool":"nope","mode":"x"}`)
	resp, err := http.Post(ts.URL+"/api/v1/cryptolab/run", "application/json", body)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", resp.StatusCode)
	}
}

func TestSPAPlaceholderWhenNoAssets(t *testing.T) {
	t.Parallel()
	ts := newTestServer(t)
	resp, err := http.Get(ts.URL + "/")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(b), "SPA not bundled") {
		t.Fatalf("placeholder missing: %s", b)
	}
}
