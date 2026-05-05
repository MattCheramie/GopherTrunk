// Package storage persists GopherTrunk's runtime data to disk.
//
// The default backend is SQLite via the pure-Go `modernc.org/sqlite`
// driver — no extra CGO beyond what librtlsdr already needs, and the
// daemon stays cross-compilable to linux/arm64 without toolchain
// gymnastics.
//
// Layout:
//
//   sqlite.go     Open + schema migrations. One-shot at startup.
//   calllog.go    CallLog: subscribes to events.KindCallStart /
//                 KindCallEnd from the trunking engine, writes rows
//                 keyed by (device serial, started_at).
//   retention.go  Background sweeper that deletes DB rows + the WAV /
//                 raw files written by internal/voice older than a
//                 configurable cutoff.
//
// Higher-level integrations (the API's /api/v1/calls/history endpoint,
// gRPC CallLogService) are layered on top via the `History` query
// helpers exposed here.
package storage
