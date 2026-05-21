package broadcast

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// RdioScannerConfig configures one RdioScanner call-upload feed.
type RdioScannerConfig struct {
	// Name is an optional log label; defaults to "rdioscanner".
	Name string
	// URL is the RdioScanner base URL (e.g. "https://scanner.example").
	// The "/api/call-upload" path is appended automatically.
	URL string
	// APIKey is the RdioScanner system API key.
	APIKey string
	// SystemID is the numeric RdioScanner system ID.
	SystemID int
	// Systems restricts the feed to these trunking-system names.
	// Empty streams every system.
	Systems []string
}

type rdioScannerBackend struct {
	systemFilter
	name     string
	endpoint string
	apiKey   string
	systemID int
	http     *http.Client
}

// NewRdioScanner builds an RdioScanner call-upload backend.
func NewRdioScanner(cfg RdioScannerConfig, hc *http.Client) (Backend, error) {
	if cfg.URL == "" {
		return nil, errors.New("broadcast/rdioscanner: url is required")
	}
	if cfg.APIKey == "" {
		return nil, errors.New("broadcast/rdioscanner: api_key is required")
	}
	if cfg.SystemID == 0 {
		return nil, errors.New("broadcast/rdioscanner: system_id is required")
	}
	name := cfg.Name
	if name == "" {
		name = "rdioscanner"
	}
	if hc == nil {
		hc = http.DefaultClient
	}
	return &rdioScannerBackend{
		systemFilter: newSystemFilter(cfg.Systems),
		name:         name,
		endpoint:     strings.TrimRight(cfg.URL, "/") + "/api/call-upload",
		apiKey:       cfg.APIKey,
		systemID:     cfg.SystemID,
		http:         hc,
	}, nil
}

func (b *rdioScannerBackend) Name() string { return b.name }

// Send uploads the call to RdioScanner as a multipart/form-data POST.
func (b *rdioScannerBackend) Send(ctx context.Context, c *Call) error {
	audio, err := c.MP3()
	if err != nil {
		return fmt.Errorf("%s: encode mp3: %w", b.name, err)
	}
	fields := []multipartField{
		{Name: "key", Value: b.apiKey},
		{Name: "system", Value: strconv.Itoa(b.systemID)},
		{Name: "dateTime", Value: c.StartedAt.UTC().Format(time.RFC3339)},
		{Name: "talkgroup", Value: strconv.FormatUint(uint64(c.Talkgroup), 10)},
		{Name: "source", Value: strconv.FormatUint(uint64(c.Source), 10)},
		{Name: "frequency", Value: strconv.FormatUint(uint64(c.FrequencyHz), 10)},
		{Name: "audioName", Value: audioFilename(c, "mp3")},
		{Name: "audioType", Value: "audio/mpeg"},
		{Name: "audio", Filename: audioFilename(c, "mp3"), Data: audio},
	}
	if c.TalkgroupLabel != "" {
		fields = append(fields, multipartField{Name: "talkgroupLabel", Value: c.TalkgroupLabel})
	}
	if c.System != "" {
		fields = append(fields, multipartField{Name: "systemLabel", Value: c.System})
	}

	body, contentType, err := buildMultipart(fields)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, b.endpoint, body)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", contentType)

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

// audioFilename builds a stable per-call filename for an upload part.
func audioFilename(c *Call, ext string) string {
	return fmt.Sprintf("%d-%d.%s", c.Talkgroup, c.StartedAt.Unix(), ext)
}
