package radioreference

import (
	"bytes"
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// DefaultEndpoint is the RadioReference SOAP web-service endpoint (v3.1).
const DefaultEndpoint = "http://api.radioreference.com/soap2/"

// ErrNoCredentials is returned by NewClient when no API key is configured, so
// callers can cleanly skip the (optional) duplicate check.
var ErrNoCredentials = errors.New("radioreference: no API key configured")

// Auth carries the RadioReference web-service credentials.
type Auth struct {
	AppKey   string
	Username string
	Password string
	Version  string // defaults to "latest"
}

// Client is a read-only RadioReference SOAP client. It is safe for sequential
// use; create one per hunt run.
type Client struct {
	auth     Auth
	endpoint string
	http     *http.Client
}

// NewClient builds a client. It returns ErrNoCredentials when auth.AppKey is
// empty so the hunt can degrade gracefully to "no duplicate check".
func NewClient(auth Auth) (*Client, error) {
	if strings.TrimSpace(auth.AppKey) == "" {
		return nil, ErrNoCredentials
	}
	if auth.Version == "" {
		auth.Version = "latest"
	}
	return &Client{
		auth:     auth,
		endpoint: DefaultEndpoint,
		http:     &http.Client{Timeout: 20 * time.Second},
	}, nil
}

// SetEndpoint overrides the SOAP endpoint (used by tests).
func (c *Client) SetEndpoint(url string) { c.endpoint = url }

// SetHTTPClient overrides the HTTP client (used by tests).
func (c *Client) SetHTTPClient(h *http.Client) { c.http = h }

// GetTrsDetails fetches a single trunked system by RadioReference system id.
// The response is parsed namespace-agnostically (the RR WSDL wraps the return
// in method-specific elements that vary by SOAP style), pulling the leaf
// fields used for duplicate detection.
func (c *Client) GetTrsDetails(ctx context.Context, sid int) (System, error) {
	body := fmt.Sprintf(`<ns1:getTrsDetails>`+
		`<sid xsi:type="xsd:int">%d</sid>`+
		`%s`+
		`</ns1:getTrsDetails>`, sid, c.authXML())
	raw, err := c.call(ctx, body)
	if err != nil {
		return System{}, err
	}
	leaves := firstLeaves(raw, "sid", "sName", "sType", "sysid", "wacn")
	sys := System{
		SID:  atoiDefault(leaves["sid"], sid),
		Name: leaves["sName"],
		Type: leaves["sType"],
	}
	// RR returns sysid/wacn as hex strings on P25 systems.
	if v, ok := parseHexOrDec(leaves["sysid"]); ok {
		sys.SystemID = uint16(v)
	}
	if v, ok := parseHexOrDec(leaves["wacn"]); ok {
		sys.WACN = uint32(v)
	}
	return sys, nil
}

// GetCountyInfo fetches the trunked systems registered in a RadioReference
// county (by ctid). Only the brief identity of each system is returned; call
// GetTrsDetails to enrich a candidate with its SYSID/WACN before matching.
func (c *Client) GetCountyInfo(ctx context.Context, ctid int) ([]System, error) {
	body := fmt.Sprintf(`<ns1:getCountyInfo>`+
		`<ctid xsi:type="xsd:int">%d</ctid>`+
		`%s`+
		`</ns1:getCountyInfo>`, ctid, c.authXML())
	raw, err := c.call(ctx, body)
	if err != nil {
		return nil, err
	}
	var out []System
	for _, block := range blocks(raw, "trsList") {
		leaves := firstLeaves(block, "sid", "sName", "sType")
		if leaves["sid"] == "" {
			continue
		}
		out = append(out, System{
			SID:  atoiDefault(leaves["sid"], 0),
			Name: leaves["sName"],
			Type: leaves["sType"],
		})
	}
	return out, nil
}

// authXML renders the shared authInfo SOAP block.
func (c *Client) authXML() string {
	return fmt.Sprintf(`<authInfo xsi:type="ns1:authInfo">`+
		`<appKey xsi:type="xsd:string">%s</appKey>`+
		`<username xsi:type="xsd:string">%s</username>`+
		`<password xsi:type="xsd:string">%s</password>`+
		`<version xsi:type="xsd:string">%s</version>`+
		`<style xsi:type="xsd:string">rpc</style>`+
		`</authInfo>`,
		xmlEscape(c.auth.AppKey), xmlEscape(c.auth.Username),
		xmlEscape(c.auth.Password), xmlEscape(c.auth.Version))
}

// call wraps a method body in a SOAP envelope, POSTs it, and returns the raw
// response body (HTTP errors and SOAP faults become Go errors).
func (c *Client) call(ctx context.Context, methodBody string) ([]byte, error) {
	envelope := `<?xml version="1.0" encoding="UTF-8"?>` +
		`<SOAP-ENV:Envelope ` +
		`xmlns:SOAP-ENV="http://schemas.xmlsoap.org/soap/envelope/" ` +
		`xmlns:ns1="urn:RadioReference" ` +
		`xmlns:xsd="http://www.w3.org/2001/XMLSchema" ` +
		`xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance">` +
		`<SOAP-ENV:Body>` + methodBody + `</SOAP-ENV:Body></SOAP-ENV:Envelope>`

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint, strings.NewReader(envelope))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "text/xml; charset=utf-8")
	req.Header.Set("SOAPAction", "")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("radioreference: request: %w", err)
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return nil, fmt.Errorf("radioreference: read response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		// RadioReference answers an unknown/mismatched operation with an
		// HTTP 500 whose body is a SOAP fault; surface the concise
		// <faultstring> rather than dumping the raw XML envelope at the
		// user. See issue #849.
		if fault := firstLeaves(raw, "faultstring")["faultstring"]; fault != "" {
			return nil, fmt.Errorf("radioreference: HTTP %d: %s", resp.StatusCode, fault)
		}
		return nil, fmt.Errorf("radioreference: HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(raw)))
	}
	if fault := firstLeaves(raw, "faultstring")["faultstring"]; fault != "" {
		return nil, fmt.Errorf("radioreference: SOAP fault: %s", fault)
	}
	return raw, nil
}

