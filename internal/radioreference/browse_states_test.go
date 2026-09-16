package radioreference

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// phpSOAPArray mimics how RadioReference's PHP SoapServer encodes a
// SOAP-encoded array in an rpc/encoded response: ONE list element wrapping
// repeated <item> children — not one list element per entry.
const countryInfoResponse = `<?xml version="1.0" encoding="UTF-8"?>
<SOAP-ENV:Envelope xmlns:SOAP-ENV="http://schemas.xmlsoap.org/soap/envelope/" xmlns:ns1="urn:RadioReference" xmlns:xsd="http://www.w3.org/2001/XMLSchema" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xmlns:SOAP-ENC="http://schemas.xmlsoap.org/soap/encoding/">
<SOAP-ENV:Body><ns1:getCountryInfoResponse><return xsi:type="ns1:CountryInfo">
 <coid xsi:type="xsd:int">1</coid>
 <countryName xsi:type="xsd:string">United States</countryName>
 <countryCode xsi:type="xsd:string">US</countryCode>
 <agencyList SOAP-ENC:arrayType="ns1:Agency[1]" xsi:type="ns1:Agencies">
  <item xsi:type="ns1:Agency"><aid xsi:type="xsd:int">7</aid><aName xsi:type="xsd:string">Federal</aName></item>
 </agencyList>
 <stateList SOAP-ENC:arrayType="ns1:State[3]" xsi:type="ns1:States">
  <item xsi:type="ns1:State"><stid xsi:type="xsd:int">1</stid><stateName xsi:type="xsd:string">Alabama</stateName><stateCode xsi:type="xsd:string">AL</stateCode></item>
  <item xsi:type="ns1:State"><stid xsi:type="xsd:int">2</stid><stateName xsi:type="xsd:string">Alaska</stateName><stateCode xsi:type="xsd:string">AK</stateCode></item>
  <item xsi:type="ns1:State"><stid xsi:type="xsd:int">3</stid><stateName xsi:type="xsd:string">Arizona</stateName><stateCode xsi:type="xsd:string">AZ</stateCode></item>
 </stateList>
</return></ns1:getCountryInfoResponse></SOAP-ENV:Body></SOAP-ENV:Envelope>`

// TestGetStateList_UsesGetCountryInfo pins the second half of issue #1197:
// the live RadioReference WSDL defines no getStateList operation (the
// reporter's GET /api/v1/config/rr/states answered "Operation 'getStateList'
// is not defined in the WSDL"). States are the stateList of getCountryInfo.
func TestGetStateList_UsesGetCountryInfo(t *testing.T) {
	var gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
		if strings.Contains(gotBody, "getStateList") {
			// What the real service does with the fabricated operation.
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`<SOAP-ENV:Envelope xmlns:SOAP-ENV="http://schemas.xmlsoap.org/soap/envelope/"><SOAP-ENV:Body><SOAP-ENV:Fault><faultstring>Operation 'getStateList' is not defined in the WSDL for this service</faultstring></SOAP-ENV:Fault></SOAP-ENV:Body></SOAP-ENV:Envelope>`))
			return
		}
		w.Header().Set("Content-Type", "text/xml")
		_, _ = w.Write([]byte(countryInfoResponse))
	}))
	defer srv.Close()
	c, _ := NewClient(Auth{AppKey: "K", Username: "u", Password: "p"})
	c.SetEndpoint(srv.URL)

	states, err := c.GetStateList(context.Background())
	if err != nil {
		t.Fatalf("GetStateList: %v (request body: %s)", err, gotBody)
	}
	if !strings.Contains(gotBody, "<ns1:getCountryInfo>") {
		t.Errorf("request must call getCountryInfo, got body: %s", gotBody)
	}
	if !strings.Contains(gotBody, `<coid xsi:type="xsd:int">1</coid>`) {
		t.Errorf("request must ask for the United States (coid 1), got body: %s", gotBody)
	}
	want := []GeoRef{{ID: 1, Name: "Alabama"}, {ID: 2, Name: "Alaska"}, {ID: 3, Name: "Arizona"}}
	if len(states) != len(want) {
		t.Fatalf("states = %+v, want %+v", states, want)
	}
	for i := range want {
		if states[i] != want[i] {
			t.Errorf("states[%d] = %+v, want %+v", i, states[i], want[i])
		}
	}
}

// TestGetCountyInfo_SOAPEncodedArray: the same PHP array shape on
// getCountyInfo's trsList — every system must be returned, not only the
// first <item> of the single wrapper element.
func TestGetCountyInfo_SOAPEncodedArray(t *testing.T) {
	const resp = `<?xml version="1.0"?>
<SOAP-ENV:Envelope xmlns:SOAP-ENV="http://schemas.xmlsoap.org/soap/envelope/" xmlns:SOAP-ENC="http://schemas.xmlsoap.org/soap/encoding/">
 <SOAP-ENV:Body><ns1:getCountyInfoResponse xmlns:ns1="urn:RadioReference"><return>
  <ctid>42</ctid>
  <trsList SOAP-ENC:arrayType="ns1:Trs[2]">
   <item><sid>10</sid><sName>Alpha</sName><sType>P25</sType></item>
   <item><sid>20</sid><sName>Bravo</sName><sType>DMR</sType></item>
  </trsList>
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

func TestGetCountyList_SOAPEncodedArray(t *testing.T) {
	const resp = `<?xml version="1.0"?>
<SOAP-ENV:Envelope xmlns:SOAP-ENV="http://schemas.xmlsoap.org/soap/envelope/" xmlns:SOAP-ENC="http://schemas.xmlsoap.org/soap/encoding/">
 <SOAP-ENV:Body><ns1:getStateInfoResponse xmlns:ns1="urn:RadioReference"><return>
  <stid>5</stid>
  <countyList SOAP-ENC:arrayType="ns1:CountyHeader[2]">
   <item><ctid>100</ctid><countyName>Maricopa</countyName></item>
   <item><ctid>101</ctid><countyName>Pima</countyName></item>
  </countyList>
 </return></ns1:getStateInfoResponse></SOAP-ENV:Body></SOAP-ENV:Envelope>`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(resp))
	}))
	defer srv.Close()
	c, _ := NewClient(Auth{AppKey: "K"})
	c.SetEndpoint(srv.URL)
	counties, err := c.GetCountyList(context.Background(), 5)
	if err != nil {
		t.Fatalf("GetCountyList: %v", err)
	}
	if len(counties) != 2 || counties[0].ID != 100 || counties[1].Name != "Pima" {
		t.Errorf("counties = %+v, want 100 Maricopa + 101 Pima", counties)
	}
}
