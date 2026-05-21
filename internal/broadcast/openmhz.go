package broadcast

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
)

// openMHzEndpoint is the production OpenMHz upload API. The system
// short name is appended as a path segment.
const openMHzEndpoint = "https://api.openmhz.com"

// OpenMHzConfig configures one OpenMHz upload feed.
type OpenMHzConfig struct {
	// Name is an optional log label; defaults to "openmhz".
	Name string
	// APIKey is the OpenMHz system API key.
	APIKey string
	// ShortName is the OpenMHz system short name (the path segment in
	// the upload URL).
	ShortName string
	// Systems restricts the feed to these trunking-system names.
	// Empty streams every system.
	Systems []string
	// Endpoint overrides the API base URL (tests). Empty uses the
	// production endpoint.
	Endpoint string
}

type openMHzBackend struct {
	systemFilter
	name     string
	apiKey   string
	endpoint string
	http     *http.Client
}

// NewOpenMHz builds an OpenMHz upload backend.
func NewOpenMHz(cfg OpenMHzConfig, hc *http.Client) (Backend, error) {
	if cfg.APIKey == "" {
		return nil, errors.New("broadcast/openmhz: api_key is required")
	}
	if cfg.ShortName == "" {
		return nil, errors.New("broadcast/openmhz: short_name is required")
	}
	name := cfg.Name
	if name == "" {
		name = "openmhz"
	}
	base := cfg.Endpoint
	if base == "" {
		base = openMHzEndpoint
	}
	if hc == nil {
		hc = http.DefaultClient
	}
	return &openMHzBackend{
		systemFilter: newSystemFilter(cfg.Systems),
		name:         name,
		apiKey:       cfg.APIKey,
		endpoint:     strings.TrimRight(base, "/") + "/" + cfg.ShortName + "/upload",
		http:         hc,
	}, nil
}

func (b *openMHzBackend) Name() string { return b.name }

// Send uploads the call to OpenMHz as a multipart/form-data POST.
func (b *openMHzBackend) Send(ctx context.Context, c *Call) error {
	audio, err := c.MP3()
	if err != nil {
		return fmt.Errorf("%s: encode mp3: %w", b.name, err)
	}
	emergency := "0"
	if c.Emergency {
		emergency = "1"
	}
	// source_list and patch_list are JSON arrays OpenMHz expects even
	// when there is only the single granting unit to report.
	sourceList := fmt.Sprintf(`[{"src":%d,"time":%d,"pos":0}]`,
		c.Source, c.StartedAt.Unix())
	patchList := "[]"
	if len(c.PatchedGroups) > 0 {
		parts := make([]string, len(c.PatchedGroups))
		for i, g := range c.PatchedGroups {
			parts[i] = strconv.FormatUint(uint64(g), 10)
		}
		patchList = "[" + strings.Join(parts, ",") + "]"
	}

	fields := []multipartField{
		{Name: "api_key", Value: b.apiKey},
		{Name: "freq", Value: strconv.FormatUint(uint64(c.FrequencyHz), 10)},
		{Name: "start_time", Value: strconv.FormatInt(c.StartedAt.Unix(), 10)},
		{Name: "stop_time", Value: strconv.FormatInt(c.EndedAt.Unix(), 10)},
		{Name: "call_length", Value: strconv.Itoa(int(c.Duration().Seconds()))},
		{Name: "talkgroup_num", Value: strconv.FormatUint(uint64(c.Talkgroup), 10)},
		{Name: "emergency", Value: emergency},
		{Name: "source_list", Value: sourceList},
		{Name: "patch_list", Value: patchList},
		{Name: "call", Filename: audioFilename(c, "mp3"), Data: audio},
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
