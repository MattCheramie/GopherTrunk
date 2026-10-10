package wmbus

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"strings"
)

// Record is one EN 13757-3 data record: a DIF/VIF header and its value.
type Record struct {
	DIF  byte
	DIFE []byte
	VIF  byte
	VIFE []byte

	// Storage number (0 = current value; higher = historic values such as
	// the reading at the last set day), tariff and subunit, from the DIF
	// and its extensions.
	Storage int
	Tariff  int
	Subunit int
	// Function is "instantaneous", "maximum", "minimum" or "error".
	Function string

	// Quantity and Unit name what the VIF measures ("volume", "m3").
	// Empty for VIFs this package does not name; the record still parses.
	Quantity string
	Unit     string

	// Value is the scaled numeric value, when HasValue is set.
	Value    float64
	HasValue bool
	// Text holds a date, date-time, string or invalid-value note.
	Text string
	// Raw is the record's data field as transmitted.
	Raw []byte
}

// String renders the record compactly: "volume 123.529 m3",
// "set day[17] date 2019-01-31".
func (r Record) String() string {
	var b strings.Builder
	q := r.Quantity
	if q == "" {
		q = fmt.Sprintf("VIF 0x%02X", r.VIF)
		if len(r.VIFE) > 0 {
			q += fmt.Sprintf(" VIFE %X", r.VIFE)
		}
	}
	b.WriteString(q)
	if r.Function != "instantaneous" {
		b.WriteString(" (" + r.Function + ")")
	}
	if r.Storage != 0 {
		fmt.Fprintf(&b, " [storage %d]", r.Storage)
	}
	if r.Tariff != 0 {
		fmt.Fprintf(&b, " [tariff %d]", r.Tariff)
	}
	switch {
	case r.Text != "":
		b.WriteString(" " + r.Text)
	case r.HasValue:
		b.WriteString(" " + formatValue(r.Value))
		if r.Unit != "" {
			b.WriteString(" " + r.Unit)
		}
	default:
		fmt.Fprintf(&b, " %X", r.Raw)
	}
	return b.String()
}

func formatValue(v float64) string {
	s := fmt.Sprintf("%.6f", v)
	s = strings.TrimRight(s, "0")
	return strings.TrimSuffix(s, ".")
}

var errRecordTruncated = errors.New("data record truncated")

// ParseRecords decodes the data records of an unencrypted application
// layer. It stops at manufacturer-specific data (DIF 0x0F / 0x1F) and on
// anything it cannot size, returning the records parsed so far with an
// error that says why.
func ParseRecords(p []byte) ([]Record, error) {
	var out []Record
	for len(p) > 0 {
		dif := p[0]
		if dif == 0x2F { // idle filler
			p = p[1:]
			continue
		}
		if dif == 0x0F || dif == 0x1F {
			return out, nil // manufacturer-specific data follows
		}
		r := Record{DIF: dif}
		r.Function = [4]string{"instantaneous", "maximum", "minimum", "error"}[dif>>4&3]
		r.Storage = int(dif >> 6 & 1)
		p = p[1:]
		ext, shift := dif&0x80 != 0, 1
		for i := 0; ext; i++ {
			if len(p) == 0 || i == 10 {
				return out, errRecordTruncated
			}
			e := p[0]
			r.DIFE = append(r.DIFE, e)
			r.Storage |= int(e&0x0F) << shift
			r.Tariff |= int(e>>4&3) << (2 * i)
			r.Subunit |= int(e>>6&1) << i
			shift += 4
			ext, p = e&0x80 != 0, p[1:]
		}
		if len(p) == 0 {
			return out, errRecordTruncated
		}
		r.VIF = p[0]
		p = p[1:]
		var plainUnit string
		if r.VIF&0x7F == 0x7C { // plain-text VIF: length, then ASCII unit
			if len(p) == 0 || len(p) < 1+int(p[0]) {
				return out, errRecordTruncated
			}
			u := make([]byte, p[0])
			for i := range u { // transmitted last character first
				u[i] = p[int(p[0])-i]
			}
			plainUnit, p = string(u), p[1+int(p[0]):]
		}
		ext = r.VIF&0x80 != 0
		for i := 0; ext; i++ {
			if len(p) == 0 || i == 10 {
				return out, errRecordTruncated
			}
			r.VIFE = append(r.VIFE, p[0])
			ext, p = p[0]&0x80 != 0, p[1:]
		}
		n, err := dataLength(dif, p)
		if err != nil {
			return out, err
		}
		if len(p) < n {
			return out, errRecordTruncated
		}
		r.Raw = append([]byte(nil), p[:n]...)
		p = p[n:]
		decodeValue(&r, plainUnit)
		out = append(out, r)
	}
	return out, nil
}

