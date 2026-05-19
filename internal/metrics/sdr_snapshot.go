package metrics

import (
	"math"

	"github.com/prometheus/client_golang/prometheus"
)

// sdrSnapshotCollector implements prometheus.Collector by snapshotting
// the SDR pool at scrape time. It exposes the per-device tuning
// parameters (gain, AGC, PPM, bias-tee) that the event-driven counters
// in prom.go don't cover. Pull-mode keeps state out of the metrics
// struct and ensures the values can't go stale relative to the pool.
type sdrSnapshotCollector struct {
	pool Snapshotter

	gainDB   *prometheus.Desc
	gainAuto *prometheus.Desc
	ppm      *prometheus.Desc
	biasTee  *prometheus.Desc
}

func newSDRSnapshotCollector(p Snapshotter) *sdrSnapshotCollector {
	labels := []string{"driver", "serial", "role"}
	return &sdrSnapshotCollector{
		pool: p,
		gainDB: prometheus.NewDesc(
			namespace+"_sdr_gain_db",
			"Configured gain in dB for the SDR. NaN when the tuner is running AGC; pair with gophertrunk_sdr_gain_auto to filter.",
			labels, nil,
		),
		gainAuto: prometheus.NewDesc(
			namespace+"_sdr_gain_auto",
			"1 when the tuner is running automatic gain control, 0 otherwise.",
			labels, nil,
		),
		ppm: prometheus.NewDesc(
			namespace+"_sdr_ppm",
			"Configured frequency-error correction in parts-per-million.",
			labels, nil,
		),
		biasTee: prometheus.NewDesc(
			namespace+"_sdr_bias_tee",
			"1 when the SDR's 5 V bias-tee output is enabled, 0 otherwise.",
			labels, nil,
		),
	}
}

func (c *sdrSnapshotCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.gainDB
	ch <- c.gainAuto
	ch <- c.ppm
	ch <- c.biasTee
}

func (c *sdrSnapshotCollector) Collect(ch chan<- prometheus.Metric) {
	for _, s := range c.pool.Snapshot() {
		if !s.Attached {
			continue
		}
		gain := math.NaN()
		if !s.GainAuto {
			gain = float64(s.GainTenthDB) / 10.0
		}
		ch <- prometheus.MustNewConstMetric(c.gainDB, prometheus.GaugeValue, gain, s.Driver, s.Serial, s.Role)
		ch <- prometheus.MustNewConstMetric(c.gainAuto, prometheus.GaugeValue, boolToFloat(s.GainAuto), s.Driver, s.Serial, s.Role)
		ch <- prometheus.MustNewConstMetric(c.ppm, prometheus.GaugeValue, float64(s.PPM), s.Driver, s.Serial, s.Role)
		ch <- prometheus.MustNewConstMetric(c.biasTee, prometheus.GaugeValue, boolToFloat(s.BiasTee), s.Driver, s.Serial, s.Role)
	}
}

func boolToFloat(b bool) float64 {
	if b {
		return 1
	}
	return 0
}
