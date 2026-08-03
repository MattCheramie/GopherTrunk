package voice

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// callMeta is the per-recording metadata sidecar, laid out to match the
// trunk-recorder call `.json` schema so the parsers people already have work
// unchanged. GopherTrunk populates every field it can; the rest carry
// trunk-recorder's own "unknown" conventions (signal/noise = 999, color_code =
// -1) or a best-effort 0 so a strict parser never trips on a missing key.
type callMeta struct {
	CallNum              int             `json:"call_num"`
	Freq                 uint32          `json:"freq"`
	FreqError            int             `json:"freq_error"`
	Signal               int             `json:"signal"`
	Noise                int             `json:"noise"`
	SourceNum            int             `json:"source_num"`
	RecorderNum          int             `json:"recorder_num"`
	TDMASlot             int             `json:"tdma_slot"`
	Phase2TDMA           int             `json:"phase2_tdma"`
	StartTime            int64           `json:"start_time"`
	StopTime             int64           `json:"stop_time"`
	StartTimeMs          int64           `json:"start_time_ms"`
	StopTimeMs           int64           `json:"stop_time_ms"`
	Emergency            int             `json:"emergency"`
	Priority             int             `json:"priority"`
	Mode                 int             `json:"mode"`
	Duplex               int             `json:"duplex"`
	Encrypted            int             `json:"encrypted"`
	CallLength           int             `json:"call_length"`
	CallLengthMs         int64           `json:"call_length_ms"`
	Talkgroup            uint32          `json:"talkgroup"`
	TalkgroupTag         string          `json:"talkgroup_tag"`
	TalkgroupDescription string          `json:"talkgroup_description"`
	TalkgroupGroupTag    string          `json:"talkgroup_group_tag"`
	TalkgroupGroup       string          `json:"talkgroup_group"`
	ColorCode            int             `json:"color_code"`
	AudioType            string          `json:"audio_type"`
	ShortName            string          `json:"short_name"`
	FreqList             []callFreqEntry `json:"freqList"`
	SrcList              []callSrcEntry  `json:"srcList"`
}

// callFreqEntry mirrors trunk-recorder's freqList element. error_count /
// spike_count have no GopherTrunk source and are emitted as 0.
type callFreqEntry struct {
	Freq       uint32  `json:"freq"`
	Time       int64   `json:"time"`
	Pos        float64 `json:"pos"`
	Len        float64 `json:"len"`
	ErrorCount int     `json:"error_count"`
	SpikeCount int     `json:"spike_count"`
}

// callSrcEntry mirrors trunk-recorder's srcList element — one per talker heard
// in this recording, in order. signal_system / tag / tag_ota are emitted empty
// (GopherTrunk resolves talker aliases elsewhere).
type callSrcEntry struct {
	Src          uint32  `json:"src"`
	Time         int64   `json:"time"`
	Pos          float64 `json:"pos"`
	Emergency    int     `json:"emergency"`
	SignalSystem string  `json:"signal_system"`
	Tag          string  `json:"tag"`
	TagOTA       string  `json:"tag_ota"`
}

// srcEvent / freqEvent are the raw talker / frequency observations accumulated
// on a recordingSession as the call runs; buildCallMeta turns them into the
// pos/len-resolved srcList / freqList above at finalize time.
type srcEvent struct {
	src       uint32
	at        time.Time
	emergency bool
}

type freqEvent struct {
	freq uint32
	at   time.Time
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

// buildCallMeta assembles the trunk-recorder metadata for a finished recording
// from the session's grant/talkgroup and its accumulated talker/frequency
// events. startedAt/endedAt bound the file (a per-transmission segment has its
// own span). callNum is the recorder's monotonic call counter.
func buildCallMeta(cs trunking.CallStart, startedAt, endedAt time.Time, callNum int, digital bool, srcs []srcEvent, freqs []freqEvent) *callMeta {
	g := cs.Grant
	dur := endedAt.Sub(startedAt)
	audioType := "analog"
	if digital {
		audioType = "digital"
	}
	m := &callMeta{
		CallNum:      callNum,
		Freq:         g.FrequencyHz,
		FreqError:    0,
		Signal:       999, // trunk-recorder "unknown" sentinel
		Noise:        999,
		TDMASlot:     int(g.Timeslot),
		StartTime:    startedAt.Unix(),
		StopTime:     endedAt.Unix(),
		StartTimeMs:  startedAt.UnixMilli(),
		StopTimeMs:   endedAt.UnixMilli(),
		Emergency:    boolToInt(g.Emergency),
		Encrypted:    boolToInt(g.Encrypted),
		CallLength:   int(dur.Seconds()),
		CallLengthMs: dur.Milliseconds(),
		Talkgroup:    g.GroupID,
		ColorCode:    -1, // trunk-recorder "unknown" sentinel
		AudioType:    audioType,
		ShortName:    g.System,
		FreqList:     []callFreqEntry{},
		SrcList:      []callSrcEntry{},
	}
	if tg := cs.Talkgroup; tg != nil {
		m.TalkgroupTag = tg.AlphaTag
		m.TalkgroupDescription = tg.Description
		m.TalkgroupGroupTag = tg.Tag
		m.TalkgroupGroup = tg.Group
		m.Priority = tg.Priority
	}

	// freqList: pos is seconds from the file start; len runs to the next entry
	// (or the call end for the last). TETRA same-carrier calls stay on one freq,
	// so this is usually a single entry spanning the call.
	if len(freqs) == 0 {
		freqs = []freqEvent{{freq: g.FrequencyHz, at: startedAt}}
	}
	for i, fe := range freqs {
		end := endedAt
		if i+1 < len(freqs) {
			end = freqs[i+1].at
		}
		m.FreqList = append(m.FreqList, callFreqEntry{
			Freq: fe.freq,
			Time: fe.at.Unix(),
			Pos:  round2(fe.at.Sub(startedAt).Seconds()),
			Len:  round2(end.Sub(fe.at).Seconds()),
		})
	}

	// srcList: one entry per talker, in order. pos is seconds from the CALL start
	// (cs.StartedAt), not this file's start — in transmission grouping the list is
	// call-complete, so a talker from an earlier over would otherwise get a negative
	// pos. For a single-transmission call the two starts coincide, so pos is
	// unchanged. Clamp at 0 as a floor against clock skew.
	srcBase := cs.StartedAt
	if srcBase.IsZero() {
		srcBase = startedAt
	}
	for _, se := range srcs {
		pos := se.at.Sub(srcBase).Seconds()
		if pos < 0 {
			pos = 0
		}
		m.SrcList = append(m.SrcList, callSrcEntry{
			Src:       se.src,
			Time:      se.at.Unix(),
			Pos:       round2(pos),
			Emergency: boolToInt(se.emergency),
		})
	}
	return m
}

// writeCallMeta writes m as <wav-basename>.json next to the recording. A nil m
// or empty wavPath is a no-op. The write is small (~1 KB) and is done off the
// recorder lock by the caller.
func writeCallMeta(wavPath string, m *callMeta) error {
	if m == nil || wavPath == "" {
		return nil
	}
	jsonPath := strings.TrimSuffix(wavPath, filepath.Ext(wavPath)) + ".json"
	b, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(jsonPath, b, 0o644)
}
