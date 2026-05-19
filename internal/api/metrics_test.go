package api

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/metrics"
)

func TestMetricsEndpointExposesPromText(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	m, err := metrics.New(nil, nil, "v9.9.9-test")
	if err != nil {
		t.Fatal(err)
	}
	defer m.Close()

	base, teardown := mkServer(t, ServerOptions{
		Bus:            bus,
		MetricsHandler: m.Handler(),
	})
	defer teardown()

	resp, err := http.Get(base + "/metrics")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), `gophertrunk_build_info{version="v9.9.9-test"}`) {
		t.Errorf("metrics scrape missing build_info; body:\n%s", body)
	}
}

func TestMetricsEndpoint404WithoutHandler(t *testing.T) {
	bus := events.NewBus(4)
	defer bus.Close()
	base, teardown := mkServer(t, ServerOptions{Bus: bus})
	defer teardown()

	resp, err := http.Get(base + "/metrics")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 404 {
		t.Errorf("status = %d, want 404 (no metrics handler configured)", resp.StatusCode)
	}
}
