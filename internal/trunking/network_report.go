package trunking

import (
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"
)

// NetworkReport is the display-ready, protocol-neutral input to
// RenderNetworkReport — GopherTrunk's equivalent of SDRtrunk's P25 network
// configuration / activity summary. It is deliberately separate from
// TopologySnapshot (a wire/JSON type with an Empty() attach contract): adapters
// resolve every channel to absolute frequencies and flatten the one-site
// (CLI/daemon) and multi-site (hunt) shapes into the same Sites slice so the
// renderer stays pure and free of band-plan math.
type NetworkReport struct {
	Name     string // system name/label ("" → "(unnamed)")
	Protocol string // "p25" etc.; uppercased in the header

	// Network identity.
	WACN     uint32
	SystemID uint32
	NAC      uint16
	LRA      uint8

	Sites []ReportSite // 1 for a single control channel; N for an aggregated hunt
	Bands []ReportBand // frequency bands (band plan), system-wide
}

// ReportSite is one site's identity and control-channel topology.
type ReportSite struct {
	RFSS        uint8
	Site        uint8
	LRA         uint8
	PrimaryCC   ReportChannel
	SecondaryCC []ReportChannel
	Neighbors   []ReportNeighbor
}

// ReportChannel is a control/voice channel with band-plan coordinates and
// resolved frequencies. Coordinates may be zero (a hunt-discovered channel
// carries only a resolved frequency); frequencies may be zero (no band plan).
type ReportChannel struct {
	ChannelID     uint8
	ChannelNumber uint16
	DownlinkHz    uint32
	UplinkHz      uint32
}

func (c ReportChannel) empty() bool { return c == ReportChannel{} }

// ReportNeighbor is an adjacent site advertised by a control channel.
type ReportNeighbor struct {
	RFSS    uint8
	Site    uint8
	Channel ReportChannel
}

// ReportBand is one P25 IDEN_UP band-plan slot.
type ReportBand struct {
	ChannelID   uint8
	BaseHz      uint64
	SpacingHz   uint32
	BandwidthHz uint32
	TxOffsetHz  int64
	AccessTDMA  bool
}

// FormatNetworkReport renders r to a string.
func FormatNetworkReport(r NetworkReport) string {
	var b strings.Builder
	RenderNetworkReport(&b, r)
	return b.String()
}

// RenderNetworkReport writes the GopherTrunk-native network-configuration report
// to w. Empty sections are suppressed; sites, neighbours and bands are emitted
// in a stable order so the output is golden-test friendly.
func RenderNetworkReport(w io.Writer, r NetworkReport) {
	proto := strings.ToUpper(strings.TrimSpace(r.Protocol))
	if proto == "" {
		proto = "P25"
	}
	name := r.Name
	if name == "" {
		name = "(unnamed)"
	}
	fmt.Fprintf(w, "%s Network Configuration — %s\n", proto, name)

	fmt.Fprintln(w, "Network")
	fmt.Fprintf(w, "  WACN:%s SYSTEM:%s NAC:%s LRA:%s\n",
		hexDec(uint64(r.WACN)), hexDec(uint64(r.SystemID)), hexDec(uint64(r.NAC)), hexDec(uint64(r.LRA)))

	sites := append([]ReportSite(nil), r.Sites...)
	sort.SliceStable(sites, func(i, j int) bool {
		if sites[i].RFSS != sites[j].RFSS {
			return sites[i].RFSS < sites[j].RFSS
		}
		return sites[i].Site < sites[j].Site
	})

	header := "Current Site"
	if len(sites) > 1 {
		header = "Sites"
	}
	for i := range sites {
		if i == 0 || len(sites) > 1 {
			fmt.Fprintln(w, header)
		}
		renderSite(w, sites[i])
	}

	if len(r.Bands) > 0 {
		bands := append([]ReportBand(nil), r.Bands...)
		sort.SliceStable(bands, func(i, j int) bool { return bands[i].ChannelID < bands[j].ChannelID })
		fmt.Fprintln(w, "Frequency Bands")
		for _, b := range bands {
			mode := "FDMA"
			if b.AccessTDMA {
				mode = "TDMA"
			}
			fmt.Fprintf(w, "  BAND:%d %s BASE:%s BANDWIDTH:%s SPACING:%s OFFSET:%s\n",
				b.ChannelID, mode, mhz(b.BaseHz), khz(uint64(b.BandwidthHz)), khz(uint64(b.SpacingHz)), offsetMHz(b.TxOffsetHz))
		}
	}
}

