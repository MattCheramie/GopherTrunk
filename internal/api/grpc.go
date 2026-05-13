package api

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"time"

	apiv1 "github.com/MattCheramie/GopherTrunk/internal/api/pb/v1"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/status"
)

// GRPCServer hosts the gRPC SystemService + TalkgroupService against the
// same in-process state as the HTTP/SSE/WebSocket server.
//
// AudioService.StreamAudio is registered but is a no-op until the demod
// pipeline composer (deferred) starts pushing PCM into a per-call
// channel. The streaming surface is in place so clients can call it
// without churning at the wire-protocol layer when audio lands.
type GRPCServer struct {
	apiv1.UnimplementedSystemServiceServer
	apiv1.UnimplementedTalkgroupServiceServer
	apiv1.UnimplementedAudioServiceServer

	addr       string
	systems    []trunking.System
	talkgroups *trunking.TalkgroupDB
	engine     EngineSnapshot
	audio      *AudioPublisher
	log        *slog.Logger

	srv *grpc.Server
}

// GRPCServerOptions configure a new GRPCServer.
type GRPCServerOptions struct {
	Addr       string
	Systems    []trunking.System
	Talkgroups *trunking.TalkgroupDB
	Engine     EngineSnapshot
	// Audio is the optional AudioPublisher backing StreamAudio.
	// When nil the RPC still registers (so clients don't churn
	// at the wire-protocol layer if audio is configured off) but
	// returns Unavailable rather than streaming frames.
	Audio *AudioPublisher
	Log   *slog.Logger
	// TLSCert and TLSKey, when both non-empty, switch the gRPC
	// server to TLS using credentials.NewServerTLSFromFile. Same
	// disk-loaded-once semantics as the HTTP server's TLS support.
	// Leave both empty for plain TCP (default; appropriate for
	// loopback / private-network deployments).
	TLSCert string
	TLSKey  string
}

// NewGRPCServer constructs the server but does not bind a listener.
func NewGRPCServer(opts GRPCServerOptions) (*GRPCServer, error) {
	if opts.Addr == "" {
		return nil, errors.New("api: gRPC Addr is required")
	}
	log := opts.Log
	if log == nil {
		log = slog.Default()
	}
	if opts.Talkgroups == nil {
		opts.Talkgroups = trunking.NewTalkgroupDB()
	}
	g := &GRPCServer{
		addr:       opts.Addr,
		systems:    append([]trunking.System(nil), opts.Systems...),
		talkgroups: opts.Talkgroups,
		engine:     opts.Engine,
		audio:      opts.Audio,
		log:        log,
	}
	// Keep-alive guards long-lived RPCs (StreamAudio in particular)
	// against silently-dead peers — without server-side pings, a
	// client whose network drops without a TCP FIN/RST would pin a
	// gRPC stream + its publisher subscription forever. The values
	// match Google's published defaults: Time = 30 s of idle before
	// the first ping, Timeout = 10 s for the ack before the
	// connection is closed. MinTime = 5 s gates client-side ping
	// floods.
	keepaliveParams := keepalive.ServerParameters{
		Time:    30 * time.Second,
		Timeout: 10 * time.Second,
	}
	keepaliveEnforcement := keepalive.EnforcementPolicy{
		MinTime:             5 * time.Second,
		PermitWithoutStream: true,
	}
	srvOpts := []grpc.ServerOption{
		grpc.KeepaliveParams(keepaliveParams),
		grpc.KeepaliveEnforcementPolicy(keepaliveEnforcement),
	}
	// TLS: same all-or-nothing semantics as the HTTP server.
	if (opts.TLSCert == "") != (opts.TLSKey == "") {
		return nil, errors.New("api: grpc tls_cert and tls_key must both be set or both be empty")
	}
	if opts.TLSCert != "" {
		creds, err := credentials.NewServerTLSFromFile(opts.TLSCert, opts.TLSKey)
		if err != nil {
			return nil, fmt.Errorf("api: load gRPC TLS credentials: %w", err)
		}
		srvOpts = append(srvOpts, grpc.Creds(creds))
	}
	g.srv = grpc.NewServer(srvOpts...)
	apiv1.RegisterSystemServiceServer(g.srv, g)
	apiv1.RegisterTalkgroupServiceServer(g.srv, g)
	apiv1.RegisterAudioServiceServer(g.srv, g)
	return g, nil
}

// Run binds the listener and serves until ctx cancels.
func (g *GRPCServer) Run(ctx context.Context) error {
	listener, err := net.Listen("tcp", g.addr)
	if err != nil {
		return err
	}
	g.log.Info("api: gRPC listening", "addr", listener.Addr().String())
	errCh := make(chan error, 1)
	go func() { errCh <- g.srv.Serve(listener) }()
	select {
	case <-ctx.Done():
		g.srv.GracefulStop()
		return nil
	case err := <-errCh:
		return err
	}
}

// Stop gracefully halts the gRPC server.
func (g *GRPCServer) Stop() { g.srv.GracefulStop() }

// --- SystemService ---

func (g *GRPCServer) ListSystems(_ context.Context, _ *apiv1.ListSystemsRequest) (*apiv1.ListSystemsResponse, error) {
	out := make([]*apiv1.System, 0, len(g.systems))
	for _, s := range g.systems {
		out = append(out, systemToPB(s))
	}
	return &apiv1.ListSystemsResponse{Systems: out}, nil
}

