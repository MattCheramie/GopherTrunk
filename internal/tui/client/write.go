package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// MutationStatus mirrors GET /api/v1/mutations. The TUI fetches it
// once at startup to decide which write keybindings to expose. A
// daemon that doesn't know the route (older build) returns 404 and
// the TUI treats that as "no mutations".
type MutationStatus struct {
	AllowMutations    bool `json:"allow_mutations"`
	EngineWritable    bool `json:"engine_writable"`
	RetentionWritable bool `json:"retention_writable"`
	TonesWritable     bool `json:"tones_writable"`
}

// MutationStatus calls GET /api/v1/mutations. Returns a zero-value
// status (and a nil error) if the daemon doesn't know the route.
func (c *Client) MutationStatus(ctx context.Context) (MutationStatus, error) {
	var s MutationStatus
	err := c.getJSON(ctx, "/api/v1/mutations", &s)
	if err != nil {
		var herr *HTTPError
		if asHTTPErr(err, &herr) && herr.Status == http.StatusNotFound {
			return MutationStatus{}, nil
		}
		return MutationStatus{}, err
	}
	return s, nil
}

// EndCall calls POST /api/v1/calls/{deviceSerial}/end. reason is
// optional; defaults to "manual".
func (c *Client) EndCall(ctx context.Context, deviceSerial, reason string) error {
	if deviceSerial == "" {
		return fmt.Errorf("client: deviceSerial required")
	}
	body := map[string]string{"reason": reason}
	return c.do(ctx, http.MethodPost,
		"/api/v1/calls/"+deviceSerial+"/end",
		body, nil)
}

// UpdateTalkgroup calls PATCH /api/v1/talkgroups/{id}. Pass nil for
// fields you don't want to change.
func (c *Client) UpdateTalkgroup(ctx context.Context, id uint32, priority *int, lockout *bool, scan *bool) (TalkgroupDTO, error) {
	body := map[string]any{}
	if priority != nil {
		body["priority"] = *priority
	}
	if lockout != nil {
		body["lockout"] = *lockout
	}
	if scan != nil {
		body["scan"] = *scan
	}
	if len(body) == 0 {
		return TalkgroupDTO{}, fmt.Errorf("client: supply priority, lockout, or scan")
	}
	var out TalkgroupDTO
	if err := c.do(ctx, http.MethodPatch,
		fmt.Sprintf("/api/v1/talkgroups/%d", id),
		body, &out); err != nil {
		return TalkgroupDTO{}, err
	}
	return out, nil
}

// SweepRetention calls POST /api/v1/retention/sweep.
func (c *Client) SweepRetention(ctx context.Context) error {
	return c.do(ctx, http.MethodPost, "/api/v1/retention/sweep", nil, nil)
}

// ScannerSetMode calls PATCH /api/v1/scanner with the new global
// scan_mode ("all" or "list").
func (c *Client) ScannerSetMode(ctx context.Context, mode string) error {
	return c.do(ctx, http.MethodPatch, "/api/v1/scanner",
		map[string]string{"scan_mode": mode}, nil)
}

// ScannerHuntHold / ScannerHuntResume / ScannerHuntRetune call the
// per-system hunt mutation endpoints. system must match a configured
// trunked system name.
func (c *Client) ScannerHuntHold(ctx context.Context, system string) error {
	return c.do(ctx, http.MethodPost,
		"/api/v1/scanner/hunt/"+system+"/hold", nil, nil)
}
func (c *Client) ScannerHuntResume(ctx context.Context, system string) error {
	return c.do(ctx, http.MethodPost,
		"/api/v1/scanner/hunt/"+system+"/resume", nil, nil)
}
func (c *Client) ScannerHuntRetune(ctx context.Context, system string) error {
	return c.do(ctx, http.MethodPost,
		"/api/v1/scanner/hunt/"+system+"/retune", nil, nil)
}

