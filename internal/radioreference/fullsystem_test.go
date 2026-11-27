package radioreference

import (
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"testing"
)

// ---- a fake service built from the WSDL tables --------------------------

// soapValue renders v as WSDL type typ in an element called name, the way
// the service's PHP SoapServer writes an rpc/encoded response: a struct as
// its fields in WSDL order, an array as one element holding an <item> per
// entry. A struct value is a map of field name to value; a key the WSDL type
// does not define fails the test, so a fixture cannot invent an element.
func soapValue(t *testing.T, name, typ string, v any) string {
	t.Helper()
	if strings.HasPrefix(typ, "xsd:") {
		if v == nil {
			return fmt.Sprintf(`<%s xsi:nil="true"/>`, name)
		}
		return fmt.Sprintf(`<%s xsi:type="%s">%s</%s>`, name, typ, xmlEscape(v.(string)), name)
	}
	tn := strings.TrimPrefix(typ, "tns:")
	if entry, ok := wsdlArrays[tn]; ok {
		var items []map[string]any
		if v != nil {
			items = v.([]map[string]any)
		}
		var b strings.Builder
		fmt.Fprintf(&b, `<%s SOAP-ENC:arrayType="ns1:%s[%d]" xsi:type="ns1:%s">`, name, entry, len(items), tn)
		for _, it := range items {
			b.WriteString(soapValue(t, "item", "tns:"+entry, it))
		}
		fmt.Fprintf(&b, `</%s>`, name)
		return b.String()
	}
	fields, ok := wsdlStructs[tn]
	if !ok {
		t.Fatalf("type %q is not in the WSDL tables", typ)
	}
	if v == nil {
		return fmt.Sprintf(`<%s xsi:nil="true"/>`, name)
	}
	vals := v.(map[string]any)
	known := make(map[string]bool, len(fields))
	var b strings.Builder
	fmt.Fprintf(&b, `<%s xsi:type="ns1:%s">`, name, tn)
	for _, f := range fields {
		known[f.name] = true
		b.WriteString(soapValue(t, f.name, f.typ, vals[f.name]))
	}
	for k := range vals {
		if !known[k] {
			t.Fatalf("fixture sets %q, which WSDL type %s does not define", k, tn)
		}
	}
	fmt.Fprintf(&b, `</%s>`, name)
	return b.String()
}

func rrEnvelope(body string) string {
	return `<?xml version="1.0" encoding="UTF-8"?>` +
		`<SOAP-ENV:Envelope xmlns:SOAP-ENV="http://schemas.xmlsoap.org/soap/envelope/" ` +
		`xmlns:ns1="http://api.radioreference.com/soap2" xmlns:xsd="http://www.w3.org/2001/XMLSchema" ` +
		`xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" ` +
		`xmlns:SOAP-ENC="http://schemas.xmlsoap.org/soap/encoding/">` +
		`<SOAP-ENV:Body>` + body + `</SOAP-ENV:Body></SOAP-ENV:Envelope>`
}

// fakeRR is a RadioReference stand-in. It holds one system and answers the
// five operations the import calls, checking every request against wsdlOps.
type fakeRR struct {
	t   *testing.T
	srv *httptest.Server

	details    map[string]any   // the Trs
	sites      []map[string]any // TrsSite entries
	talkgroups []map[string]any // Talkgroup entries
	types      map[int]string   // sType → sTypeDescr
	flavors    map[int]string   // sFlavor → sFlavorDescr
	faults     map[string]string

	mu    sync.Mutex
	calls []string
}

func newFakeRR(t *testing.T) *fakeRR {
	f := &fakeRR{t: t, faults: map[string]string{}}
	f.srv = httptest.NewServer(http.HandlerFunc(f.serve))
	t.Cleanup(f.srv.Close)
	return f
}

func (f *fakeRR) client() *Client {
	c, err := NewClient(Auth{AppKey: "KEY", Username: "user", Password: "pass"})
	if err != nil {
		f.t.Fatal(err)
	}
	c.SetEndpoint(f.srv.URL)
	return c
}

func (f *fakeRR) called() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]string(nil), f.calls...)
}

