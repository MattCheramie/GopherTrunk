//go:build darwin

package usb

import (
	"errors"
	"fmt"
	"os"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"

	"github.com/ebitengine/purego"
)

// Asynchronous macOS bulk-IN path.
//
// The legacy path (usb_darwin.go bulkLoop) runs one OS-thread-pinned goroutine
// per ring slot, each blocking in a synchronous ReadPipe on the SAME pipe. That
// is not a supported IOUSBLib usage: under sustained streaming macOS aborts the
// concurrent synchronous reads with kIOReturnAborted (0xe00002eb), independent
// of data rate (reproduced on an Airspy R2 at both 2.5 and 10 MS/s). SDRtrunk
// streams the same hardware fine because libusb uses asynchronous transfers.
//
// This path mirrors that model: it queues a ring of ReadPipeAsync transfers and
// services their completions on a single dedicated CFRunLoop thread, re-arming
// each transfer as it completes so the kernel always has the full ring
// outstanding. One process-wide IOAsyncCallback1 (created once via purego) routes
// each completion — keyed by an integer refcon, never a Go pointer — back to the
// owning transport and slot.

// syncBulkEnabled reports whether the operator forced the legacy synchronous
// reaper path via GT_USB_SYNC_BULK (a truthy value). Default (unset/empty) uses
// the async path.
func syncBulkEnabled() bool {
	switch os.Getenv("GT_USB_SYNC_BULK") {
	case "", "0", "false", "no", "off":
		return false
	default:
		return true
	}
}

// Async completion routing. IOKit hands the callback only the integer refcon we
// registered, so a process-wide registry maps refcon → slot. The single C
// callback is created once (purego.NewCallback registrations are permanent, so
// re-creating per stream would leak the callback pool).
var (
	asyncReg    sync.Map // uintptr refcon -> *darwinBulkSlot
	asyncRefctr atomic.Uintptr
	asyncCBOnce sync.Once
	asyncCBPtr  uintptr
)

func nextAsyncRefcon() uintptr {
	// Start at 1 so a zero refcon (never registered) can't accidentally match.
	return asyncRefctr.Add(1)
}

// usbRunLoopMode returns a private CFRunLoop mode (CFStringRef) for the async USB
// event source, created once. A dedicated mode — rather than the
// kCFRunLoopDefaultMode global — is used deliberately: reading that global means
// dereferencing its Dlsym'd address (a bare uintptr→unsafe.Pointer conversion go
// vet flags), whereas creating our own mode string is clean. The source is both
// added to and run in this mode, so completions are delivered normally.
var (
	usbModeOnce sync.Once
	usbModeRef  uintptr
)

func usbRunLoopMode() uintptr {
	usbModeOnce.Do(func() {
		name := append([]byte("GopherTrunkUSBBulk"), 0)
		usbModeRef = cfStringCreateWithCString(kCFAllocatorDefault, &name[0], kCFStringEncodingUTF8)
	})
	return usbModeRef
}

// asyncCompletionCallback returns the process-wide IOAsyncCallback1 trampoline,
// creating it once. IOAsyncCallback1 is void(*)(void *refcon, IOReturn result,
// void *arg0); for ReadPipeAsync arg0 is the byte count transferred. The Go
// callback returns uintptr(0) — IOKit expects void and ignores the return.
func asyncCompletionCallback() uintptr {
	asyncCBOnce.Do(func() {
		asyncCBPtr = purego.NewCallback(func(refcon, result, arg0 uintptr) uintptr {
			if v, ok := asyncReg.Load(refcon); ok {
				s := v.(*darwinBulkSlot)
				s.t.onAsyncComplete(s, result, arg0)
			}
			return 0
		})
	})
	return asyncCBPtr
}