// ScannerConvHold / ScannerConvResume / ScannerConvDwell drive the
// conventional FM scanner.
func (c *Client) ScannerConvHold(ctx context.Context) error {
	return c.do(ctx, http.MethodPost, "/api/v1/scanner/conventional/hold", nil, nil)
}
func (c *Client) ScannerConvResume(ctx context.Context) error {
	return c.do(ctx, http.MethodPost, "/api/v1/scanner/conventional/resume", nil, nil)
}
func (c *Client) ScannerConvDwell(ctx context.Context, index int) error {
	return c.do(ctx, http.MethodPost,
		fmt.Sprintf("/api/v1/scanner/conventional/%d/dwell", index),
		nil, nil)
}

// AudioStatusDTO mirrors api.AudioStatusDTO for the wire layer.
type AudioStatusDTO struct {
	BackendEnabled   bool    `json:"backend_enabled"`
	SampleRate       uint32  `json:"sample_rate"`
	Volume           float32 `json:"volume"`
	Muted            bool    `json:"muted"`
	RecordingEnabled bool    `json:"recording_enabled"`
	DropsTotal       uint64  `json:"drops_total"`
}

// AudioStatus calls GET /api/v1/audio. Returns a zero value (no
// error) when the daemon doesn't have the audio cockpit wired so
// older daemons don't break the TUI.
func (c *Client) AudioStatus(ctx context.Context) (AudioStatusDTO, error) {
	var s AudioStatusDTO
	err := c.getJSON(ctx, "/api/v1/audio", &s)
	if err != nil {
		var herr *HTTPError
		if asHTTPErr(err, &herr) && (herr.Status == http.StatusNotFound || herr.Status == http.StatusServiceUnavailable) {
			return AudioStatusDTO{}, nil
		}
		return AudioStatusDTO{}, err
	}
	return s, nil
}

// SetAudio calls PATCH /api/v1/audio with whichever knobs are non-nil.
// Pass nil to leave a field unchanged.
func (c *Client) SetAudio(ctx context.Context, volume *float32, muted *bool, recording *bool) (AudioStatusDTO, error) {
	body := map[string]any{}
	if volume != nil {
		body["volume"] = *volume
	}
	if muted != nil {
		body["muted"] = *muted
	}
	if recording != nil {
		body["recording_enabled"] = *recording
	}
	if len(body) == 0 {
		return AudioStatusDTO{}, fmt.Errorf("client: supply volume, muted, or recording_enabled")
	}
	var out AudioStatusDTO
	if err := c.do(ctx, http.MethodPatch, "/api/v1/audio", body, &out); err != nil {
		return AudioStatusDTO{}, err
	}
	return out, nil
}

// ResetToneDevice calls POST /api/v1/devices/{serial}/tone-reset.
func (c *Client) ResetToneDevice(ctx context.Context, serial string) error {
	if serial == "" {
		return fmt.Errorf("client: serial required")
	}
	return c.do(ctx, http.MethodPost,
		"/api/v1/devices/"+serial+"/tone-reset",
		nil, nil)
}

// do is the generic JSON request/response helper for write methods.
// Responses with 2xx + a non-nil out get JSON-decoded; nil out
// discards the body. Non-2xx returns *HTTPError.
func (c *Client) do(ctx context.Context, method, path string, in any, out any) error {
	ctx, cancel := c.withTimeout(ctx)
	defer cancel()
	var bodyR *bytes.Reader
	if in != nil {
		buf, err := json.Marshal(in)
		if err != nil {
			return err
		}
		bodyR = bytes.NewReader(buf)
	}
	var req *http.Request
	var err error
	if bodyR != nil {
		req, err = http.NewRequestWithContext(ctx, method, c.base+path, bodyR)
	} else {
		req, err = http.NewRequestWithContext(ctx, method, c.base+path, nil)
	}
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "application/json")
	if in != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.hc.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		return c.httpErr(method, req.URL.String(), resp)
	}
	if out == nil {
		return nil
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

// asHTTPErr is a tiny wrapper around errors.As that avoids dragging
// the import into the call site.
func asHTTPErr(err error, target **HTTPError) bool {
	if err == nil {
		return false
	}
	if h, ok := err.(*HTTPError); ok {
		*target = h
		return true
	}
	return false
}
