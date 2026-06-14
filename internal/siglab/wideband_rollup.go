package siglab

import (
	"fmt"
	"sort"
	"strings"
)

// dmrMemberConfidence is the floor below which a non-locked DMR identification
// is too weak to count as a real carrier of the system. A locked carrier always
// counts regardless of confidence.
const dmrMemberConfidence = 0.30

// isDMRProtocol reports whether an identify winner is a DMR-family protocol.
func isDMRProtocol(p string) bool {
	return p == "dmr" || p == "dmr-tier2" || p == "dmr-tier1"
}

// isDMRMember reports whether a surveyed carrier is a real DMR carrier of a
// trunked system — a DMR-family winner that either locked or cleared the
// confidence floor.
func isDMRMember(c CarrierResult) bool {
	if !isDMRProtocol(c.Protocol) {
		return false
	}
	return c.Locked || (!c.Inconclusive && c.Confidence >= dmrMemberConfidence)
}

// roleForCarrier classifies a DMR carrier as control vs voice. A control
// channel locks the Tier III state machine and carries an Aloha SystemID; a
// voice carrier is any other DMR member (traffic channel — it may carry grants
// or just the BS sync + slot type of an in-progress call).
func roleForCarrier(c CarrierResult) CarrierRole {
	if !isDMRMember(c) {
		return RoleUnknown
	}
	if c.Protocol == "dmr" && c.Locked && c.SystemID != 0 {
		return RoleControl
	}
	return RoleVoice
}

// clusterSystems groups DMR members into trunked-system rollups. Carriers are
// grouped by the SystemID of their nearest control channel; when no control
// carries a SystemID they fold into a single unidentified DMR system. Each
// system yields the verdict string.
func clusterSystems(cs []CarrierResult, centerHz uint32, sampleRateHz, channelRateHz float64) []SystemRollup {
	type cluster struct {
		systemID  uint32
		colorCode uint8
		idx       []int
		control   int
		voice     int
		lo, hi    uint32
	}

	// Distinct control SystemIDs anchor the systems.
	controls := []int{}
	for i, c := range cs {
		if c.Role == RoleControl {
			controls = append(controls, i)
		}
	}

	clusters := map[uint32]*cluster{}
	keyForVoice := func(c CarrierResult) uint32 {
		// Attach a voice carrier to the control with the nearest frequency.
		if len(controls) == 0 {
			return 0
		}
		best, bestD := controls[0], absU32(c.FreqHz, cs[controls[0]].FreqHz)
		for _, ci := range controls[1:] {
			if d := absU32(c.FreqHz, cs[ci].FreqHz); d < bestD {
				best, bestD = ci, d
			}
		}
		return cs[best].SystemID
	}

	get := func(key uint32) *cluster {
		cl := clusters[key]
		if cl == nil {
			cl = &cluster{systemID: key, lo: ^uint32(0)}
			clusters[key] = cl
		}
		return cl
	}

	for i, c := range cs {
		if !isDMRMember(c) {
			continue
		}
		var key uint32
		if c.Role == RoleControl {
			key = c.SystemID
		} else {
			key = keyForVoice(c)
		}
		cl := get(key)
		cl.idx = append(cl.idx, i)
		if c.Role == RoleControl {
			cl.control++
			if cl.systemID == 0 {
				cl.systemID = c.SystemID
			}
			if cl.colorCode == 0 {
				cl.colorCode = c.ColorCode
			}
		} else {
			cl.voice++
		}
		if c.FreqHz != 0 {
			if c.FreqHz < cl.lo {
				cl.lo = c.FreqHz
			}
			if c.FreqHz > cl.hi {
				cl.hi = c.FreqHz
			}
		}
	}

	// Stable order: most control channels first, then by SystemID.
	keys := make([]uint32, 0, len(clusters))
	for k := range clusters {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(a, b int) bool {
		ca, cb := clusters[keys[a]], clusters[keys[b]]
		if ca.control != cb.control {
			return ca.control > cb.control
		}
		return keys[a] < keys[b]
	})

	var out []SystemRollup
	for _, k := range keys {
		cl := clusters[k]
		if cl.lo == ^uint32(0) {
			cl.lo = 0
		}
		sr := SystemRollup{
			Protocol:     "dmr",
			Tier3:        cl.control >= 1,
			SystemID:     cl.systemID,
			ColorCode:    cl.colorCode,
			ControlCount: cl.control,
			VoiceCount:   cl.voice,
			LoHz:         cl.lo,
			HiHz:         cl.hi,
			CarrierIdx:   cl.idx,
		}
		sr.Verdict = verdictString(sr, centerHz, sampleRateHz)
		out = append(out, sr)
	}
	return out
}

// verdictString renders the operator-facing one-liner.
func verdictString(sr SystemRollup, centerHz uint32, sampleRateHz float64) string {
	var b strings.Builder
	if sr.Tier3 {
		b.WriteString("DMR Tier III trunked system")
	} else {
		b.WriteString("DMR carriers (no control channel locked)")
	}
	if centerHz != 0 {
		fmt.Fprintf(&b, " at %.4f MHz / %.1f MS/s wideband", float64(centerHz)/1e6, sampleRateHz/1e6)
	} else {
		fmt.Fprintf(&b, " / %.1f MS/s wideband", sampleRateHz/1e6)
	}
	if sr.SystemID != 0 {
		fmt.Fprintf(&b, ", SystemID=%d", sr.SystemID)
	}
	fmt.Fprintf(&b, ", CC=%d", sr.ColorCode)
	fmt.Fprintf(&b, ", %d control + %d voice carrier%s", sr.ControlCount, sr.VoiceCount, plural(sr.ControlCount+sr.VoiceCount))
	if sr.LoHz != 0 && sr.HiHz != 0 {
		fmt.Fprintf(&b, " spanning %.4f–%.4f MHz", float64(sr.LoHz)/1e6, float64(sr.HiHz)/1e6)
	}
	return b.String()
}

func plural(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}

func absU32(a, b uint32) uint32 {
	if a > b {
		return a - b
	}
	return b - a
}