// startBulkInAsyncLocked launches the async streaming path. Called under bulkMu
// from StartBulkIn with the slots, pipe, and stream-death/onPacket callbacks
// already installed on t. On error the caller unwinds the shared bulk state.
func (t *darwinTransport) startBulkInAsyncLocked() error {
	if cfRunLoopGetCurrent == nil || cfRunLoopRunInMode == nil ||
		cfRunLoopAddSource == nil || cfStringCreateWithCString == nil {
		return errors.New("usb: CoreFoundation run-loop symbols unavailable for async bulk-IN")
	}
	if usbRunLoopMode() == 0 {
		return errors.New("usb: could not create CFRunLoop mode for async bulk-IN")
	}
	// Create the async event source once per transport and reuse it across
	// streams; it is released in releaseAsyncSource (ReleaseInterface/Close).
	if t.bulkAsyncSource == 0 {
		var src uintptr
		rc := vtableCall(t.ifaceIface, ifaceCreateInterfaceAsyncEventSource,
			uintptr(unsafe.Pointer(&src)))
		if err := translateIOReturn(rc); err != nil {
			return fmt.Errorf("usb: CreateInterfaceAsyncEventSource: %w", err)
		}
		if src == 0 {
			return errors.New("usb: CreateInterfaceAsyncEventSource returned a nil source")
		}
		t.bulkAsyncSource = src
	}
	// Register every slot so the shared completion callback can route to it.
	for _, s := range t.bulkSlots {
		s.refcon = nextAsyncRefcon()
		asyncReg.Store(s.refcon, s)
	}
	t.bulkDone = make(chan struct{})
	ready := make(chan error, 1)
	go t.bulkAsyncRunLoop(ready)
	if err := <-ready; err != nil {
		for _, s := range t.bulkSlots {
			asyncReg.Delete(s.refcon)
		}
		return err
	}
	return nil
}

// bulkAsyncRunLoop owns the CFRunLoop that services async completions. It pins
// its OS thread (CFRunLoop is thread-local), publishes the run-loop ref for
// teardown, adds the async event source, submits the initial ring, then runs
// bounded run-loop slices until StopBulkIn sets the stop flag.
func (t *darwinTransport) bulkAsyncRunLoop(ready chan<- error) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	defer close(t.bulkDone)

	mode := usbRunLoopMode()
	rl := cfRunLoopGetCurrent()
	t.bulkRunLoopRef.Store(rl)
	cfRunLoopAddSource(rl, t.bulkAsyncSource, mode)
	defer func() {
		cfRunLoopRemoveSource(rl, t.bulkAsyncSource, mode)
		t.bulkRunLoopRef.Store(0)
	}()

	// Submit the initial ring. A submit failure here aborts whatever was queued
	// and fails the StartBulkIn.
	for i, s := range t.bulkSlots {
		if rc := t.submitAsyncRead(s); uint32(rc) != kIOReturnSuccess {
			if i > 0 && t.ifaceIface != 0 {
				vtableCall(t.ifaceIface, ifaceAbortPipe, uintptr(t.bulkPipeRef))
			}
			ready <- fmt.Errorf("usb: ReadPipeAsync submit: 0x%08x", uint32(rc))
			return
		}
	}
	ready <- nil

	// Service completions in bounded slices, checking the stop flag between
	// them. 0.25 s keeps teardown latency low; when idle (e.g. after a stream
	// death, before StopBulkIn) the slice simply times out and loops.
	for t.bulkStopFlag.Load() == 0 {
		cfRunLoopRunInMode(mode, 0.25, false)
	}
}

// submitAsyncRead arms one ReadPipeAsync transfer for slot s. Returns the raw
// IOReturn.
func (t *darwinTransport) submitAsyncRead(s *darwinBulkSlot) uintptr {
	return vtableCall(t.ifaceIface, ifaceReadPipeAsync,
		uintptr(s.pipeRef),
		uintptr(unsafe.Pointer(&s.buf[0])),
		uintptr(uint32(len(s.buf))),
		asyncCompletionCallback(),
		s.refcon,
	)
}

