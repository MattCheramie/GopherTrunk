// Auxiliary-decoder configuration: the non-trunking side channels GopherTrunk
// can decode alongside the trunked systems — ADS-B, M17, LoRa/LoRaWAN, APRS,
// AIS, DSC, MDC1200 and paging (POCSAG/FLEX). Split out of config.go to keep
// the core schema focused; these are plain YAML-mapped DTOs.
package config

// ADSBConfig configures the ADS-B aircraft-tracking input. The
// native 1 Msps PPM DSP frontend is planned; for now the BEAST
// upstream lets operators consume Mode-S frames from a separately-
// running dump1090 / readsb / BeastSplitter / commercial hub. Most
// 1090 MHz receiver chains already run dump1090 on a dedicated
// RTL-SDR + 1090 MHz filter + LNA; pointing GopherTrunk at it
// is a one-line config away.
type ADSBConfig struct {
	BeastUpstreams []ADSBBeastConfig   `yaml:"beast_upstreams"`
	Channels       []ADSBChannelConfig `yaml:"channels"`
}

// ADSBChannelConfig describes one SDR pinned to 1090 MHz for the
// native PPM Mode-S receiver — the alternative to a BEAST upstream for
// operators who want GopherTrunk to own the whole 1090 MHz chain
// rather than running a separate dump1090 / readsb. Serial picks the
// SDR; the daemon tunes it to FrequencyHz (default 1090 MHz) and runs
// the PPM demodulator against its full IQ stream. A 1090 MHz SAW
// filter + LNA ahead of the SDR is strongly recommended — Mode-S is a
// weak, bursty signal. The SDR must sample at ≥ 2 Msps; the receiver
// resamples to 2 Msps internally. Decoded frames merge into the same
// events.KindAircraftReport stream the BEAST upstreams feed, so the
// /aircraft panel and storage are shared.
type ADSBChannelConfig struct {
	Serial      string `yaml:"serial"`
	FrequencyHz uint32 `yaml:"frequency_hz"` // defaults to 1090 MHz when zero
}

// ADSBBeastConfig describes one BEAST upstream to consume. Addr is
// typically "host:30005" — the standard dump1090 / readsb BEAST
// output port. Multiple upstreams can run side-by-side (e.g. a
// local antenna + a remote hub at the airport) and their frames
// merge into the same `events.KindAircraftReport` stream.
type ADSBBeastConfig struct {
	Addr string `yaml:"addr"`
	Name string `yaml:"name"` // log + metrics label
}

// M17Config configures the M17 digital-voice link-layer receiver.
// Each entry pins an SDR to an M17 frequency and runs the DSP frontend
// (FM demod → C4FM matched filter → symbol-timing recovery → 4FSK
// slice → sync hunt → LICH reassembly → Link Setup Frame parse).
// Decoded link metadata (source / destination callsigns, mode)
// publishes on events.KindM17LinkSetup; storage.M17Log persists it to
// the m17_log table and the REST endpoint at /api/v1/m17/linksetups
// returns the recent rows. Voice (Codec2) decode is a later milestone.
type M17Config struct {
	Channels []M17ChannelConfig `yaml:"channels"`
}

// M17ChannelConfig describes one M17 channel to decode. Serial picks
// the SDR; the daemon tunes it to FrequencyHz and runs the receiver
// against its full IQ stream. M17 simplex calling is commonly
// 144.975 MHz (2 m) / 433.475 MHz (70 cm) in many regions.
type M17ChannelConfig struct {
	Serial      string `yaml:"serial"`
	FrequencyHz uint32 `yaml:"frequency_hz"`
}

