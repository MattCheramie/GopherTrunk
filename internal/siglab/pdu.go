package siglab

import (
	"encoding/hex"

	p25phase1 "github.com/MattCheramie/GopherTrunk/internal/radio/p25/phase1"
)

// pduMaxRecords caps the per-signaling-block dissection a run retains, so a long
// busy control-channel capture cannot grow Result.PDUs without bound. Mirrors
// the IQ-tap cap discipline.
const pduMaxRecords = 5000

// PDURecord is one decoded (or attempted) data/signaling PDU, surfaced for
// SigLab's data-over-RF inspector. It is a flattened, export-friendly view of a
// decoder SignalingBlock with the common entity IDs promoted out of Fields for
// filtering.
type PDURecord struct {
	Seq        int            `json:"seq" yaml:"seq"`
	OffsetSec  float64        `json:"offset_sec" yaml:"offset_sec"`
	Protocol   string         `json:"protocol" yaml:"protocol"`
	Opcode     uint8          `json:"opcode" yaml:"opcode"`
	OpcodeName string         `json:"opcode_name" yaml:"opcode_name"`
	MFID       uint8          `json:"mfid,omitempty" yaml:"mfid,omitempty"`
	NAC        uint16         `json:"nac,omitempty" yaml:"nac,omitempty"`
	SourceID   uint32         `json:"source_id,omitempty" yaml:"source_id,omitempty"`
	DestID     uint32         `json:"dest_id,omitempty" yaml:"dest_id,omitempty"`
	Talkgroup  uint32         `json:"talkgroup,omitempty" yaml:"talkgroup,omitempty"`
	Fields     map[string]any `json:"fields,omitempty" yaml:"fields,omitempty"`
	RawHex     string         `json:"raw_hex" yaml:"raw_hex"`
	CRCOK      bool           `json:"crc_ok" yaml:"crc_ok"`
	FECMetric  int            `json:"fec_metric" yaml:"fec_metric"`
	DibitStart int            `json:"dibit_start" yaml:"dibit_start"`
	DibitLen   int            `json:"dibit_len" yaml:"dibit_len"`
}

// pduCollector accumulates the per-block dissection from the decoder's PDU sink.
// It is driven from a single goroutine (the read loop), so it needs no locking.
type pduCollector struct {
	recs      []PDURecord
	max       int
	truncated bool
	baudHz    float64 // for OffsetSec from DibitStart
}

func newPDUCollector(baudHz float64) *pduCollector {
	return &pduCollector{max: pduMaxRecords, baudHz: baudHz}
}

// observeP25 folds one P25 Phase 1 signaling block into the collection.
func (p *pduCollector) observeP25(b p25phase1.SignalingBlock) {
	if len(p.recs) >= p.max {
		p.truncated = true
		return
	}
	rec := PDURecord{
		Seq:        len(p.recs),
		Protocol:   "p25p1",
		Opcode:     b.Opcode,
		OpcodeName: b.OpcodeName,
		MFID:       b.MFID,
		NAC:        b.NAC,
		RawHex:     hex.EncodeToString(b.RawPayload),
		CRCOK:      b.CRCOK,
		FECMetric:  b.FECMetric,
		DibitStart: b.DibitStart,
		DibitLen:   b.DibitLen,
	}
	if p.baudHz > 0 {
		rec.OffsetSec = float64(b.DibitStart) / p.baudHz
	}
	if b.CRCOK {
		dissectP25TSBK(&rec, b) // only a CRC-valid payload is worth field-decoding
	}
	p.recs = append(p.recs, rec)
}

// dissectP25TSBK decodes the high-value P25 opcodes into promoted entity IDs
// and a Fields map, reusing the decoder's own Parse* field decoders. Opcodes
// not handled here still carry their name + raw hex.
func dissectP25TSBK(rec *PDURecord, b p25phase1.SignalingBlock) {
	var payload [8]byte
	copy(payload[:], b.RawPayload)
	f := map[string]any{}
	switch p25phase1.Opcode(b.Opcode) {
	case p25phase1.OpGroupVoiceChannelGrant:
		g := p25phase1.ParseGroupVoiceChannelGrant(payload)
		rec.Talkgroup, rec.SourceID = uint32(g.GroupAddress), g.SourceID
		f["GroupAddress"], f["SourceID"] = g.GroupAddress, g.SourceID
		f["ChannelID"], f["ChannelNumber"] = g.ChannelID, g.ChannelNumber
	case p25phase1.OpUnitToUnitVoiceChannelGrant:
		g := p25phase1.ParseUnitToUnitVoiceChannelGrant(payload)
		rec.DestID, rec.SourceID = g.TargetID, g.SourceID
		f["TargetID"], f["SourceID"] = g.TargetID, g.SourceID
		f["ChannelID"], f["ChannelNumber"] = g.ChannelID, g.ChannelNumber
	case p25phase1.OpGroupAffiliationResponse:
		a := p25phase1.ParseGroupAffiliationResponse(payload)
		rec.Talkgroup, rec.SourceID = uint32(a.GroupAddress), a.TargetID
		f["GroupAddress"], f["TargetID"] = a.GroupAddress, a.TargetID
	case p25phase1.OpUnitRegistrationResponse:
		u := p25phase1.ParseUnitRegistrationResponse(payload)
		rec.SourceID = u.SourceID
		f["SourceID"], f["SourceAddress"] = u.SourceID, u.SourceAddress
	}
	if len(f) > 0 {
		rec.Fields = f
	}
}
