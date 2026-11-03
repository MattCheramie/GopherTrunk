// Package iio implements an sdr.Driver for Analog Devices AD936x boards
// that run the Pluto firmware (ADALM-Pluto, OpenSourceSDRLab PlutoSky, and
// compatibles). It talks to iiod, the libiio network daemon running ON the
// board, over its plain-text TCP protocol, so the host needs no SoapySDR,
// no SoapyPlutoSDR and no libiio (issue #1329; design in
// docs/design/native-pluto-driver.md).
//
// Every protocol fact below is pinned to analogdevicesinc/libiio tag v0.25
// (the server in iiod/ops.c + iiod/parser.y, the client in iiod-client.c);
// every AD9361 attribute to pothosware/SoapyPlutoSDR. Neither was ported:
// both are read for the wire format only.
package iio

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// DefaultPort is iiod's TCP port (libiio dns_sd.h, IIOD_PORT).
const DefaultPort = 30431

// ErrnoError is a negative status iiod returned for a command. iiod answers
// every command with a decimal integer line: a byte count or 0 on success,
// a negative errno on failure (print_value, iiod/ops.c).
type ErrnoError struct {
	Op   string
	Code int // positive errno
}

func (e *ErrnoError) Error() string {
	return fmt.Sprintf("iiod: %s: %s (errno %d)", e.Op, syscall.Errno(e.Code).Error(), e.Code)
}

// client speaks iiod's ASCII protocol over one TCP connection. It is not
// safe for concurrent use; the device serialises access.
type client struct {
	conn    net.Conn
	r       *bufio.Reader
	timeout time.Duration // per-command I/O deadline; 0 = none
}

func dial(addr string, timeout time.Duration) (*client, error) {
	conn, err := net.DialTimeout("tcp", addr, timeout)
	if err != nil {
		return nil, err
	}
	if tc, ok := conn.(*net.TCPConn); ok {
		_ = tc.SetNoDelay(true)
	}
	return newClient(conn, timeout), nil
}

func newClient(conn net.Conn, timeout time.Duration) *client {
	return &client{conn: conn, r: bufio.NewReaderSize(conn, 64<<10), timeout: timeout}
}

func (c *client) Close() error { return c.conn.Close() }

func (c *client) arm() {
	if c.timeout > 0 {
		_ = c.conn.SetDeadline(time.Now().Add(c.timeout))
	}
}

func (c *client) disarm() {
	if c.timeout > 0 {
		_ = c.conn.SetDeadline(time.Time{})
	}
}

func (c *client) writeAll(b []byte) error {
	for len(b) > 0 {
		n, err := c.conn.Write(b)
		if err != nil {
			return err
		}
		b = b[n:]
	}
	return nil
}

// readInteger reads iiod's reply integer. Like iiod_client_read_integer it
// skips empty lines (the '\n' after an attribute value or the PRINT XML may
// still be pending) and parses the first non-empty one.
func (c *client) readInteger() (int, error) {
	for {
		line, err := c.r.ReadString('\n')
		if err != nil {
			return 0, err
		}
		line = strings.TrimRight(line, "\r\n")
		if line == "" {
			continue
		}
		v, perr := strconv.Atoi(strings.TrimSpace(line))
		if perr != nil {
			// A server started with -v prints "ERROR: <text>" instead of
			// the negative errno (print_value's verbose branch).
			return 0, fmt.Errorf("iiod: unexpected reply %q", line)
		}
		return v, nil
	}
}

// exec writes cmd and returns iiod's status integer, turning a negative
// one into an *ErrnoError.
func (c *client) exec(op, cmd string) (int, error) {
	c.arm()
	defer c.disarm()
	if err := c.writeAll([]byte(cmd)); err != nil {
		return 0, fmt.Errorf("iiod: %s: %w", op, err)
	}
	v, err := c.readInteger()
	if err != nil {
		return 0, fmt.Errorf("iiod: %s: %w", op, err)
	}
	if v < 0 {
		return 0, &ErrnoError{Op: op, Code: -v}
	}
	return v, nil
}

// Version sends VERSION. iiod replies one line, "major.minor.gittag"
// ("%u.%u.%-7.7s\n", iiod/parser.y), with no status integer in front.
func (c *client) Version() (string, error) {
	c.arm()
	defer c.disarm()
	if err := c.writeAll([]byte("VERSION\r\n")); err != nil {
		return "", fmt.Errorf("iiod: VERSION: %w", err)
	}
	line, err := c.r.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("iiod: VERSION: %w", err)
	}
	v := strings.TrimSpace(line)
	if v == "" || !strings.Contains(v, ".") {
		return "", fmt.Errorf("iiod: VERSION: unexpected reply %q", line)
	}
	return v, nil
}

// PrintXML sends PRINT: a length line, then that many bytes of context XML,
// then '\n' (iiod_client_create_context_private reads xml_len + 1).
func (c *client) PrintXML() ([]byte, error) {
	n, err := c.exec("PRINT", "PRINT\r\n")
	if err != nil {
		return nil, err
	}
	if n <= 0 || n > 16<<20 {
		return nil, fmt.Errorf("iiod: PRINT: implausible XML length %d", n)
	}
	c.arm()
	defer c.disarm()
	buf := make([]byte, n+1)
	if _, err := io.ReadFull(c.r, buf); err != nil {
		return nil, fmt.Errorf("iiod: PRINT: %w", err)
	}
	return buf[:n], nil
}

