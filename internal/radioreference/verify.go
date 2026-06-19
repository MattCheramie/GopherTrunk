package radioreference

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// Account is the subset of a RadioReference user account returned by the
// getUserData call. It is used to confirm that the supplied username/password
// are valid and that the membership carries an active (premium) subscription —
// the app key alone cannot prove premium access.
type Account struct {
	Username string `json:"username,omitempty"`
	// Expires is the subscription expiry as RR reports it (left as the raw
	// string so the UI can display whatever format the service returns).
	Expires string `json:"expires,omitempty"`
	Premium bool   `json:"premium"`
}

// ErrNoLogin is returned by VerifyCredentials when the username or password is
// empty: with a built-in app key present, a "verify" can otherwise reach RR and
// come back with an opaque authorization fault instead of a clear local error.
var ErrNoLogin = errors.New("radioreference: username and password are required to verify a subscription")

// VerifyCredentials confirms the configured username/password against the RR
// account service (getUserData) and reports whether the account has an active
// (premium) subscription. A SOAP fault (bad login / expired membership)
// surfaces as an error so callers can show the message verbatim.
func (c *Client) VerifyCredentials(ctx context.Context) (Account, error) {
	if strings.TrimSpace(c.auth.Username) == "" || strings.TrimSpace(c.auth.Password) == "" {
		return Account{}, ErrNoLogin
	}
	body := fmt.Sprintf(`<ns1:getUserData>%s</ns1:getUserData>`, c.authXML())
	raw, err := c.call(ctx, body)
	if err != nil {
		return Account{}, err
	}
	leaves := firstLeaves(raw, "username", "userName", "uid", "subExpireDate", "expiration", "expires")
	acct := Account{
		Username: firstNonEmpty(leaves["username"], leaves["userName"]),
		Expires:  firstNonEmpty(leaves["subExpireDate"], leaves["expiration"], leaves["expires"]),
	}
	acct.Premium = subscriptionActive(acct.Expires)
	// A response with no identifying field means the call did not authenticate
	// (RR sometimes returns an empty userInfo rather than a SOAP fault).
	if acct.Username == "" && acct.Expires == "" && leaves["uid"] == "" {
		return Account{}, errors.New("radioreference: could not verify account — check the username and password")
	}
	return acct, nil
}

// subscriptionActive reports whether an RR expiry timestamp is in the future.
// RR returns dates like "2025-12-31 00:00:00" or "2025-12-31" (and some
// accounts a UNIX-seconds value); unparseable or empty values are treated as
// "not premium". Feed Provider and Admin accounts keep premium for as long as
// their feed is active, so RR reports their expiry as the sentinel strings
// "Never - Feed Provider" / "Never - Admin" rather than a date — those count
// as active.
func subscriptionActive(expires string) bool {
	expires = strings.TrimSpace(expires)
	if expires == "" {
		return false
	}
	if strings.HasPrefix(strings.ToLower(expires), "never") {
		return true
	}
	for _, layout := range []string{"2006-01-02 15:04:05", "2006-01-02T15:04:05", "2006-01-02", time.RFC3339} {
		if t, err := time.Parse(layout, expires); err == nil {
			return t.After(time.Now())
		}
	}
	if n, err := strconv.ParseInt(expires, 10, 64); err == nil {
		return time.Unix(n, 0).After(time.Now())
	}
	return false
}