// LoRaConfig configures the wide-band LoRa decoder. Each entry pins an SDR
// to a centre frequency and splits its IQ band into one or more parallel
// LoRa sub-channels (a tuner channelizer/DDC bank), each running a
// dechirp/FFT demodulator with spreading-factor auto-detection. Decoded
// frames publish on events.KindLoRaFrame; storage.LoRaLog persists them to
// the lora_log table, the REST endpoint at /api/v1/lora/frames and the
// /lora web panel render them. When a sub-channel carries the LoRaWAN
// public sync word (0x34) and matching session keys are supplied, the MAC
// layer is parsed, the MIC verified and the payload decrypted.
type LoRaConfig struct {
	Channels []LoRaChannelConfig `yaml:"channels"`
}

// LoRaChannelConfig describes one SDR fanned out into LoRa sub-channels.
// Serial picks the SDR; the daemon tunes it to CenterHz and runs the
// wide-band receiver against its full IQ stream. Bandwidth applies to every
// sub-channel (one bank per bandwidth class). Oversample defaults to 2.
type LoRaChannelConfig struct {
	Serial      string                 `yaml:"serial"`
	CenterHz    uint32                 `yaml:"center_hz"`
	Bandwidth   uint32                 `yaml:"bandwidth"`  // 125000 | 250000 | 500000
	Oversample  int                    `yaml:"oversample"` // samples per chip; 0 → 2
	SubChannels []LoRaSubChannelConfig `yaml:"sub_channels"`
	LoRaWANKeys []LoRaWANKeyConfig     `yaml:"lorawan_keys"`
}

// LoRaSubChannelConfig is one LoRa carrier within the dongle's IQ band.
// OffsetHz is the carrier's offset from CenterHz. SpreadingFactor pins the
// SF (7..12); 0 auto-detects across SF7..12. SyncWord defaults to 0x12
// (private); set 0x34 for LoRaWAN. LowDataRateOptimize is "auto" (default),
// "on", or "off" — auto applies the Semtech Ts >= 16 ms rule (SF11/SF12 at
// 125 kHz, SF12 at 250 kHz); set it explicitly for a network that deviates.
type LoRaSubChannelConfig struct {
	OffsetHz            int32  `yaml:"offset_hz"`
	SpreadingFactor     int    `yaml:"spreading_factor"`
	SyncWord            uint8  `yaml:"sync_word"`
	Label               string `yaml:"label"`
	LowDataRateOptimize string `yaml:"low_data_rate_optimize"`
}

// LoRaWANKeyConfig is one operator-supplied LoRaWAN device session-key set,
// keyed by DevAddr. DevAddr / NwkSKey / AppSKey are hex; an optional "0x"
// prefix and internal whitespace are tolerated. GopherTrunk decrypts only
// with keys the operator already holds — it performs no key recovery.
type LoRaWANKeyConfig struct {
	DevAddr string `yaml:"dev_addr"`
	NwkSKey string `yaml:"nwk_skey"`
	AppSKey string `yaml:"app_skey"`
}

// APRSConfig configures the APRS / AX.25 Bell-202 AFSK receiver.
// Each entry pins an SDR to a 2 m / 70 cm APRS frequency and runs
// the DSP frontend (FM demod → FFSK discriminator → symbol-timing
// recovery → NRZI decode → HDLC framer → AX.25 + APRS info-field
// parsing) against its full IQ stream. Decoded packets publish on
// events.KindAPRSPacket; the storage.APRSLog subscriber persists
// them, the REST endpoint at /api/v1/aprs/packets and the /aprs
// web panel render them.
type APRSConfig struct {
	Channels []APRSChannelConfig `yaml:"channels"`
}

// APRSChannelConfig describes one APRS channel to decode. Serial
// picks the SDR; the daemon tunes it to FrequencyHz and runs the
// AFSK receiver against its full IQ stream. The 144.39 MHz North-
// America primary channel is the most common target; other
// regions use 144.575 (EU R1), 144.64 (JP), 144.80 (EU R1 short-
// distance), 145.825 (ISS digipeater), 144.575 (AU). The DropBadFCS
// and DropNonUI toggles match the receiver's options — leave both
// false to see marginal traffic on the panel (highlighted in
// yellow); flip them on if the channel is dominated by noise.
type APRSChannelConfig struct {
	Serial      string `yaml:"serial"`
	FrequencyHz uint32 `yaml:"frequency_hz"`
	DropBadFCS  bool   `yaml:"drop_bad_fcs"`
	DropNonUI   bool   `yaml:"drop_non_ui"`
}