// parsedRequest is the operation element of a request and its parameters in
// the order sent. It is read with encoding/xml directly, not with the
// client's own XML helpers.
type parsedRequest struct {
	op     string
	params []wsdlField       // name + xsi:type as sent
	values map[string]string // text of each scalar parameter
}

func parseRequest(body []byte) (parsedRequest, error) {
	var req parsedRequest
	req.values = map[string]string{}
	dec := xml.NewDecoder(strings.NewReader(string(body)))
	depth, opDepth := 0, -1
	inBody := false
	var cur string
	for {
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return req, err
		}
		switch e := tok.(type) {
		case xml.StartElement:
			depth++
			switch {
			case e.Name.Local == "Body" && opDepth < 0:
				inBody = true
			case inBody && opDepth < 0:
				req.op, opDepth = e.Name.Local, depth
			case opDepth > 0 && depth == opDepth+1:
				typ := ""
				for _, a := range e.Attr {
					if a.Name.Local == "type" {
						typ = a.Value
					}
				}
				req.params = append(req.params, wsdlField{e.Name.Local, typ})
				cur = e.Name.Local
			}
		case xml.CharData:
			if opDepth > 0 && depth == opDepth+1 && cur != "" {
				req.values[cur] += strings.TrimSpace(string(e))
			}
		case xml.EndElement:
			if opDepth > 0 && depth == opDepth+1 {
				cur = ""
			}
			depth--
		}
	}
	if req.op == "" {
		return req, errors.New("no operation element in the SOAP body")
	}
	return req, nil
}

func (f *fakeRR) fault(w http.ResponseWriter, msg string) {
	w.Header().Set("Content-Type", "text/xml")
	w.WriteHeader(http.StatusInternalServerError)
	_, _ = io.WriteString(w, rrEnvelope(`<SOAP-ENV:Fault><faultcode>SOAP-ENV:Server</faultcode><faultstring>`+
		xmlEscape(msg)+`</faultstring></SOAP-ENV:Fault>`))
}

func (f *fakeRR) serve(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)
	req, err := parseRequest(body)
	if err != nil {
		f.fault(w, "Bad Request: "+err.Error())
		return
	}
	f.mu.Lock()
	f.calls = append(f.calls, req.op)
	f.mu.Unlock()

	op, ok := wsdlOps[req.op]
	if !ok {
		// What the real service answers for an operation it does not have
		// (issue #1197).
		f.fault(w, fmt.Sprintf("Operation '%s' is not defined in the WSDL for this service", req.op))
		return
	}
	// Parameters must be the WSDL's message parts: same names, same order,
	// and the declared type on every scalar.
	want := make([]wsdlField, len(op.params))
	copy(want, op.params)
	for i := range want {
		want[i].typ = strings.Replace(want[i].typ, "tns:", "ns1:", 1)
	}
	if !reflect.DeepEqual(req.params, want) {
		f.t.Errorf("%s request parameters = %v, the WSDL declares %v", req.op, req.params, want)
		f.fault(w, "SOAP-ERROR: Encoding: Violation of encoding rules")
		return
	}
	if msg := f.faults[req.op]; msg != "" {
		f.fault(w, msg)
		return
	}

	var ret any
	switch req.op {
	case "getTrsDetails":
		ret = f.details
	case "getTrsSites":
		ret = f.sites
	case "getTrsTalkgroups":
		for _, filter := range []string{"tgCid", "tgTag", "tgDec"} {
			if req.values[filter] != "0" {
				f.t.Errorf("getTrsTalkgroups %s = %q; 0 (no filter) returns every talkgroup", filter, req.values[filter])
			}
		}
		ret = f.talkgroups
	case "getTrsType":
		id, _ := strconv.Atoi(req.values["id"])
		var defs []map[string]any
		if d, ok := f.types[id]; ok {
			defs = append(defs, map[string]any{"sType": strconv.Itoa(id), "sTypeDescr": d})
		}
		ret = defs
	case "getTrsFlavor":
		id, _ := strconv.Atoi(req.values["id"])
		var defs []map[string]any
		if d, ok := f.flavors[id]; ok {
			defs = append(defs, map[string]any{"sType": "8", "sFlavor": strconv.Itoa(id), "sFlavorDescr": d})
		}
		ret = defs
	}
	w.Header().Set("Content-Type", "text/xml; charset=utf-8")
	_, _ = io.WriteString(w, rrEnvelope(`<ns1:`+req.op+`Response>`+
		soapValue(f.t, "return", op.returns, ret)+`</ns1:`+req.op+`Response>`))
}