// onAsyncComplete handles one ReadPipeAsync completion (on the run-loop thread).
// On success it delivers the payload and re-arms the slot; on error or during
// teardown it retires the slot. bulkOnPacket must not block — it runs on the
// single servicing thread, so a stall there stalls every slot (the Airspy driver
// copies + hands off, keeping it O(1)).
func (t *darwinTransport) onAsyncComplete(s *darwinBulkSlot, result, arg0 uintptr) {
	if t.bulkStopFlag.Load() != 0 {
		t.asyncSlotDone(s)
		return
	}
	if uint32(result) != kIOReturnSuccess {
		// The stall watchdog records ErrStreamStalled before it AbortPipes, and
		// recordBulkErr is first-wins, so a watchdog abort keeps that cause; any
		// other kernel error surfaces its raw code.
		t.recordBulkErr(fmt.Errorf("usb: ReadPipeAsync: 0x%08x", uint32(result)))
		t.asyncSlotDone(s)
		return
	}
	if n := int(arg0); n > 0 {
		t.markBulkActivity(s.idx, n)
		if t.bulkOnPacket != nil {
			t.bulkOnPacket(s.buf[:n])
		}
	}
	if rc := t.submitAsyncRead(s); uint32(rc) != kIOReturnSuccess {
		if t.bulkStopFlag.Load() == 0 {
			t.recordBulkErr(fmt.Errorf("usb: ReadPipeAsync resubmit: 0x%08x", uint32(rc)))
		}
		t.asyncSlotDone(s)
	}
}

// asyncSlotDone retires one slot from the outstanding ring. When the last slot
// retires with the stop flag unset, the stream died on its own (device gone /
// stall / kernel error): fire bulkOnDead exactly once with the recorded cause so
// the driver surfaces EOF and the daemon reacquires — the issue-#345 path.
func (t *darwinTransport) asyncSlotDone(s *darwinBulkSlot) {
	if t.bulkAlive.Add(-1) != 0 {
		return
	}
	if t.bulkStopFlag.Load() != 0 || t.bulkOnDead == nil {
		return
	}
	t.bulkDeadOnce.Do(func() {
		t.bulkErrMu.Lock()
		err := t.bulkDeadErr
		t.bulkErrMu.Unlock()
		if err == nil {
			err = ErrDeviceGone
		}
		// Dispatch from a fresh goroutine: bulkOnDead typically calls StopBulkIn,
		// which waits on the run loop this callback is running under.
		go t.bulkOnDead(err)
	})
}

// stopBulkInAsync tears down the async stream: abort + reset the pipe (which
// completes outstanding transfers), let the run-loop goroutine observe the stop
// flag and exit, then drop the slot registrations. Called from StopBulkIn after
// the stop flag is set and the watchdog is disarmed.
func (t *darwinTransport) stopBulkInAsync(pipeRef uint8) error {
	if t.ifaceIface != 0 {
		// AbortPipe completes every outstanding async transfer with
		// kIOReturnAborted; ResetPipe clears the halt so the next StartBulkIn
		// works without re-Open.
		vtableCall(t.ifaceIface, ifaceAbortPipe, uintptr(pipeRef))
		vtableCall(t.ifaceIface, ifaceResetPipe, uintptr(pipeRef))
	}
	var err error
	if t.bulkDone != nil {
		select {
		case <-t.bulkDone:
		case <-time.After(2 * time.Second):
			err = errors.New("usb: async bulk run loop did not exit within 2 s")
		}
	}
	for _, s := range t.bulkSlots {
		asyncReg.Delete(s.refcon)
	}
	return err
}

// releaseAsyncSource drops the per-transport async event source. Called from
// ReleaseInterface/Close before the interface handle goes away; safe to call
// when no source was created.
func (t *darwinTransport) releaseAsyncSource() {
	if t.bulkAsyncSource != 0 {
		cfRelease(t.bulkAsyncSource)
		t.bulkAsyncSource = 0
	}
}
