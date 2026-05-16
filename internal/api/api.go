// Package api exposes GopherTrunk's read + write control surface, the
// streaming events feed, and the gRPC mirror of the same state.
//
// The default daemon links both the HTTP+SSE+WebSocket server defined
// here and the gRPC server in grpc.go. Mutation endpoints (end call,
// set talkgroup priority/lockout/scan, retention sweep, tone-out
// reset, scanner cockpit) are gated behind api.allow_mutations so a
// daemon bound to a trusted interface can expose them while a default
// build stays read-only.
//
// gRPC bindings (proto/*.proto under the repo root) generate Go code
// at internal/api/pb/v1 when `make proto` is invoked with protoc and
// the standard plugins installed.
//
// Layout:
//
//	server.go               HTTP server lifecycle (Run, Close), routing, mux
//	handlers.go             REST read handlers (health/version/systems/talkgroups/calls/devices)
//	handlers_mutations.go   REST mutation handlers (end-call, retention, talkgroup, tone-reset)
//	handlers_scanner.go     Scanner cockpit REST handlers (status + 6 mutation routes)
//	sse.go                  Server-Sent Events stream of internal/events bus events
//	ws.go                   WebSocket bridge that streams the same events as JSON
//	grpc.go                 gRPC server: SystemService + TalkgroupService + AudioService
//	types.go                JSON-friendly DTOs (mirroring the proto definitions)
package api
