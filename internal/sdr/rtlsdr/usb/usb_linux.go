//go:build linux && (amd64 || arm64 || 386 || arm || riscv64 || loong64)

package usb

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"unsafe"

	"golang.org/x/sys/unix"
)

// usbdevfsCtrltransfer mirrors struct usbdevfs_ctrltransfer in
// <linux/usbdevice_fs.h>. The build tag above restricts this file to
// architectures where Go's natural field alignment matches the kernel's
// (asm-generic ioctl encoding + LP64/ILP32 with no pragma pack), so the
// Go compiler's auto-padding produces a wire-identical struct.
type usbdevfsCtrltransfer struct {
	BmRequestType uint8
	BRequest      uint8
	WValue        uint16
	WIndex        uint16
	WLength       uint16
	Timeout       uint32
	Data          *byte
}

// usbdevfsURB mirrors struct usbdevfs_urb (without the trailing
// iso_frame_desc flexible array — RTL-SDR only uses bulk).
type usbdevfsURB struct {
	Type            uint8
	Endpoint        uint8
	Status          int32
	Flags           uint32
	Buffer          *byte
	BufferLength    int32
	ActualLength    int32
	StartFrame      int32
	NumberOfPackets int32 // union with stream_id; unused for bulk
	ErrorCount      int32
	Signr           uint32
	Usercontext     *byte
}

// usbdevfsIoctlArg mirrors struct usbdevfs_ioctl in
// <linux/usbdevice_fs.h> — the argument to USBDEVFS_IOCTL, the
// "ioctl within an ioctl" used to drive a bound kernel driver. We use
// it to issue USBDEVFS_DISCONNECT against a single interface so the
// kernel unbinds whatever driver currently owns it. The build tag at
// the top of this file keeps Go's natural layout (int32, int32,
// pointer) wire-identical to the kernel's struct on every supported
// arch.
type usbdevfsIoctlArg struct {
	Ifno      int32 // interface number the inner ioctl targets
	IoctlCode int32 // the inner ioctl request (USBDEVFS_DISCONNECT)
	Data      *byte // inner ioctl payload; nil for DISCONNECT
}

const (
	usbdevfsURBTypeBULK = 3
)

// asm-generic ioctl encoding.
const (
	iocNRShift   = 0
	iocTypeShift = 8
	iocSizeShift = 16
	iocDirShift  = 30

	iocNone  = 0
	iocWrite = 1
	iocRead  = 2
)

func ioc(dir, typ, nr, size uintptr) uintptr {
	return (dir << iocDirShift) | (typ << iocTypeShift) | (nr << iocNRShift) | (size << iocSizeShift)
}

var (
	usbdevfsControl          = ioc(iocRead|iocWrite, 'U', 0, unsafe.Sizeof(usbdevfsCtrltransfer{}))
	usbdevfsSubmitURB        = ioc(iocRead, 'U', 10, unsafe.Sizeof(usbdevfsURB{}))
	usbdevfsDiscardURB       = ioc(iocNone, 'U', 11, 0)
	usbdevfsReapURB          = ioc(iocWrite, 'U', 12, unsafe.Sizeof(uintptr(0)))
	usbdevfsClaimInterface   = ioc(iocRead, 'U', 15, 4)
	usbdevfsReleaseInterface = ioc(iocRead, 'U', 16, 4)
	usbdevfsIoctlCmd         = ioc(iocRead|iocWrite, 'U', 18, unsafe.Sizeof(usbdevfsIoctlArg{}))
	usbdevfsReset            = ioc(iocNone, 'U', 20, 0)
	usbdevfsDisconnect       = ioc(iocNone, 'U', 22, 0)
)

func platformEnumerator() Enumerator { return &linuxEnumerator{} }

// linuxEnumerator walks /sys/bus/usb/devices for [Descriptor]s and opens
// matching device nodes under /dev/bus/usb. Both roots can be overridden
// from tests by setting [linuxEnumerator.sysfsRoot] / [linuxEnumerator.devfsRoot].
type linuxEnumerator struct {
	sysfsRoot string
	devfsRoot string
}

func (l *linuxEnumerator) Name() string { return "usbdevfs" }