// exampleSystem fills f with a made-up P25 system: two sites with
// frequencies, one site with none, and four talkgroups.
func exampleSystem(f *fakeRR) {
	rect := []map[string]any{{"nw_lat": "39.1", "nw_lon": "-78.2", "se_lat": "37.9", "se_lon": "-76.4"}}
	f.types = map[int]string{8: "Project 25", 12: "DMR"}
	f.flavors = map[int]string{42: "Phase II", 43: "Tier III"}
	f.details = map[string]any{
		"sName": "Example County Public Safety", "sType": "8", "sFlavor": "42", "sVoice": "3",
		"sCity": "Exampleville", "sCounty": []map[string]any{{"ctid": "2881"}}, "sState": []map[string]any{{"stid": "51"}},
		"sCountry": "1", "lat": "38.5", "lon": "-77.3", "range": "25", "rectangles": rect,
		"lastUpdated": "2026-11-01T12:00:00+00:00",
		"sysid":       []map[string]any{{"sysid": "49A", "ct": "", "wacn": "BEE99", "model": ""}},
		"bandplan":    []map[string]any{{"base": "851.00625", "spacing": "6.25", "offset": "-45"}},
	}
	f.sites = []map[string]any{
		{
			"siteId": "30001", "sid": "7715", "siteNumber": "1", "siteDescr": "North Tower", "rfss": "1",
			"nac": "293", "siteCtid": "2881", "siteCt": "Example", "siteModulation": "CQPSK",
			"lat": "38.250000", "lon": "-77.100000", "range": "15", "rectangles": rect,
			"splinter": "0", "rebanded": "1", "tdma_cc": "0",
			"siteLicenses": []map[string]any{{"license": "WQZZ123"}},
			"siteFreqs": []map[string]any{
				{"lcn": "1", "freq": "851.01250", "use": "d"},
				{"lcn": "2", "freq": "851.26250", "use": ""},
				{"lcn": "3", "freq": "851.51250", "use": "a"},
			},
		},
		{
			// Not geolocated: lat / lon are nil, while its rectangle still
			// carries coordinates under other names.
			"siteId": "30002", "sid": "7715", "siteNumber": "2", "siteDescr": "South Tower", "rfss": "1",
			"nac": "293", "siteCt": "Example", "rectangles": rect,
			"siteFreqs": []map[string]any{
				{"lcn": "1", "freq": "852.01250", "use": "d"},
				{"lcn": "2", "freq": "852.26250"},
			},
		},
		{"siteId": "30003", "sid": "7715", "siteNumber": "3", "siteDescr": "Planned site", "rfss": "1"},
	}
	f.talkgroups = []map[string]any{
		{"tgId": "1", "tgDec": "101", "tgDescr": "Police Dispatch", "tgAlpha": "PD Disp", "tgMode": "D", "enc": "0",
			"tags": []map[string]any{{"tagId": "3", "tagDescr": "Law Dispatch"}}, "tgCid": "9", "tgSort": "1"},
		{"tgId": "2", "tgDec": "202", "tgDescr": "Fire Tactical", "tgAlpha": "FD Tac", "tgMode": "T", "enc": "2",
			"tags": []map[string]any{{"tagId": "8", "tagDescr": "Fire-Tac"}}, "tgCid": "9", "tgSort": "2"},
		{"tgId": "3", "tgDec": "303", "tgDescr": "Public Works", "tgAlpha": "PW", "tgMode": "A", "enc": "1", "tgCid": "9"},
		{"tgId": "4", "tgDec": "0", "tgDescr": "placeholder", "tgAlpha": "ignored"},
	}
}

// ---- the regression tests -----------------------------------------------

