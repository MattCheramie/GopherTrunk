// Package beast implements a client for the BEAST binary
// protocol — the de-facto wire format dump1090, readsb,
// dump1090-fa, BeastSplitter, and every commercial ADS-B
// hub speak. BEAST upstreams typically listen on TCP port
// 30005; pointing GopherTrunk at one decodes the Mode-S
// frames it emits and feeds them into the standard
// `events.KindAircraftReport` bus event the rest of the
// stack already consumes.
//
// This lets operators keep their existing dump1090 /
// readsb installation (running on a dedicated RTL-SDR with
// a 1090 MHz filter + LNA) and just point GopherTrunk at
// it — no native PPM demod required.
//
// BEAST frame layout (de Magnus de Beer's original spec):
//
//	0x1A <type> <timestamp 6B> <signal 1B> <payload>
//
// Type codes:
//   - 0x31 = Mode-AC (4-byte payload; we skip)
//   - 0x32 = Mode-S short  (7-byte payload, 56-bit frame)
//   - 0x33 = Mode-S long   (14-byte payload, 112-bit frame)
//
// Any 0x1A byte inside the timestamp / signal / payload is
// escaped as 0x1A 0x1A by the transmitter; the receiver
// un-stuffs back to a single 0x1A. This lets the framer
// resync on a missed 0x1A boundary by simply hunting for
// the next 0x1A that's NOT followed by 0x1A.
//
// References:
//   - https://wiki.jetvision.de/wiki/Mode-S_Beast:Data_Output_Formats
//   - https://github.com/firestuff/dump1090/blob/master/beast.h
package beast

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"sync/atomic"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/radio/adsb"
)

// Frame is one decoded BEAST envelope. The Payload bytes hold the
// raw 56- or 112-bit Mode-S frame ready to feed into
// adsb.Decode.
type Frame struct {
	Type      byte   // 0x31 / 0x32 / 0x33
	Timestamp uint64 // 6-byte MLAT timestamp from the receiver
	Signal    byte   // 0..255 signal level
	Payload   []byte // 2 / 7 / 14 bytes depending on Type
}

// payloadLen returns the BEAST type's payload byte count.
// 0 means unknown / unsupported.
func payloadLen(t byte) int {
	switch t {
	case 0x31:
		return 2 // Mode-AC
	case 0x32:
		return 7 // Mode-S short
	case 0x33:
		return 14 // Mode-S long
	}
	return 0
}

// Options configures a Client.
type Options struct {
	// Addr is the BEAST upstream TCP address — typically
	// "host:30005". Required.
	Addr string

	// Bus is required — frames publish onto KindAircraftReport.
	Bus *events.Bus

	// SourceName is stamped on log lines. Typically "beast" or
	// the upstream's hostname.
	SourceName string

	// ReconnectDelay is how long to wait between reconnect
	// attempts after a connection drops. Default 2 s; small
	// values keep the panel responsive to upstream restarts.
	ReconnectDelay time.Duration

	// DialTimeout caps the initial TCP connect attempt.
	// Default 5 s.
	DialTimeout time.Duration

	// ReadDeadline applies to each individual read; if the
	// upstream stops sending for this long the client treats
	// it as a dropped connection and reconnects. Default 30 s.
	// (dump1090 / readsb typically send frames every second
	// on a busy channel.)
	ReadDeadline time.Duration

	// Log is optional; defaults to slog.Default.
	Log *slog.Logger
}

// Client consumes a BEAST TCP upstream, decodes the Mode-S
// frames it emits, pairs CPR halves via an embedded Tracker,
// and publishes one events.KindAircraftReport per parsed
// message.
type Client struct {
	addr           string
	source         string
	bus            *events.Bus
	tracker        *adsb.Tracker
	reconnectDelay time.Duration
	dialTimeout    time.Duration
	readDeadline   time.Duration
	log            *slog.Logger

	framesRead    atomic.Uint64
	framesDecoded atomic.Uint64
	framesEmitted atomic.Uint64
	reconnects    atomic.Uint64
}

// New constructs a Client. Returns an error if required Options
// are missing.
func New(opts Options) (*Client, error) {
	if opts.Bus == nil {
		return nil, errors.New("beast: events.Bus is required")
	}
	if opts.Addr == "" {
		return nil, errors.New("beast: Addr is required")
	}
	c := &Client{
		addr:           opts.Addr,
		source:         opts.SourceName,
		bus:            opts.Bus,
		tracker:        adsb.NewTracker(),
		reconnectDelay: opts.ReconnectDelay,
		dialTimeout:    opts.DialTimeout,
		readDeadline:   opts.ReadDeadline,
		log:            opts.Log,
	}
	if c.reconnectDelay == 0 {
		c.reconnectDelay = 2 * time.Second
	}
	if c.dialTimeout == 0 {
		c.dialTimeout = 5 * time.Second
	}
	if c.readDeadline == 0 {
		c.readDeadline = 30 * time.Second
	}
	if c.log == nil {
		c.log = slog.Default()
	}
	return c, nil
}

// Run dials the upstream and consumes frames until ctx cancels.
// Reconnects with backoff on disconnects; never returns an
// error of its own (logs them instead so a transient upstream
// outage doesn't bring down the daemon).
func (c *Client) Run(ctx context.Context) error {
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		conn, err := c.dial(ctx)
		if err != nil {
			c.log.Warn("beast: dial failed",
				"addr", c.addr, "err", err,
				"retry_in", c.reconnectDelay)
			if waitErr := sleepCtx(ctx, c.reconnectDelay); waitErr != nil {
				return waitErr
			}
			continue
		}
		c.log.Info("beast: connected", "addr", c.addr, "source", c.source)
		c.consume(ctx, conn)
		c.tracker.Reset()
		c.reconnects.Add(1)
		if err := sleepCtx(ctx, c.reconnectDelay); err != nil {
			return err
		}
	}
}