// firstLeaves walks the XML token stream and records the first text content of
// each requested leaf element, matched by local name (namespace/prefix
// ignored). This is resilient to the differing wrapper elements the RR WSDL
// emits across SOAP styles.
func firstLeaves(raw []byte, names ...string) map[string]string {
	want := make(map[string]struct{}, len(names))
	for _, n := range names {
		want[n] = struct{}{}
	}
	out := make(map[string]string, len(names))
	dec := xml.NewDecoder(bytes.NewReader(raw))
	var cur string
	for {
		tok, err := dec.Token()
		if err != nil {
			break
		}
		switch t := tok.(type) {
		case xml.StartElement:
			if _, ok := want[t.Name.Local]; ok {
				cur = t.Name.Local
			} else {
				cur = ""
			}
		case xml.CharData:
			if cur != "" {
				if _, seen := out[cur]; !seen {
					if s := strings.TrimSpace(string(t)); s != "" {
						out[cur] = s
					}
				}
			}
		case xml.EndElement:
			cur = ""
		}
	}
	return out
}

// blocks returns the raw XML of each element whose local name matches name,
// so a list response (repeated <trsList>…</trsList> entries) can be parsed
// one entry at a time with firstLeaves.
func blocks(raw []byte, name string) [][]byte {
	var out [][]byte
	dec := xml.NewDecoder(bytes.NewReader(raw))
	for {
		tok, err := dec.Token()
		if err != nil {
			break
		}
		se, ok := tok.(xml.StartElement)
		if !ok || se.Name.Local != name {
			continue
		}
		var buf bytes.Buffer
		enc := xml.NewEncoder(&buf)
		_ = enc.EncodeToken(se)
		depth := 1
		for depth > 0 {
			t, err := dec.Token()
			if err != nil {
				break
			}
			switch t.(type) {
			case xml.StartElement:
				depth++
			case xml.EndElement:
				depth--
			}
			if depth >= 0 {
				_ = enc.EncodeToken(xml.CopyToken(t))
			}
		}
		_ = enc.Flush()
		out = append(out, buf.Bytes())
	}
	return out
}

func atoiDefault(s string, def int) int {
	if v, err := strconv.Atoi(strings.TrimSpace(s)); err == nil {
		return v
	}
	return def
}

// parseHexOrDec accepts RR's hex identity strings (sysid/wacn) and falls back
// to decimal. Returns ok=false for empty/garbage input.
func parseHexOrDec(s string) (uint64, bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, false
	}
	if v, err := strconv.ParseUint(s, 16, 64); err == nil {
		return v, true
	}
	if v, err := strconv.ParseUint(s, 10, 64); err == nil {
		return v, true
	}
	return 0, false
}

func xmlEscape(s string) string {
	var b strings.Builder
	_ = xml.EscapeText(&b, []byte(s))
	return b.String()
}