// Issue #1403: importing a system through the RadioReference login created a
// blank system with no control channels and no talkgroups, and no error.
// The parser looked for <siteList>, <siteFreq> and <tgList>, which the
// service's WSDL does not define, and the two calls' errors were dropped.
// Against a service that answers as the WSDL says, the import must return
// the sites, their control channels and the talkgroups.
func TestGetFullSystemReadsWSDLShapedResponses(t *testing.T) {
	f := newFakeRR(t)
	exampleSystem(f)

	fs, err := f.client().GetFullSystem(context.Background(), 7715)
	if err != nil {
		t.Fatalf("GetFullSystem: %v", err)
	}
	if fs.SID != 7715 || fs.Name != "Example County Public Safety" || fs.City != "Exampleville" {
		t.Errorf("identity = sid %d %q city %q", fs.SID, fs.Name, fs.City)
	}
	if fs.SystemID != 0x49A || fs.WACN != 0xBEE99 {
		t.Errorf("SystemID %X WACN %X, want 49A BEE99 (the sysid array's first entry)", fs.SystemID, fs.WACN)
	}

	wantSites := []SiteDetail{
		{RFSS: 1, SiteNumber: 1, Description: "North Tower", County: "Example",
			ControlChannels: []uint32{851012500, 851512500},
			Frequencies:     []uint32{851012500, 851262500, 851512500},
			Latitude:        38.25, Longitude: -77.1},
		{RFSS: 1, SiteNumber: 2, Description: "South Tower", County: "Example",
			ControlChannels: []uint32{852012500},
			Frequencies:     []uint32{852012500, 852262500}},
	}
	if !reflect.DeepEqual(fs.Sites, wantSites) {
		t.Errorf("sites =\n %+v\nwant\n %+v", fs.Sites, wantSites)
	}

	wantTGs := []TalkgroupDetail{
		{Dec: 101, AlphaTag: "PD Disp", Description: "Police Dispatch", Tag: "Law Dispatch", Mode: "D"},
		{Dec: 202, AlphaTag: "FD Tac", Description: "Fire Tactical", Tag: "Fire-Tac", Mode: "D", Encrypted: true},
		{Dec: 303, AlphaTag: "PW", Description: "Public Works", Mode: "A"},
	}
	if !reflect.DeepEqual(fs.Talkgroups, wantTGs) {
		t.Errorf("talkgroups =\n %+v\nwant\n %+v", fs.Talkgroups, wantTGs)
	}
	if len(fs.Notes) != 0 {
		t.Errorf("notes = %q, want none for a complete system", fs.Notes)
	}
}

// sType and sFlavor are numeric ids in a Trs. Matched as text they never
// named a protocol, so every system imported as "p25". They are resolved
// through getTrsType / getTrsFlavor before the mapping.
func TestGetFullSystemMapsProtocolFromTypeIDs(t *testing.T) {
	cases := []struct {
		sType, sFlavor string
		wantType       string
		wantProtocol   string
	}{
		{"8", "42", "Project 25", "p25"},
		{"12", "43", "DMR", "dmr"},
	}
	for _, tc := range cases {
		f := newFakeRR(t)
		exampleSystem(f)
		f.details["sType"], f.details["sFlavor"] = tc.sType, tc.sFlavor
		fs, err := f.client().GetFullSystem(context.Background(), 7715)
		if err != nil {
			t.Fatalf("GetFullSystem: %v", err)
		}
		if fs.Type != tc.wantType || fs.Protocol != tc.wantProtocol {
			t.Errorf("sType %s sFlavor %s: type %q flavor %q protocol %q, want type %q protocol %q",
				tc.sType, tc.sFlavor, fs.Type, fs.Flavor, fs.Protocol, tc.wantType, tc.wantProtocol)
		}
	}
}

// A type the lookup cannot name must not fail the import, and must not pass
// silently either: the system still imports, with a note that the protocol
// is a guess.
func TestGetFullSystemNotesAnUnreadableType(t *testing.T) {
	f := newFakeRR(t)
	exampleSystem(f)
	f.details["sType"] = "99" // not in the fake's type list

	fs, err := f.client().GetFullSystem(context.Background(), 7715)
	if err != nil {
		t.Fatalf("GetFullSystem: %v", err)
	}
	if len(fs.Sites) != 2 || len(fs.Talkgroups) != 3 {
		t.Errorf("got %d sites / %d talkgroups, want the whole system", len(fs.Sites), len(fs.Talkgroups))
	}
	if len(fs.Notes) != 1 || !strings.Contains(fs.Notes[0], "system type") {
		t.Errorf("notes = %q, want one about the system type", fs.Notes)
	}
}