// dial connects to the upstream with a configurable timeout.
func (c *Client) dial(ctx context.Context) (net.Conn, error) {
	d := net.Dialer{Timeout: c.dialTimeout}
	return d.DialContext(ctx, "tcp", c.addr)
}

// consume reads BEAST frames off conn until the connection drops
// or ctx cancels. Each Mode-S frame is decoded, run through the
// CPR tracker, and published on the bus.
func (c *Client) consume(ctx context.Context, conn net.Conn) {
	defer conn.Close()

	// Spawn a goroutine to close the conn on ctx cancel so the
	// blocking read unblocks.
	cancelCh := make(chan struct{})
	defer close(cancelCh)
	go func() {
		select {
		case <-ctx.Done():
			_ = conn.Close()
		case <-cancelCh:
		}
	}()

	r := bufio.NewReaderSize(conn, 64*1024)
	for {
		if err := ctx.Err(); err != nil {
			return
		}
		if c.readDeadline > 0 {
			_ = conn.SetReadDeadline(time.Now().Add(c.readDeadline))
		}
		f, err := ReadFrame(r)
		if err != nil {
			if !errors.Is(err, io.EOF) && ctx.Err() == nil {
				c.log.Warn("beast: read error", "addr", c.addr, "err", err)
			}
			return
		}
		c.framesRead.Add(1)
		c.handleFrame(f)
	}
}

// handleFrame decodes one BEAST-extracted Mode-S frame and
// publishes the resulting AircraftReport.
func (c *Client) handleFrame(f Frame) {
	if f.Type != 0x32 && f.Type != 0x33 {
		// Mode-AC frames carry no useful ADS-B data; skip.
		return
	}
	rep, ok := adsb.ProcessFrame(f.Payload, c.tracker, time.Now())
	if !ok {
		return
	}
	c.framesDecoded.Add(1)
	c.bus.Publish(events.Event{Kind: events.KindAircraftReport, Payload: rep})
	c.framesEmitted.Add(1)
}

// Stats reports cumulative counters for /metrics + debugging.
type Stats struct {
	FramesRead    uint64
	FramesDecoded uint64
	FramesEmitted uint64
	Reconnects    uint64
	TrackerSize   int
}

func (c *Client) Stats() Stats {
	return Stats{
		FramesRead:    c.framesRead.Load(),
		FramesDecoded: c.framesDecoded.Load(),
		FramesEmitted: c.framesEmitted.Load(),
		Reconnects:    c.reconnects.Load(),
		TrackerSize:   c.tracker.Size(),
	}
}

// ReadFrame parses one BEAST frame from a buffered reader.
// Handles the 0x1A 0x1A byte-stuffing escape transparently.
// Returns io.EOF on clean close; any other error indicates a
// transport problem.
func ReadFrame(r *bufio.Reader) (Frame, error) {
	// Sync: hunt for a non-stuffed 0x1A. Anywhere a 0x1A 0x1A
	// pair appears we've landed inside an escaped data byte
	// rather than a frame boundary; advance and try again.
	for {
		b, err := r.ReadByte()
		if err != nil {
			return Frame{}, err
		}
		if b != 0x1A {
			continue
		}
		next, err := r.Peek(1)
		if err != nil {
			return Frame{}, err
		}
		if next[0] == 0x1A {
			// Escaped pair inside a data byte — consume and
			// continue scanning for the next sync.
			_, _ = r.Discard(1)
			continue
		}
		// b is a real frame-start 0x1A. Read the type.
		t, err := r.ReadByte()
		if err != nil {
			return Frame{}, err
		}
		bodyLen := payloadLen(t)
		if bodyLen == 0 {
			return Frame{}, fmt.Errorf("beast: unknown frame type 0x%02X", t)
		}
		// Body = 6-byte timestamp + 1-byte signal + payload.
		body, err := readUnstuffed(r, 6+1+bodyLen)
		if err != nil {
			return Frame{}, err
		}
		var ts uint64
		for i := 0; i < 6; i++ {
			ts = ts<<8 | uint64(body[i])
		}
		return Frame{
			Type:      t,
			Timestamp: ts,
			Signal:    body[6],
			Payload:   body[7:],
		}, nil
	}
}

// readUnstuffed reads exactly n logical bytes from r, transparently
// un-escaping 0x1A 0x1A pairs back to a single 0x1A.
func readUnstuffed(r *bufio.Reader, n int) ([]byte, error) {
	out := make([]byte, 0, n)
	for len(out) < n {
		b, err := r.ReadByte()
		if err != nil {
			return nil, err
		}
		if b == 0x1A {
			next, err := r.ReadByte()
			if err != nil {
				return nil, err
			}
			if next != 0x1A {
				// Should not happen mid-frame; un-stuffing
				// failure means the upstream broke the
				// protocol or we lost sync.
				return nil, fmt.Errorf("beast: unescaped 0x1A in body")
			}
		}
		out = append(out, b)
	}
	return out, nil
}

// sleepCtx is time.Sleep that respects ctx — returns ctx.Err()
// if ctx fires before the duration elapses.
func sleepCtx(ctx context.Context, d time.Duration) error {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}
