// Package metrics exposes a Prometheus collector for GopherTrunk.
//
// The `Metrics` type owns a private prometheus.Registry, registers a set
// of counters / gauges, and runs a goroutine that subscribes to the
// internal events bus and increments counters as engine events flow by.
// Subsystems (SDR pool, protocol decoders, recorder) push their own
// metrics through the public Record* methods.
//
// The handler is exposed via Handler() so cmd/gophertrunk can mount it
// at /metrics on the API server.
package metrics

import (
	"context"
	"errors"
	"net/http"
	"sync"

	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/sdr"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const namespace = "gophertrunk"

// Snapshotter is the subset of sdr.Pool the snapshot collector needs.
// Declared here (rather than imported from sdr) so callers can pass a
// nil Pool or a fake without dragging in the full sdr.Pool surface.
type Snapshotter interface {
	Snapshot() []sdr.SDRStatus
}

// Metrics owns the Prometheus registry and counters/gauges used by the
// daemon.
type Metrics struct {
	reg *prometheus.Registry

	eventsTotal   *prometheus.CounterVec
	callsTotal    *prometheus.CounterVec // by system,protocol,encrypted,reason
	callsStarted  *prometheus.CounterVec // by system,protocol,encrypted
	activeCalls   *prometheus.GaugeVec   // by system,protocol
	ccLockedGauge *prometheus.GaugeVec   // by system (1 when CC locked)
	ccFrequencyHz *prometheus.GaugeVec   // by system; deleted on CC loss
	ccTransitions *prometheus.CounterVec // by system,event (locked|lost)
	iqUnderruns   *prometheus.CounterVec
	usbReconnects *prometheus.CounterVec
	decodeErrors  *prometheus.CounterVec
	sdrAttached   *prometheus.GaugeVec
	versionInfo   *prometheus.GaugeVec
	sdrSnap       *sdrSnapshotCollector

	bus       *events.Bus
	sub       *events.Subscription
	runDone   chan struct{}
	closeOnce sync.Once
}

// New constructs the metrics registry and (optionally) subscribes to the
// supplied events bus. Pass nil for the bus to use the metrics package
// purely for the manual Record* methods. Pass nil for pool to skip the
// SDR snapshot collector.
func New(bus *events.Bus, pool Snapshotter, version string) (*Metrics, error) {
	reg := prometheus.NewRegistry()
	m := &Metrics{
		reg:     reg,
		bus:     bus,
		runDone: make(chan struct{}),
	}

	m.eventsTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Name:      "events_total",
		Help:      "Total events observed on the internal events bus, by kind.",
	}, []string{"kind"})

	m.callsTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Name:      "calls_total",
		Help:      "Total calls completed, by system, protocol, encryption state, and end reason.",
	}, []string{"system", "protocol", "encrypted", "reason"})

	m.callsStarted = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Name:      "calls_started_total",
		Help:      "Total calls started. More reliable as a rate signal than calls_total because CallEnd can be missed when the daemon dies mid-call.",
	}, []string{"system", "protocol", "encrypted"})

	m.activeCalls = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: namespace,
		Name:      "calls_active",
		Help:      "Active calls currently being followed, by system and protocol. Use sum() for the daemon-wide total.",
	}, []string{"system", "protocol"})

	m.ccLockedGauge = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: namespace,
		Name:      "control_channel_locked",
		Help:      "1 when GopherTrunk is locked to a control channel for the named system, 0 otherwise.",
	}, []string{"system"})

	m.ccFrequencyHz = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: namespace,
		Name:      "control_channel_frequency_hz",
		Help:      "Currently-locked control-channel frequency for the named system, in Hz. Series is deleted when the CC is lost.",
	}, []string{"system"})

	m.ccTransitions = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Name:      "control_channel_transitions_total",
		Help:      "Control-channel lock/lost transitions per system, useful for spotting churn under poor SNR.",
	}, []string{"system", "event"})

	m.iqUnderruns = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Subsystem: "sdr",
		Name:      "iq_underruns_total",
		Help:      "Times the IQ stream pipeline dropped samples because a downstream stage was too slow.",
	}, []string{"driver", "serial"})

	m.usbReconnects = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Subsystem: "sdr",
		Name:      "usb_reconnects_total",
		Help:      "Times the SDR USB driver had to re-open a device after a transient error.",
	}, []string{"driver", "serial"})

	// Stage names for decodeErrors are an open taxonomy — see
	// events.DecodeError for the canonical list. Protocol packages either
	// call RecordDecodeError directly or publish events.KindDecodeError on
	// the bus; both paths land in this counter.
	m.decodeErrors = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Subsystem: "decode",
		Name:      "errors_total",
		Help:      "Decode failures by protocol and stage (e.g. p25/nid-bch, dmr/slottype-hamming).",
	}, []string{"protocol", "stage"})

	m.sdrAttached = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: namespace,
		Subsystem: "sdr",
		Name:      "attached",
		Help:      "1 for each currently-attached SDR device, by serial.",
	}, []string{"driver", "serial", "role"})

	m.versionInfo = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: namespace,
		Name:      "build_info",
		Help:      "Always 1; carries the build version as a label.",
	}, []string{"version"})

	collectors := []prometheus.Collector{
		m.eventsTotal,
		m.callsTotal,
		m.callsStarted,
		m.activeCalls,
		m.ccLockedGauge,
		m.ccFrequencyHz,
		m.ccTransitions,
		m.iqUnderruns,
		m.usbReconnects,
		m.decodeErrors,
		m.sdrAttached,
		m.versionInfo,
	}
	if pool != nil {
		m.sdrSnap = newSDRSnapshotCollector(pool)
		collectors = append(collectors, m.sdrSnap)
	}
	for _, c := range collectors {
		if err := reg.Register(c); err != nil {
			return nil, err
		}
	}
	if version == "" {
		version = "dev"
	}
	m.versionInfo.WithLabelValues(version).Set(1)

	if bus != nil {
		m.sub = bus.Subscribe()
	}
	return m, nil
}

