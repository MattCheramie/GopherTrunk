package broadcast

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

// p25SiteCall is testCall enriched with the P25 site identity the webhook
// payload surfaces (issue #698).
func p25SiteCall(t *testing.T) *Call {
	t.Helper()
	c := testCall(t)
	c.ChannelID = 1
	c.RFSS = 1
	c.Site = 9
	c.NAC = 359
	c.Emergency = true
	c.PatchedGroups = []uint32{201, 212}
	return c
}

func TestWebhookPostsJSONMetadata(t *testing.T) {
	var got webhookPayload
	var gotAuth, gotCT string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		gotAuth = r.Header.Get("Authorization")
		gotCT = r.Header.Get("Content-Type")
		if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
			t.Errorf("decode body: %v", err)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	be, err := NewWebhook(WebhookConfig{
		URL:        srv.URL,
		AuthHeader: "Bearer t0ken",
	}, srv.Client())
	if err != nil {
		t.Fatalf("NewWebhook: %v", err)
	}
	if err := be.Send(context.Background(), p25SiteCall(t)); err != nil {
		t.Fatalf("Send: %v", err)
	}

	if gotCT != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", gotCT)
	}
	if gotAuth != "Bearer t0ken" {
		t.Errorf("Authorization = %q, want Bearer t0ken", gotAuth)
	}
	if got.Event != "call" {
		t.Errorf("event = %q, want call", got.Event)
	}
	if got.System != "Metro" || got.Talkgroup != 4321 || got.Source != 9001 {
		t.Errorf("core fields wrong: %+v", got)
	}
	if got.RFSSID != 1 || got.SiteID != 9 || got.ChannelID != 1 || got.NAC != 359 {
		t.Errorf("site identity not surfaced: rfss=%d site=%d ch=%d nac=%d",
			got.RFSSID, got.SiteID, got.ChannelID, got.NAC)
	}
	if !got.Emergency {
		t.Error("emergency flag not surfaced")
	}
	if len(got.PatchedGroups) != 2 || got.PatchedGroups[0] != 201 {
		t.Errorf("patched_groups = %v, want [201 212]", got.PatchedGroups)
	}
	if got.StartedAt == "" || got.EndedAt == "" || got.DurationMs <= 0 {
		t.Errorf("timing fields wrong: start=%q end=%q dur=%d", got.StartedAt, got.EndedAt, got.DurationMs)
	}
	// Metadata-only by default: no audio embedded.
	if got.AudioBase64 != "" || got.AudioFormat != "" {
		t.Errorf("audio embedded without include_audio: fmt=%q len=%d", got.AudioFormat, len(got.AudioBase64))
	}
}

// TestWebhookCallTypeAndEncrypted pins issue #897: the payload must carry a
// call_type so a unit (individual) call's destination RID in the talkgroup
// field isn't mistaken for a talkgroup, and it must faithfully report the
// call's encryption state (the serializer used to hardcode neither, so unit
// RIDs looked like talkgroups and every call read encrypted=false).
func TestWebhookCallTypeAndEncrypted(t *testing.T) {
	send := func(t *testing.T, mutate func(*Call)) webhookPayload {
		t.Helper()
		var got webhookPayload
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
				t.Errorf("decode body: %v", err)
			}
			w.WriteHeader(http.StatusNoContent)
		}))
		defer srv.Close()
		be, err := NewWebhook(WebhookConfig{URL: srv.URL}, srv.Client())
		if err != nil {
			t.Fatalf("NewWebhook: %v", err)
		}
		c := testCall(t)
		mutate(c)
		if err := be.Send(context.Background(), c); err != nil {
			t.Fatalf("Send: %v", err)
		}
		return got
	}

	t.Run("group call", func(t *testing.T) {
		got := send(t, func(c *Call) {})
		if got.CallType != "group" {
			t.Errorf("call_type = %q, want group", got.CallType)
		}
		if got.Encrypted {
			t.Error("clear call reported encrypted")
		}
	})

	t.Run("unit call surfaces call_type and encryption", func(t *testing.T) {
		got := send(t, func(c *Call) {
			c.Individual = true
			c.Talkgroup = 206606 // a destination RID, not a talkgroup
			c.Encrypted = true
			c.AlgorithmID = 0x84
			c.KeyID = 0x1234
		})
		if got.CallType != "unit" {
			t.Errorf("call_type = %q, want unit", got.CallType)
		}
		if !got.Encrypted {
			t.Error("encrypted call reported clear")
		}
		if got.AlgorithmID != 0x84 || got.KeyID != 0x1234 {
			t.Errorf("alg/key not surfaced: alg=0x%02X key=0x%04X", got.AlgorithmID, got.KeyID)
		}
	})

	t.Run("data call", func(t *testing.T) {
		got := send(t, func(c *Call) { c.DataCall = true })
		if got.CallType != "data" {
			t.Errorf("call_type = %q, want data", got.CallType)
		}
	})
}

func TestWebhookIncludeAudioEmbedsMP3(t *testing.T) {
	var got webhookPayload
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&got)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	be, _ := NewWebhook(WebhookConfig{URL: srv.URL, IncludeAudio: true}, srv.Client())
	if err := be.Send(context.Background(), testCall(t)); err != nil {
		t.Fatalf("Send: %v", err)
	}
	if got.AudioFormat != "mp3" || got.AudioBase64 == "" {
		t.Fatalf("audio not embedded: fmt=%q len=%d", got.AudioFormat, len(got.AudioBase64))
	}
}

func TestWebhookReportsHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "nope", http.StatusInternalServerError)
	}))
	defer srv.Close()
	be, _ := NewWebhook(WebhookConfig{URL: srv.URL}, srv.Client())
	if err := be.Send(context.Background(), testCall(t)); err == nil {
		t.Fatal("Send should fail on HTTP 500")
	}
}

func TestWebhookOmitsAuthHeaderWhenUnset(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, ok := r.Header["Authorization"]; ok {
			t.Errorf("Authorization header sent when unset")
		}
		io.Copy(io.Discard, r.Body)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()
	be, _ := NewWebhook(WebhookConfig{URL: srv.URL}, srv.Client())
	if err := be.Send(context.Background(), testCall(t)); err != nil {
		t.Fatalf("Send: %v", err)
	}
}

func TestWebhookRequiresURL(t *testing.T) {
	if _, err := NewWebhook(WebhookConfig{}, nil); err == nil {
		t.Error("Webhook without url should error")
	}
}

func TestWebhookSystemFilter(t *testing.T) {
	be, _ := NewWebhook(WebhookConfig{URL: "http://x", Systems: []string{"OnlyThis"}}, nil)
	if be.Accepts("Metro") {
		t.Error("filter should reject unlisted system")
	}
	if !be.Accepts("OnlyThis") {
		t.Error("filter should accept listed system")
	}
}
