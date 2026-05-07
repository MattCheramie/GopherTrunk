package client

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestEndCall_PostsBody(t *testing.T) {
	var got struct {
		Method, Path, Body string
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got.Method = r.Method
		got.Path = r.URL.Path
		body, _ := io.ReadAll(r.Body)
		got.Body = string(body)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()
	c := New(srv.URL, time.Second, false)

	if err := c.EndCall(context.Background(), "00000001", "manual"); err != nil {
		t.Fatal(err)
	}
	if got.Method != "POST" {
		t.Errorf("method = %s", got.Method)
	}
	if got.Path != "/api/v1/calls/00000001/end" {
		t.Errorf("path = %s", got.Path)
	}
	if !strings.Contains(got.Body, `"reason":"manual"`) {
		t.Errorf("body = %s", got.Body)
	}
}

func TestEndCall_403_SurfacesHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(403)
		_, _ = w.Write([]byte(`{"error":"mutations disabled"}`))
	}))
	defer srv.Close()
	c := New(srv.URL, time.Second, false)
	err := c.EndCall(context.Background(), "abc", "")
	if err == nil {
		t.Fatal("want error")
	}
	var herr *HTTPError
	if !asHTTPErr(err, &herr) {
		t.Fatalf("want *HTTPError, got %T", err)
	}
	if herr.Status != 403 {
		t.Errorf("status = %d", herr.Status)
	}
}

func TestUpdateTalkgroup_OmitsNilFields(t *testing.T) {
	var body string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		buf, _ := io.ReadAll(r.Body)
		body = string(buf)
		_, _ = w.Write([]byte(`{"id":42,"alpha_tag":"x","priority":3}`))
	}))
	defer srv.Close()
	c := New(srv.URL, time.Second, false)
	pri := 3
	out, err := c.UpdateTalkgroup(context.Background(), 42, &pri, nil)
	if err != nil {
		t.Fatal(err)
	}
	if out.ID != 42 {
		t.Errorf("ID = %d", out.ID)
	}
	if !strings.Contains(body, `"priority":3`) {
		t.Errorf("body = %s", body)
	}
	if strings.Contains(body, `"lockout"`) {
		t.Errorf("body should omit lockout when nil: %s", body)
	}
}

func TestUpdateTalkgroup_RequiresAtLeastOneField(t *testing.T) {
	c := New("http://example.invalid", time.Second, false)
	_, err := c.UpdateTalkgroup(context.Background(), 42, nil, nil)
	if err == nil {
		t.Fatal("want error when both fields are nil")
	}
}

func TestSweepRetention_PostsEmpty(t *testing.T) {
	got := struct {
		method, path string
		bodyLen      int
	}{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got.method, got.path = r.Method, r.URL.Path
		body, _ := io.ReadAll(r.Body)
		got.bodyLen = len(body)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()
	c := New(srv.URL, time.Second, false)
	if err := c.SweepRetention(context.Background()); err != nil {
		t.Fatal(err)
	}
	if got.method != "POST" || got.path != "/api/v1/retention/sweep" {
		t.Errorf("method=%s path=%s", got.method, got.path)
	}
	if got.bodyLen != 0 {
		t.Errorf("body should be empty, got %d bytes", got.bodyLen)
	}
}

func TestResetToneDevice_PathHasSerial(t *testing.T) {
	var path string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()
	c := New(srv.URL, time.Second, false)
	if err := c.ResetToneDevice(context.Background(), "00000001"); err != nil {
		t.Fatal(err)
	}
	if path != "/api/v1/devices/00000001/tone-reset" {
		t.Errorf("path = %s", path)
	}
}

func TestMutationStatus_Decodes(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]bool{
			"allow_mutations":    true,
			"engine_writable":    true,
			"retention_writable": false,
			"tones_writable":     true,
		})
	}))
	defer srv.Close()
	c := New(srv.URL, time.Second, false)
	s, err := c.MutationStatus(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !s.AllowMutations || !s.EngineWritable || s.RetentionWritable || !s.TonesWritable {
		t.Errorf("decoded unexpectedly: %+v", s)
	}
}

func TestMutationStatus_404_ReturnsZero(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	}))
	defer srv.Close()
	c := New(srv.URL, time.Second, false)
	s, err := c.MutationStatus(context.Background())
	if err != nil {
		t.Fatalf("404 should not return error, got %v", err)
	}
	if s != (MutationStatus{}) {
		t.Errorf("want zero MutationStatus on 404, got %+v", s)
	}
}
