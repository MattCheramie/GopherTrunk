package trunking

import (
	"fmt"
	"io"
	"math"
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
// StatusFlags is the human-readable CFVA summary from the adjacent-status
// broadcast ("" when never observed, "none" when observed all-clear).
type ReportNeighbor struct {
	RFSS        uint8
	Site        uint8
	LRA         uint8
	Channel     ReportChannel
	StatusFlags string
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
		out := channelLine(line, n.Channel)
		if n.StatusFlags != "" {
			out += " STATUS:[" + strings.ToUpper(strings.ReplaceAll(n.StatusFlags, ",", " ")) + "]"
		}
		fmt.Fprintf(w, "  %s\n", out)
	}
}

// RenderNeighborLines returns one line per adjacent site in a topology snapshot,
// each like "RFSS:01[1] SITE:07[7] CHANNEL:2-1754 DOWNLINK:450.962500 MHz
// UPLINK:461.437500 MHz", sorted by (RFSS, Site). It reuses the same band-plan
// resolution and channel formatting as the full network report. The slice is
// empty when the snapshot is nil or carries no neighbours. The decoded-message
// log uses it to surface adjacent sites the way SDRtrunk's "Neighbor Sites"
// block does, without re-deriving uplink frequencies.
func RenderNeighborLines(t *TopologySnapshot) []string {
	if t == nil {
		return nil
	}
	r := ReportFromTopology(t)
	if len(r.Sites) == 0 {
		return nil
	}
	neighbors := append([]ReportNeighbor(nil), r.Sites[0].Neighbors...)
	sort.SliceStable(neighbors, func(i, j int) bool {
		if neighbors[i].RFSS != neighbors[j].RFSS {
			return neighbors[i].RFSS < neighbors[j].RFSS
		}
		return neighbors[i].Site < neighbors[j].Site
	})
	lines := make([]string, 0, len(neighbors))
	for _, n := range neighbors {
		prefix := fmt.Sprintf("RFSS:%s SITE:%s CHANNEL", hexDec(uint64(n.RFSS)), hexDec(uint64(n.Site)))
		lines = append(lines, channelLine(prefix, n.Channel))
	}
	return lines
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

// IDHex renders a P25 identity number (WACN, System ID, NAC, RFSS, Site) as an
// uppercase hex string with no "0x" prefix — the convention these are quoted in
// the field, e.g. System ID 706 as "2C2". Returns "" for zero so it drops out
// of omitempty JSON fields (an unknown/absent identity carries no hex).
func IDHex(v uint64) string {
	if v == 0 {
		return ""
	}
	return fmt.Sprintf("%X", v)
}

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
	bands := make(map[uint8]TopoBandPlanSlot, len(t.BandPlan))
	for _, b := range t.BandPlan {
		offsets[b.ChannelID] = b.TxOffsetHz
		bands[b.ChannelID] = b
	}
	conv := func(id uint8, num uint16, downHz, upHz uint32) ReportChannel {
		// Resolve the downlink from the band plan when the protocol didn't carry
		// an explicit frequency. A P25 neighbour (ADJ_STS_BCST) is reported as a
		// (channel id, channel number) pair, not a frequency, so its downlink is
		// base + number*spacing from the matching IDEN_UP — the same computation
		// voice-grant channels use (identifier.BandPlan.Frequency). Without this a
		// neighbour whose band arrived after it stays at 0 Hz forever, which is
		// what left the web "Neighbor sites" downlink blank (SDRtrunk shows it).
		if downHz == 0 && num != 0 {
			if b, ok := bands[id]; ok && b.BaseHz != 0 && b.SpacingHz != 0 {
				if hz := b.BaseHz + uint64(num)*uint64(b.SpacingHz); hz > 0 && hz <= math.MaxUint32 {
					downHz = uint32(hz)
				}
			}
		}
		c := ReportChannel{ChannelID: id, ChannelNumber: num, DownlinkHz: downHz}
		switch {
		case upHz != 0:
			// The protocol resolved the uplink directly (TETRA duplex spacing).
			c.UplinkHz = upHz
		case downHz != 0:
			// Otherwise derive it from the band-plan transmit offset (P25).
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
		site.PrimaryCC = conv(t.PrimaryCC.ChannelID, t.PrimaryCC.ChannelNumber, t.PrimaryCC.FrequencyHz, t.PrimaryCC.UplinkHz)
	}
	for _, s := range t.Secondary {
		site.SecondaryCC = append(site.SecondaryCC, conv(s.ChannelID, s.ChannelNumber, s.FrequencyHz, s.UplinkHz))
	}
	for _, n := range t.Neighbors {
		// An explicit uplink (the P25 AMBT adjacent-status form) wins; a
		// pair-only explicit uplink resolves via plain base + number*spacing
		// (no transmit offset — the uplink channel number already encodes the
		// uplink frequency). Otherwise conv derives it from the band-plan
		// transmit offset as before.
		upHz := n.UplinkHz
		if upHz == 0 && n.UplinkChannelNumber != 0 {
			if b, ok := bands[n.UplinkChannelID]; ok && b.BaseHz != 0 && b.SpacingHz != 0 {
				if hz := b.BaseHz + uint64(n.UplinkChannelNumber)*uint64(b.SpacingHz); hz > 0 && hz <= math.MaxUint32 {
					upHz = uint32(hz)
				}
			}
		}
		site.Neighbors = append(site.Neighbors, ReportNeighbor{
			RFSS:        n.RFSS,
			Site:        n.Site,
			LRA:         n.LRA,
			Channel:     conv(n.ChannelID, n.ChannelNumber, n.FrequencyHz, upHz),
			StatusFlags: n.StatusFlags,
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
