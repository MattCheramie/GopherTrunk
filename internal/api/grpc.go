package api

import (
	"context"
	"errors"
	"log/slog"
	"net"

	apiv1 "github.com/MattCheramie/GopherTrunk/internal/api/pb/v1"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
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
	log        *slog.Logger

	srv *grpc.Server
}

// GRPCServerOptions configure a new GRPCServer.
type GRPCServerOptions struct {
	Addr       string
	Systems    []trunking.System
	Talkgroups *trunking.TalkgroupDB
	Engine     EngineSnapshot
	Log        *slog.Logger
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
		log:        log,
	}
	g.srv = grpc.NewServer()
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

// --- AudioService (stub) ---
// StreamAudio is registered so clients can call it without churn at the
// wire-protocol layer once the demod pipeline composer (deferred) starts
// producing PCM. For now it returns Unimplemented so downstream code
// fails cleanly rather than silently hanging.
func (g *GRPCServer) StreamAudio(_ *apiv1.StreamAudioRequest, _ apiv1.AudioService_StreamAudioServer) error {
	return status.Error(codes.Unimplemented, "audio streaming lands with the demod pipeline composer")
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
