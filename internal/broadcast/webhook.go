package broadcast

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// WebhookConfig configures one generic JSON-webhook call sink.
type WebhookConfig struct {
	// Name is an optional log label; defaults to "webhook".
	Name string
	// URL is the endpoint the call payload is POSTed to (required).
	URL string
	// AuthHeader, when set, is sent verbatim as the Authorization header
	// (e.g. "Bearer <token>"). Empty omits the header.
	AuthHeader string
	// IncludeAudio, when true, embeds the MP3-encoded call audio as a
	// base64 string in the payload (audio_base64 / audio_format). Off by
	// default so the webhook stays a lightweight metadata feed; consumers
	// that want audio can opt in.
	IncludeAudio bool
	// Systems restricts the feed to these trunking-system names. Empty
	// streams every system.
	Systems []string
	// Loc is the timezone the StartedAt/EndedAt RFC3339 timestamps render in.
	// Nil means time.Local. The daemon passes display.timezone so the payload
	// matches the operator's other timestamps; the format stays RFC3339, so the
	// offset (e.g. +02:00, or Z for UTC) keeps the instant unambiguous.
	Loc *time.Location
}

// rfc3339In formats t as RFC3339 in loc (default time.Local). Centralises the
// display-timezone rendering shared by the webhook and rdioscanner payloads.
func rfc3339In(t time.Time, loc *time.Location) string {
	if loc == nil {
		loc = time.Local
	}
	return t.In(loc).Format(time.RFC3339)
}

type webhookBackend struct {
	systemFilter
	name         string
	endpoint     string
	authHeader   string
	includeAudio bool
	loc          *time.Location
	http         *http.Client
}

// NewWebhook builds a generic JSON-webhook backend that POSTs one JSON
// object per completed call (issue #404 / #268). The payload is a stable,
// documented schema (see webhookPayload) carrying the call metadata and,
// when IncludeAudio is set, the base64 MP3.
func NewWebhook(cfg WebhookConfig, hc *http.Client) (Backend, error) {
	if cfg.URL == "" {
		return nil, errors.New("broadcast/webhook: url is required")
	}
	name := cfg.Name
	if name == "" {
		name = "webhook"
	}
	if hc == nil {
		hc = http.DefaultClient
	}
	return &webhookBackend{
		systemFilter: newSystemFilter(cfg.Systems),
		name:         name,
		endpoint:     cfg.URL,
		authHeader:   cfg.AuthHeader,
		includeAudio: cfg.IncludeAudio,
		loc:          cfg.Loc,
		http:         hc,
	}, nil
}

func (b *webhookBackend) Name() string { return b.name }

// webhookPayload is the stable JSON schema POSTed for each call. Fields
// that don't apply to a given call (site identity on non-P25, encryption
// params on clear calls, audio when not requested) are omitted via
// omitempty so a consumer can rely on presence meaning "known".
type webhookPayload struct {
	Event         string   `json:"event"` // always "call"
	System        string   `json:"system"`
	Protocol      string   `json:"protocol"`
	Talkgroup     uint32   `json:"talkgroup"`
	TalkgroupID   string   `json:"talkgroup_label,omitempty"`
	Source        uint32   `json:"source,omitempty"`
	FrequencyHz   uint32   `json:"frequency_hz"`
	ChannelID     uint8    `json:"channel_id,omitempty"`
	RFSSID        uint8    `json:"rfss_id,omitempty"`
	SiteID        uint8    `json:"site_id,omitempty"`
	NAC           uint16   `json:"nac,omitempty"`
	Timeslot      uint8    `json:"timeslot,omitempty"`
	Encrypted     bool     `json:"encrypted"`
	AlgorithmID   uint8    `json:"algorithm_id,omitempty"`
	KeyID         uint16   `json:"key_id,omitempty"`
	Emergency     bool     `json:"emergency"`
	PatchedGroups []uint32 `json:"patched_groups,omitempty"`
	StartedAt     string   `json:"started_at"` // RFC3339 UTC
	EndedAt       string   `json:"ended_at"`   // RFC3339 UTC
	DurationMs    int64    `json:"duration_ms"`
	AudioFilename string   `json:"audio_filename,omitempty"`
	AudioFormat   string   `json:"audio_format,omitempty"`
	AudioBase64   string   `json:"audio_base64,omitempty"`
}

// Send POSTs the call as a single application/json object. The audio is
// only MP3-encoded when IncludeAudio is set, so a metadata-only webhook
// never pays the encode cost.
func (b *webhookBackend) Send(ctx context.Context, c *Call) error {
	p := webhookPayload{
		Event:         "call",
		System:        c.System,
		Protocol:      c.Protocol,
		Talkgroup:     c.Talkgroup,
		TalkgroupID:   c.TalkgroupLabel,
		Source:        c.Source,
		FrequencyHz:   c.FrequencyHz,
		ChannelID:     c.ChannelID,
		RFSSID:        c.RFSS,
		SiteID:        c.Site,
		NAC:           c.NAC,
		Timeslot:      c.Timeslot,
		Encrypted:     c.Encrypted,
		AlgorithmID:   c.AlgorithmID,
		KeyID:         c.KeyID,
		Emergency:     c.Emergency,
		PatchedGroups: c.PatchedGroups,
		StartedAt:     rfc3339In(c.StartedAt, b.loc),
		EndedAt:       rfc3339In(c.EndedAt, b.loc),
		DurationMs:    c.Duration().Milliseconds(),
		AudioFilename: audioFilename(c, "mp3"),
	}
	if b.includeAudio {
		audio, err := c.MP3()
		if err != nil {
			return fmt.Errorf("%s: encode mp3: %w", b.name, err)
		}
		p.AudioFormat = "mp3"
		p.AudioBase64 = base64.StdEncoding.EncodeToString(audio)
	}

	body, err := json.Marshal(p)
	if err != nil {
		return fmt.Errorf("%s: marshal: %w", b.name, err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, b.endpoint, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if b.authHeader != "" {
		req.Header.Set("Authorization", b.authHeader)
	}

	resp, err := b.http.Do(req)
	if err != nil {
		return fmt.Errorf("%s: post: %w", b.name, err)
	}
	defer resp.Body.Close()
	msg, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if resp.StatusCode/100 != 2 {
		return fmt.Errorf("%s: HTTP %d: %s", b.name, resp.StatusCode,
			strings.TrimSpace(string(msg)))
	}
	return nil
}
