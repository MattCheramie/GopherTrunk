package radioreference

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"strings"
)

// SearchHit is a single trunked system returned by a geography search
// (zip / county / state). Enrich it with GetFullSystem before importing.
type SearchHit struct {
	SID    int    `json:"sid"`
	Name   string `json:"name"`
	Type   string `json:"type"`
	County string `json:"county,omitempty"`
	State  string `json:"state,omitempty"`
}

// FullSystem is the import-oriented projection of a RadioReference trunked
// system: identity plus its sites (with control-channel + voice
// frequencies) and talkgroups. It carries enough to fold into a
// config.SystemConfig (control channels collapse across sites, since the
// config schema is flat) and to write a talkgroup CSV sidecar.
type FullSystem struct {
	SID        int               `json:"sid"`
	Name       string            `json:"name"`
	Type       string            `json:"type"`             // e.g. "Project 25"
	Flavor     string            `json:"flavor,omitempty"` // e.g. "Phase II"
	Voice      string            `json:"voice,omitempty"`
	Protocol   string            `json:"protocol"` // mapped GopherTrunk protocol id
	SystemID   uint16            `json:"system_id,omitempty"`
	WACN       uint32            `json:"wacn,omitempty"`
	NAC        uint16            `json:"nac,omitempty"`
	City       string            `json:"city,omitempty"`
	County     string            `json:"county,omitempty"`
	State      string            `json:"state,omitempty"`
	Sites      []SiteDetail      `json:"sites"`
	Talkgroups []TalkgroupDetail `json:"talkgroups"`
}

// SiteDetail is one RF site of a trunked system. ControlChannels and
// Frequencies are in Hz.
type SiteDetail struct {
	RFSS            int      `json:"rfss,omitempty"`
	SiteNumber      int      `json:"site_number,omitempty"`
	Description     string   `json:"description,omitempty"`
	County          string   `json:"county,omitempty"`
	ControlChannels []uint32 `json:"control_channels"`
	Frequencies     []uint32 `json:"frequencies"`
}

// TalkgroupDetail is one talkgroup of a trunked system, shaped to match
// the columns trunking.TalkGroup loads from a Trunk Recorder–style CSV.
type TalkgroupDetail struct {
	Dec         uint32 `json:"dec"`
	AlphaTag    string `json:"alpha_tag,omitempty"`
	Description string `json:"description,omitempty"`
	Tag         string `json:"tag,omitempty"`
	Group       string `json:"group,omitempty"`
	Mode        string `json:"mode,omitempty"` // D / A / M
	Encrypted   bool   `json:"encrypted,omitempty"`
}

// SearchByZip resolves a US ZIP code to its county, then lists the
// trunked systems registered in that county.
func (c *Client) SearchByZip(ctx context.Context, zip string) ([]SearchHit, error) {
	zip = strings.TrimSpace(zip)
	if zip == "" {
		return nil, fmt.Errorf("radioreference: empty zip")
	}
	body := fmt.Sprintf(`<ns1:getZipcodeInfo>`+
		`<zipcode xsi:type="xsd:string">%s</zipcode>`+
		`%s`+
		`</ns1:getZipcodeInfo>`, xmlEscape(zip), c.authXML())
	raw, err := c.call(ctx, body)
	if err != nil {
		return nil, err
	}
	leaves := firstLeaves(raw, "ctid")
	ctid := atoiDefault(leaves["ctid"], 0)
	if ctid == 0 {
		return nil, fmt.Errorf("radioreference: zip %s resolved no county", zip)
	}
	return c.SearchByCounty(ctx, ctid)
}

// SearchByCounty lists the trunked systems registered in a RadioReference
// county (by ctid).
func (c *Client) SearchByCounty(ctx context.Context, ctid int) ([]SearchHit, error) {
	systems, err := c.GetCountyInfo(ctx, ctid)
	if err != nil {
		return nil, err
	}
	hits := make([]SearchHit, 0, len(systems))
	for _, s := range systems {
		hits = append(hits, SearchHit{SID: s.SID, Name: s.Name, Type: s.Type})
	}
	return hits, nil
}