// The sites and talkgroups calls used to drop their errors, so a rejected
// request became an empty list. A failure of either must fail the import
// with the service's own message.
func TestGetFullSystemSurfacesFailedCalls(t *testing.T) {
	for _, op := range []string{"getTrsDetails", "getTrsSites", "getTrsTalkgroups"} {
		f := newFakeRR(t)
		exampleSystem(f)
		f.faults[op] = "AUTH: Invalid username or password"
		_, err := f.client().GetFullSystem(context.Background(), 7715)
		if err == nil {
			t.Errorf("%s faulted and GetFullSystem returned no error", op)
			continue
		}
		for _, want := range []string{"Invalid username or password", op} {
			if !strings.Contains(err.Error(), want) {
				t.Errorf("%s fault: error %q does not mention %q", op, err, want)
			}
		}
	}
}

// A system with neither sites nor talkgroups is the blank import of #1403.
// It is an error now; a system missing only one of the two imports with a note.
func TestGetFullSystemEmptyAndPartial(t *testing.T) {
	f := newFakeRR(t)
	exampleSystem(f)
	f.sites, f.talkgroups = nil, nil
	if _, err := f.client().GetFullSystem(context.Background(), 7715); !errors.Is(err, ErrEmptySystem) {
		t.Errorf("no sites and no talkgroups: err = %v, want ErrEmptySystem", err)
	}

	f = newFakeRR(t)
	exampleSystem(f)
	f.talkgroups = nil
	fs, err := f.client().GetFullSystem(context.Background(), 7715)
	if err != nil {
		t.Fatalf("sites but no talkgroups: %v", err)
	}
	if len(fs.Sites) != 2 || len(fs.Notes) != 1 || !strings.Contains(fs.Notes[0], "no talkgroups") {
		t.Errorf("sites %d notes %q, want 2 sites and a note about talkgroups", len(fs.Sites), fs.Notes)
	}

	f = newFakeRR(t)
	exampleSystem(f)
	f.sites = nil
	fs, err = f.client().GetFullSystem(context.Background(), 7715)
	if err != nil {
		t.Fatalf("talkgroups but no sites: %v", err)
	}
	if len(fs.Talkgroups) != 3 || len(fs.Notes) != 1 || !strings.Contains(fs.Notes[0], "no site frequencies") {
		t.Errorf("talkgroups %d notes %q, want 3 talkgroups and a note about sites", len(fs.Talkgroups), fs.Notes)
	}
}

// A county listing carries each system's sType as a number, so a search
// result read "8". It is resolved once per type; an id the service cannot
// name is left as it was.
func TestSearchHitsNameTheSystemType(t *testing.T) {
	f := newFakeRR(t)
	exampleSystem(f)
	c := f.client()
	hits := []SearchHit{
		{SID: 1, Name: "A", Type: "8"},
		{SID: 2, Name: "B", Type: "8"},
		{SID: 3, Name: "C", Type: "12"},
		{SID: 4, Name: "D", Type: "99"},
		{SID: 5, Name: "E", Type: ""},
	}
	c.describeHitTypes(context.Background(), hits)
	got := []string{hits[0].Type, hits[1].Type, hits[2].Type, hits[3].Type, hits[4].Type}
	if want := []string{"Project 25", "Project 25", "DMR", "99", ""}; !reflect.DeepEqual(got, want) {
		t.Errorf("types = %q, want %q", got, want)
	}
	// 8 twice, 12 and 99 once each: three lookups, the repeat from the cache.
	if calls := f.called(); len(calls) != 3 {
		t.Errorf("made %d getTrsType calls (%v), want 3", len(calls), calls)
	}
}