// dataLength is the size of a record's data field from its DIF; for
// variable-length data it reads the LVAR byte at the start of p and
// includes it.
func dataLength(dif byte, p []byte) (int, error) {
	switch dif & 0x0F {
	case 0x0, 0x8:
		return 0, nil
	case 0x1, 0x9:
		return 1, nil
	case 0x2, 0xA:
		return 2, nil
	case 0x3, 0xB:
		return 3, nil
	case 0x4, 0x5, 0xC:
		return 4, nil
	case 0x6, 0xE:
		return 6, nil
	case 0x7:
		return 8, nil
	case 0xD:
		if len(p) == 0 {
			return 0, errRecordTruncated
		}
		l := p[0]
		switch {
		case l < 0xC0:
			return 1 + int(l), nil
		case l < 0xE0:
			return 1 + int(l&0x1F), nil
		case l < 0xF0:
			return 1 + int(l&0x0F), nil
		}
		return 0, fmt.Errorf("variable-length data: unsupported LVAR 0x%02X", l)
	}
	return 0, fmt.Errorf("DIF 0x%02X: special function", dif)
}

// decodeValue fills the record's quantity, unit and value from its VIF and
// raw data.
func decodeValue(r *Record, plainUnit string) {
	v := r.VIF & 0x7F
	exp, timeUnit := 0, ""
	switch {
	case v <= 0x07:
		r.Quantity, r.Unit, exp = "energy", "Wh", int(v&7)-3
	case v <= 0x0F:
		r.Quantity, r.Unit, exp = "energy", "J", int(v&7)
	case v >= 0x10 && v <= 0x17:
		r.Quantity, r.Unit, exp = "volume", "m3", int(v&7)-6
	case v >= 0x20 && v <= 0x23:
		r.Quantity, timeUnit = "on time", durationUnit(v)
	case v >= 0x24 && v <= 0x27:
		r.Quantity, timeUnit = "operating time", durationUnit(v)
	case v >= 0x28 && v <= 0x2F:
		r.Quantity, r.Unit, exp = "power", "W", int(v&7)-3
	case v >= 0x30 && v <= 0x37:
		r.Quantity, r.Unit, exp = "power", "J/h", int(v&7)
	case v >= 0x38 && v <= 0x3F:
		r.Quantity, r.Unit, exp = "volume flow", "m3/h", int(v&7)-6
	case v >= 0x58 && v <= 0x5B:
		r.Quantity, r.Unit, exp = "flow temperature", "°C", int(v&3)-3
	case v >= 0x5C && v <= 0x5F:
		r.Quantity, r.Unit, exp = "return temperature", "°C", int(v&3)-3
	case v >= 0x60 && v <= 0x63:
		r.Quantity, r.Unit, exp = "temperature difference", "K", int(v&3)-3
	case v >= 0x64 && v <= 0x67:
		r.Quantity, r.Unit, exp = "external temperature", "°C", int(v&3)-3
	case v >= 0x68 && v <= 0x6B:
		r.Quantity, r.Unit, exp = "pressure", "bar", int(v&3)-3
	case v == 0x6C:
		r.Quantity = "date"
	case v == 0x6D:
		r.Quantity = "date time"
	case v == 0x6E:
		r.Quantity = "heat cost allocation"
	case v >= 0x74 && v <= 0x77:
		r.Quantity, timeUnit = "actuality duration", durationUnit(v)
	case v == 0x78:
		r.Quantity = "fabrication number"
	case v == 0x79:
		r.Quantity = "enhanced identification"
	case v == 0x7C:
		r.Quantity, r.Unit = "value", plainUnit
	}
	if timeUnit != "" {
		r.Unit = timeUnit
	}

	if r.Quantity == "date" || r.Quantity == "date time" {
		r.Text = decodeDate(r.Raw)
		return
	}
	dt := r.DIF & 0x0F
	if dt == 0xD {
		decodeVariable(r)
		return
	}
	val, ok := rawNumber(dt, r.Raw)
	if !ok {
		if len(r.Raw) > 0 {
			r.Text = fmt.Sprintf("invalid %X", r.Raw)
		}
		return
	}
	if r.Quantity == "fabrication number" || r.Quantity == "enhanced identification" {
		r.Text = formatValue(val)
		return
	}
	r.Value, r.HasValue = val*math.Pow10(exp), true
}

