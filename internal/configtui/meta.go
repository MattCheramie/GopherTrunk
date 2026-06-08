// Package configtui is a standalone Bubble Tea terminal Config Builder/Editor
// that mirrors the web Config Builder (web/configbuilder). It is reflection-
// driven: it walks config.Config so every field is editable automatically (it
// cannot drift from the schema), with a metadata table supplying the web-like
// polish — labels, help, select options, Hz formatting, fieldset grouping and
// the AdvancedJSON long-tail. It operates on the local filesystem via the
// config package directly (no daemon, no browser).
package configtui

import (
	"reflect"
	"strings"
	"unicode"

	"github.com/MattCheramie/GopherTrunk/internal/configbuilder"
)

// selOpt is one select option (value + display label).
type selOpt struct{ Value, Label string }

// fieldMeta overrides the reflection defaults for one struct field, keyed
// "StructName.FieldName". Absent ⇒ label is the humanized field name and the
// widget is derived from the Go kind.
type fieldMeta struct {
	Label    string
	Help     string
	Options  []selOpt // makes a string render as a select
	Hz       bool     // uint32 → Hz widget (MHz/Hz)
	FreqList bool     // []uint32 → frequency-list widget
	Hidden   bool
}

func role4() []selOpt {
	return []selOpt{{"", "(auto)"}, {"control", "control"}, {"voice", "voice"}, {"auto", "auto"}}
}

// metaTable mirrors the web sections' SelectField options, HzField/FreqList
// usage, and labels. Only fields needing polish appear; everything else uses
// reflection defaults (and is still fully editable).
var metaTable = map[string]fieldMeta{
	// Log
	"LogConfig.Level":  {Options: []selOpt{{"", "(default: info)"}, {"debug", "debug"}, {"info", "info"}, {"warn", "warn"}, {"error", "error"}}},
	"LogConfig.Format": {Options: []selOpt{{"", "(default: text)"}, {"text", "text"}, {"json", "json"}}},
	// API auth / cors
	"APIAuthConfig.Mode": {Options: []selOpt{{"", "(auto)"}, {"auto", "auto"}, {"required", "required"}, {"disabled", "disabled"}}},
	// SDR devices
	"DeviceConfig.Role":                {Options: []selOpt{{"", "(auto)"}, {"control", "control"}, {"voice", "voice"}, {"auto", "auto"}, {"wideband", "wideband"}}},
	"DeviceConfig.CenterFreqHz":        {Hz: true},
	"DeviceConfig.TunerStrategy":       {Options: []selOpt{{"", "(auto)"}, {"ddc", "ddc"}, {"polyphase", "polyphase"}}},
	"DeviceChannelConfig.FrequencyHz":  {Hz: true, Label: "Frequency"},
	"RTLTCPConfig.Role":                {Options: role4()},
	"SoapyRemoteConfig.Role":           {Options: role4()},
	"SoapyRemoteConfig.Format":         {Options: []selOpt{{"", "(default)"}, {"CS16", "CS16"}, {"CF32", "CF32"}}},
	"SoapyRemoteConfig.StreamProtocol": {Options: []selOpt{{"", "(default)"}, {"tcp", "tcp"}}},
	// Trunking systems
	"SystemConfig.Protocol":              {Options: protocolOpts()},
	"SystemConfig.ControlChannels":       {FreqList: true, Label: "Control channels"},
	"P25BandPlanEntryConfig.BaseHz":      {Hz: true},
	"P25BandPlanEntryConfig.SpacingHz":   {Hz: true},
	"P25BandPlanEntryConfig.BandwidthHz": {Hz: true},
	"DMRLinearBandPlanConfig.BaseHz":     {Hz: true},
	"DMRLinearBandPlanConfig.SpacingHz":  {Hz: true},
	"DMRBandPlanTableEntryConfig.FreqHz": {Hz: true},
	"EncryptionKeyConfig.Algorithm":      {Options: []selOpt{{"rc4", "rc4 (DMR Enhanced Privacy)"}}},
	// Scanner
	"ScannerConfig.ScanMode":        {Options: []selOpt{{"", "(default: all)"}, {"all", "all"}, {"list", "list"}}},
	"ConvChannelConfig.FrequencyHz": {Hz: true},
	"ConvChannelConfig.Mode":        {Options: []selOpt{{"", "(fm)"}, {"fm", "fm"}, {"nfm", "nfm"}}},
	"ConvToneConfig.Mode":           {Options: []selOpt{{"", "none"}, {"none", "none"}, {"ctcss", "ctcss"}, {"dcs", "dcs"}}},
	// Baseband
	"BasebandReplayConfig.Role": {Options: role4()},
	// Paging
	"PagingPOCSAGConfig.FrequencyHz":    {Hz: true},
	"PagingPOCSAGConfig.BaudHz":         {Options: baudOpts()},
	"PagingFLEXConfig.FrequencyHz":      {Hz: true},
	"PagingWidebandConfig.CenterFreqHz": {Hz: true},
	"PagingWidebandChannel.Protocol":    {Options: []selOpt{{"pocsag", "POCSAG"}, {"flex", "FLEX"}}},
	"PagingWidebandChannel.FrequencyHz": {Hz: true},
	"PagingWidebandChannel.BaudHz":      {Options: baudOpts()},
	// Decoder channels
	"APRSChannelConfig.FrequencyHz":    {Hz: true},
	"AISChannelConfig.FrequencyHz":     {Hz: true},
	"DSCChannelConfig.FrequencyHz":     {Hz: true},
	"MDC1200ChannelConfig.FrequencyHz": {Hz: true},
	"ADSBChannelConfig.FrequencyHz":    {Hz: true},
	"M17ChannelConfig.FrequencyHz":     {Hz: true},
}

func protocolOpts() []selOpt {
	ps := []string{"p25", "p25-phase2", "dmr", "dmr-tier2", "dmr-tier1", "nxdn", "dpmr",
		"edacs", "motorola", "ltr", "mpt1327", "tetra", "ysf", "dstar"}
	out := make([]selOpt, len(ps))
	for i, p := range ps {
		out[i] = selOpt{p, p}
	}
	return out
}

func baudOpts() []selOpt {
	return []selOpt{{"512", "512"}, {"1200", "1200"}, {"2400", "2400"}}
}

// metaFor returns the metadata for structName.field (zero value if none).
func metaFor(structName, field string) fieldMeta {
	return metaTable[structName+"."+field]
}

// isAdvanced reports whether a field is in the AdvancedJSON long-tail.
func isAdvanced(structName, field string) bool {
	return configbuilder.IsAdvanced(structName, field)
}

// humanize turns a Go field name into a label ("CallTimeoutMs" → "Call timeout
// ms", "HTTPAddr" → "HTTP addr").
func humanize(name string) string {
	var b strings.Builder
	runes := []rune(name)
	for i, r := range runes {
		if i > 0 && unicode.IsUpper(r) {
			prev := runes[i-1]
			next := rune(0)
			if i+1 < len(runes) {
				next = runes[i+1]
			}
			// Insert a space at a lower→upper boundary, or at the end of an
			// acronym run (upper followed by lower).
			if !unicode.IsUpper(prev) || (next != 0 && unicode.IsLower(next)) {
				b.WriteRune(' ')
			}
		}
		b.WriteRune(r)
	}
	out := b.String()
	// Lower-case the tail words but keep the first character as-is.
	return out
}

// labelFor resolves a field's display label (metadata override or humanized).
func labelFor(structName string, f reflect.StructField) string {
	if m := metaFor(structName, f.Name); m.Label != "" {
		return m.Label
	}
	return humanize(f.Name)
}