func inOut(output bool) string {
	if output {
		return "OUTPUT"
	}
	return "INPUT"
}

// ReadChnAttr sends READ <dev> INPUT|OUTPUT <chn> <attr>: a length line,
// then the value, then '\n' (iiod_client_read_attr reads ret + 1).
func (c *client) ReadChnAttr(dev string, output bool, chn, attr string) (string, error) {
	op := "READ " + dev + " " + chn + " " + attr
	n, err := c.exec(op, fmt.Sprintf("READ %s %s %s %s\r\n", dev, inOut(output), chn, attr))
	if err != nil {
		return "", err
	}
	c.arm()
	defer c.disarm()
	buf := make([]byte, n+1)
	if _, err := io.ReadFull(c.r, buf); err != nil {
		return "", fmt.Errorf("iiod: %s: %w", op, err)
	}
	return strings.TrimRight(string(buf[:n]), "\x00\r\n "), nil
}

// WriteChnAttr sends WRITE <dev> INPUT|OUTPUT <chn> <attr> <len> followed by
// the value. Like iio_channel_attr_write, the value goes out with its
// terminating NUL (strlen(src) + 1 bytes, libiio channel.c).
func (c *client) WriteChnAttr(dev string, output bool, chn, attr, value string) error {
	payload := append([]byte(value), 0)
	op := "WRITE " + dev + " " + chn + " " + attr
	c.arm()
	defer c.disarm()
	hdr := fmt.Sprintf("WRITE %s %s %s %s %d\r\n", dev, inOut(output), chn, attr, len(payload))
	if err := c.writeAll(append([]byte(hdr), payload...)); err != nil {
		return fmt.Errorf("iiod: %s: %w", op, err)
	}
	v, err := c.readInteger()
	if err != nil {
		return fmt.Errorf("iiod: %s: %w", op, err)
	}
	if v < 0 {
		return &ErrnoError{Op: op, Code: -v}
	}
	return nil
}

// Open sends OPEN <dev> <samples_count> <mask>, the mask as 8 hex digits
// per 32-bit word, most significant word first (iiod_client_open_unlocked).
func (c *client) Open(dev string, samples int, mask string) error {
	_, err := c.exec("OPEN "+dev, fmt.Sprintf("OPEN %s %d %s\r\n", dev, samples, mask))
	return err
}

// CloseDev sends CLOSE <dev>.
func (c *client) CloseDev(dev string) error {
	_, err := c.exec("CLOSE "+dev, fmt.Sprintf("CLOSE %s\r\n", dev))
	return err
}

// errReadbufShort marks a READBUF that ended before the requested byte
// count: iiod sends a 0 count when its buffer could not fill the request
// (rw_buffer's "ret > 0 && ret < nb" branch).
var errReadbufShort = errors.New("iiod: READBUF ended short")

// ReadBuf sends READBUF <dev> <nbytes> and hands each chunk's sample bytes
// to emit. The reply is a sequence of chunks, each a decimal byte-count
// line then that many raw bytes; the FIRST chunk of every READBUF also
// carries the channel mask (8 hex digits per word, then '\n') between the
// count and the data — the server sets new_client in rw_buffer on each
// READBUF and send_data clears it after the first chunk, and the client
// reads the mask once per call (iiod_client_read_unlocked). A zero count
// ends the transfer early; a negative one is an error. The transfer
// otherwise ends when nbytes have arrived, with no terminator.
//
// emit receives a slice of an internal buffer that is reused by the next
// call; it must copy what it keeps. maskWords is the device's word count.
func (c *client) ReadBuf(dev string, nbytes, maskWords int, emit func([]byte) error) error {
	op := "READBUF " + dev
	if err := c.writeAll([]byte(fmt.Sprintf("READBUF %s %d\r\n", dev, nbytes))); err != nil {
		return fmt.Errorf("iiod: %s: %w", op, err)
	}
	first := true
	remaining := nbytes
	buf := make([]byte, 0, nbytes)
	for remaining > 0 {
		n, err := c.readInteger()
		if err != nil {
			return fmt.Errorf("iiod: %s: %w", op, err)
		}
		if n < 0 {
			return &ErrnoError{Op: op, Code: -n}
		}
		if n == 0 {
			return errReadbufShort
		}
		if n > remaining {
			return fmt.Errorf("iiod: %s: chunk of %d bytes exceeds the %d still owed", op, n, remaining)
		}
		if first {
			mask := make([]byte, maskWords*8+1)
			if _, err := io.ReadFull(c.r, mask); err != nil {
				return fmt.Errorf("iiod: %s: mask: %w", op, err)
			}
			if mask[len(mask)-1] != '\n' {
				return fmt.Errorf("iiod: %s: malformed mask %q", op, mask)
			}
			first = false
		}
		buf = buf[:n]
		if _, err := io.ReadFull(c.r, buf); err != nil {
			return fmt.Errorf("iiod: %s: %w", op, err)
		}
		remaining -= n
		if err := emit(buf); err != nil {
			return err
		}
	}
	return nil
}
