package api

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"strings"
	"time"

	apiv1 "github.com/MattCheramie/GopherTrunk/internal/api/pb/v1"
	gtdiag "github.com/MattCheramie/GopherTrunk/internal/diag"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/metadata"
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
	apiv1.UnimplementedRIDServiceServer
	apiv1.UnimplementedAudioServiceServer

	addr         string
	systems      []trunking.System
	talkgroups   *trunking.TalkgroupDB
	rids         *trunking.RIDDB
	affiliations AffiliationProvider
	history      HistoryQuery
	engine       EngineSnapshot
	audio        *AudioPublisher
	log          *slog.Logger

	diagnostics   *gtdiag.Collector
	verboseErrors bool

	srv *grpc.Server
}

// GRPCServerOptions configure a new GRPCServer.
type GRPCServerOptions struct {
	Addr       string
	Systems    []trunking.System
	Talkgroups *trunking.TalkgroupDB
	// RIDs is the operator-configured radio-ID alias table. When nil
	// the server allocates an empty one so RIDService still serves a
	// stable shape.
	RIDs *trunking.RIDDB
	// Affiliations is the read side of the affiliation tracker —
	// supplies the live UnitActivity overlay for RIDService. Optional.
	Affiliations AffiliationProvider
	// History supplies per-RID call history for ListRIDHistory.
	// Optional; without it ListRIDHistory returns Unavailable.
	History HistoryQuery
	Engine  EngineSnapshot
	// Audio is the optional AudioPublisher backing StreamAudio.
	// When nil the RPC still registers (so clients don't churn
	// at the wire-protocol layer if audio is configured off) but
	// returns Unavailable rather than streaming frames.
	Audio *AudioPublisher
	Log   *slog.Logger
	// Diagnostics is the shared error-diagnostics collector. When set
	// with VerboseErrors (or a request carrying the gophertrunk-verbose
	// metadata key), returned errors are decorated with the diagnostics
	// banner + wrapped chain via a server interceptor.
	Diagnostics *gtdiag.Collector
	// VerboseErrors mirrors diagnostics.verbose_errors.
	VerboseErrors bool
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
	if opts.RIDs == nil {
		opts.RIDs = trunking.NewRIDDB()
	}
	g := &GRPCServer{
		addr:          opts.Addr,
		systems:       append([]trunking.System(nil), opts.Systems...),
		talkgroups:    opts.Talkgroups,
		rids:          opts.RIDs,
		affiliations:  opts.Affiliations,
		history:       opts.History,
		engine:        opts.Engine,
		audio:         opts.Audio,
		log:           log,
		diagnostics:   opts.Diagnostics,
		verboseErrors: opts.VerboseErrors,
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
		grpc.UnaryInterceptor(g.diagUnaryInterceptor),
		grpc.StreamInterceptor(g.diagStreamInterceptor),
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
	apiv1.RegisterRIDServiceServer(g.srv, g)
	apiv1.RegisterAudioServiceServer(g.srv, g)
	return g, nil
}

// diagUnaryInterceptor decorates a failing unary RPC's error with the
// diagnostics banner + wrapped chain when verbose reporting is enabled
// (config) or the caller opted in via the gophertrunk-verbose metadata
// key. The status code is always preserved; only the message text is
// extended, so existing clients that don't parse the suffix are
// unaffected.
func (g *GRPCServer) diagUnaryInterceptor(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
	resp, err := handler(ctx, req)
	if err != nil {
		return resp, g.decorateErr(ctx, err)
	}
	return resp, nil
}

// diagStreamInterceptor is the streaming-RPC analogue of
// diagUnaryInterceptor.
func (g *GRPCServer) diagStreamInterceptor(srv any, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
	if err := handler(srv, ss); err != nil {
		return g.decorateErr(ss.Context(), err)
	}
	return nil
}

// decorateErr appends the diagnostics banner + error chain to a gRPC
// status message when verbose is on (or requested via metadata) and a
// collector is wired. The original status code is retained.
func (g *GRPCServer) decorateErr(ctx context.Context, err error) error {
	if g.diagnostics == nil || !g.verboseRequested(ctx) {
		return err
	}
	st, _ := status.FromError(err)
	banner := gtdiag.FormatBannerPlain(g.diagnostics.SysInfo())
	chain := gtdiag.Chain(err)
	msg := st.Message()
	if banner != "" {
		msg += "\n\n" + banner
	}
	if len(chain) > 1 {
		msg += "\n\nerror chain: " + strings.Join(chain, " <- ")
	}
	return status.Error(st.Code(), msg)
}

// verboseRequested reports whether verbose diagnostics should be
// attached: either the daemon enabled them globally, or the caller set
// the gophertrunk-verbose metadata key to a truthy value.
func (g *GRPCServer) verboseRequested(ctx context.Context) bool {
	if g.verboseErrors {
		return true
	}
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return false
	}
	for _, v := range md.Get("gophertrunk-verbose") {
		if v == "1" || strings.EqualFold(v, "true") {
			return true
		}
	}
	return false
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

// --- RIDService ---

func (g *GRPCServer) ListRIDs(_ context.Context, _ *apiv1.ListRIDsRequest) (*apiv1.ListRIDsResponse, error) {
	out := g.mergedRIDs()
	return &apiv1.ListRIDsResponse{Rids: out}, nil
}

func (g *GRPCServer) GetRID(_ context.Context, req *apiv1.GetRIDRequest) (*apiv1.GetRIDResponse, error) {
	id := req.GetId()
	rid := ridToPB(g.rids.Lookup(id), true)
	if g.affiliations != nil {
		for _, u := range g.affiliations.Affiliations() {
			if u.RadioID != id {
				continue
			}
			rid = mergeRIDLivePB(rid, u)
			break
		}
	}
	if rid == nil || rid.Id == 0 {
		return nil, status.Errorf(codes.NotFound, "rid %d not found", id)
	}
	return &apiv1.GetRIDResponse{Rid: rid}, nil
}

func (g *GRPCServer) ListRIDHistory(ctx context.Context, req *apiv1.ListRIDHistoryRequest) (*apiv1.ListRIDHistoryResponse, error) {
	if g.history == nil {
		return nil, status.Error(codes.Unavailable, "call log persistence is not enabled")
	}
	if req.GetId() == 0 {
		return nil, status.Error(codes.InvalidArgument, "rid id required")
	}
	limit := int(req.GetLimit())
	if limit <= 0 {
		limit = 100
	}
	if limit > 1000 {
		limit = 1000
	}
	rows, err := g.history.History(ctx, HistoryFilter{
		SourceID: req.GetId(),
		System:   req.GetSystem(),
		Limit:    limit,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "history query: %v", err)
	}
	out := make([]*apiv1.RIDCallRow, 0, len(rows))
	for _, r := range rows {
		out = append(out, callRowToPB(r))
	}
	return &apiv1.ListRIDHistoryResponse{Calls: out}, nil
}

// mergedRIDs walks both the static RIDDB and the live affiliation
// tracker, merging by RadioID. Mirrors handleListRIDs / mergedRIDList
// in the HTTP layer.
func (g *GRPCServer) mergedRIDs() []*apiv1.RID {
	byID := map[uint32]*apiv1.RID{}
	for _, r := range g.rids.All() {
		byID[r.ID] = ridToPB(r, true)
	}
	if g.affiliations != nil {
		for _, u := range g.affiliations.Affiliations() {
			byID[u.RadioID] = mergeRIDLivePB(byID[u.RadioID], u)
		}
	}
	out := make([]*apiv1.RID, 0, len(byID))
	for _, rid := range byID {
		out = append(out, rid)
	}
	return out
}

func ridToPB(r *trunking.RID, configured bool) *apiv1.RID {
	if r == nil {
		return nil
	}
	return &apiv1.RID{
		Id:          r.ID,
		Alias:       r.Alias,
		Description: r.Description,
		Tag:         r.Tag,
		Group:       r.Group,
		Owner:       r.Owner,
		Priority:    uint32(r.Priority),
		Lockout:     r.Lockout,
		Watch:       r.Watch,
		Icon:        r.Icon,
		Configured:  configured,
	}
}

func mergeRIDLivePB(p *apiv1.RID, u trunking.UnitActivity) *apiv1.RID {
	if p == nil {
		p = &apiv1.RID{Id: u.RadioID, Watch: true}
	}
	p.System = u.System
	p.Protocol = u.Protocol
	p.LastTalkgroup = u.Talkgroup
	p.TalkerAlias = u.TalkerAlias
	if !u.TalkerAliasAt.IsZero() {
		p.TalkerAliasAt = u.TalkerAliasAt.UTC().Format(time.RFC3339)
	}
	p.CallCount = u.CallCount
	if !u.FirstSeen.IsZero() {
		p.FirstSeen = u.FirstSeen.UTC().Format(time.RFC3339)
	}
	if !u.LastSeen.IsZero() {
		p.LastSeen = u.LastSeen.UTC().Format(time.RFC3339)
	}
	return p
}

func callRowToPB(r CallRow) *apiv1.RIDCallRow {
	pb := &apiv1.RIDCallRow{
		Id:             r.ID,
		System:         r.System,
		Protocol:       r.Protocol,
		GroupId:        r.GroupID,
		SourceId:       r.SourceID,
		FrequencyHz:    r.FrequencyHz,
		Encrypted:      r.Encrypted,
		Emergency:      r.Emergency,
		DataCall:       r.DataCall,
		DeviceSerial:   r.DeviceSerial,
		DurationMs:     r.DurationMs,
		EndReason:      r.EndReason,
		TalkgroupAlpha: r.TalkgroupAlpha,
	}
	if !r.StartedAt.IsZero() {
		pb.StartedAt = r.StartedAt.UTC().Format(time.RFC3339)
	}
	if !r.EndedAt.IsZero() {
		pb.EndedAt = r.EndedAt.UTC().Format(time.RFC3339)
	}
	return pb
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
		Timeslot:  uint32(g.Timeslot),
		Encrypted: g.Encrypted, Emergency: g.Emergency, DataCall: g.DataCall,
		AlgorithmId: uint32(g.AlgorithmID), KeyId: uint32(g.KeyID),
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
