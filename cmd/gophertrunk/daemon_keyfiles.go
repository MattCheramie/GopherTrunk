package main

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"sort"
	"strings"
	"syscall"

	"github.com/charmbracelet/x/term"

	"github.com/MattCheramie/GopherTrunk/internal/config"
	"github.com/MattCheramie/GopherTrunk/internal/ekc"
	"github.com/MattCheramie/GopherTrunk/internal/radio/p25"
)

// ekcAlgorithm maps a KFDtool key's TIA-102 ALGID to the encryption_keys
// algorithm GopherTrunk decrypts it with, or "" when it has no decoder for
// it (DES-XL 0x9F, and anything not in the registry).
func ekcAlgorithm(algID int) string {
	switch algID {
	case 0x81:
		return "des"
	case 0x83, 0x86:
		return "tdes"
	case 0x84, 0x85, 0x89:
		return "aes"
	case int(p25.AlgorithmADP):
		return "rc4"
	}
	return ""
}

// ekcLoad is what a container contributed: its keys as encryption_keys
// entries, and why the rest were left out (for one log line; never key
// material).
type ekcLoad struct {
	keys    []config.EncryptionKeyConfig
	kek     int            // key-encryption keys: no use to a receiver
	keyset  int            // in another keyset than the one asked for
	shadow  int            // same key ID as a key from the active keyset
	badLen  int            // key length the algorithm does not accept
	badID   int            // key ID outside 0..65535
	unknown map[string]int // ALGID with no decoder → count
}

// ekcKeys turns a decrypted container into encryption_keys entries. Only
// traffic keys are taken (a KEK wraps other keys and decrypts no voice).
// With keyset != 0 only that keyset's keys are taken; otherwise, where two
// keysets carry the same key ID, the active keyset's key wins — a KFD holds
// the next keyset alongside the current one, under the same IDs.
func ekcKeys(c *ekc.Container, keyset int) ekcLoad {
	var out ekcLoad
	type pick struct {
		cfg    config.EncryptionKeyConfig
		active bool
	}
	byID := map[uint16]pick{}
	var order []uint16
	for _, k := range c.Keys {
		switch {
		case k.KEK:
			out.kek++
			continue
		case keyset != 0 && k.KeysetID != keyset:
			out.keyset++
			continue
		case k.KeyID < 0 || k.KeyID > 0xFFFF:
			out.badID++
			continue
		}
		alg := ekcAlgorithm(k.AlgorithmID)
		if alg == "" {
			if out.unknown == nil {
				out.unknown = map[string]int{}
			}
			out.unknown[p25.FormatAlgorithm(uint8(k.AlgorithmID))]++
			continue
		}
		cfg := config.EncryptionKeyConfig{KeyID: uint16(k.KeyID), Algorithm: alg, Key: fmt.Sprintf("%X", k.Material)}
		if cfg.Validate() != nil {
			out.badLen++
			continue
		}
		prev, seen := byID[cfg.KeyID]
		switch {
		case !seen:
			order = append(order, cfg.KeyID)
		case prev.active || !k.ActiveKeyset:
			out.shadow++
			continue
		default:
			out.shadow++ // the inactive one it replaces
		}
		byID[cfg.KeyID] = pick{cfg, k.ActiveKeyset}
	}
	for _, id := range order {
		out.keys = append(out.keys, byID[id].cfg)
	}
	return out
}

// skipped renders why keys were left out, e.g. "1 KEK, 1 0x9F (DES-XL)".
func (l ekcLoad) skipped() string {
	var parts []string
	add := func(n int, what string) {
		if n > 0 {
			parts = append(parts, fmt.Sprintf("%d %s", n, what))
		}
	}
	add(l.kek, "KEK")
	add(l.keyset, "in other keysets")
	add(l.shadow, "superseded by the active keyset")
	add(l.badLen, "with a key length the algorithm does not take")
	add(l.badID, "with a key ID over 65535")
	algs := make([]string, 0, len(l.unknown))
	for a := range l.unknown {
		algs = append(algs, a)
	}
	sort.Strings(algs)
	for _, a := range algs {
		add(l.unknown[a], a+", which GopherTrunk does not decrypt")
	}
	return strings.Join(parts, ", ")
}

