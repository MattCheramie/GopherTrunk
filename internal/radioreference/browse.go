package radioreference

import (
	"context"
	"errors"
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
	// Notes are things the operator should know about a system that did
	// import: a protocol that had to be guessed, or a system RadioReference
	// lists no sites or no talkgroups for.
	Notes []string `json:"notes,omitempty"`
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
	// Latitude / Longitude are the site's geographic position in decimal
	// degrees (RadioReference siteLat / siteLng), zero when RR did not
	// geolocate the site. Carried through import so a discovered site can be
	// plotted on the console map.
	Latitude  float64 `json:"latitude,omitempty"`
	Longitude float64 `json:"longitude,omitempty"`
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
	c.describeHitTypes(ctx, hits)
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
	for _, block := range listEntries(raw, "countyList") {
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
	c.describeHitTypes(ctx, hits)
	return hits, nil
}

// GeoRef is a RadioReference geography entry (a state or county) used to
// back name-based pickers so operators don't have to know numeric ctid/stid.
type GeoRef struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// CountryIDUnitedStates is RadioReference's coid for the United States
// (https://www.radioreference.com/db/browse/coid/1), the default country for
// the state picker.
const CountryIDUnitedStates = 1

// GetStateList returns the United States' states (stid + name) for a name
// dropdown. See GetStatesByCountry.
func (c *Client) GetStateList(ctx context.Context) ([]GeoRef, error) {
	return c.GetStatesByCountry(ctx, CountryIDUnitedStates)
}

// GetStatesByCountry returns the states/provinces of a country (by coid) via
// getCountryInfo, whose stateList is the only state enumeration the
// RadioReference WSDL defines — there is no getStateList operation (the
// service answers one with "Operation 'getStateList' is not defined in the
// WSDL", issue #1197). Element names are matched tolerantly; an unrecognised
// response yields an empty list.
func (c *Client) GetStatesByCountry(ctx context.Context, coid int) ([]GeoRef, error) {
	body := fmt.Sprintf(`<ns1:getCountryInfo>`+
		`<coid xsi:type="xsd:int">%d</coid>`+
		`%s`+
		`</ns1:getCountryInfo>`, coid, c.authXML())
	raw, err := c.call(ctx, body)
	if err != nil {
		return nil, err
	}
	var out []GeoRef
	for _, block := range listEntries(raw, "stateList") {
		l := firstLeaves(block, "stid", "stateId", "stateName", "stateCode")
		id := atoiDefault(firstNonEmpty(l["stid"], l["stateId"]), 0)
		if id == 0 {
			continue
		}
		out = append(out, GeoRef{ID: id, Name: firstNonEmpty(l["stateName"], l["stateCode"])})
	}
	return out, nil
}

// GetCountyList returns the counties in a state (ctid + name) via
// getStateInfo (the same call SearchByState walks), so the UI can offer a
// county dropdown after a state is chosen.
func (c *Client) GetCountyList(ctx context.Context, stid int) ([]GeoRef, error) {
	body := fmt.Sprintf(`<ns1:getStateInfo>`+
		`<stid xsi:type="xsd:int">%d</stid>`+
		`%s`+
		`</ns1:getStateInfo>`, stid, c.authXML())
	raw, err := c.call(ctx, body)
	if err != nil {
		return nil, err
	}
	var out []GeoRef
	for _, block := range listEntries(raw, "countyList") {
		l := firstLeaves(block, "ctid", "countyName", "name")
		id := atoiDefault(l["ctid"], 0)
		if id == 0 {
			continue
		}
		out = append(out, GeoRef{ID: id, Name: firstNonEmpty(l["countyName"], l["name"])})
	}
	return out, nil
}

// ErrEmptySystem is returned by GetFullSystem when the service answered every
// call but listed neither a site nor a talkgroup, so there is nothing to
// import. Before #1403 that was returned as a blank system with no error.
var ErrEmptySystem = errors.New("radioreference: the service lists no sites and no talkgroups for this system")

// GetFullSystem fetches a system's identity, sites, and talkgroups and
// returns an import-ready FullSystem with a mapped GopherTrunk protocol.
//
// The shapes read here are the WSDL's (v=latest, service version 18), not
// guesses (#1403):
//
//   - getTrsDetails returns a Trs: sName, sCity, numeric sType / sFlavor ids,
//     and a sysid array of {sysid, ct, wacn, model}. It carries no sites and
//     no talkgroups.
//   - getTrsSites returns an array of TrsSite, each with a siteFreqs array of
//     {lcn, freq, use, colorCode, ch_id}.
//   - getTrsTalkgroups takes sid, tgCid, tgTag, tgDec (zero = no filter) and
//     returns an array of Talkgroup.
//   - getTrsType / getTrsFlavor turn the numeric ids into the descriptions
//     ("Project 25", "DMR", …) protocolFromType maps.
//
// A failed sites or talkgroups call fails the import: both used to be
// swallowed, which is how a rejected request became a blank system. The two
// description lookups only choose the protocol, so their failure is reported
// in Notes and the protocol left at protocolFromType's default.
func (c *Client) GetFullSystem(ctx context.Context, sid int) (FullSystem, error) {
	det, err := c.callOp(ctx, "getTrsDetails", intParam("sid", sid))
	if err != nil {
		return FullSystem{}, err
	}
	fs := FullSystem{
		SID:  sid,
		Name: det.val("sName"),
		City: det.val("sCity"),
	}
	for _, id := range det.child("sysid").items() {
		if v, ok := parseHexOrDec(id.val("sysid")); ok && fs.SystemID == 0 {
			fs.SystemID = uint16(v)
		}
		if v, ok := parseHexOrDec(id.val("wacn")); ok && fs.WACN == 0 {
			fs.WACN = uint32(v)
		}
	}

	typeID := atoiDefault(det.val("sType"), 0)
	flavorID := atoiDefault(det.val("sFlavor"), 0)
	if fs.Type, err = c.describe(ctx, "getTrsType", "sType", "sTypeDescr", typeID); err != nil {
		fs.Notes = append(fs.Notes, fmt.Sprintf(
			"could not read the system type (%v); the protocol is a guess, check it", err))
	}
	if fs.Type != "" && flavorID != 0 {
		// The flavor only refines a known type (DMR Tier II vs Tier III),
		// so a failure here costs less than the type's.
		if fs.Flavor, err = c.describe(ctx, "getTrsFlavor", "sFlavor", "sFlavorDescr", flavorID); err != nil {
			fs.Notes = append(fs.Notes, fmt.Sprintf(
				"could not read the system flavor (%v); check the protocol", err))
		}
	}
	fs.Protocol = protocolFromType(fs.Type, fs.Flavor)

	if fs.Sites, err = c.getTrsSites(ctx, sid); err != nil {
		return FullSystem{}, err
	}
	if fs.Talkgroups, err = c.getTrsTalkgroups(ctx, sid); err != nil {
		return FullSystem{}, err
	}
	switch {
	case len(fs.Sites) == 0 && len(fs.Talkgroups) == 0:
		return FullSystem{}, fmt.Errorf("%w (sid %d)", ErrEmptySystem, sid)
	case len(fs.Sites) == 0:
		fs.Notes = append(fs.Notes, "RadioReference lists no site frequencies for this system; add its control channels by hand")
	case len(fs.Talkgroups) == 0:
		fs.Notes = append(fs.Notes, "RadioReference lists no talkgroups for this system")
	}
	return fs, nil
}

// describe resolves a numeric system type or flavor id to its description
// through getTrsType / getTrsFlavor. Both take the id and return an array of
// definitions ({sType, sTypeDescr} / {sType, sFlavor, sFlavorDescr}); the
// entry whose idField equals id is the answer. Results are cached on the
// client, so a search listing many systems of one type asks once.
func (c *Client) describe(ctx context.Context, op, idField, descrField string, id int) (string, error) {
	if id == 0 {
		return "", fmt.Errorf("radioreference: the system has no %s", idField)
	}
	key := fmt.Sprintf("%s/%d", op, id)
	c.mu.Lock()
	descr, ok := c.descr[key]
	c.mu.Unlock()
	if ok {
		return descr, nil
	}
	ret, err := c.callOp(ctx, op, intParam("id", id))
	if err != nil {
		return "", err
	}
	for _, def := range ret.items() {
		if atoiDefault(def.val(idField), 0) == id && def.val(descrField) != "" {
			descr = def.val(descrField)
			c.mu.Lock()
			if c.descr == nil {
				c.descr = make(map[string]string)
			}
			c.descr[key] = descr
			c.mu.Unlock()
			return descr, nil
		}
	}
	return "", fmt.Errorf("radioreference: %s has no entry for %s %d", op, idField, id)
}

// describeHitTypes replaces the numeric sType a county listing carries with
// its description, so a search result reads "Project 25" and not "8". It is
// cosmetic: an id that cannot be resolved is left as it was.
func (c *Client) describeHitTypes(ctx context.Context, hits []SearchHit) {
	for i := range hits {
		id, err := strconv.Atoi(strings.TrimSpace(hits[i].Type))
		if err != nil || id == 0 {
			continue
		}
		if descr, err := c.describe(ctx, "getTrsType", "sType", "sTypeDescr", id); err == nil {
			hits[i].Type = descr
		}
	}
}

func (c *Client) getTrsSites(ctx context.Context, sid int) ([]SiteDetail, error) {
	ret, err := c.callOp(ctx, "getTrsSites", intParam("sid", sid))
	if err != nil {
		return nil, err
	}
	return parseSites(ret), nil
}

func (c *Client) getTrsTalkgroups(ctx context.Context, sid int) ([]TalkgroupDetail, error) {
	// tgCid / tgTag / tgDec are filters; the WSDL documents that with none
	// specified every talkgroup of the system is returned.
	ret, err := c.callOp(ctx, "getTrsTalkgroups",
		intParam("sid", sid), intParam("tgCid", 0), intParam("tgTag", 0), intParam("tgDec", 0))
	if err != nil {
		return nil, err
	}
	return parseTalkgroups(ret), nil
}

// parseSites reads a getTrsSites <return>: an array of TrsSite. A frequency
// whose "use" flag marks it a control channel lands in ControlChannels; all
// frequencies (control + voice) land in Frequencies. A site with no usable
// frequency is dropped, so a metadata-only site doesn't surface as a
// zero-channel site.
func parseSites(ret *node) []SiteDetail {
	var sites []SiteDetail
	for _, it := range ret.items() {
		site := SiteDetail{
			RFSS:        atoiDefault(it.val("rfss"), 0),
			SiteNumber:  atoiDefault(it.val("siteNumber"), 0),
			Description: it.val("siteDescr"),
			County:      it.val("siteCt"),
			Latitude:    parseDegrees(it.val("lat")),
			Longitude:   parseDegrees(it.val("lon")),
		}
		for _, f := range it.child("siteFreqs").items() {
			hz := mhzToHz(f.val("freq"))
			if hz == 0 {
				continue
			}
			site.Frequencies = append(site.Frequencies, hz)
			if isControlUse(f.val("use")) {
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

// parseTalkgroups reads a getTrsTalkgroups <return>: an array of Talkgroup.
func parseTalkgroups(ret *node) []TalkgroupDetail {
	var tgs []TalkgroupDetail
	for _, it := range ret.items() {
		dec := atoiDefault(it.val("tgDec"), 0)
		if dec <= 0 {
			continue
		}
		tg := TalkgroupDetail{
			Dec:         uint32(dec),
			AlphaTag:    it.val("tgAlpha"),
			Description: it.val("tgDescr"),
			Mode:        normalizeMode(it.val("tgMode")),
			// enc (service version 15): 0 = no encryption, 1 = mixed,
			// 2 = fully encrypted. Only a talkgroup that is always
			// encrypted is marked; a mixed one still carries clear calls.
			Encrypted: atoiDefault(it.val("enc"), 0) == 2,
		}
		// tags is an array of {tagId, tagDescr}; the first is the service tag.
		for _, tag := range it.child("tags").items() {
			if d := tag.val("tagDescr"); d != "" {
				tg.Tag = d
				break
			}
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
	case "D", "DE", "T", "TE", "TDMA":
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
//
// A "Project 25 Phase II" system maps to "p25", NOT "p25-phase2" (#1316):
// RadioReference's Phase II flavor describes the VOICE channels. Such a
// system's control channel is still Phase 1 C4FM/FDMA, and `protocol: p25`
// already follows its TDMA voice grants (IDEN_UP_TDMA → the Phase 2 voice
// chain). `p25-phase2` selects the H-DQPSK TDMA CONTROL-CHANNEL pipeline,
// which never locks on an FDMA control channel, so every imported Phase II
// system used to come out with a decoder that could not lock.
func protocolFromType(sType, flavor string) string {
	t := strings.ToLower(sType + " " + flavor)
	switch {
	case strings.Contains(t, "project 25"), strings.Contains(t, "p25"),
		strings.Contains(t, "phase ii"), strings.Contains(t, "phase 2"):
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

// parseDegrees parses a decimal-degrees coordinate string, returning 0 for an
// empty/unparseable value or one outside the valid ±180 range (a longitude's
// bound; latitude is a tighter subset). Tolerant like the other RR parsers so a
// missing/renamed leaf degrades to "no position" rather than an error.
func parseDegrees(s string) float64 {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}
	f, err := strconv.ParseFloat(s, 64)
	if err != nil || f < -180 || f > 180 {
		return 0
	}
	return f
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
