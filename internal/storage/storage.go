// Package storage persists GopherTrunk's runtime data to disk.
//
// The default backend is SQLite via the pure-Go `modernc.org/sqlite`
// driver — CGO_ENABLED=0 stays true across the daemon and the daemon
// cross-compiles to linux/arm64 without toolchain gymnastics.
//
// Layout:
//
//	sqlite.go     Open + schema migrations. One-shot at startup.
//	calllog.go    CallLog: subscribes to events.KindCallStart /
//	              KindCallEnd from the trunking engine, writes rows
//	              keyed by (device serial, started_at).
//	retention.go  Background sweeper that deletes DB rows + the WAV /
//	              raw files written by internal/voice older than a
//	              configurable cutoff.
//
// The API's /api/v1/calls/history endpoint reads through the `History`
// query helpers exposed here. There is no gRPC call-log service today
// (history is REST only).
package storage
