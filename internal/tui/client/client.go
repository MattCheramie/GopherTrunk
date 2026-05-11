package client

import (
	"bufio"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const userAgent = "gophertrunk-tui/1"

// Client is a typed HTTP client for the daemon's read API.
//
// The base URL is normalised to drop any trailing slash; methods
// build paths relative to it.
type Client struct {
	base    string
	hc      *http.Client
	timeout time.Duration
}

// New constructs a Client. timeout applies per request; the SSE
// stream uses its own context-bounded request that ignores this
// timeout (a long-lived stream is the normal case for SSE).
//
// insecure disables TLS certificate verification — useful when
// pointing at a self-signed daemon over HTTPS in a lab.
func New(baseURL string, timeout time.Duration, insecure bool) *Client {
	base := strings.TrimRight(baseURL, "/")
	tr := http.DefaultTransport.(*http.Transport).Clone()
	if insecure {
		tr.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	}
	return &Client{
		base:    base,
		timeout: timeout,
		hc: &http.Client{
			Transport: tr,
		},
	}
}

// Base returns the daemon base URL the client is pointed at. Used
// in the TUI's status bar.
func (c *Client) Base() string { return c.base }

// HTTPClient returns the underlying http.Client so the SSE reader
// can issue a long-lived request without the per-call timeout.
func (c *Client) HTTPClient() *http.Client { return c.hc }

// Health calls GET /api/v1/health.
func (c *Client) Health(ctx context.Context) (Health, error) {
	var h Health
	return h, c.getJSON(ctx, "/api/v1/health", &h)
}

// Version calls GET /api/v1/version.
func (c *Client) Version(ctx context.Context) (string, error) {
	var v Version
	if err := c.getJSON(ctx, "/api/v1/version", &v); err != nil {
		return "", err
	}
	return v.Version, nil
}

type systemsResp struct {
	Systems []SystemDTO `json:"systems"`
}
type talkgroupsResp struct {
	Talkgroups []TalkgroupDTO `json:"talkgroups"`
}
type activeCallsResp struct {
	Calls []ActiveCallDTO `json:"calls"`
}
type historyResp struct {
	Calls []CallRow `json:"calls"`
}
type devicesResp struct {
	Devices []SDRStatus `json:"devices"`
}

// Systems calls GET /api/v1/systems.
func (c *Client) Systems(ctx context.Context) ([]SystemDTO, error) {
	var r systemsResp
	if err := c.getJSON(ctx, "/api/v1/systems", &r); err != nil {
		return nil, err
	}
	return r.Systems, nil
}

// Talkgroups calls GET /api/v1/talkgroups.
func (c *Client) Talkgroups(ctx context.Context) ([]TalkgroupDTO, error) {
	var r talkgroupsResp
	if err := c.getJSON(ctx, "/api/v1/talkgroups", &r); err != nil {
		return nil, err
	}
	return r.Talkgroups, nil
}

// Scanner calls GET /api/v1/scanner. Always returns 200 even when
// no scanner subsystem is wired (an empty ScannerStatusDTO).
func (c *Client) Scanner(ctx context.Context) (ScannerStatusDTO, error) {
	var s ScannerStatusDTO
	if err := c.getJSON(ctx, "/api/v1/scanner", &s); err != nil {
		return ScannerStatusDTO{}, err
	}
	return s, nil
}

// Devices calls GET /api/v1/devices.
func (c *Client) Devices(ctx context.Context) ([]SDRStatus, error) {
	var r devicesResp
	if err := c.getJSON(ctx, "/api/v1/devices", &r); err != nil {
		return nil, err
	}
	return r.Devices, nil
}

// System calls GET /api/v1/systems/{name} and returns the detail
// record for one system. Used by the TUI's drill-in modal.
func (c *Client) System(ctx context.Context, name string) (SystemDTO, error) {
	var s SystemDTO
	if err := c.getJSON(ctx, "/api/v1/systems/"+url.PathEscape(name), &s); err != nil {
		return SystemDTO{}, err
	}
	return s, nil
}

// Talkgroup calls GET /api/v1/talkgroups/{id} and returns the detail
// record for one talkgroup. Used by the TUI's drill-in modal.
func (c *Client) Talkgroup(ctx context.Context, id uint32) (TalkgroupDTO, error) {
	var t TalkgroupDTO
	if err := c.getJSON(ctx, "/api/v1/talkgroups/"+strconv.FormatUint(uint64(id), 10), &t); err != nil {
		return TalkgroupDTO{}, err
	}
	return t, nil
}

// ActiveCalls calls GET /api/v1/calls/active.
func (c *Client) ActiveCalls(ctx context.Context) ([]ActiveCallDTO, error) {
	var r activeCallsResp
	if err := c.getJSON(ctx, "/api/v1/calls/active", &r); err != nil {
		return nil, err
	}
	return r.Calls, nil
}

// History calls GET /api/v1/calls/history with the supplied filter.
func (c *Client) History(ctx context.Context, f HistoryFilter) ([]CallRow, error) {
	q := url.Values{}
	if f.System != "" {
		q.Set("system", f.System)
	}
	if f.GroupID != 0 {
		q.Set("group_id", strconv.FormatUint(uint64(f.GroupID), 10))
	}
	if !f.Since.IsZero() {
		q.Set("since", f.Since.UTC().Format(time.RFC3339))
	}
	if !f.Until.IsZero() {
		q.Set("until", f.Until.UTC().Format(time.RFC3339))
	}
	if f.Limit > 0 {
		q.Set("limit", strconv.Itoa(f.Limit))
	}
	if f.OnlyEnded {
		q.Set("only_ended", "true")
	}
	path := "/api/v1/calls/history"
	if len(q) > 0 {
		path += "?" + q.Encode()
	}
	var r historyResp
	if err := c.getJSON(ctx, path, &r); err != nil {
		return nil, err
	}
	return r.Calls, nil
}

// Metrics calls GET /metrics, parses Prometheus text-format output,
// and returns a curated map of name → most recent sample value. We
// don't try to be a full Prometheus parser — just enough to fuel the
// Metrics panel's small set of named series.
func (c *Client) Metrics(ctx context.Context) (map[string]float64, error) {
	ctx, cancel := c.withTimeout(ctx)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.base+"/metrics", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", userAgent)
	resp, err := c.hc.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		return nil, c.httpErr(http.MethodGet, req.URL.String(), resp)
	}

	out := make(map[string]float64)
	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 0, 64*1024), 1<<20)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		// Format: name{labels} value [timestamp]
		// We strip {labels} and aggregate by base name (sum across
		// label sets) — a lossy projection good enough for at-a-
		// glance counters.
		name, valStr := splitMetricLine(line)
		if name == "" {
			continue
		}
		val, err := strconv.ParseFloat(valStr, 64)
		if err != nil {
			continue
		}
		out[name] += val
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