func (l *linuxEnumerator) sysfs() string {
	if l.sysfsRoot != "" {
		return l.sysfsRoot
	}
	return "/sys/bus/usb/devices"
}

func (l *linuxEnumerator) devfs() string {
	if l.devfsRoot != "" {
		return l.devfsRoot
	}
	return "/dev/bus/usb"
}

func (l *linuxEnumerator) List(vid, pid uint16) ([]Descriptor, error) {
	root := l.sysfs()
	entries, err := os.ReadDir(root)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("usbdevfs: read %s: %w", root, err)
	}
	var out []Descriptor
	for _, e := range entries {
		path := filepath.Join(root, e.Name())
		v, ok1 := readHex16(filepath.Join(path, "idVendor"))
		p, ok2 := readHex16(filepath.Join(path, "idProduct"))
		if !ok1 || !ok2 {
			continue
		}
		if vid != 0 && v != vid {
			continue
		}
		if pid != 0 && p != pid {
			continue
		}
		bus, busOK := readUint8(filepath.Join(path, "busnum"))
		addr, addrOK := readUint8(filepath.Join(path, "devnum"))
		if !busOK || !addrOK {
			continue
		}
		d := Descriptor{
			Bus:          bus,
			Address:      addr,
			VID:          v,
			PID:          p,
			Serial:       readTrim(filepath.Join(path, "serial")),
			Manufacturer: readTrim(filepath.Join(path, "manufacturer")),
			Product:      readTrim(filepath.Join(path, "product")),
			Path:         filepath.Join(l.devfs(), fmt.Sprintf("%03d", bus), fmt.Sprintf("%03d", addr)),
		}
		out = append(out, d)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Bus != out[j].Bus {
			return out[i].Bus < out[j].Bus
		}
		return out[i].Address < out[j].Address
	})
	return out, nil
}

func (l *linuxEnumerator) Open(d Descriptor) (Transport, error) {
	path := d.Path
	if path == "" {
		path = filepath.Join(l.devfs(), fmt.Sprintf("%03d", d.Bus), fmt.Sprintf("%03d", d.Address))
	}
	fd, err := unix.Open(path, unix.O_RDWR|unix.O_CLOEXEC, 0)
	if err != nil {
		return nil, fmt.Errorf("usbdevfs: open %s: %w", path, err)
	}
	return &linuxTransport{fd: fd, path: path, desc: d}, nil
}

// linuxTransport is the USBDEVFS-backed [Transport].
type linuxTransport struct {
	fd     int
	path   string
	desc   Descriptor
	closed atomic.Bool

	claimMu sync.Mutex
	claimed []int

	bulkMu        sync.Mutex
	bulkActive    bool
	bulkSlots     []*bulkSlot
	bulkSubmitted int
	bulkStopFlag  atomic.Int32
	bulkDone      chan struct{}
}

type bulkSlot struct {
	urb *usbdevfsURB
	buf []byte
}

func (t *linuxTransport) ControlIn(bRequest uint8, wValue, wIndex uint16, n int, timeoutMs int) ([]byte, error) {
	if t.closed.Load() {
		return nil, ErrClosed
	}
	if n < 0 || n > 0xFFFF {
		return nil, fmt.Errorf("usbdevfs: control IN length %d out of range", n)
	}
	var buf []byte
	var dataPtr *byte
	if n > 0 {
		buf = make([]byte, n)
		dataPtr = &buf[0]
	}
	ctrl := usbdevfsCtrltransfer{
		BmRequestType: VendorIn,
		BRequest:      bRequest,
		WValue:        wValue,
		WIndex:        wIndex,
		WLength:       uint16(n),
		Timeout:       uint32(timeoutMs),
		Data:          dataPtr,
	}
	ret, _, errno := unix.Syscall(unix.SYS_IOCTL, uintptr(t.fd), usbdevfsControl, uintptr(unsafe.Pointer(&ctrl)))
	if errno != 0 {
		return nil, translateErrno(errno)
	}
	return buf[:int(ret)], nil
}

