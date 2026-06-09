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