// splitMetricLine parses one Prometheus line into base name + value.
// Returns ("", "") on malformed input.
func splitMetricLine(line string) (string, string) {
	i := strings.IndexAny(line, "{ ")
	if i < 0 {
		return "", ""
	}
	name := line[:i]
	rest := line[i:]
	if rest[0] == '{' {
		end := strings.Index(rest, "} ")
		if end < 0 {
			return "", ""
		}
		rest = rest[end+2:]
	} else {
		rest = strings.TrimLeft(rest, " ")
	}
	// Value is the first whitespace-delimited token.
	parts := strings.Fields(rest)
	if len(parts) == 0 {
		return "", ""
	}
	return name, parts[0]
}

// getJSON performs an authenticated GET, decodes JSON, surfaces
// non-2xx as *HTTPError.
func (c *Client) getJSON(ctx context.Context, path string, out any) error {
	ctx, cancel := c.withTimeout(ctx)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.base+path, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "application/json")
	resp, err := c.hc.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		return c.httpErr(http.MethodGet, req.URL.String(), resp)
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

func (c *Client) withTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	if c.timeout <= 0 {
		return ctx, func() {}
	}
	return context.WithTimeout(ctx, c.timeout)
}

func (c *Client) httpErr(method, urlStr string, resp *http.Response) error {
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
	return &HTTPError{
		Status: resp.StatusCode,
		Method: method,
		URL:    urlStr,
		Body:   strings.TrimSpace(string(body)),
	}
}

// Quote-style helpers — kept here so tests can lean on them without
// re-implementing.

// FormatFreqMHz formats a frequency in Hz as "851.012500 MHz" with
// six decimal places. Zero returns "—".
func FormatFreqMHz(hz uint32) string {
	if hz == 0 {
		return "—"
	}
	mhz := float64(hz) / 1_000_000
	return fmt.Sprintf("%.6f MHz", mhz)
}
