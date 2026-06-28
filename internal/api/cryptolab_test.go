//go:build cryptolab

package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// TestCryptolabMountedOnDaemon verifies that, in a build tagged `cryptolab`,
// the daemon mux mounts the cryptolab research console: the tool-schema route
// answers and the closer it owns is registered for shutdown cleanup. The
// default (untagged) build links the no-op mountCryptolab instead, so this
// test only exists behind the tag.
func TestCryptolabMountedOnDaemon(t *testing.T) {
	bus := events.NewBus(4)
	defer bus.Close()
	s := &Server{
		bus:        bus,
		systems:    []trunking.System{{Name: "X", Protocol: trunking.ProtocolP25, ControlChannels: []uint32{1_000_000}}},
		talkgroups: trunking.NewTalkgroupDB(),
		log:        nopLogger(),
		version:    "test",
	}
	mux := s.routes()
	t.Cleanup(func() {
		if s.cryptolabCloser != nil {
			_ = s.cryptolabCloser.Close()
		}
	})
	if s.cryptolabCloser == nil {
		t.Fatal("cryptolab subsystem not registered for shutdown cleanup")
	}

	srv := httptest.NewServer(mux)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/api/v1/cryptolab/tools")
	if err != nil {
		t.Fatalf("GET cryptolab tools: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("cryptolab tools status = %d, want 200", resp.StatusCode)
	}
}