// ekcPasswordPrompt asks for a container's password on the terminal, with no
// echo. It is a variable so tests can stand in for the operator.
var ekcPasswordPrompt = func(prompt string) (string, bool, error) {
	fd := os.Stdin.Fd()
	if !term.IsTerminal(fd) {
		return "", false, nil
	}
	// The daemon has already taken SIGINT (main's NotifyContext) by the time
	// it asks, so Ctrl-C would not end the process: the prompt would just sit
	// there. Watch for it here, put the terminal's echo back, and give up.
	state, err := term.GetState(fd)
	if err != nil {
		return "", false, err
	}
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(interrupt)
	type answer struct {
		b   []byte
		err error
	}
	got := make(chan answer, 1)
	fmt.Fprint(os.Stderr, prompt)
	go func() {
		b, err := term.ReadPassword(fd)
		got <- answer{b, err}
	}()
	select {
	case a := <-got:
		fmt.Fprintln(os.Stderr)
		return string(a.b), true, a.err
	case <-interrupt:
		_ = term.Restore(fd, state)
		fmt.Fprintln(os.Stderr)
		return "", true, errors.New("password prompt interrupted")
	}
}

// ekcPromptTries is how many times a wrong password typed at the terminal is
// asked again before startup gives up.
const ekcPromptTries = 3

// openKeyFile decrypts one encryption_key_files entry. The password comes
// from the entry, from its password_env variable, or — when neither is set —
// from the terminal; without a terminal it is an error, so a service cannot
// start silently without its keys.
func openKeyFile(system string, kf config.EncryptionKeyFileConfig) (*ekc.Container, error) {
	c, _, err := openKeyFileWithPassword(system, kf)
	return c, err
}

// openKeyFileWithPassword is openKeyFile that also returns the password that
// opened the container, for `gophertrunk ekc` commands that save it again.
func openKeyFileWithPassword(system string, kf config.EncryptionKeyFileConfig) (*ekc.Container, string, error) {
	data, err := os.ReadFile(kf.Path)
	if err != nil {
		return nil, "", err
	}
	switch {
	case kf.Password != "":
		c, err := ekc.Open(data, kf.Password)
		return c, kf.Password, err
	case kf.PasswordEnv != "":
		pw, ok := os.LookupEnv(kf.PasswordEnv)
		if !ok {
			return nil, "", fmt.Errorf("password_env %s is not set", kf.PasswordEnv)
		}
		c, err := ekc.Open(data, pw)
		return c, pw, err
	}
	prompt := fmt.Sprintf("Password for key container %s: ", kf.Path)
	if system != "" {
		prompt = fmt.Sprintf("Password for key container %s (system %s): ", kf.Path, system)
	}
	for try := 1; ; try++ {
		pw, ok, err := ekcPasswordPrompt(prompt)
		if err != nil {
			return nil, "", err
		}
		if !ok {
			return nil, "", errors.New("no password given: set password or password_env, or start from a terminal to be asked for it")
		}
		c, err := ekc.Open(data, pw)
		if !errors.Is(err, ekc.ErrWrongPassword) || try == ekcPromptTries {
			return c, pw, err
		}
		fmt.Fprintln(os.Stderr, "Wrong password, try again.")
	}
}

// withKeyFileKeys returns a copy of systems with each system's
// encryption_key_files decrypted and added to its EncryptionKeys. An
// encryption_keys entry with the same key_id wins (the operator's explicit
// entry). The result feeds the key consumers only (engine key-ID set, voice
// key resolver, recorder) — never the daemon's config, which the API can
// serve and write back, so decrypted keys never reach a YAML file.
func withKeyFileKeys(systems []config.SystemConfig, log *slog.Logger) ([]config.SystemConfig, error) {
	out := make([]config.SystemConfig, len(systems))
	copy(out, systems)
	for i := range out {
		s := &out[i]
		if len(s.EncryptionKeyFiles) == 0 {
			continue
		}
		have := make(map[uint16]bool, len(s.EncryptionKeys))
		for _, k := range s.EncryptionKeys {
			have[k.KeyID] = true
		}
		keys := append([]config.EncryptionKeyConfig(nil), s.EncryptionKeys...)
		for j, kf := range s.EncryptionKeyFiles {
			c, err := openKeyFile(s.Name, kf)
			if err != nil {
				return nil, fmt.Errorf("trunking.systems[%d].encryption_key_files[%d] (%s): %w", i, j, kf.Path, err)
			}
			load := ekcKeys(c, kf.Keyset)
			added, overridden := 0, 0
			for _, k := range load.keys {
				if have[k.KeyID] {
					overridden++
					continue
				}
				have[k.KeyID] = true
				keys = append(keys, k)
				added++
			}
			args := []any{"system", s.Name, "file", kf.Path, "keys", added}
			if overridden > 0 {
				args = append(args, "overridden_by_encryption_keys", overridden)
			}
			if sk := load.skipped(); sk != "" {
				args = append(args, "skipped", sk)
			}
			log.Info("daemon: decryption keys loaded from key container", args...)
		}
		s.EncryptionKeys = keys
	}
	return out, nil
}
