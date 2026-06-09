package radioreference

import "testing"

func TestResolveAuth_KeyPrecedence(t *testing.T) {
	// Explicit key wins over env and the built-in default.
	t.Run("explicit wins", func(t *testing.T) {
		t.Setenv("GOPHERTRUNK_RR_KEY", "env-key")
		setDefaultAppKey(t, "builtin-key")
		got := ResolveAuth(Auth{AppKey: "explicit", Username: "u", Password: "p"})
		if got.AppKey != "explicit" {
			t.Errorf("AppKey = %q, want explicit", got.AppKey)
		}
		if got.Username != "u" || got.Password != "p" {
			t.Errorf("user/pass should be untouched, got %q/%q", got.Username, got.Password)
		}
	})

	// Env beats the built-in default when no explicit key.
	t.Run("env over builtin", func(t *testing.T) {
		t.Setenv("GOPHERTRUNK_RR_KEY", "env-key")
		setDefaultAppKey(t, "builtin-key")
		if got := ResolveAuth(Auth{}).AppKey; got != "env-key" {
			t.Errorf("AppKey = %q, want env-key", got)
		}
	})

	// Built-in default is the last resort.
	t.Run("builtin fallback", func(t *testing.T) {
		t.Setenv("GOPHERTRUNK_RR_KEY", "")
		setDefaultAppKey(t, "builtin-key")
		if got := ResolveAuth(Auth{}).AppKey; got != "builtin-key" {
			t.Errorf("AppKey = %q, want builtin-key", got)
		}
		if got := EffectiveAppKey(""); got != "builtin-key" {
			t.Errorf("EffectiveAppKey = %q, want builtin-key", got)
		}
	})

	// No key anywhere → empty (NewClient then returns ErrNoCredentials).
	t.Run("none", func(t *testing.T) {
		t.Setenv("GOPHERTRUNK_RR_KEY", "")
		setDefaultAppKey(t, "")
		if got := ResolveAuth(Auth{}).AppKey; got != "" {
			t.Errorf("AppKey = %q, want empty", got)
		}
	})
}

// setDefaultAppKey overrides the build-time DefaultAppKey for the duration of a
// test and restores it afterwards.
func setDefaultAppKey(t *testing.T, key string) {
	t.Helper()
	prev := DefaultAppKey
	DefaultAppKey = key
	t.Cleanup(func() { DefaultAppKey = prev })
}