// GOPHERTRUNK_RR_DUMP saves every reply for a bug report, and nothing from
// the request: no password and no application key may reach the files.
func TestDumpSavesRepliesOnly(t *testing.T) {
	f := newFakeRR(t)
	exampleSystem(f)
	dir := filepath.Join(t.TempDir(), "rr-dump")
	t.Setenv(DumpEnv, dir)

	if _, err := f.client().GetFullSystem(context.Background(), 7715); err != nil {
		t.Fatal(err)
	}
	files, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	seen := map[string]bool{}
	for _, e := range files {
		data, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			t.Fatal(err)
		}
		for _, secret := range []string{"KEY", "pass", "appKey", "password"} {
			if strings.Contains(string(data), secret) {
				t.Errorf("%s contains %q: a dump must hold the reply only", e.Name(), secret)
			}
		}
		if m := regexp.MustCompile(`-([A-Za-z]+)\.xml$`).FindStringSubmatch(e.Name()); m != nil {
			seen[m[1]] = true
		}
	}
	for _, op := range []string{"getTrsDetails", "getTrsType", "getTrsFlavor", "getTrsSites", "getTrsTalkgroups"} {
		if !seen[op] {
			t.Errorf("no dump file for %s (have %v)", op, seen)
		}
	}
}

// TestWSDLContract checks the tables in wsdl_contract_test.go against the
// service's WSDL. It is skipped unless GT_RR_WSDL names a saved copy or a
// URL, e.g.
//
//	GT_RR_WSDL='http://api.radioreference.com/soap2/?wsdl&v=latest' \
//	  go test ./internal/radioreference -run TestWSDLContract -v
//
// The WSDL is public and needs no account. A failure means the service
// changed and the import's parsers need another look.
func TestWSDLContract(t *testing.T) {
	src := os.Getenv("GT_RR_WSDL")
	if src == "" {
		t.Skip("set GT_RR_WSDL to a saved WSDL or its URL to check the contract tables")
	}
	var data []byte
	var err error
	if strings.HasPrefix(src, "http://") || strings.HasPrefix(src, "https://") {
		resp, gerr := http.Get(src)
		if gerr != nil {
			t.Fatal(gerr)
		}
		defer resp.Body.Close()
		data, err = io.ReadAll(resp.Body)
	} else {
		data, err = os.ReadFile(src)
	}
	if err != nil {
		t.Fatal(err)
	}
	wsdl := string(data)

	partsOf := func(message string) []wsdlField {
		m := regexp.MustCompile(`(?s)<message name="` + message + `">(.*?)</message>`).FindStringSubmatch(wsdl)
		if m == nil {
			t.Errorf("the WSDL has no message %q", message)
			return nil
		}
		var out []wsdlField
		for _, p := range regexp.MustCompile(`<part name="([^"]+)" type="([^"]+)"`).FindAllStringSubmatch(m[1], -1) {
			out = append(out, wsdlField{p[1], p[2]})
		}
		return out
	}
	for name, op := range wsdlOps {
		if got := partsOf(name); !reflect.DeepEqual(got, op.params) {
			t.Errorf("%s parameters: WSDL %v, table %v", name, got, op.params)
		}
		if got := partsOf(name + "Response"); len(got) != 1 || got[0] != (wsdlField{"return", op.returns}) {
			t.Errorf("%s response: WSDL %v, table return %s", name, got, op.returns)
		}
	}
	typeBody := func(name string) string {
		m := regexp.MustCompile(`(?s)<xsd:complexType name="` + name + `">(.*?)</xsd:complexType>`).FindStringSubmatch(wsdl)
		if m == nil {
			t.Errorf("the WSDL has no type %q", name)
			return ""
		}
		return m[1]
	}
	for name, want := range wsdlStructs {
		var got []wsdlField
		for _, e := range regexp.MustCompile(`<xsd:element name="([^"]+)" type="([^"]+)"`).FindAllStringSubmatch(typeBody(name), -1) {
			got = append(got, wsdlField{e[1], e[2]})
		}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("type %s: WSDL %v, table %v", name, got, want)
		}
	}
	for name, entry := range wsdlArrays {
		m := regexp.MustCompile(`wsdl:arrayType="tns:([^"\[]+)\[\]"`).FindStringSubmatch(typeBody(name))
		if m == nil || m[1] != entry {
			t.Errorf("array %s: WSDL entry type %v, table %s", name, m, entry)
		}
	}
}