func durationUnit(v byte) string {
	return [4]string{"s", "min", "h", "d"}[v&3]
}

// rawNumber decodes an integer (two's complement), real or BCD data field.
func rawNumber(dt byte, b []byte) (float64, bool) {
	switch dt {
	case 0x1, 0x2, 0x3, 0x4, 0x6, 0x7:
		var u uint64
		for i := len(b) - 1; i >= 0; i-- {
			u = u<<8 | uint64(b[i])
		}
		bits := uint(8 * len(b))
		if u&(1<<(bits-1)) != 0 { // sign-extend
			u |= ^uint64(0) << bits
		}
		return float64(int64(u)), true
	case 0x5:
		return float64(math.Float32frombits(binary.LittleEndian.Uint32(b))), true
	case 0x9, 0xA, 0xB, 0xC, 0xE:
		return bcdValue(b)
	}
	return 0, false
}

// bcdValue decodes little-endian packed BCD; an 0xF top digit marks a
// negative number. Any other non-decimal digit makes it invalid.
func bcdValue(b []byte) (float64, bool) {
	neg := false
	var v float64
	for i := len(b) - 1; i >= 0; i-- {
		for _, d := range [2]byte{b[i] >> 4, b[i] & 0xF} {
			if d == 0xF && i == len(b)-1 && v == 0 && !neg {
				neg = true
				continue
			}
			if d > 9 {
				return 0, false
			}
			v = v*10 + float64(d)
		}
	}
	if neg {
		v = -v
	}
	return v, true
}

func decodeVariable(r *Record) {
	if len(r.Raw) == 0 {
		return
	}
	l, d := r.Raw[0], r.Raw[1:]
	switch {
	case l < 0xC0: // ASCII, transmitted last character first
		s := make([]byte, len(d))
		for i := range d {
			s[i] = d[len(d)-1-i]
		}
		r.Text = strings.TrimSpace(string(s))
	case l < 0xE0: // BCD, 0xD_ = negative
		if v, ok := bcdValue(d); ok {
			if l >= 0xD0 {
				v = -v
			}
			r.Value, r.HasValue = v, true
		}
	default: // binary
		if v, ok := rawNumber(0x7, padTo8(d)); ok && len(d) <= 8 {
			r.Value, r.HasValue = v, true
		}
	}
}

func padTo8(d []byte) []byte {
	if len(d) >= 8 {
		return d[:8]
	}
	return append(append([]byte(nil), d...), make([]byte, 8-len(d))...)
}

// decodeDate renders EN 13757-3 date types G (2 bytes: date), F (4 bytes:
// date and time) and I (6 bytes: date and time with seconds).
func decodeDate(b []byte) string {
	date := func(hi, lo byte) (string, bool) {
		day := int(lo & 0x1F)
		month := int(hi & 0x0F)
		year := 2000 + int(lo&0xE0)>>5 + int(hi&0xF0)>>1
		if day == 0 || month == 0 || month > 12 {
			return "", false
		}
		return fmt.Sprintf("%04d-%02d-%02d", year, month, day), true
	}
	clock := func(hi, lo byte) (string, bool) {
		h, m := int(hi&0x1F), int(lo&0x3F)
		if h > 23 || m > 59 || lo&0x80 != 0 { // bit 7 of the minute byte: invalid
			return "", false
		}
		return fmt.Sprintf("%02d:%02d", h, m), true
	}
	switch len(b) {
	case 2:
		if d, ok := date(b[1], b[0]); ok {
			return d
		}
	case 4:
		d, ok1 := date(b[3], b[2])
		t, ok2 := clock(b[1], b[0])
		if ok1 && ok2 {
			return d + " " + t
		}
	case 6:
		d, ok1 := date(b[4], b[3])
		t, ok2 := clock(b[2], b[1])
		if ok1 && ok2 {
			return fmt.Sprintf("%s %s:%02d", d, t, b[0]&0x3F)
		}
	}
	return fmt.Sprintf("invalid date %X", b)
}
