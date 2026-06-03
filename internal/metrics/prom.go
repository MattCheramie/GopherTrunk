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

	eventsTotal    *prometheus.CounterVec
	callsTotal     *prometheus.CounterVec // by system,protocol,encrypted,reason
	callsStarted   *prometheus.CounterVec // by system,protocol,encrypted
	activeCalls    *prometheus.GaugeVec   // by system,protocol
	ccLockedGauge  *prometheus.GaugeVec   // by system (1 when CC locked)
	ccFrequencyHz  *prometheus.GaugeVec   // by system; deleted on CC loss
	ccTransitions  *prometheus.CounterVec // by system,event (locked|lost)
	iqUnderruns    *prometheus.CounterVec
	decodeOverruns prometheus.Counter   // chunks dropped at the decode queue (issue #402)
	iqPowerDbFS    *prometheus.GaugeVec // by system; mean IQ power on the control SDR
	iqDCRatioDb    *prometheus.GaugeVec // by system; DC-bin power relative to total IQ power, in dB (issue #402)
	iqClipRatio    *prometheus.GaugeVec // by system; fraction of IQ samples pinned to the ADC rail (issue #402)
	usbReconnects  *prometheus.CounterVec
	decodeErrors   *prometheus.CounterVec
	sdrAttached    *prometheus.GaugeVec
	versionInfo    *prometheus.GaugeVec
	sdrSnap        *sdrSnapshotCollector

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

	// Chunks the live decoder dropped at its internal decode queue because
	// decode could not keep up with real time (issue #402). Distinct from
	// sdr_iq_underruns_total: that counts the SDR driver's own delivery
	// channel overrunning, while this counts the decode goroutine being the
	// bottleneck (CPU-starved host, contention). A climbing value here, with
	// iq_underruns_total flat, says "the machine can't sustain this decode"
	// rather than "the USB transport hiccuped".
	m.decodeOverruns = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: namespace,
		Subsystem: "ccdecoder",
		Name:      "decode_overruns_total",
		Help:      "Live IQ chunks dropped at the decode queue because decode can't keep up with real time (issue #402). Distinct from sdr_iq_underruns_total (driver-delivery overruns).",
	})

	// Mean IQ power on the control SDR, expressed in dBFS (0 dBFS = ±1.0
	// full-scale per the convertU8IQ normalization). Surfaces silent
	// failure modes that issue #275 was the first instance of: cc-hunt
	// retunes forever with no decoder progress because the dongle is
	// either delivering near-zero samples (no antenna, gain at 0, USB
	// stuck) or clipping. -45 dBFS is typical idle noise; -25 dBFS is
	// healthy signal; > -3 dBFS indicates clipping.
	m.iqPowerDbFS = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: namespace,
		Subsystem: "sdr",
		Name:      "iq_power_dbfs",
		Help:      "Mean IQ power on the control SDR in dBFS (window ~= 1 s). Idle ≈ -45; healthy signal ≈ -25. This is RMS and averages away peak clipping — watch iq_clip_ratio for the authoritative overload signal (issue #402).",
	}, []string{"system"})

	// Diagnostic for the RTL-SDR R820T2 zero-IF DC-spike-on-channel
	// failure mode (issue #402): the DC bin power of the same window
	// the iq_power_dbfs gauge is computed over, expressed relative to
	// total IQ power in dB. 0 dB means all channel power is in the
	// DC bin; -20 dB or lower is a clean off-DC signal. Watch for
	// values above -10 dB on a tuned-on-channel P25 control: that's
	// the spike sitting on top of the channel.
	m.iqDCRatioDb = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: namespace,
		Subsystem: "sdr",
		Name:      "iq_dc_ratio_db",
		Help:      "DC-bin power relative to total IQ power, in dB (window ~= 1 s). 0 = all power is DC; -20 or lower = clean. > -10 on a tuned-on-channel control SDR hints at the R820T2 DC-spike failure mode tracked in issue #402.",
	}, []string{"system"})

	// Fraction of IQ samples whose I or Q component hit the ADC rail over
	// the same window iq_power_dbfs is computed over (issue #402). The
	// power gauge is RMS, so a high-crest signal can clip on peaks while
	// the average still reads merely "hot" (~ -5 dBFS); this gauge is the
	// authoritative overload signal. 0 = no clipping; a sustained value
	// above ~0.002 means the front end is overloaded — reduce gain or add
	// attenuation (raising gain makes it worse).
	m.iqClipRatio = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: namespace,
		Subsystem: "sdr",
		Name:      "iq_clip_ratio",
		Help:      "Fraction of control-SDR IQ samples pinned to the ADC rail (window ~= 1 s). 0 = no clipping; a sustained value above ~0.002 means front-end overload — reduce gain or add attenuation, do not raise gain (issue #402).",
	}, []string{"system"})

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
		m.decodeOverruns,
		m.iqPowerDbFS,
		m.iqDCRatioDb,
		m.iqClipRatio,
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

// RecordDecodeOverrun increments the decode-overrun counter: one live IQ
// chunk dropped at the decode queue because decode can't keep up with
// real time (issue #402). Distinct from RecordIQUnderrun, which is the
// SDR driver's own delivery-channel overrun.
func (m *Metrics) RecordDecodeOverrun() {
	m.decodeOverruns.Inc()
}

// RecordIQPowerDbFS sets the mean-IQ-power gauge for system. Caller
// computes the window mean; the metrics package just stores it.
func (m *Metrics) RecordIQPowerDbFS(system string, dbfs float64) {
	if system == "" {
		system = "unknown"
	}
	m.iqPowerDbFS.WithLabelValues(system).Set(dbfs)
}

// ClearIQPowerDbFS drops the gauge series for system. Called when a
// decoder pipeline tears down so stale dBFS values don't linger after
// the SDR is no longer being driven for that system.
func (m *Metrics) ClearIQPowerDbFS(system string) {
	if system == "" {
		return
	}
	m.iqPowerDbFS.DeleteLabelValues(system)
}

// RecordIQDCRatioDb sets the DC-bin-to-total IQ power ratio gauge for
// system. The decoder computes this once per IQ-power window; the
// metrics package just stores it. See iqDCRatioDb's Help text for
// interpretation guidance (issue #402).
func (m *Metrics) RecordIQDCRatioDb(system string, ratioDb float64) {
	if system == "" {
		system = "unknown"
	}
	m.iqDCRatioDb.WithLabelValues(system).Set(ratioDb)
}

// ClearIQDCRatioDb drops the DC-ratio gauge series for system. Called
// alongside ClearIQPowerDbFS when a decoder pipeline tears down.
func (m *Metrics) ClearIQDCRatioDb(system string) {
	if system == "" {
		return
	}
	m.iqDCRatioDb.DeleteLabelValues(system)
}

// RecordIQClipRatio sets the rail-clipping-fraction gauge for system.
// The decoder computes this once per IQ-power window; the metrics package
// just stores it. See iqClipRatio's Help text for interpretation (issue
// #402).
func (m *Metrics) RecordIQClipRatio(system string, ratio float64) {
	if system == "" {
		system = "unknown"
	}
	m.iqClipRatio.WithLabelValues(system).Set(ratio)
}

// ClearIQClipRatio drops the clip-ratio gauge series for system. Called
// alongside ClearIQPowerDbFS when a decoder pipeline tears down.
func (m *Metrics) ClearIQClipRatio(system string) {
	if system == "" {
		return
	}
	m.iqClipRatio.DeleteLabelValues(system)
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