func (t *linuxTransport) ControlOut(bRequest uint8, wValue, wIndex uint16, data []byte, timeoutMs int) error {
	if t.closed.Load() {
		return ErrClosed
	}
	if len(data) > 0xFFFF {
		return fmt.Errorf("usbdevfs: control OUT length %d out of range", len(data))
	}
	var dataPtr *byte
	if len(data) > 0 {
		dataPtr = &data[0]
	}
	ctrl := usbdevfsCtrltransfer{
		BmRequestType: VendorOut,
		BRequest:      bRequest,
		WValue:        wValue,
		WIndex:        wIndex,
		WLength:       uint16(len(data)),
		Timeout:       uint32(timeoutMs),
		Data:          dataPtr,
	}
	_, _, errno := unix.Syscall(unix.SYS_IOCTL, uintptr(t.fd), usbdevfsControl, uintptr(unsafe.Pointer(&ctrl)))
	if errno != 0 {
		return translateErrno(errno)
	}
	return nil
}

// ClaimInterface tells the kernel this transport owns the given USB
// interface. If the first attempt fails with EBUSY a kernel driver is
// bound to the interface — on RTL-SDR dongles that is dvb_usb_rtl28xxu,
// the DVB-T TV-tuner driver the kernel binds at plug time. We detach it
// and retry once, mirroring libusb's LIBUSB_OPTION_AUTO_DETACH_KERNEL_-
// DRIVER, so GopherTrunk claims the dongle even when the operator never
// blacklisted the module. The driver is intentionally left detached:
// re-binding it would only let it re-grab the interface and reintroduce
// the race on the next open.
func (t *linuxTransport) ClaimInterface(num int) error {
	if t.closed.Load() {
		return ErrClosed
	}
	if err := claimWithAutoDetach(
		func() error { return t.claimInterface(num) },
		func() error { return t.detachKernelDriver(num) },
	); err != nil {
		return err
	}
	t.claimMu.Lock()
	t.claimed = append(t.claimed, num)
	t.claimMu.Unlock()
	return nil
}

// claimInterface issues a single USBDEVFS_CLAIMINTERFACE ioctl with no
// retry and no bookkeeping — the raw building block ClaimInterface
// drives through claimWithAutoDetach.
func (t *linuxTransport) claimInterface(num int) error {
	n := uint32(num)
	_, _, errno := unix.Syscall(unix.SYS_IOCTL, uintptr(t.fd), usbdevfsClaimInterface, uintptr(unsafe.Pointer(&n)))
	if errno != 0 {
		return translateErrno(errno)
	}
	return nil
}

// detachKernelDriver issues USBDEVFS_DISCONNECT against interface num
// via the USBDEVFS_IOCTL wrapper, telling the kernel to unbind whatever
// driver currently owns it. Mirrors libusb_detach_kernel_driver.
func (t *linuxTransport) detachKernelDriver(num int) error {
	arg := usbdevfsIoctlArg{
		Ifno:      int32(num),
		IoctlCode: int32(usbdevfsDisconnect),
	}
	_, _, errno := unix.Syscall(unix.SYS_IOCTL, uintptr(t.fd), usbdevfsIoctlCmd, uintptr(unsafe.Pointer(&arg)))
	if errno != 0 {
		return translateErrno(errno)
	}
	return nil
}

// claimWithAutoDetach runs claim; on EBUSY (a kernel driver owns the
// interface) it runs detach once and retries claim a single time.
// Factored out of ClaimInterface as a pure function so the EBUSY →
// detach → re-claim policy is unit-testable without a real usbdevfs
// device node.
func claimWithAutoDetach(claim, detach func() error) error {
	err := claim()
	if !errors.Is(err, unix.EBUSY) {
		return err
	}
	if detachErr := detach(); detachErr != nil {
		return fmt.Errorf("usbdevfs: interface busy, kernel-driver auto-detach failed: %w (original: %w)", detachErr, err)
	}
	return claim()
}

