package radioreference

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

// soapEnvelope wraps a getUserData body in a minimal SOAP envelope.
func soapEnvelope(body string) string {
	return `<?xml version="1.0" encoding="UTF-8"?>` +
		`<SOAP-ENV:Envelope xmlns:SOAP-ENV="http://schemas.xmlsoap.org/soap/envelope/">` +
		`<SOAP-ENV:Body><ns1:getUserDataResponse xmlns:ns1="urn:RadioReference"><return>` +
		body +
		`</return></ns1:getUserDataResponse></SOAP-ENV:Body></SOAP-ENV:Envelope>`
}

func verifyClient(t *testing.T, response string) *Client {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/xml")
		_, _ = w.Write([]byte(response))
	}))
	t.Cleanup(srv.Close)
	c, err := NewClient(Auth{AppKey: "KEY", Username: "alice", Password: "pw"})
	if err != nil {
		t.Fatal(err)
	}
	c.SetEndpoint(srv.URL)
	return c
}

func TestVerifyCredentials_Premium(t *testing.T) {
	c := verifyClient(t, soapEnvelope(`<username>alice</username><subExpireDate>2999-01-01 00:00:00</subExpireDate>`))
	acct, err := c.VerifyCredentials(context.Background())
	if err != nil {
		t.Fatalf("VerifyCredentials: %v", err)
	}
	if !acct.Premium {
		t.Errorf("expected Premium=true for a future expiry, got %+v", acct)
	}
	if acct.Username != "alice" {
		t.Errorf("Username = %q, want alice", acct.Username)
	}
}

func TestVerifyCredentials_Expired(t *testing.T) {
	c := verifyClient(t, soapEnvelope(`<username>bob</username><subExpireDate>2000-01-01 00:00:00</subExpireDate>`))
	acct, err := c.VerifyCredentials(context.Background())
	if err != nil {
		t.Fatalf("VerifyCredentials: %v", err)
	}
	if acct.Premium {
		t.Errorf("expected Premium=false for a past expiry, got %+v", acct)
	}
	if acct.Username != "bob" {
		t.Errorf("Username = %q, want bob", acct.Username)
	}
}

func TestVerifyCredentials_FeedProvider(t *testing.T) {
	// Feed Provider / Admin accounts have no expiry date — RR reports the
	// subscription as the sentinel "Never - Feed Provider" / "Never - Admin".
	for _, sentinel := range []string{"Never - Feed Provider", "Never - Admin"} {
		t.Run(sentinel, func(t *testing.T) {
			c := verifyClient(t, soapEnvelope(`<username>carol</username><subExpireDate>`+sentinel+`</subExpireDate>`))
			acct, err := c.VerifyCredentials(context.Background())
			if err != nil {
				t.Fatalf("VerifyCredentials: %v", err)
			}
			if !acct.Premium {
				t.Errorf("expected Premium=true for %q, got %+v", sentinel, acct)
			}
			if acct.Username != "carol" {
				t.Errorf("Username = %q, want carol", acct.Username)
			}
		})
	}
}

func TestVerifyCredentials_Fault(t *testing.T) {
	fault := `<?xml version="1.0"?><SOAP-ENV:Envelope xmlns:SOAP-ENV="http://schemas.xmlsoap.org/soap/envelope/">` +
		`<SOAP-ENV:Body><SOAP-ENV:Fault><faultstring>Invalid username or password</faultstring></SOAP-ENV:Fault>` +
		`</SOAP-ENV:Body></SOAP-ENV:Envelope>`
	c := verifyClient(t, fault)
	if _, err := c.VerifyCredentials(context.Background()); err == nil {
		t.Fatal("expected an error on a SOAP fault")
	}
}

func TestVerifyCredentials_RequiresLogin(t *testing.T) {
	c, err := NewClient(Auth{AppKey: "KEY"}) // key only, no user/pass
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.VerifyCredentials(context.Background()); err != ErrNoLogin {
		t.Errorf("err = %v, want ErrNoLogin", err)
	}
}

// TestVerifyCredentials_USDateFormat pins issue #1197: RadioReference reports
// subExpireDate as a US-style MM-DD-YYYY string ("03-02-2027" on the
// reporter's Premium account), which the ISO-only layouts rejected, so an
// active subscription verified as premium=false while still echoing a
// future expiry.
func TestVerifyCredentials_USDateFormat(t *testing.T) {
	// Literal response body from the issue (only username + subExpireDate,
	// no premium boolean).
	c := verifyClient(t, soapEnvelope(`<username xsi:type="xsd:string">Mkubit</username>`+
		`<subExpireDate xsi:type="xsd:string">03-02-2027</subExpireDate>`))
	acct, err := c.VerifyCredentials(context.Background())
	if err != nil {
		t.Fatalf("VerifyCredentials: %v", err)
	}
	if !acct.Premium {
		t.Errorf("expected Premium=true for a future MM-DD-YYYY expiry, got %+v", acct)
	}
	if acct.Expires != "03-02-2027" {
		t.Errorf("Expires should echo the raw RR string, got %q", acct.Expires)
	}
}

func TestSubscriptionActive_Layouts(t *testing.T) {
	cases := []struct {
		in   string
		want bool
	}{
		{"03-02-2027", true},          // RR's live MM-DD-YYYY (issue #1197)
		{"03-02-2020", false},         // past, same layout
		{"25-03-2999", true},          // day > 12: only DD-MM-YYYY parses — accept it
		{"25-03-2000", false},         // past, DD-MM-YYYY fallback
		{"2999-01-01 00:00:00", true}, // legacy ISO layouts still accepted
		{"2000-01-01", false},
		{"Never - Feed Provider", true},
		{"", false},
		{"not a date", false},
	}
	for _, tc := range cases {
		if got := subscriptionActive(tc.in); got != tc.want {
			t.Errorf("subscriptionActive(%q) = %v, want %v", tc.in, got, tc.want)
		}
	}
}