// SearchByState enumerates a state's counties (by stid) and aggregates the
// trunked systems registered in each. Duplicate systems (registered in
// multiple counties) are de-duplicated by SID.
func (c *Client) SearchByState(ctx context.Context, stid int) ([]SearchHit, error) {
	body := fmt.Sprintf(`<ns1:getStateInfo>`+
		`<stid xsi:type="xsd:int">%d</stid>`+
		`%s`+
		`</ns1:getStateInfo>`, stid, c.authXML())
	raw, err := c.call(ctx, body)
	if err != nil {
		return nil, err
	}
	var hits []SearchHit
	seen := make(map[int]struct{})
	for _, block := range blocks(raw, "countyList") {
		ctid := atoiDefault(firstLeaves(block, "ctid")["ctid"], 0)
		if ctid == 0 {
			continue
		}
		systems, err := c.GetCountyInfo(ctx, ctid)
		if err != nil {
			return nil, err
		}
		for _, s := range systems {
			if _, dup := seen[s.SID]; dup {
				continue
			}
			seen[s.SID] = struct{}{}
			hits = append(hits, SearchHit{SID: s.SID, Name: s.Name, Type: s.Type})
		}
	}
	return hits, nil
}

// GetFullSystem fetches a system's identity, sites, and talkgroups and
// returns an import-ready FullSystem with a mapped GopherTrunk protocol.
func (c *Client) GetFullSystem(ctx context.Context, sid int) (FullSystem, error) {
	body := fmt.Sprintf(`<ns1:getTrsDetails>`+
		`<sid xsi:type="xsd:int">%d</sid>`+
		`%s`+
		`</ns1:getTrsDetails>`, sid, c.authXML())
	raw, err := c.call(ctx, body)
	if err != nil {
		return FullSystem{}, err
	}

	leaves := firstLeaves(raw, "sid", "sName", "sType", "sFlavor", "sVoice",
		"sysid", "wacn", "nac", "city", "cName", "sName2", "stateName")
	fs := FullSystem{
		SID:    atoiDefault(leaves["sid"], sid),
		Name:   leaves["sName"],
		Type:   leaves["sType"],
		Flavor: leaves["sFlavor"],
		Voice:  leaves["sVoice"],
		City:   leaves["city"],
		County: leaves["cName"],
		State:  leaves["stateName"],
	}
	if v, ok := parseHexOrDec(leaves["sysid"]); ok {
		fs.SystemID = uint16(v)
	}
	if v, ok := parseHexOrDec(leaves["wacn"]); ok {
		fs.WACN = uint32(v)
	}
	if v, ok := parseHexOrDec(leaves["nac"]); ok {
		fs.NAC = uint16(v)
	}
	fs.Protocol = protocolFromType(fs.Type, fs.Flavor)
	fs.Sites = parseSites(raw)
	fs.Talkgroups = parseTalkgroups(raw)

	// getTrsDetails responses don't always embed sites/talkgroups; fall
	// back to the dedicated endpoints when the embedded lists are empty.
	if len(fs.Sites) == 0 {
		if sites, err := c.getTrsSites(ctx, sid); err == nil {
			fs.Sites = sites
		}
	}
	if len(fs.Talkgroups) == 0 {
		if tgs, err := c.getTrsTalkgroups(ctx, sid); err == nil {
			fs.Talkgroups = tgs
		}
	}
	return fs, nil
}

func (c *Client) getTrsSites(ctx context.Context, sid int) ([]SiteDetail, error) {
	body := fmt.Sprintf(`<ns1:getTrsSites>`+
		`<sid xsi:type="xsd:int">%d</sid>`+
		`%s`+
		`</ns1:getTrsSites>`, sid, c.authXML())
	raw, err := c.call(ctx, body)
	if err != nil {
		return nil, err
	}
	return parseSites(raw), nil
}

func (c *Client) getTrsTalkgroups(ctx context.Context, sid int) ([]TalkgroupDetail, error) {
	body := fmt.Sprintf(`<ns1:getTrsTalkgroups>`+
		`<sid xsi:type="xsd:int">%d</sid>`+
		`%s`+
		`</ns1:getTrsTalkgroups>`, sid, c.authXML())
	raw, err := c.call(ctx, body)
	if err != nil {
		return nil, err
	}
	return parseTalkgroups(raw), nil
}

// parseSites extracts every <siteList> entry and its frequencies. A
// frequency whose "use" flag marks it a control/data channel lands in
// ControlChannels; all frequencies (control + voice) land in Frequencies.
//
// RadioReference's XML is namespaced SOAP; firstLeaves/blocks are
// intentionally tolerant first-match scanners (not schema-validated), so a
// renamed/missing element degrades to a zero value rather than an error.
// Site entries that yield no usable frequency are dropped so an empty or
// metadata-only <siteList> doesn't surface as a zero-channel site.
func parseSites(raw []byte) []SiteDetail {
	var sites []SiteDetail
	for _, block := range blocks(raw, "siteList") {
		leaves := firstLeaves(block, "rfss", "siteNumber", "siteDescr", "siteCounty", "cName")
		site := SiteDetail{
			RFSS:        atoiDefault(leaves["rfss"], 0),
			SiteNumber:  atoiDefault(leaves["siteNumber"], 0),
			Description: leaves["siteDescr"],
			County:      firstNonEmpty(leaves["siteCounty"], leaves["cName"]),
		}
		for _, fb := range blocks(block, "siteFreq") {
			fl := firstLeaves(fb, "freq", "use")
			hz := mhzToHz(fl["freq"])
			if hz == 0 {
				continue
			}
			site.Frequencies = append(site.Frequencies, hz)
			if isControlUse(fl["use"]) {
				site.ControlChannels = append(site.ControlChannels, hz)
			}
		}
		if len(site.Frequencies) == 0 {
			continue
		}
		sites = append(sites, site)
	}
	return sites
}

