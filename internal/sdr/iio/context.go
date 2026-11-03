package iio

import (
	"encoding/xml"
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// Context is the part of iiod's PRINT XML the driver needs.
type Context struct {
	Name        string
	Description string
	Attrs       map[string]string // context-attribute name → value
	Devices     []*Device
}

// Device is one IIO device in the context.
type Device struct {
	ID       string // "iio:device3"; what commands address
	Name     string // "cf-ad9361-lpc"
	Channels []*Channel
}

// Channel is one IIO channel. Number is libiio's channel number: the index
// into the device's channel list after reorder_channels (context.c) sorts
// scan elements by index, which is the bit the channel takes in a buffer
// mask.
type Channel struct {
	ID     string
	Name   string
	Output bool
	Scan   *ScanFormat // nil when the channel is not a scan element
	Attrs  []string
	Number int
}

// ScanFormat is a scan element's sample layout, parsed from the XML
// "format" attribute ("le:S12/16>>0"), as libiio's setup_scan_element does.
type ScanFormat struct {
	Index     int
	BigEndian bool
	Signed    bool
	Bits      int // significant bits
	Length    int // storage bits
	Repeat    int
	Shift     int
	Raw       string
}

// FindDevice looks a device up by id or name.
func (c *Context) FindDevice(name string) *Device {
	for _, d := range c.Devices {
		if d.ID == name || d.Name == name {
			return d
		}
	}
	return nil
}

// FindChannel looks a channel up by id (or name) and direction.
func (d *Device) FindChannel(id string, output bool) *Channel {
	for _, ch := range d.Channels {
		if (ch.ID == id || (ch.Name != "" && ch.Name == id)) && ch.Output == output {
			return ch
		}
	}
	return nil
}

// HasAttr reports whether the channel lists attr.
func (ch *Channel) HasAttr(attr string) bool {
	for _, a := range ch.Attrs {
		if a == attr {
			return true
		}
	}
	return false
}

// MaskWords is the device's buffer-mask length in 32-bit words: one bit per
// channel, scan element or not (xml.c: words = (nb_channels + 31) / 32).
func (d *Device) MaskWords() int { return (len(d.Channels) + 31) / 32 }

// Mask renders the buffer mask enabling chans, as OPEN expects it: 8 lower-
// case hex digits per word, most significant word first.
func (d *Device) Mask(chans ...*Channel) string {
	words := make([]uint32, d.MaskWords())
	for _, ch := range chans {
		words[ch.Number/32] |= 1 << uint(ch.Number%32)
	}
	var b strings.Builder
	for i := len(words) - 1; i >= 0; i-- {
		fmt.Fprintf(&b, "%08x", words[i])
	}
	return b.String()
}

type xmlContext struct {
	Name        string `xml:"name,attr"`
	Description string `xml:"description,attr"`
	Attrs       []struct {
		Name  string `xml:"name,attr"`
		Value string `xml:"value,attr"`
	} `xml:"context-attribute"`
	Devices []struct {
		ID       string `xml:"id,attr"`
		Name     string `xml:"name,attr"`
		Channels []struct {
			ID   string `xml:"id,attr"`
			Name string `xml:"name,attr"`
			Type string `xml:"type,attr"`
			Scan *struct {
				Index  string `xml:"index,attr"`
				Format string `xml:"format,attr"`
			} `xml:"scan-element"`
			Attrs []struct {
				Name string `xml:"name,attr"`
			} `xml:"attribute"`
		} `xml:"channel"`
	} `xml:"device"`
}

// ParseContext parses iiod's PRINT XML.
func ParseContext(data []byte) (*Context, error) {
	var x xmlContext
	// The XML opens with a DOCTYPE carrying an internal DTD; encoding/xml
	// reads it as a directive and skips it.
	if err := xml.Unmarshal(data, &x); err != nil {
		return nil, fmt.Errorf("iiod: context XML: %w", err)
	}
	ctx := &Context{Name: x.Name, Description: x.Description, Attrs: map[string]string{}}
	for _, a := range x.Attrs {
		ctx.Attrs[a.Name] = a.Value
	}
	for _, xd := range x.Devices {
		dev := &Device{ID: xd.ID, Name: xd.Name}
		for _, xc := range xd.Channels {
			ch := &Channel{ID: xc.ID, Name: xc.Name, Output: xc.Type == "output"}
			for _, a := range xc.Attrs {
				ch.Attrs = append(ch.Attrs, a.Name)
			}
			if xc.Scan != nil {
				f, err := parseScanFormat(xc.Scan.Index, xc.Scan.Format)
				if err != nil {
					return nil, fmt.Errorf("iiod: %s channel %s: %w", xd.Name, xc.ID, err)
				}
				ch.Scan = f
			}
			dev.Channels = append(dev.Channels, ch)
		}
		reorderChannels(dev)
		ctx.Devices = append(ctx.Devices, dev)
	}
	return ctx, nil
}

func parseScanFormat(index, format string) (*ScanFormat, error) {
	idx, err := strconv.ParseInt(index, 0, 64)
	if err != nil || idx < 0 {
		return nil, fmt.Errorf("bad scan-element index %q", index)
	}
	f := &ScanFormat{Index: int(idx), Repeat: 1, Raw: format}
	// "%ce:%c%u/%u>>%u", optionally "%ce:%c%u/%uX%u>>%u" (libiio xml.c).
	colon := strings.IndexByte(format, ':')
	if colon != 2 || format[1] != 'e' || len(format) < 4 {
		return nil, fmt.Errorf("bad scan-element format %q", format)
	}
	f.BigEndian = format[0] == 'b'
	s := format[colon+1]
	f.Signed = s == 's' || s == 'S'
	rest := format[colon+2:]
	shiftAt := strings.Index(rest, ">>")
	if shiftAt < 0 {
		return nil, fmt.Errorf("bad scan-element format %q", format)
	}
	if f.Shift, err = strconv.Atoi(rest[shiftAt+2:]); err != nil {
		return nil, fmt.Errorf("bad scan-element format %q", format)
	}
	rest = rest[:shiftAt]
	if x := strings.IndexByte(rest, 'X'); x >= 0 {
		if f.Repeat, err = strconv.Atoi(rest[x+1:]); err != nil {
			return nil, fmt.Errorf("bad scan-element format %q", format)
		}
		rest = rest[:x]
	}
	slash := strings.IndexByte(rest, '/')
	if slash < 0 {
		return nil, fmt.Errorf("bad scan-element format %q", format)
	}
	if f.Bits, err = strconv.Atoi(rest[:slash]); err != nil {
		return nil, fmt.Errorf("bad scan-element format %q", format)
	}
	if f.Length, err = strconv.Atoi(rest[slash+1:]); err != nil {
		return nil, fmt.Errorf("bad scan-element format %q", format)
	}
	return f, nil
}

// reorderChannels assigns libiio's channel numbers: scan elements sorted by
// index (ties by shift), then non-scan channels in document order
// (reorder_channels, context.c — a stable bubble sort that only moves a
// channel with an index ahead of a larger index or of a non-scan channel).
func reorderChannels(dev *Device) {
	key := func(ch *Channel) (int, int) {
		if ch.Scan == nil {
			return -1, 0
		}
		return ch.Scan.Index, ch.Scan.Shift
	}
	sort.SliceStable(dev.Channels, func(i, j int) bool {
		ai, as := key(dev.Channels[i])
		bi, bs := key(dev.Channels[j])
		if ai < 0 {
			return false
		}
		if bi < 0 {
			return true
		}
		if ai != bi {
			return ai < bi
		}
		return as < bs
	})
	for i, ch := range dev.Channels {
		ch.Number = i
	}
}