func renderSite(w io.Writer, s ReportSite) {
	fmt.Fprintf(w, "  RFSS:%s SITE:%s LRA:%s\n", hexDec(uint64(s.RFSS)), hexDec(uint64(s.Site)), hexDec(uint64(s.LRA)))
	if !s.PrimaryCC.empty() {
		fmt.Fprintf(w, "  %s\n", channelLine("PRI CONTROL CHANNEL", s.PrimaryCC))
	}
	for _, c := range s.SecondaryCC {
		fmt.Fprintf(w, "  %s\n", channelLine("SEC CONTROL CHANNEL", c))
	}
	neighbors := append([]ReportNeighbor(nil), s.Neighbors...)
	sort.SliceStable(neighbors, func(i, j int) bool {
		if neighbors[i].RFSS != neighbors[j].RFSS {
			return neighbors[i].RFSS < neighbors[j].RFSS
		}
		return neighbors[i].Site < neighbors[j].Site
	})
	for _, n := range neighbors {
		line := fmt.Sprintf("NEIGHBOR RFSS:%s SITE:%s CHANNEL", hexDec(uint64(n.RFSS)), hexDec(uint64(n.Site)))
		fmt.Fprintf(w, "  %s\n", channelLine(line, n.Channel))
	}
}

// channelLine renders "<prefix>[:<id>-<num>] DOWNLINK:<mhz> UPLINK:<mhz>",
// omitting the coordinate token when unknown (hunt) and each frequency when
// unresolved.
func channelLine(prefix string, c ReportChannel) string {
	var b strings.Builder
	b.WriteString(prefix)
	if c.ChannelID != 0 || c.ChannelNumber != 0 {
		fmt.Fprintf(&b, ":%d-%d", c.ChannelID, c.ChannelNumber)
	}
	if c.DownlinkHz != 0 {
		fmt.Fprintf(&b, " DOWNLINK:%s", mhz(uint64(c.DownlinkHz)))
	}
	if c.UplinkHz != 0 {
		fmt.Fprintf(&b, " UPLINK:%s", mhz(uint64(c.UplinkHz)))
	}
	return b.String()
}

// hexDec renders a value as "HEX[decimal]" — e.g. WACN BEE00 as "BEE00[781824]".
func hexDec(v uint64) string { return fmt.Sprintf("%X[%d]", v, v) }

// mhz formats a frequency in Hz as MHz with 6 decimals (1 Hz resolution).
func mhz(hz uint64) string { return fmt.Sprintf("%.6f MHz", float64(hz)/1e6) }

// khz formats a value in Hz as kHz, trimming trailing zeros (12500 → "12.5 kHz").
func khz(hz uint64) string {
	return strconv.FormatFloat(float64(hz)/1e3, 'f', -1, 64) + " kHz"
}

// offsetMHz formats a signed transmit offset in Hz as MHz, keeping the sign.
func offsetMHz(hz int64) string { return fmt.Sprintf("%+.6f MHz", float64(hz)/1e6) }

// ReportFromTopology builds a single-site NetworkReport from a TopologySnapshot
// (the CLI and live-daemon path). Uplink frequencies are derived from each
// channel's matching band-plan transmit offset.
func ReportFromTopology(t *TopologySnapshot) NetworkReport {
	if t == nil {
		return NetworkReport{}
	}
	offsets := make(map[uint8]int64, len(t.BandPlan))
	for _, b := range t.BandPlan {
		offsets[b.ChannelID] = b.TxOffsetHz
	}
	conv := func(id uint8, num uint16, downHz uint32) ReportChannel {
		c := ReportChannel{ChannelID: id, ChannelNumber: num, DownlinkHz: downHz}
		if downHz != 0 {
			if off, ok := offsets[id]; ok && off != 0 {
				if up := int64(downHz) + off; up > 0 {
					c.UplinkHz = uint32(up)
				}
			}
		}
		return c
	}

	r := NetworkReport{
		Name:     t.SystemName,
		Protocol: t.Protocol,
		WACN:     t.WACN,
		SystemID: t.SystemID,
		NAC:      t.NAC,
		LRA:      t.LRA,
	}
	site := ReportSite{RFSS: t.RFSS, Site: t.Site, LRA: t.LRA}
	if t.PrimaryCC != nil {
		site.PrimaryCC = conv(t.PrimaryCC.ChannelID, t.PrimaryCC.ChannelNumber, t.PrimaryCC.FrequencyHz)
	}
	for _, s := range t.Secondary {
		site.SecondaryCC = append(site.SecondaryCC, conv(s.ChannelID, s.ChannelNumber, s.FrequencyHz))
	}
	for _, n := range t.Neighbors {
		site.Neighbors = append(site.Neighbors, ReportNeighbor{
			RFSS:    n.RFSS,
			Site:    n.Site,
			Channel: conv(n.ChannelID, n.ChannelNumber, n.FrequencyHz),
		})
	}
	r.Sites = []ReportSite{site}
	for _, b := range t.BandPlan {
		r.Bands = append(r.Bands, ReportBand{
			ChannelID:   b.ChannelID,
			BaseHz:      b.BaseHz,
			SpacingHz:   b.SpacingHz,
			BandwidthHz: b.BandwidthHz,
			TxOffsetHz:  b.TxOffsetHz,
			AccessTDMA:  b.AccessTDMA,
		})
	}
	return r
}
