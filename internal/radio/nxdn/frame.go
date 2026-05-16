// Package nxdn decodes NXDN frame structure (TIA-102.AABG / NXDN technical
// specification rev 1.4). NXDN runs at one of two channel rates:
//
//   - 4800 baud (BFSK): 1 bit per symbol, 4800 bps
//   - 9600 baud (4-FSK): 2 bits per symbol = 1 dibit per symbol,
//     9600 bps over 4800 symbols/sec
//
// Frames are 80 ms long and (regardless of channel rate) carry the same
// logical structure: Frame Sync Word, LICH, SACCH, and an Information
// Field that holds CAC (control channel), VCH/UDCH (voice/data), or
// FACCH (fast associated control). The exact dibit lengths below match
// the most-cited interpretation of the spec; final cross-check against
// the published technical document will be wanted before live captures.
package nxdn

// BaudRate selects the channel rate. The structural layout (in dibits) is
// identical for both; the difference is symbol mapping (BFSK vs 4-FSK).
type BaudRate uint16

const (
	Rate4800 BaudRate = 4800
	Rate9600 BaudRate = 9600
)

func (b BaudRate) String() string {
	switch b {
	case Rate4800:
		return "4800"
	case Rate9600:
		return "9600"
	default:
		return "unknown"
	}
}

// Frame layout in dibits (1 dibit = 2 bits at 9600; 1 dibit ≡ 2 sym at 4800).
const (
	FSWDibits       = 8   // 16 bits
	LICHWireDibits  = 8   // 16 bits transmitted
	LICHInfoBits    = 8   // 8 information bits (each on-air bit doubled)
	SACCHDibits     = 32  // 64 bits
	InfoFieldDibits = 144 // 288 bits — CAC / VCH / UDCH / FACCH
	FrameDibits     = FSWDibits + LICHWireDibits + SACCHDibits + InfoFieldDibits
	FrameBits       = FrameDibits * 2
	FrameDurationMs = 80
)

// Layout offsets within the 192-dibit frame.
const (
	OffsetFSW   = 0
	OffsetLICH  = OffsetFSW + FSWDibits
	OffsetSACCH = OffsetLICH + LICHWireDibits
	OffsetInfo  = OffsetSACCH + SACCHDibits
)

// Frame holds one full 192-dibit NXDN frame.
type Frame struct {
	Dibits [FrameDibits]uint8
}

// FSW returns the 8 dibits of the Frame Sync Word.
func (f *Frame) FSW() []uint8 { return f.Dibits[OffsetFSW : OffsetFSW+FSWDibits] }

// LICH returns the 8 wire dibits of the LICH (which carry the doubled
// 8-bit information field).
func (f *Frame) LICH() []uint8 { return f.Dibits[OffsetLICH : OffsetLICH+LICHWireDibits] }

// SACCH returns the 32 dibits of the Slow Associated Control Channel.
func (f *Frame) SACCH() []uint8 { return f.Dibits[OffsetSACCH : OffsetSACCH+SACCHDibits] }

// Info returns the 144 dibits of the Information Field (CAC / VCH / UDCH).
func (f *Frame) Info() []uint8 { return f.Dibits[OffsetInfo : OffsetInfo+InfoFieldDibits] }