// Handler returns an http.Handler that serves the registered metrics.
// Mount this at /metrics on the API server.
func (m *Metrics) Handler() http.Handler {
	return promhttp.HandlerFor(m.reg, promhttp.HandlerOpts{Registry: m.reg})
}

// Registry returns the underlying registry. Tests use it to scrape
// individual counters.
func (m *Metrics) Registry() *prometheus.Registry { return m.reg }

// Run consumes events from the subscription until ctx cancels. Returns
// nil when the subscription closed before ctx, ctx.Err() otherwise.
// Safe to call without a bus configured (returns immediately).
func (m *Metrics) Run(ctx context.Context) error {
	if m.sub == nil {
		return nil
	}
	defer close(m.runDone)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case ev, ok := <-m.sub.C:
			if !ok {
				return nil
			}
			m.observeEvent(ev)
		}
	}
}

// Close releases the subscription and waits for Run to drain. Idempotent.
func (m *Metrics) Close() error {
	if m.sub == nil {
		return nil
	}
	m.closeOnce.Do(func() {
		m.sub.Close()
		// Best-effort wait for Run to finish; bounded so a never-started
		// Run doesn't block forever.
		<-m.runDone
	})
	return nil
}

func (m *Metrics) observeEvent(ev events.Event) {
	m.eventsTotal.WithLabelValues(string(ev.Kind)).Inc()
	switch ev.Kind {
	case events.KindCallStart:
		if cs, ok := ev.Payload.(trunking.CallStart); ok {
			sys, proto, enc := callLabels(cs.Grant)
			m.activeCalls.WithLabelValues(sys, proto).Inc()
			m.callsStarted.WithLabelValues(sys, proto, enc).Inc()
		}
	case events.KindCallEnd:
		if ce, ok := ev.Payload.(trunking.CallEnd); ok {
			sys, proto, enc := callLabels(ce.Grant)
			m.activeCalls.WithLabelValues(sys, proto).Dec()
			m.callsTotal.WithLabelValues(sys, proto, enc, ce.Reason.String()).Inc()
		}
	case events.KindCCLocked:
		// Best-effort system-name extraction; both phase1.LockState and
		// the DMR / NXDN LockStates have FrequencyHz but not all carry
		// the system name. We default to "unknown" so the gauge always
		// has at least one label set.
		sys := systemLabel(ev)
		m.ccLockedGauge.WithLabelValues(sys).Set(1)
		m.ccTransitions.WithLabelValues(sys, "locked").Inc()
		if lp, ok := ev.Payload.(trunking.LockedPayload); ok {
			if hz := lp.LockedFrequencyHz(); hz != 0 {
				m.ccFrequencyHz.WithLabelValues(sys).Set(float64(hz))
			}
		}
	case events.KindCCLost:
		sys := systemLabel(ev)
		m.ccLockedGauge.WithLabelValues(sys).Set(0)
		m.ccTransitions.WithLabelValues(sys, "lost").Inc()
		m.ccFrequencyHz.DeleteLabelValues(sys)
	case events.KindDecodeError:
		if de, ok := ev.Payload.(events.DecodeError); ok {
			m.decodeErrors.WithLabelValues(de.Protocol, string(de.Stage)).Inc()
		}
	case events.KindSDRAttached:
		if st, ok := ev.Payload.(sdr.SDRStatus); ok {
			m.SetSDRAttached(st.Driver, st.Serial, st.Role, true)
		}
	case events.KindSDRDetached:
		if st, ok := ev.Payload.(sdr.SDRStatus); ok {
			m.SetSDRAttached(st.Driver, st.Serial, st.Role, false)
		}
	}
}

