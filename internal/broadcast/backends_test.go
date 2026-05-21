package broadcast

import (
	"bufio"
	"context"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"
)

func testCall(t *testing.T) *Call {
	t.Helper()
	now := time.Now()
	return &Call{
		System:         "Metro",
		Protocol:       "p25",
		Talkgroup:      4321,
		TalkgroupLabel: "Dispatch",
		Source:         9001,
		FrequencyHz:    851_012_500,
		StartedAt:      now.Add(-3 * time.Second),
		EndedAt:        now,
		AudioPath:      writeWAV(t, 0.4),
		SampleRate:     8000,
	}
}

func TestBroadcastifyTwoStepUpload(t *testing.T) {
	var gotForm, gotAudio bool
	mux := http.NewServeMux()
	srv := httptest.NewServer(mux)
	defer srv.Close()

	mux.HandleFunc("/metadata", func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Errorf("parse form: %v", err)
		}
		if r.PostForm.Get("apiKey") != "KEY" || r.PostForm.Get("systemId") != "77" {
			t.Errorf("bad metadata form: %v", r.PostForm)
		}
		if r.PostForm.Get("tg") != "4321" {
			t.Errorf("tg = %q, want 4321", r.PostForm.Get("tg"))
		}
		gotForm = true
		io.WriteString(w, "0 "+srv.URL+"/upload")
	})
	mux.HandleFunc("/upload", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("audio upload method = %s, want PUT", r.Method)
		}
		body, _ := io.ReadAll(r.Body)
		if len(body) < 2 || body[0] != 0xFF {
			t.Errorf("audio body is not MP3 (%d bytes)", len(body))
		}
		gotAudio = true
		w.WriteHeader(http.StatusOK)
	})

	be, err := NewBroadcastify(BroadcastifyConfig{
		APIKey: "KEY", SystemID: 77, Endpoint: srv.URL + "/metadata",
	}, srv.Client())
	if err != nil {
		t.Fatalf("NewBroadcastify: %v", err)
	}
	if err := be.Send(context.Background(), testCall(t)); err != nil {
		t.Fatalf("Send: %v", err)
	}
	if !gotForm || !gotAudio {
		t.Fatalf("two-step incomplete: form=%v audio=%v", gotForm, gotAudio)
	}
}

func TestBroadcastifyRejectsErrorResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, "1 invalid api key")
	}))
	defer srv.Close()
	be, _ := NewBroadcastify(BroadcastifyConfig{
		APIKey: "BAD", SystemID: 1, Endpoint: srv.URL,
	}, srv.Client())
	if err := be.Send(context.Background(), testCall(t)); err == nil {
		t.Fatal("Send should fail on an error metadata response")
	}
}

func TestParseBroadcastifyUploadURL(t *testing.T) {
	cases := []struct {
		body, want string
		wantErr    bool
	}{
		{"0 https://up.example/x", "https://up.example/x", false},
		{"0\nhttps://up.example/y", "https://up.example/y", false},
		{"https://up.example/z", "https://up.example/z", false},
		{"1 bad key", "", true},
		{"", "", true},
	}
	for _, c := range cases {
		got, err := parseBroadcastifyUploadURL(c.body)
		if c.wantErr {
			if err == nil {
				t.Errorf("%q: expected error", c.body)
			}
			continue
		}
		if err != nil || got != c.want {
			t.Errorf("%q: got (%q, %v), want %q", c.body, got, err, c.want)
		}
	}
}

func TestRdioScannerUpload(t *testing.T) {
	var ok bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/call-upload" {
			t.Errorf("path = %s", r.URL.Path)
		}
		if err := r.ParseMultipartForm(8 << 20); err != nil {
			t.Errorf("parse multipart: %v", err)
			return
		}
		if r.FormValue("key") != "RKEY" || r.FormValue("system") != "12" {
			t.Errorf("bad fields: key=%q system=%q", r.FormValue("key"), r.FormValue("system"))
		}
		if r.FormValue("talkgroup") != "4321" {
			t.Errorf("talkgroup = %q", r.FormValue("talkgroup"))
		}
		f, _, err := r.FormFile("audio")
		if err != nil {
			t.Errorf("audio part missing: %v", err)
			return
		}
		defer f.Close()
		body, _ := io.ReadAll(f)
		if len(body) < 2 || body[0] != 0xFF {
			t.Errorf("audio part is not MP3")
		}
		ok = true
		io.WriteString(w, "Call imported successfully.")
	}))
	defer srv.Close()

	be, err := NewRdioScanner(RdioScannerConfig{
		URL: srv.URL, APIKey: "RKEY", SystemID: 12,
	}, srv.Client())
	if err != nil {
		t.Fatalf("NewRdioScanner: %v", err)
	}
	if err := be.Send(context.Background(), testCall(t)); err != nil {
		t.Fatalf("Send: %v", err)
	}
	if !ok {
		t.Fatal("server did not receive a well-formed upload")
	}
}

func TestRdioScannerReportsHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "bad key", http.StatusUnauthorized)
	}))
	defer srv.Close()
	be, _ := NewRdioScanner(RdioScannerConfig{URL: srv.URL, APIKey: "X", SystemID: 1}, srv.Client())
	if err := be.Send(context.Background(), testCall(t)); err == nil {
		t.Fatal("Send should fail on HTTP 401")
	}
}

func TestOpenMHzUpload(t *testing.T) {
	var ok bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/metro911/upload" {
			t.Errorf("path = %s, want /metro911/upload", r.URL.Path)
		}
		if err := r.ParseMultipartForm(8 << 20); err != nil {
			t.Errorf("parse multipart: %v", err)
			return
		}
		if r.FormValue("api_key") != "OKEY" {
			t.Errorf("api_key = %q", r.FormValue("api_key"))
		}
		if r.FormValue("talkgroup_num") != "4321" {
			t.Errorf("talkgroup_num = %q", r.FormValue("talkgroup_num"))
		}
		if _, _, err := r.FormFile("call"); err != nil {
			t.Errorf("call part missing: %v", err)
		}
		ok = true
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	be, err := NewOpenMHz(OpenMHzConfig{
		APIKey: "OKEY", ShortName: "metro911", Endpoint: srv.URL,
	}, srv.Client())
	if err != nil {
		t.Fatalf("NewOpenMHz: %v", err)
	}
	if err := be.Send(context.Background(), testCall(t)); err != nil {
		t.Fatalf("Send: %v", err)
	}
	if !ok {
		t.Fatal("server did not receive a well-formed upload")
	}
}

func TestBackendConstructorsValidate(t *testing.T) {
	if _, err := NewBroadcastify(BroadcastifyConfig{SystemID: 1}, nil); err == nil {
		t.Error("Broadcastify without api_key should error")
	}
	if _, err := NewRdioScanner(RdioScannerConfig{APIKey: "x", SystemID: 1}, nil); err == nil {
		t.Error("RdioScanner without url should error")
	}
	if _, err := NewOpenMHz(OpenMHzConfig{APIKey: "x"}, nil); err == nil {
		t.Error("OpenMHz without short_name should error")
	}
	if _, err := NewIcecast(IcecastConfig{Host: "h", Port: 8000}, nil); err == nil {
		t.Error("Icecast without password should error")
	}
}

func TestIcecastSourceHandshakeAndStream(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	host, port, _ := net.SplitHostPort(ln.Addr().String())

	handshake := make(chan string, 1)
	streamed := make(chan int, 1)
	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		br := bufio.NewReader(conn)
		reqLine, _ := br.ReadString('\n')
		// Drain the remaining request headers up to the blank line.
		for {
			line, err := br.ReadString('\n')
			if err != nil || strings.TrimSpace(line) == "" {
				break
			}
		}
		handshake <- strings.TrimSpace(reqLine)
		io.WriteString(conn, "HTTP/1.0 200 OK\r\n\r\n")
		conn.SetReadDeadline(time.Now().Add(2 * time.Second))
		buf := make([]byte, 64*1024)
		n, _ := br.Read(buf)
		streamed <- n
	}()

	portNum, _ := strconv.Atoi(port)
	be, err := NewIcecast(IcecastConfig{
		Host: host, Port: portNum, Mount: "/gt", Password: "hackme",
	}, nil)
	if err != nil {
		t.Fatalf("NewIcecast: %v", err)
	}
	defer be.(interface{ Close() error }).Close()

	select {
	case req := <-handshake:
		if !strings.HasPrefix(req, "SOURCE /gt") {
			t.Fatalf("handshake request = %q, want SOURCE /gt ...", req)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("no source handshake received")
	}

	if err := be.Send(context.Background(), testCall(t)); err != nil {
		t.Fatalf("Send: %v", err)
	}
	select {
	case n := <-streamed:
		if n == 0 {
			t.Fatal("no bytes streamed to the mountpoint")
		}
	case <-time.After(3 * time.Second):
		t.Fatal("no audio streamed")
	}
}
