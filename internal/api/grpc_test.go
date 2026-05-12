package api

import (
	"context"
	"log/slog"
	"net"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/events"

	apiv1 "github.com/MattCheramie/GopherTrunk/internal/api/pb/v1"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
)

// mkGRPC wires a GRPCServer over an in-memory bufconn listener and
// returns a connected client.
func mkGRPC(t *testing.T, opts GRPCServerOptions) (*grpc.ClientConn, func()) {
	t.Helper()
	lis := bufconn.Listen(64 * 1024)
	g, err := NewGRPCServer(GRPCServerOptions{
		Addr:       "bufconn",
		Systems:    opts.Systems,
		Talkgroups: opts.Talkgroups,
		Engine:     opts.Engine,
		Audio:      opts.Audio,
		Log:        opts.Log,
	})
	if err != nil {
		t.Fatal(err)
	}
	go g.srv.Serve(lis)
	dial := func(context.Context, string) (net.Conn, error) { return lis.Dial() }
	conn, err := grpc.NewClient("passthrough://bufconn",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithContextDialer(dial),
	)
	if err != nil {
		t.Fatal(err)
	}
	return conn, func() {
		conn.Close()
		g.Stop()
		lis.Close()
	}
}

func TestGRPCListSystems(t *testing.T) {
	systems := []trunking.System{
		{Name: "Alpha", Protocol: trunking.ProtocolP25, ControlChannels: []uint32{851_000_000}},
		{Name: "Bravo", Protocol: trunking.ProtocolDMR, ControlChannels: []uint32{460_000_000}},
	}
	conn, teardown := mkGRPC(t, GRPCServerOptions{Systems: systems})
	defer teardown()

	cli := apiv1.NewSystemServiceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	resp, err := cli.ListSystems(ctx, &apiv1.ListSystemsRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.Systems) != 2 {
		t.Fatalf("len = %d, want 2", len(resp.Systems))
	}

	got, err := cli.GetSystem(ctx, &apiv1.GetSystemRequest{Name: "Bravo"})
	if err != nil {
		t.Fatal(err)
	}
	if got.System.Name != "Bravo" {
		t.Errorf("got %q", got.System.Name)
	}

	_, err = cli.GetSystem(ctx, &apiv1.GetSystemRequest{Name: "missing"})
	if status.Code(err) != codes.NotFound {
		t.Errorf("missing got %v, want NotFound", err)
	}
}

func TestGRPCListAndGetTalkgroup(t *testing.T) {
	db := trunking.NewTalkgroupDB()
	db.Add(&trunking.TalkGroup{ID: 100, AlphaTag: "OPS-1", Priority: 1})
	conn, teardown := mkGRPC(t, GRPCServerOptions{Talkgroups: db})
	defer teardown()

	cli := apiv1.NewTalkgroupServiceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	resp, err := cli.GetTalkgroup(ctx, &apiv1.GetTalkgroupRequest{Id: 100})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Talkgroup.AlphaTag != "OPS-1" {
		t.Errorf("got %q", resp.Talkgroup.AlphaTag)
	}

	_, err = cli.GetTalkgroup(ctx, &apiv1.GetTalkgroupRequest{Id: 9999})
	if status.Code(err) != codes.NotFound {
		t.Errorf("missing got %v, want NotFound", err)
	}
}

func TestGRPCActiveCalls(t *testing.T) {
	dev := &trunking.VoiceDevice{Serial: "VOICE-1"}
	engine := &fakeEngine{
		calls: []*trunking.ActiveCall{{
			Device:    dev,
			Grant:     trunking.Grant{System: "Alpha", Protocol: "p25", GroupID: 1234, FrequencyHz: 851_000_000},
			Talkgroup: &trunking.TalkGroup{ID: 1234, AlphaTag: "FIRE-DISP"},
			StartedAt: time.Now().UTC(),
		}},
	}
	conn, teardown := mkGRPC(t, GRPCServerOptions{Engine: engine})
	defer teardown()

	cli := apiv1.NewTalkgroupServiceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	resp, err := cli.ListActiveCalls(ctx, &apiv1.ListActiveCallsRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.Calls) != 1 || resp.Calls[0].Talkgroup.AlphaTag != "FIRE-DISP" {
		t.Errorf("calls = %+v", resp.Calls)
	}
}

// TestGRPCAudioStreamUnavailable verifies the server reports
// Unavailable when no AudioPublisher is wired (degraded daemon path —
// composer disabled, audio off, etc.). Older code returned
// Unimplemented; we now ship a real publisher when one is supplied,
// so an absent publisher means "audio temporarily unavailable"
// rather than "RPC not implemented".
func TestGRPCAudioStreamUnavailable(t *testing.T) {
	conn, teardown := mkGRPC(t, GRPCServerOptions{})
	defer teardown()

	cli := apiv1.NewAudioServiceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	stream, err := cli.StreamAudio(ctx, &apiv1.StreamAudioRequest{})
	if err != nil {
		t.Fatal(err)
	}
	_, err = stream.Recv()
	if status.Code(err) != codes.Unavailable {
		t.Errorf("expected Unavailable, got %v", err)
	}
}

// TestGRPCAudioStreamPublishes verifies the end-to-end gRPC streaming
// path: construct a server with an AudioPublisher, publish a
// CallStart on the bus, drive WritePCM, and confirm the client
// reads an AudioFrame off the stream.
func TestGRPCAudioStreamPublishes(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	pub, err := NewAudioPublisher(bus, slog.Default())
	if err != nil {
		t.Fatal(err)
	}
	pubCtx, pubCancel := context.WithCancel(context.Background())
	pubDone := make(chan struct{})
	go func() { _ = pub.Run(pubCtx); close(pubDone) }()
	defer func() {
		pubCancel()
		<-pubDone
		_ = pub.Close()
	}()

	conn, teardown := mkGRPC(t, GRPCServerOptions{Audio: pub})
	defer teardown()

	cli := apiv1.NewAudioServiceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	stream, err := cli.StreamAudio(ctx, &apiv1.StreamAudioRequest{})
	if err != nil {
		t.Fatal(err)
	}

	// Publish a CallStart so the publisher knows which Grant
	// VOICE-1 belongs to.
	bus.Publish(events.Event{Kind: events.KindCallStart, Payload: trunking.CallStart{
		Grant:        trunking.Grant{System: "Alpha", GroupID: 42},
		DeviceSerial: "VOICE-1",
	}})

	// Spin until the publisher reports a tracked grant + at least
	// one subscriber (the gRPC stream registers asynchronously
	// after the client call).
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		st := pub.Stats()
		if st.TrackedGrants >= 1 && st.Subscribers >= 1 {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}

	_ = pub.WritePCM("VOICE-1", []int16{1, -2, 3, -4})

	frame, err := stream.Recv()
	if err != nil {
		t.Fatalf("stream.Recv: %v", err)
	}
	if frame.GetGrant().GroupId != 42 {
		t.Errorf("frame.grant.group_id = %d, want 42", frame.GetGrant().GroupId)
	}
	if frame.DeviceSerial != "VOICE-1" {
		t.Errorf("frame.device_serial = %q, want VOICE-1", frame.DeviceSerial)
	}
	pcm := frame.GetPcm()
	if pcm == nil || len(pcm.Samples) != 8 {
		t.Errorf("PCM body missing or wrong length: %+v", pcm)
	}
}
