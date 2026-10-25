package metrics

import (
	"sort"

	"github.com/prometheus/client_golang/prometheus"
)

// Pull collectors: metrics whose values live in another subsystem's own
// counters, read at scrape time instead of being pushed on every change.
// Issue #1288 §13 asked for conventional-channel and external-delivery
// visibility; both already keep the numbers (the scanners' channel state,
// the broadcast manager's per-backend counters), so the collectors only
// read them. Label sets stay bounded: kind is "scan" / "wideband", backend
// is one per configured feed.

// ConventionalChannelCounts is one kind of conventional channel: "scan"
// (the sequential scan list) or "wideband" (continuous channels on a
// role: wideband dongle).
type ConventionalChannelCounts struct {
	Kind       string
	Configured int
	// Active counts channels whose squelch (and tone / NAC / call gate) is
	// open right now.
	Active int
}

// DeliveryCounts is a snapshot of the outbound call-delivery counters:
// cumulative sent / failed uploads per backend, calls dropped because the
// delivery queue was full, and the queue's current depth.
type DeliveryCounts struct {
	Sent    map[string]int
	Failed  map[string]int
	Dropped int
	Queued  int
}

// funcCollector is a prometheus.Collector whose Collect is a function.
type funcCollector struct {
	descs   []*prometheus.Desc
	collect func(chan<- prometheus.Metric)
}

func (c funcCollector) Describe(ch chan<- *prometheus.Desc) {
	for _, d := range c.descs {
		ch <- d
	}
}

func (c funcCollector) Collect(ch chan<- prometheus.Metric) { c.collect(ch) }

// RegisterConventionalChannels exposes, from fn at every scrape,
// gophertrunk_conventional_channels_configured and
// gophertrunk_conventional_channels_active by kind. Calls on those channels
// are already counted by calls_started_total / calls_total / calls_active
// under their protocol (fm-conv, am-conv, p25, …). Call it once.
func (m *Metrics) RegisterConventionalChannels(fn func() []ConventionalChannelCounts) error {
	configured := prometheus.NewDesc(prometheus.BuildFQName(namespace, "conventional", "channels_configured"),
		"Conventional channels configured, by kind (scan = the sequential scan list, wideband = continuous channels on a wideband dongle).",
		[]string{"kind"}, nil)
	active := prometheus.NewDesc(prometheus.BuildFQName(namespace, "conventional", "channels_active"),
		"Conventional channels whose squelch (and tone / NAC / call gate) is open now, by kind.",
		[]string{"kind"}, nil)
	return m.reg.Register(funcCollector{
		descs: []*prometheus.Desc{configured, active},
		collect: func(ch chan<- prometheus.Metric) {
			for _, c := range fn() {
				ch <- prometheus.MustNewConstMetric(configured, prometheus.GaugeValue, float64(c.Configured), c.Kind)
				ch <- prometheus.MustNewConstMetric(active, prometheus.GaugeValue, float64(c.Active), c.Kind)
			}
		},
	})
}

// RegisterDelivery exposes the outbound call-delivery counters from fn at
// every scrape: gophertrunk_delivery_sent_total and
// gophertrunk_delivery_failures_total by backend (a failure is a call the
// backend gave up on after its retries), gophertrunk_delivery_dropped_total
// (calls never queued because the queue was full) and
// gophertrunk_delivery_queue_depth. Delivery runs off the SDR path on a
// bounded queue, so these are where a slow or dead consumer shows up.
// Call it once.
func (m *Metrics) RegisterDelivery(fn func() DeliveryCounts) error {
	sent := prometheus.NewDesc(prometheus.BuildFQName(namespace, "delivery", "sent_total"),
		"Items delivered to an outbound feed, by backend: completed calls, or grants for a grant_webhook:<name> backend.", []string{"backend"}, nil)
	failed := prometheus.NewDesc(prometheus.BuildFQName(namespace, "delivery", "failures_total"),
		"Items an outbound feed gave up on after its retries, by backend.", []string{"backend"}, nil)
	dropped := prometheus.NewDesc(prometheus.BuildFQName(namespace, "delivery", "dropped_total"),
		"Items dropped before delivery because a delivery queue was full.", nil, nil)
	queued := prometheus.NewDesc(prometheus.BuildFQName(namespace, "delivery", "queue_depth"),
		"Items waiting in the delivery queues.", nil, nil)
	return m.reg.Register(funcCollector{
		descs: []*prometheus.Desc{sent, failed, dropped, queued},
		collect: func(ch chan<- prometheus.Metric) {
			d := fn()
			backends := map[string]bool{}
			for b := range d.Sent {
				backends[b] = true
			}
			for b := range d.Failed {
				backends[b] = true
			}
			names := make([]string, 0, len(backends))
			for b := range backends {
				names = append(names, b)
			}
			sort.Strings(names)
			for _, b := range names {
				ch <- prometheus.MustNewConstMetric(sent, prometheus.CounterValue, float64(d.Sent[b]), b)
				ch <- prometheus.MustNewConstMetric(failed, prometheus.CounterValue, float64(d.Failed[b]), b)
			}
			ch <- prometheus.MustNewConstMetric(dropped, prometheus.CounterValue, float64(d.Dropped))
			ch <- prometheus.MustNewConstMetric(queued, prometheus.GaugeValue, float64(d.Queued))
		},
	})
}