// AISConfig configures the marine-AIS GMSK receiver. Each entry
// pins an SDR to one of the AIS channels (87B = 161.975 MHz,
// 88B = 162.025 MHz — class A vessels alternate between them
// every second) and runs the DSP frontend (FM demod → GFSK
// matched filter → symbol-timing recovery → NRZI decode → HDLC
// framer → ITU-R M.1371-5 message parser) against its full IQ
// stream. Decoded messages publish on events.KindAISMessage;
// storage.VesselLog persists them, the REST endpoint at
// /api/v1/ais/vessels and the /ais web panel render them.
type AISConfig struct {
	Channels []AISChannelConfig `yaml:"channels"`
}

// AISChannelConfig describes one AIS channel to decode. Serial
// picks the SDR; the daemon tunes it to FrequencyHz and runs the
// GMSK receiver against its full IQ stream. Most operators pin
// one SDR to 161.975 (channel 87B) and another to 162.025 (88B)
// to catch both halves of the class-A alternation; one channel
// is enough for class-B-only or quiet-area monitoring. The
// DropBadFCS and DropNonPosition toggles match the receiver's
// options.
type AISChannelConfig struct {
	Serial          string `yaml:"serial"`
	FrequencyHz     uint32 `yaml:"frequency_hz"`
	DropBadFCS      bool   `yaml:"drop_bad_fcs"`
	DropNonPosition bool   `yaml:"drop_non_position"`
}

// DSCConfig configures the marine Digital Selective Calling receiver.
// Each entry pins an SDR to a DSC channel — VHF channel 70 is
// 156.525 MHz; HF DSC rides 2187.5 / 8414.5 / 12577 / 16804.5 kHz
// among others — and runs the DSP frontend (FM demod → FFSK tone
// discriminator at 1300/2100 Hz → symbol-timing recovery → direct-FSK
// slicer → BCH(10,7) character sync → ITU-R M.493 sequence parser)
// against its full IQ stream. Decoded sequences publish on
// events.KindDSCMessage; storage.DSCLog persists them, the REST
// endpoint at /api/v1/dsc/messages and the /dsc web panel render them.
type DSCConfig struct {
	Channels []DSCChannelConfig `yaml:"channels"`
}

// DSCChannelConfig describes one DSC channel to decode. Serial picks
// the SDR; the daemon tunes it to FrequencyHz and runs the FFSK
// receiver against its full IQ stream. The VHF calling channel 70
// (156.525 MHz) is the most common target — it carries distress /
// urgency / safety alerts and the routine call-ups that precede a
// voice working-channel hand-off. DropBadFCS matches the receiver's
// option: leave it false to see BCH-marginal sequences on the panel
// (flagged), flip it on for noisy channels.
type DSCChannelConfig struct {
	Serial      string `yaml:"serial"`
	FrequencyHz uint32 `yaml:"frequency_hz"`
	DropBadFCS  bool   `yaml:"drop_bad_fcs"`
}

// MDC1200Config configures the Motorola MDC1200 FFSK signaling
// receiver. Each entry pins an SDR to a conventional analog VHF / UHF
// voice channel and runs the DSP frontend (FM demod → FFSK
// discriminator at 1200/1800 Hz → symbol-timing recovery → NRZ
// slicer → sync framer → op/arg/unit-ID parser) against its full IQ
// stream. Decoded bursts publish on events.KindMDC1200Message;
// storage.MDC1200Log persists them, the REST endpoint at
// /api/v1/mdc1200/messages and the /mdc1200 web panel render them.
type MDC1200Config struct {
	Channels []MDC1200ChannelConfig `yaml:"channels"`
}

