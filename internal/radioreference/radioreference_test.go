package radioreference

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestMatchAgainst(t *testing.T) {
	cand := Candidate{
		WACN: 0xBEE99, SystemID: 0x49A, Name: "New County P25",
		ControlChannels: []uint32{851012500, 851262500},
	}
	existing := []System{
		{SID: 100, Name: "Exact Identity", WACN: 0xBEE99, SystemID: 0x49A},
		{SID: 200, Name: "Shared Freq", ControlChannels: []uint32{851262500}},
		{SID: 300, Name: "new-county-p25"},
		{SID: 400, Name: "Totally Different", WACN: 1, SystemID: 2, ControlChannels: []uint32{460000000}},
	}
	hints := MatchAgainst(cand, existing)
	if len(hints) != 3 {
		t.Fatalf("len(hints) = %d, want 3 (%+v)", len(hints), hints)
	}
	bySID := map[int]Hint{}
	for _, h := range hints {
		bySID[h.SID] = h
	}
	if h, ok := bySID[100]; !ok || h.Confidence < 0.9 {
		t.Errorf("SID 100 should match strongly on WACN+SYSID, got %+v", h)
	}
	if h, ok := bySID[200]; !ok || h.Confidence != 0.80 {
		t.Errorf("SID 200 should match on shared freq, got %+v", h)
	}
	if h, ok := bySID[300]; !ok || h.Confidence != 0.50 {
		t.Errorf("SID 300 should match on name fold, got %+v", h)
	}
	if _, ok := bySID[400]; ok {
		t.Error("SID 400 should not match")
	}
}

func TestNewClient_NoCredentials(t *testing.T) {
	if _, err := NewClient(Auth{}); err != ErrNoCredentials {
		t.Errorf("NewClient(empty) err = %v, want ErrNoCredentials", err)
	}
	if _, err := NewClient(Auth{AppKey: "k"}); err != nil {
		t.Errorf("NewClient(key) err = %v, want nil", err)
	}
}

const trsDetailsResponse = `<?xml version="1.0" encoding="UTF-8"?>
<SOAP-ENV:Envelope xmlns:SOAP-ENV="http://schemas.xmlsoap.org/soap/envelope/">
 <SOAP-ENV:Body>
  <ns1:getTrsDetailsResponse xmlns:ns1="urn:RadioReference">
   <return>
    <sid>7715</sid>
    <sName>Example Regional P25</sName>
    <sType>Project 25 Phase II</sType>
    <sysid>49A</sysid>
    <wacn>BEE99</wacn>
   </return>
  </ns1:getTrsDetailsResponse>
 </SOAP-ENV:Body>
</SOAP-ENV:Envelope>`

func TestGetTrsDetails_ParsesResponse(t *testing.T) {
	var gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		buf := make([]byte, r.ContentLength)
		_, _ = r.Body.Read(buf)
		gotBody = string(buf)
		w.Header().Set("Content-Type", "text/xml")
		_, _ = w.Write([]byte(trsDetailsResponse))
	}))
	defer srv.Close()

	c, err := NewClient(Auth{AppKey: "KEY", Username: "u", Password: "p"})
	if err != nil {
		t.Fatal(err)
	}
	c.SetEndpoint(srv.URL)

	sys, err := c.GetTrsDetails(context.Background(), 7715)
	if err != nil {
		t.Fatalf("GetTrsDetails: %v", err)
	}
	if sys.SID != 7715 || sys.Name != "Example Regional P25" {
		t.Errorf("sys = %+v, want SID 7715 Example Regional P25", sys)
	}
	if sys.SystemID != 0x49A {
		t.Errorf("SystemID = %X, want 49A", sys.SystemID)
	}
	if sys.WACN != 0xBEE99 {
		t.Errorf("WACN = %X, want BEE99", sys.WACN)
	}
	// The request must carry the auth block + the sid.
	for _, want := range []string{"getTrsDetails", "<appKey", "KEY", "<sid", "7715"} {
		if !contains(gotBody, want) {
			t.Errorf("request body missing %q:\n%s", want, gotBody)
		}
	}
}

const faultResponse = `<?xml version="1.0"?>
<SOAP-ENV:Envelope xmlns:SOAP-ENV="http://schemas.xmlsoap.org/soap/envelope/">
 <SOAP-ENV:Body><SOAP-ENV:Fault>
  <faultcode>SOAP-ENV:Server</faultcode>
  <faultstring>Invalid API key</faultstring>
 </SOAP-ENV:Fault></SOAP-ENV:Body></SOAP-ENV:Envelope>`

func TestCall_SurfacesSOAPFault(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(faultResponse))
	}))
	defer srv.Close()
	c, _ := NewClient(Auth{AppKey: "BAD"})
	c.SetEndpoint(srv.URL)
	if _, err := c.GetTrsDetails(context.Background(), 1); err == nil {
		t.Fatal("expected SOAP fault error, got nil")
	}
}

func TestGetCountyInfo_ParsesList(t *testing.T) {
	const resp = `<?xml version="1.0"?>
<SOAP-ENV:Envelope xmlns:SOAP-ENV="http://schemas.xmlsoap.org/soap/envelope/">
 <SOAP-ENV:Body><ns1:getCountyInfoResponse xmlns:ns1="urn:RadioReference"><return>
  <trsList><sid>10</sid><sName>Alpha</sName><sType>P25</sType></trsList>
  <trsList><sid>20</sid><sName>Bravo</sName><sType>DMR</sType></trsList>
 </return></ns1:getCountyInfoResponse></SOAP-ENV:Body></SOAP-ENV:Envelope>`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(resp))
	}))
	defer srv.Close()
	c, _ := NewClient(Auth{AppKey: "K"})
	c.SetEndpoint(srv.URL)
	sysList, err := c.GetCountyInfo(context.Background(), 42)
	if err != nil {
		t.Fatalf("GetCountyInfo: %v", err)
	}
	if len(sysList) != 2 || sysList[0].SID != 10 || sysList[1].Name != "Bravo" {
		t.Errorf("county systems = %+v, want SID 10 Alpha + SID 20 Bravo", sysList)
	}
}

func contains(haystack, needle string) bool {
	return len(needle) == 0 || (len(haystack) >= len(needle) && indexOf(haystack, needle) >= 0)
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
