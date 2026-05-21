package broadcast

import (
	"bufio"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/voice/mp3"
)

// Icecast streaming constants. The shine encoder emits a 128 kbps CBR
// MP3, so the live mountpoint is paced at 16000 bytes/second; between
// calls the pacer tops the stream up with pre-encoded silence so the
// source connection is never starved and the Icecast server does not
// time it out.
const (
	icecastBitrateBps  = 128000
	icecastBytesPerSec = icecastBitrateBps / 8
	icecastTick        = 200 * time.Millisecond
	icecastReconnect   = 5 * time.Second
	icecastMaxQueue    = icecastBytesPerSec * 120 // ~2 min of audio
)

// IcecastConfig configures one live Icecast/ShoutCast feed.
type IcecastConfig struct {
	// Name is an optional log label; defaults to "icecast".
	Name string
	// Host and Port address the Icecast server.
	Host string
	Port int
	// Mount is the mountpoint path (e.g. "/gophertrunk"). A leading
	// slash is added when missing.
	Mount string
	// Username is the source username; defaults to "source".
	Username string
	// Password is the source password.
	Password string
	// StreamName is advertised to listeners via the Ice-Name header.
	StreamName string
	// SampleRate is the PCM rate used to synthesise inter-call
	// silence; defaults to 8000.
	SampleRate int
	// Systems restricts the feed to these trunking-system names.
	// Empty streams every system.
	Systems []string
}

type icecastBackend struct {
	systemFilter
	name    string
	addr    string
	mount   string
	auth    string
	stream  string
	log     *slog.Logger
	silence []byte

	mu     sync.Mutex
	queue  []byte
	closed bool

	cancel context.CancelFunc
	done   chan struct{}
}

// NewIcecast builds a live Icecast/ShoutCast streaming backend and
// starts its background source connection.
func NewIcecast(cfg IcecastConfig, log *slog.Logger) (Backend, error) {
	if cfg.Host == "" {
		return nil, errors.New("broadcast/icecast: host is required")
	}
	if cfg.Port == 0 {
		return nil, errors.New("broadcast/icecast: port is required")
	}
	if cfg.Password == "" {
		return nil, errors.New("broadcast/icecast: password is required")
	}
	name := cfg.Name
	if name == "" {
		name = "icecast"
	}
	user := cfg.Username
	if user == "" {
		user = "source"
	}
	mount := cfg.Mount
	if mount == "" {
		mount = "/gophertrunk"
	}
	if !strings.HasPrefix(mount, "/") {
		mount = "/" + mount
	}
	rate := cfg.SampleRate
	if rate == 0 {
		rate = 8000
	}
	if log == nil {
		log = slog.Default()
	}
	stream := cfg.StreamName
	if stream == "" {
		stream = "GopherTrunk"
	}
	// One second of digital silence, encoded once and cycled by the
	// pacer to keep the source connection alive between calls.
	silence, err := mp3.Encode(make([]int16, rate), rate)
	if err != nil {
		return nil, fmt.Errorf("broadcast/icecast: encode silence: %w", err)
	}
	b := &icecastBackend{
		systemFilter: newSystemFilter(cfg.Systems),
		name:         name,
		addr:         net.JoinHostPort(cfg.Host, strconv.Itoa(cfg.Port)),
		mount:        mount,
		auth:         base64.StdEncoding.EncodeToString([]byte(user + ":" + cfg.Password)),
		stream:       stream,
		log:          log,
		silence:      silence,
		done:         make(chan struct{}),
	}
	ctx, cancel := context.WithCancel(context.Background())
	b.cancel = cancel
	go b.run(ctx)
	return b, nil
}

func (b *icecastBackend) Name() string { return b.name }