func (g *GRPCServer) GetSystem(_ context.Context, req *apiv1.GetSystemRequest) (*apiv1.GetSystemResponse, error) {
	for _, s := range g.systems {
		if s.Name == req.GetName() {
			return &apiv1.GetSystemResponse{System: systemToPB(s)}, nil
		}
	}
	return nil, status.Errorf(codes.NotFound, "system %q not found", req.GetName())
}

// --- TalkgroupService ---

func (g *GRPCServer) ListTalkgroups(_ context.Context, _ *apiv1.ListTalkgroupsRequest) (*apiv1.ListTalkgroupsResponse, error) {
	all := g.talkgroups.All()
	out := make([]*apiv1.TalkGroup, 0, len(all))
	for _, tg := range all {
		out = append(out, talkgroupToPB(tg))
	}
	return &apiv1.ListTalkgroupsResponse{Talkgroups: out}, nil
}

func (g *GRPCServer) GetTalkgroup(_ context.Context, req *apiv1.GetTalkgroupRequest) (*apiv1.GetTalkgroupResponse, error) {
	tg := g.talkgroups.Lookup(req.GetId())
	if tg == nil {
		return nil, status.Errorf(codes.NotFound, "talkgroup %d not found", req.GetId())
	}
	return &apiv1.GetTalkgroupResponse{Talkgroup: talkgroupToPB(tg)}, nil
}

func (g *GRPCServer) ListActiveCalls(_ context.Context, _ *apiv1.ListActiveCallsRequest) (*apiv1.ListActiveCallsResponse, error) {
	if g.engine == nil {
		return &apiv1.ListActiveCallsResponse{}, nil
	}
	active := g.engine.ActiveCalls()
	out := make([]*apiv1.ActiveCall, 0, len(active))
	for _, ac := range active {
		out = append(out, activeCallToPB(ac))
	}
	return &apiv1.ListActiveCallsResponse{Calls: out}, nil
}

// --- AudioService ---
// StreamAudio fans decoded PCM from the per-call composer to the
// gRPC client. The request's device_serials / talkgroup_ids filters
// act as allow-lists; empty matches everything. PCM samples are
// 16-bit little-endian mono at the recorder's configured rate
// (typically 8 kHz).
//
// Returns:
//
//	codes.Unavailable when the daemon was started without an audio
//	  publisher (no composer wired, audio off, or older
//	  configuration).
//	nil on graceful client cancel.
//	any send-side error from the gRPC stream — typically the
//	  caller hung up.
func (g *GRPCServer) StreamAudio(req *apiv1.StreamAudioRequest, srv apiv1.AudioService_StreamAudioServer) error {
	if g.audio == nil {
		return status.Error(codes.Unavailable, "audio publisher not wired (no composer or audio off)")
	}
	filter := AudioSubFilter{
		DeviceSerials: append([]string(nil), req.GetDeviceSerials()...),
		TalkgroupIDs:  append([]uint32(nil), req.GetTalkgroupIds()...),
		IncludeRaw:    req.GetIncludeRaw(),
	}
	sub := g.audio.Subscribe(filter)
	defer g.audio.Unsubscribe(sub)
	ctx := srv.Context()
	for {
		select {
		case <-ctx.Done():
			return nil
		case frame, ok := <-sub.ch:
			if !ok {
				return nil
			}
			if err := srv.Send(frame); err != nil {
				return err
			}
		}
	}
}

// --- helpers: trunking/* → *apiv1.* ---

func systemToPB(s trunking.System) *apiv1.System {
	return &apiv1.System{
		Name:            s.Name,
		Protocol:        s.Protocol.String(),
		ControlChannels: append([]uint32(nil), s.ControlChannels...),
		Wacn:            s.WACN,
		SystemId:        uint32(s.SystemID),
		Rfss:            uint32(s.RFSS),
		Site:            uint32(s.Site),
	}
}

func talkgroupToPB(tg *trunking.TalkGroup) *apiv1.TalkGroup {
	if tg == nil {
		return nil
	}
	return &apiv1.TalkGroup{
		Id:          tg.ID,
		AlphaTag:    tg.AlphaTag,
		Description: tg.Description,
		Tag:         tg.Tag,
		Group:       tg.Group,
		Mode:        tg.Mode,
		Priority:    int32(tg.Priority),
		Lockout:     tg.Lockout,
	}
}

func grantToPB(g trunking.Grant) *apiv1.Grant {
	return &apiv1.Grant{
		System: g.System, Protocol: g.Protocol,
		GroupId: g.GroupID, SourceId: g.SourceID,
		FrequencyHz: g.FrequencyHz,
		ChannelId:   uint32(g.ChannelID), ChannelNumber: uint32(g.ChannelNum),
		Encrypted: g.Encrypted, Emergency: g.Emergency, DataCall: g.DataCall,
	}
}

func activeCallToPB(ac *trunking.ActiveCall) *apiv1.ActiveCall {
	return &apiv1.ActiveCall{
		Grant:        grantToPB(ac.Grant),
		Talkgroup:    talkgroupToPB(ac.Talkgroup),
		DeviceSerial: ac.Device.Serial,
		StartedAt:    ac.StartedAt.UTC().Format("2006-01-02T15:04:05Z"),
		LastHeardAt:  ac.LastHeardAt.UTC().Format("2006-01-02T15:04:05Z"),
	}
}