// MDC1200ChannelConfig describes one MDC1200 channel to decode. Serial
// picks the SDR; the daemon tunes it to FrequencyHz and runs the FFSK
// receiver against its full IQ stream. Target the conventional analog
// voice channels of the systems you monitor — MDC1200 bursts ride at
// the head (and optionally tail) of each transmission. DropBadCRC
// matches the receiver's option — leave it false to see CRC-failed
// bursts on the panel (flagged), flip it on for noisy channels.
type MDC1200ChannelConfig struct {
	Serial      string `yaml:"serial"`
	FrequencyHz uint32 `yaml:"frequency_hz"`
	DropBadCRC  bool   `yaml:"drop_bad_crc"`
}

// PagingConfig configures pager decoders. POCSAG and FLEX each pin an
// SDR to a single paging frequency and run the per-protocol receiver
// against its full IQ stream. Wideband groups several paging channels
// (any mix of POCSAG / FLEX) onto one dongle: the daemon tunes the SDR
// to a center frequency and a digital down-converter splits out each
// channel, so two pagers a few hundred kHz apart fit on one stick.
type PagingConfig struct {
	POCSAG   []PagingPOCSAGConfig   `yaml:"pocsag"`
	FLEX     []PagingFLEXConfig     `yaml:"flex"`
	Wideband []PagingWidebandConfig `yaml:"wideband"`
}

// PagingWidebandConfig groups multiple paging channels onto a single
// SDR. The daemon tunes the dongle to CenterFreqHz (auto-computed as the
// midpoint of the channel frequencies when left 0), then runs an
// internal/dsp/tuner DDC bank with one tap per channel — each tap feeds
// the matching POCSAG / FLEX receiver. Every channel frequency must fall
// within CenterFreqHz ± sample_rate/2 (with a small guard band); channels
// outside the usable IQ window are skipped with a startup warning.
type PagingWidebandConfig struct {
	Serial       string                  `yaml:"serial"`
	CenterFreqHz uint32                  `yaml:"center_freq_hz"`
	Channels     []PagingWidebandChannel `yaml:"channels"`
}

// PagingWidebandChannel is one paging channel inside a wideband group.
// Protocol selects the decoder ("pocsag" or "flex"). BaudHz applies to
// POCSAG only (defaults to 1200); FLEX is fixed at 1600 bps and ignores
// it.
type PagingWidebandChannel struct {
	Protocol    string `yaml:"protocol"`
	FrequencyHz uint32 `yaml:"frequency_hz"`
	BaudHz      uint32 `yaml:"baud_hz"`
}

// PagingFLEXConfig describes one FLEX paging channel to decode. Serial
// picks the SDR; the daemon tunes it to FrequencyHz and runs the FLEX
// receiver against its full IQ stream. The frontend handles the
// 1600 bps / 2-level mode. Decoded pages publish on
// events.KindPagerMessage with protocol="flex" and share the pager_log
// table / web panel with POCSAG.
type PagingFLEXConfig struct {
	Serial      string `yaml:"serial"`
	FrequencyHz uint32 `yaml:"frequency_hz"`
}

// PagingPOCSAGConfig describes one POCSAG paging channel to
// decode. Serial picks the SDR; the daemon tunes it to FrequencyHz
// and runs the POCSAG receiver against its full IQ stream. Baud
// defaults to 1200 — the most common POCSAG rate; configure 512
// for legacy networks (e.g. some commercial paging providers) or
// 2400 for higher-throughput systems (DAPNET).
type PagingPOCSAGConfig struct {
	Serial      string `yaml:"serial"`
	FrequencyHz uint32 `yaml:"frequency_hz"`
	BaudHz      uint32 `yaml:"baud_hz"`
}
