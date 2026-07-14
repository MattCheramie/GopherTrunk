// Package gtbundle reads and writes the GopherTrunk Bundle — a single
// portable .gtb.tar.gz archive that packages one SDR capture-to-analysis
// "case": the raw IQ capture(s), a narrowband slice, the session logs, the
// SigLab signal-analysis result, the CryptoLab crypto-assessment result, and
// the hunt site/network mapping, all tied together by a human-and-machine
// readable MANIFEST.yaml.
//
// One bundle carries a whole investigation so it can be shared with the
// community as a single file and flow seamlessly capture → SigLab → CryptoLab
// → back into the scanner config.
//
// Disambiguation: the word "bundle" is overloaded in this repo. The hunt
// package's FormatBundle (internal/hunt/export.go) is a multi-section CSV that
// round-trips a discovered system into config.yaml — the "hunt CSV import
// bundle". That CSV is *stored inside* a GopherTrunk Bundle (as
// mapping/system.bundle.csv, role mapping-export); it is not this archive. This
// package is always spelled "GopherTrunk Bundle" / .gtb, and its Go identifier
// is gtbundle, never bundle.
//
// On-disk layout, under a single top-level directory named after the bundle:
//
//	<name>/
//	  MANIFEST.yaml          index: schema_version, provenance, file inventory
//	  README.md              auto-generated human summary
//	  notes.md               free-form operator notes (optional)
//	  capture/               raw IQ (.cfile), narrowband slice (.wav), meta (.yaml)
//	  logs/                  events.jsonl (all bus events), messages.log (human)
//	  mapping/               system.json (DiscoveredSystem), survey.jsonl, exports
//	  siglab/                result.yaml (siglab.Result), events.jsonl
//	  cryptolab/             frames.jsonl, assess.result.yaml (cryptolab.Result)
//
// Serialization follows the repo conventions: YAML (snake_case) for the
// manifest and analysis summaries whose models carry yaml tags; JSON for
// hunt.DiscoveredSystem (which has only json tags); JSONL for streams; raw
// binary for IQ. Frequencies are integer Hz, timestamps UTC RFC3339.
package gtbundle
