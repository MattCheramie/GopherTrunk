// Package api exposes GopherTrunk's read-only control surface and the
// streaming events feed.
//
// The default daemon links the HTTP+SSE+WebSocket server defined here.
// gRPC bindings (proto/*.proto under the repo root) generate Go code
// at `internal/api/pb/v1` when `make proto` is invoked with protoc and
// the standard plugins installed; the gRPC server then sits alongside
// this HTTP server in a follow-up phase.
//
// Layout:
//
//   server.go    HTTP server lifecycle (Run, Close), routing, mux
//   handlers.go  REST handlers (health/version/systems/talkgroups/calls)
//   sse.go       Server-Sent Events stream of internal/events bus events
//   ws.go        WebSocket bridge that streams the same events as JSON
//   types.go     JSON-friendly DTOs (mirroring the proto definitions)
package api
