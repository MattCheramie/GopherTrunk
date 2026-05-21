package broadcast

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

// broadcastifyEndpoint is the production Broadcastify Calls upload API.
const broadcastifyEndpoint = "https://api.broadcastify.com/call-upload"

// BroadcastifyConfig configures one Broadcastify Calls feed.
type BroadcastifyConfig struct {
	// Name is an optional label used in logs; defaults to
	// "broadcastify".
	Name string
	// APIKey is the Broadcastify Calls API key for the system.
	APIKey string
	// SystemID is the numeric Broadcastify Calls system ID.
	SystemID int
	// Systems restricts the feed to these trunking-system names.
	// Empty streams every system.
	Systems []string
	// Endpoint overrides the upload API URL (tests). Empty uses the
	// production endpoint.
	Endpoint string
}

type broadcastifyBackend struct {
	systemFilter
	name     string
	apiKey   string
	systemID int
	endpoint string
	http     *http.Client
}

// NewBroadcastify builds a Broadcastify Calls backend.
func NewBroadcastify(cfg BroadcastifyConfig, hc *http.Client) (Backend, error) {
	if cfg.APIKey == "" {
		return nil, errors.New("broadcast/broadcastify: api_key is required")
	}
	if cfg.SystemID == 0 {
		return nil, errors.New("broadcast/broadcastify: system_id is required")
	}
	name := cfg.Name
	if name == "" {
		name = "broadcastify"
	}
	endpoint := cfg.Endpoint
	if endpoint == "" {
		endpoint = broadcastifyEndpoint
	}
	if hc == nil {
		hc = http.DefaultClient
	}
	return &broadcastifyBackend{
		systemFilter: newSystemFilter(cfg.Systems),
		name:         name,
		apiKey:       cfg.APIKey,
		systemID:     cfg.SystemID,
		endpoint:     endpoint,
		http:         hc,
	}, nil
}

func (b *broadcastifyBackend) Name() string { return b.name }

// Send performs the Broadcastify Calls two-step upload: a metadata POST
// that returns a one-time upload URL, then a PUT of the MP3 audio.
func (b *broadcastifyBackend) Send(ctx context.Context, c *Call) error {
	audio, err := c.MP3()
	if err != nil {
		return fmt.Errorf("%s: encode mp3: %w", b.name, err)
	}
	uploadURL, err := b.requestUploadURL(ctx, c)
	if err != nil {
		return err
	}
	return b.putAudio(ctx, uploadURL, audio)
}

func (b *broadcastifyBackend) requestUploadURL(ctx context.Context, c *Call) (string, error) {
	form := url.Values{}
	form.Set("apiKey", b.apiKey)
	form.Set("systemId", strconv.Itoa(b.systemID))
	form.Set("callDuration", strconv.Itoa(int(c.Duration().Seconds())))
	form.Set("ts", strconv.FormatInt(c.StartedAt.Unix(), 10))
	form.Set("tg", strconv.FormatUint(uint64(c.Talkgroup), 10))
	form.Set("src", strconv.FormatUint(uint64(c.Source), 10))
	form.Set("freq", strconv.FormatUint(uint64(c.FrequencyHz), 10))

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, b.endpoint,
		strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := b.http.Do(req)
	if err != nil {
		return "", fmt.Errorf("%s: metadata post: %w", b.name, err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("%s: metadata post: HTTP %d: %s",
			b.name, resp.StatusCode, strings.TrimSpace(string(body)))
	}
	return parseBroadcastifyUploadURL(string(body))
}

// parseBroadcastifyUploadURL extracts the one-time upload URL from the
// Broadcastify Calls metadata response. The API answers either
// "0 <url>" / "0\n<url>" on success or an error string otherwise.
func parseBroadcastifyUploadURL(body string) (string, error) {
	fields := strings.Fields(strings.TrimSpace(body))
	if len(fields) == 0 {
		return "", errors.New("broadcastify: empty metadata response")
	}
	if fields[0] == "0" && len(fields) >= 2 {
		return fields[1], nil
	}
	if strings.HasPrefix(fields[0], "http") {
		return fields[0], nil
	}
	return "", fmt.Errorf("broadcastify: metadata response rejected: %s",
		strings.TrimSpace(body))
}

func (b *broadcastifyBackend) putAudio(ctx context.Context, uploadURL string, audio []byte) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, uploadURL,
		bytes.NewReader(audio))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "audio/mpeg")
	req.ContentLength = int64(len(audio))

	resp, err := b.http.Do(req)
	if err != nil {
		return fmt.Errorf("%s: audio put: %w", b.name, err)
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))
	if resp.StatusCode/100 != 2 {
		return fmt.Errorf("%s: audio put: HTTP %d", b.name, resp.StatusCode)
	}
	return nil
}