func (t *linuxTransport) ReleaseInterface(num int) error {
	if t.closed.Load() {
		return nil
	}
	n := uint32(num)
	_, _, errno := unix.Syscall(unix.SYS_IOCTL, uintptr(t.fd), usbdevfsReleaseInterface, uintptr(unsafe.Pointer(&n)))
	t.claimMu.Lock()
	for i, c := range t.claimed {
		if c == num {
			t.claimed = append(t.claimed[:i], t.claimed[i+1:]...)
			break
		}
	}
	t.claimMu.Unlock()
	if errno != 0 && errno != unix.ENODEV {
		return translateErrno(errno)
	}
	return nil
}

func (t *linuxTransport) Reset() error {
	if t.closed.Load() {
		return ErrClosed
	}
	_, _, errno := unix.Syscall(unix.SYS_IOCTL, uintptr(t.fd), usbdevfsReset, 0)
	if errno != 0 {
		return translateErrno(errno)
	}
	return nil
}

func (t *linuxTransport) StartBulkIn(epAddr byte, ringBufs, bufLen int, onPacket func([]byte), onStreamDead func(error)) error {
	if t.closed.Load() {
		return ErrClosed
	}
	if ringBufs <= 0 || bufLen <= 0 {
		return fmt.Errorf("usbdevfs: invalid bulk ring geometry (bufs=%d len=%d)", ringBufs, bufLen)
	}
	t.bulkMu.Lock()
	defer t.bulkMu.Unlock()
	if t.bulkActive {
		return ErrBulkActive
	}
	slots := make([]*bulkSlot, 0, ringBufs)
	for i := 0; i < ringBufs; i++ {
		buf := make([]byte, bufLen)
		urb := &usbdevfsURB{
			Type:         usbdevfsURBTypeBULK,
			Endpoint:     epAddr,
			Buffer:       &buf[0],
			BufferLength: int32(bufLen),
		}
		_, _, errno := unix.Syscall(unix.SYS_IOCTL, uintptr(t.fd), usbdevfsSubmitURB, uintptr(unsafe.Pointer(urb)))
		if errno != 0 {
			for _, s := range slots {
				_, _, _ = unix.Syscall(unix.SYS_IOCTL, uintptr(t.fd), usbdevfsDiscardURB, uintptr(unsafe.Pointer(s.urb)))
			}
			t.drainSubmitted(len(slots))
			return fmt.Errorf("usbdevfs: SUBMITURB[%d]: %w", i, translateErrno(errno))
		}
		slots = append(slots, &bulkSlot{urb: urb, buf: buf})
	}
	t.bulkSlots = slots
	t.bulkSubmitted = len(slots)
	t.bulkActive = true
	t.bulkStopFlag.Store(0)
	t.bulkDone = make(chan struct{})

	go t.reapLoop(onPacket, onStreamDead, slots, t.bulkSubmitted, t.bulkDone)
	return nil
}

// drainSubmitted reaps the given number of URBs without dispatching them;
// used to recover from a partial StartBulkIn failure.
func (t *linuxTransport) drainSubmitted(n int) {
	for i := 0; i < n; i++ {
		var ptr uintptr
		_, _, errno := unix.Syscall(unix.SYS_IOCTL, uintptr(t.fd), usbdevfsReapURB, uintptr(unsafe.Pointer(&ptr)))
		if errno != 0 && errno != unix.EINTR {
			return
		}
	}
}

