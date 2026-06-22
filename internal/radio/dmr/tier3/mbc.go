package tier3

import (
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/radio/dmr"
)

// Multi Block Control (MBC) — ETSI TS 102 361-1 §9.x / 361-4. An MBC
// carries a CSBK-opcode control message too large for a single CSBK across
// a header burst (Slot Type DTMBCHeader) followed by one or more
// continuation bursts (DTMBCContinuation), the final block flagged LB=1.
// Each block is a 96-bit information block carried by BPTC(196,96) exactly
// like a CSBK, so decodeInfoBlock (control.go) recovers every block and its
// per-block FEC already rejects uncorrectable bursts.
//
// Honesty note: there is no real off-air MBC capture pinning the last-block
// CRC convention or the multi-block payload layouts (the CSBK CRC history
// in csbk.go warns how an unverified CRC convention rejects every genuine
// frame). So this assembler relies on the per-block BPTC FEC for
// protection, surfaces the assembled opcode + payload at debug regardless,
// and structurally parses only the opcodes whose header-block content
// reuses the validated CSBK grant layout — everything else is assembled +
// logged ("pending capture validation"), the same posture the vendor
// (Connect Plus / Hytera) CSBKs get in vendor.go.

const (
	// mbcMaxBlocks bounds the per-color-code assembly against lost or noisy
	// continuation bursts that never deliver an LB=1 terminator.
	mbcMaxBlocks = 8
	// mbcMaxAge drops a partial assembly whose terminator never arrived, so
	// a stale header can't shadow a fresh message on the same color code.
	mbcMaxAge = time.Second
)

// MBCBlock is one decoded MBC block. The header block (DTMBCHeader) carries
// the CSBK-style leader — LB / PF / CSBKO opcode / FID — in its first two
// octets; continuation blocks reuse the same struct but only LB and the raw
// Content are meaningful for them.
type MBCBlock struct {
	LB      bool
	PF      bool
	Opcode  CSBKOpcode
	FID     uint8
	Content [12]byte // the full 96-bit block, MSB-first
}

// ParseMBCBlock interprets a 12-byte decoded block. LB (bit 0) is valid for
// every block; Opcode/FID are only meaningful on a header block.
func ParseMBCBlock(info []byte) MBCBlock {
	var blk MBCBlock
	blk.LB = info[0]&0x80 != 0
	blk.PF = info[0]&0x40 != 0
	blk.Opcode = CSBKOpcode(info[0] & 0x3F)
	blk.FID = info[1]
	copy(blk.Content[:], info)
	return blk
}

// mbcAssembly is an MBC message under construction, keyed by color code on
// ControlChannel.mbc. blocks holds each block's raw 12-byte content in
// arrival order (header first).
type mbcAssembly struct {
	opcode  CSBKOpcode
	fid     uint8
	blocks  [][12]byte
	started time.Time
}

// handleMBC routes one MBC burst (header or continuation). A header opens a
// fresh assembly on the burst's color code (discarding any partial one),
// continuations append, and the LB=1 block closes and dispatches the
// message.
func (c *ControlChannel) handleMBC(cc uint8, dt dmr.DataType, b *dmr.Burst) {
	info, ok := c.decodeInfoBlock(b)
	if !ok {
		return
	}
	blk := ParseMBCBlock(info)

	if c.mbc == nil {
		c.mbc = make(map[uint8]*mbcAssembly)
	}
	// Evict a stale partial assembly on this color code first: a dropped
	// terminator would otherwise wedge the slot until the next header.
	if a, exists := c.mbc[cc]; exists && c.now().Sub(a.started) > mbcMaxAge {
		delete(c.mbc, cc)
	}

	switch dt {
	case dmr.DTMBCHeader:
		c.mbc[cc] = &mbcAssembly{
			opcode:  blk.Opcode,
			fid:     blk.FID,
			blocks:  [][12]byte{blk.Content},
			started: c.now(),
		}
	case dmr.DTMBCContinuation:
		a := c.mbc[cc]
		if a == nil {
			// Joined mid-message or lost the header — nothing to attach to.
			c.log.Debug("dmr/tier3: MBC continuation without header", "cc", cc)
			return
		}
		if len(a.blocks) >= mbcMaxBlocks {
			c.log.Debug("dmr/tier3: MBC over block limit, dropping", "cc", cc, "blocks", len(a.blocks))
			delete(c.mbc, cc)
			return
		}
		a.blocks = append(a.blocks, blk.Content)
	}

	if blk.LB {
		a := c.mbc[cc]
		delete(c.mbc, cc)
		if a != nil {
			c.dispatchMBC(cc, a)
		}
	}
}

// dispatchMBC handles a fully-assembled MBC message. It logs the assembled
// opcode/payload and structurally parses only the channel-grant opcodes,
// whose header-block content reuses the validated 8-octet CSBK grant
// layout (octets 2-9 of the header block). All other opcodes are surfaced
// assembled-only, pending an off-air capture to validate their layout.
func (c *ControlChannel) dispatchMBC(cc uint8, a *mbcAssembly) {
	c.csbkDecoded.Add(1)
	payload := make([]byte, 0, len(a.blocks)*12)
	for _, blk := range a.blocks {
		payload = append(payload, blk[:]...)
	}
	c.log.Debug("dmr/tier3: mbc assembled",
		"opcode", a.opcode, "fid", a.fid, "blocks", len(a.blocks), "bytes", len(payload))

	var grant [8]byte
	copy(grant[:], a.blocks[0][2:10]) // header octets 2-9 = CSBK payload
	switch a.opcode {
	case OpTVGrant, OpBTVGrant:
		c.publishTVGrant(cc, ParseTVGrant(grant))
	case OpPVGrant:
		c.publishPVGrant(cc, ParsePVGrant(grant))
	case OpPDGrant:
		c.observeDataGrant(cc, ParseDataGrant(grant, true))
	case OpTDGrant:
		c.observeDataGrant(cc, ParseDataGrant(grant, false))
	default:
		c.log.Debug("dmr/tier3: mbc payload decode pending capture validation",
			"opcode", a.opcode, "fid", a.fid, "cc", cc)
	}
}