// parseTalkgroups extracts every <tgList> entry.
func parseTalkgroups(raw []byte) []TalkgroupDetail {
	var tgs []TalkgroupDetail
	for _, block := range blocks(raw, "tgList") {
		leaves := firstLeaves(block, "tgDec", "tgAlpha", "tgDescr", "tgSort", "tgTag", "tag", "enc", "mode")
		dec := atoiDefault(leaves["tgDec"], 0)
		if dec == 0 {
			continue
		}
		tg := TalkgroupDetail{
			Dec:         uint32(dec),
			AlphaTag:    leaves["tgAlpha"],
			Description: leaves["tgDescr"],
			Tag:         firstNonEmpty(leaves["tag"], leaves["tgTag"]),
			Mode:        normalizeMode(leaves["mode"]),
			Encrypted:   leaves["enc"] == "1" || strings.EqualFold(leaves["enc"], "true"),
		}
		tgs = append(tgs, tg)
	}
	return tgs
}

// isControlUse reports whether a site-frequency "use" flag marks a
// control/data channel. RadioReference uses "d" for the primary control
// channel and "a" for alternate control; voice channels carry no flag.
func isControlUse(use string) bool {
	switch strings.ToLower(strings.TrimSpace(use)) {
	case "d", "a", "c":
		return true
	}
	return false
}

// normalizeMode maps RadioReference's mode flags to the D/A/M codes
// trunking.TalkGroup expects.
func normalizeMode(mode string) string {
	switch strings.ToUpper(strings.TrimSpace(mode)) {
	case "D", "DE", "TDMA":
		return "D"
	case "A", "AE":
		return "A"
	case "M":
		return "M"
	}
	return ""
}

// protocolFromType maps a RadioReference system type + flavor string to
// the protocol id trunking.ParseProtocol accepts, so an imported system
// passes config validation. Unrecognised types fall back to "p25" (the
// most common case) and should be reviewed by the operator.
func protocolFromType(sType, flavor string) string {
	t := strings.ToLower(sType + " " + flavor)
	switch {
	case strings.Contains(t, "phase ii"), strings.Contains(t, "phase 2"):
		return "p25-phase2"
	case strings.Contains(t, "project 25"), strings.Contains(t, "p25"):
		return "p25"
	case strings.Contains(t, "tier iii"), strings.Contains(t, "tier 3"),
		strings.Contains(t, "capacity plus"), strings.Contains(t, "connect plus"):
		return "dmr"
	case strings.Contains(t, "tier ii"), strings.Contains(t, "tier 2"):
		return "dmr-tier2"
	case strings.Contains(t, "dmr"), strings.Contains(t, "mototrbo"):
		return "dmr"
	case strings.Contains(t, "nxdn"):
		return "nxdn"
	case strings.Contains(t, "edacs"):
		return "edacs"
	case strings.Contains(t, "ltr"):
		return "ltr"
	case strings.Contains(t, "mpt"):
		return "mpt1327"
	case strings.Contains(t, "tetra"):
		return "tetra"
	case strings.Contains(t, "motorola"), strings.Contains(t, "smartzone"),
		strings.Contains(t, "type ii"):
		return "motorola"
	}
	return "p25"
}

// mhzToHz parses a decimal-MHz string (RadioReference's frequency format,
// e.g. "851.0125") and returns the rounded value in Hz. Returns 0 for
// empty / unparseable / non-positive input, and for values that would
// overflow the uint32 Hz field the config schema uses (≈4294.97 MHz) so a
// garbage MHz string can't wrap to a plausible-looking frequency.
func mhzToHz(s string) uint32 {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}
	f, err := strconv.ParseFloat(s, 64)
	if err != nil || f <= 0 {
		return 0
	}
	hz := math.Round(f * 1e6)
	if hz > math.MaxUint32 {
		return 0
	}
	return uint32(hz)
}

// firstNonEmpty returns the first non-empty string of its arguments.
func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}
