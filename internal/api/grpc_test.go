package api

import (
	"context"
	"net"
	"testing"
	"time"

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

func TestGRPCAudioStreamUnimplemented(t *testing.T) {
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
	if status.Code(err) != codes.Unimplemented {
		t.Errorf("expected Unimplemented, got %v", err)
	}
}