func (t *linuxTransport) reapLoop(onPacket func([]byte), onStreamDead func(error), slots []*bulkSlot, submitted int, done chan struct{}) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	defer close(done)
	// firstErr keeps the first non-EINTR errno we see — either a ReapURB
	// failure or a resubmit failure. When the loop exits with the stop
	// flag unset (i.e. every URB died of an unrecoverable error rather
	// than being aborted by StopBulkIn), we hand it to onStreamDead so
	// the driver can surface "stream died" to its IQ consumer.
	var firstErr error
	remaining := submitted
	for remaining > 0 {
		var ptr uintptr
		_, _, errno := unix.Syscall(unix.SYS_IOCTL, uintptr(t.fd), usbdevfsReapURB, uintptr(unsafe.Pointer(&ptr)))
		if errno != 0 {
			if errno == unix.EINTR {
				continue
			}
			if firstErr == nil {
				firstErr = fmt.Errorf("usbdevfs: REAPURB: %w", translateErrno(errno))
			}
			break
		}
		// The kernel returns the address of the URB we previously submitted.
		// Don't deref the raw uintptr — look it up against our slot ring
		// (whose entries Go's GC keeps alive) and use that pointer instead.
		slot := findSlotByAddr(slots, ptr)
		if slot == nil {
			continue
		}
		urb := slot.urb
		stop := t.bulkStopFlag.Load() != 0
		if !stop {
			if urb.Status == 0 && urb.ActualLength > 0 {
				onPacket(slot.buf[:urb.ActualLength])
			}
			urb.ActualLength = 0
			urb.Status = 0
			_, _, errno = unix.Syscall(unix.SYS_IOCTL, uintptr(t.fd), usbdevfsSubmitURB, uintptr(unsafe.Pointer(urb)))
			if errno != 0 {
				if firstErr == nil {
					firstErr = fmt.Errorf("usbdevfs: SUBMITURB resubmit: %w", translateErrno(errno))
				}
				remaining--
			}
			continue
		}
		remaining--
	}
	if t.bulkStopFlag.Load() == 0 && onStreamDead != nil {
		if firstErr == nil {
			firstErr = ErrDeviceGone
		}
		// Dispatch from a fresh goroutine: onStreamDead typically calls
		// StopBulkIn (via the driver's cancel path) which waits on
		// `done` — which is only closed after this function returns.
		go onStreamDead(firstErr)
	}
}

func (t *linuxTransport) StopBulkIn() error {
	t.bulkMu.Lock()
	if !t.bulkActive {
		t.bulkMu.Unlock()
		return ErrBulkInactive
	}
	t.bulkStopFlag.Store(1)
	slots := t.bulkSlots
	done := t.bulkDone
	t.bulkActive = false
	t.bulkMu.Unlock()

	for _, s := range slots {
		_, _, _ = unix.Syscall(unix.SYS_IOCTL, uintptr(t.fd), usbdevfsDiscardURB, uintptr(unsafe.Pointer(s.urb)))
	}
	<-done

	t.bulkMu.Lock()
	t.bulkSlots = nil
	t.bulkSubmitted = 0
	t.bulkMu.Unlock()
	return nil
}

func (t *linuxTransport) Close() error {
	if !t.closed.CompareAndSwap(false, true) {
		return nil
	}
	t.bulkMu.Lock()
	active := t.bulkActive
	t.bulkMu.Unlock()
	if active {
		// Reset the closed flag transiently so StopBulkIn can run.
		t.closed.Store(false)
		_ = t.StopBulkIn()
		t.closed.Store(true)
	}
	t.claimMu.Lock()
	for _, num := range t.claimed {
		n := uint32(num)
		_, _, _ = unix.Syscall(unix.SYS_IOCTL, uintptr(t.fd), usbdevfsReleaseInterface, uintptr(unsafe.Pointer(&n)))
	}
	t.claimed = nil
	t.claimMu.Unlock()
	if t.fd >= 0 {
		unix.Close(t.fd)
		t.fd = -1
	}
	return nil
}

func findSlotByAddr(slots []*bulkSlot, addr uintptr) *bulkSlot {
	for _, s := range slots {
		if uintptr(unsafe.Pointer(s.urb)) == addr {
			return s
		}
	}
	return nil
}

func translateErrno(errno syscall.Errno) error {
	switch errno {
	case unix.ENODEV, unix.ENXIO, unix.ESHUTDOWN:
		return ErrDeviceGone
	case unix.ETIMEDOUT:
		return ErrTimeout
	default:
		return errno
	}
}

// ----------------------------------------------------------------------
// sysfs helpers

func readTrim(path string) string {
	b, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(b))
}

func readHex16(path string) (uint16, bool) {
	s := readTrim(path)
	if s == "" {
		return 0, false
	}
	v, err := strconv.ParseUint(s, 16, 16)
	if err != nil {
		return 0, false
	}
	return uint16(v), true
}

func readUint8(path string) (uint8, bool) {
	s := readTrim(path)
	if s == "" {
		return 0, false
	}
	v, err := strconv.ParseUint(s, 10, 8)
	if err != nil {
		return 0, false
	}
	return uint8(v), true
}