// Send queues the call audio onto the live stream. Live streaming is
// best-effort: a wedged or oversized queue drops the call rather than
// failing (which would trigger pointless Manager retries).
func (b *icecastBackend) Send(_ context.Context, c *Call) error {
	audio, err := c.MP3()
	if err != nil {
		return fmt.Errorf("%s: encode mp3: %w", b.name, err)
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.closed {
		return nil
	}
	if len(b.queue)+len(audio) > icecastMaxQueue {
		b.log.Warn("broadcast: icecast queue full, dropping call",
			"system", c.System, "tg", c.Talkgroup)
		return nil
	}
	b.queue = append(b.queue, audio...)
	return nil
}

// Close stops the background source connection.
func (b *icecastBackend) Close() error {
	b.mu.Lock()
	if b.closed {
		b.mu.Unlock()
		return nil
	}
	b.closed = true
	b.mu.Unlock()
	b.cancel()
	select {
	case <-b.done:
	case <-time.After(2 * time.Second):
	}
	return nil
}

// run maintains the source connection, reconnecting with a fixed
// backoff on failure, until ctx is cancelled.
func (b *icecastBackend) run(ctx context.Context) {
	defer close(b.done)
	for {
		if ctx.Err() != nil {
			return
		}
		if err := b.runStream(ctx); err != nil {
			b.log.Warn("broadcast: icecast source disconnected",
				"backend", b.name, "err", err)
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(icecastReconnect):
		}
	}
}

// runStream dials the server, performs the source handshake, then
// paces the queued audio (topped up with silence) until ctx ends or a
// write fails.
func (b *icecastBackend) runStream(ctx context.Context) error {
	conn, err := (&net.Dialer{Timeout: 10 * time.Second}).DialContext(ctx, "tcp", b.addr)
	if err != nil {
		return err
	}
	defer conn.Close()
	if err := b.handshake(conn); err != nil {
		return err
	}
	b.log.Info("broadcast: icecast source connected", "backend", b.name, "mount", b.mount)

	ticker := time.NewTicker(icecastTick)
	defer ticker.Stop()
	chunk := icecastBytesPerSec * int(icecastTick) / int(time.Second)
	silenceOff := 0
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
		}
		payload := b.takeChunk(chunk, &silenceOff)
		if err := conn.SetWriteDeadline(time.Now().Add(10 * time.Second)); err != nil {
			return err
		}
		if _, err := conn.Write(payload); err != nil {
			return err
		}
	}
}

// handshake sends the Icecast source request and verifies the server
// accepted it.
func (b *icecastBackend) handshake(conn net.Conn) error {
	req := strings.Join([]string{
		"SOURCE " + b.mount + " HTTP/1.0",
		"Authorization: Basic " + b.auth,
		"User-Agent: GopherTrunk",
		"Content-Type: audio/mpeg",
		"Ice-Name: " + b.stream,
		"Ice-Public: 0",
		"Ice-Audio-Info: bitrate=128",
		"", "",
	}, "\r\n")
	if err := conn.SetWriteDeadline(time.Now().Add(10 * time.Second)); err != nil {
		return err
	}
	if _, err := conn.Write([]byte(req)); err != nil {
		return err
	}
	if err := conn.SetReadDeadline(time.Now().Add(10 * time.Second)); err != nil {
		return err
	}
	status, err := bufio.NewReader(conn).ReadString('\n')
	if err != nil {
		return fmt.Errorf("read source response: %w", err)
	}
	if !strings.Contains(status, "200") {
		return fmt.Errorf("source rejected: %s", strings.TrimSpace(status))
	}
	return nil
}

// takeChunk returns up to n bytes of queued call audio, padding any
// shortfall with cycled silence so the live stream never starves.
func (b *icecastBackend) takeChunk(n int, silenceOff *int) []byte {
	out := make([]byte, 0, n)
	b.mu.Lock()
	if len(b.queue) > 0 {
		take := n
		if take > len(b.queue) {
			take = len(b.queue)
		}
		out = append(out, b.queue[:take]...)
		b.queue = b.queue[take:]
	}
	b.mu.Unlock()
	for len(out) < n {
		if len(b.silence) == 0 {
			break
		}
		if *silenceOff >= len(b.silence) {
			*silenceOff = 0
		}
		end := *silenceOff + (n - len(out))
		if end > len(b.silence) {
			end = len(b.silence)
		}
		out = append(out, b.silence[*silenceOff:end]...)
		*silenceOff = end
	}
	return out
}
