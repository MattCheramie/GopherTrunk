package main

import (
	"github.com/MattCheramie/GopherTrunk/internal/config"
	"github.com/MattCheramie/GopherTrunk/internal/voice/composer"
)

// conventionalKeySystems turns the encryption_keys of every conventional
// scan-list channel (scanner.conventional and sdr.devices[].conventional)
// into pseudo trunking systems named composer.ConventionalKeySystem(hz), so
// buildKeyResolver / buildKeyConfigured serve them unchanged and the voice
// chain of a scanner call — whose grant names the scanner, not a system —
// resolves the channel's keys by its frequency (issue #1261). Two channels
// on one frequency (distinct tones / NACs) pool their keys; a duplicate
// key_id between them keeps the first.
func conventionalKeySystems(cfg config.Config) []config.SystemConfig {
	byHz := map[uint32]*config.SystemConfig{}
	var order []uint32
	add := func(ch config.ConvChannelConfig) {
		if len(ch.EncryptionKeys) == 0 {
			return
		}
		s := byHz[ch.FrequencyHz]
		if s == nil {
			s = &config.SystemConfig{Name: composer.ConventionalKeySystem(ch.FrequencyHz)}
			byHz[ch.FrequencyHz] = s
			order = append(order, ch.FrequencyHz)
		}
	next:
		for _, k := range ch.EncryptionKeys {
			for _, have := range s.EncryptionKeys {
				if have.Scope() == k.Scope() {
					continue next
				}
			}
			s.EncryptionKeys = append(s.EncryptionKeys, k)
		}
	}
	for _, ch := range cfg.Scanner.Conventional {
		add(ch)
	}
	for _, dev := range cfg.SDR.Devices {
		for _, ch := range dev.Conventional {
			add(ch)
		}
	}
	out := make([]config.SystemConfig, 0, len(order))
	for _, hz := range order {
		out = append(out, *byHz[hz])
	}
	return out
}