// --- public Record* hooks for non-engine subsystems ---

// RecordIQUnderrun increments the underrun counter for the supplied SDR.
func (m *Metrics) RecordIQUnderrun(driver, serial string) {
	m.iqUnderruns.WithLabelValues(driver, serial).Inc()
}

// RecordUSBReconnect increments the reconnect counter for the supplied SDR.
func (m *Metrics) RecordUSBReconnect(driver, serial string) {
	m.usbReconnects.WithLabelValues(driver, serial).Inc()
}

// RecordDecodeError increments the per-protocol/stage decode-error counter.
func (m *Metrics) RecordDecodeError(protocol, stage string) {
	m.decodeErrors.WithLabelValues(protocol, stage).Inc()
}

// SetSDRAttached toggles the attached-gauge for a device.
func (m *Metrics) SetSDRAttached(driver, serial, role string, attached bool) {
	v := 0.0
	if attached {
		v = 1.0
	}
	m.sdrAttached.WithLabelValues(driver, serial, role).Set(v)
}

// systemLabel returns a stable label for a CCLocked / CCLost event. The
// internal/events package keeps the payload protocol-agnostic; we look
// at common shapes via reflection-free type assertions.
func systemLabel(ev events.Event) string {
	type withSystem interface{ SystemName() string }
	if v, ok := ev.Payload.(withSystem); ok {
		if s := v.SystemName(); s != "" {
			return s
		}
	}
	return "unknown"
}

// callLabels derives the (system, protocol, encrypted) label triple for
// the per-call CounterVecs and GaugeVec. Empty strings fall back to
// "unknown" so cardinality stays bounded.
func callLabels(g trunking.Grant) (system, protocol, encrypted string) {
	system = g.System
	if system == "" {
		system = "unknown"
	}
	protocol = g.Protocol
	if protocol == "" {
		protocol = "unknown"
	}
	if g.Encrypted {
		encrypted = "true"
	} else {
		encrypted = "false"
	}
	return
}

// ErrAlreadyRegistered is returned by New when the supplied registry
// already has a collector with the same descriptor (used in tests).
var ErrAlreadyRegistered = errors.New("metrics: collector already registered")
