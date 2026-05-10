package rtl2832u

import "fmt"

// SetI2CRepeater enables/disables the demod's I2C bridge so the
// tuner-driver layer (R820T/R820T2/E4000/...) can talk to its chip
// through the same USB control endpoint. librtlsdr toggles this
// around every tuner I2C burst — pulling the chain low between
// bursts so the demod doesn't compete with the tuner for the bus.
//
// Caches the last-pushed value; calling repeatedly with the same
// argument is free.
func (d *Demod) SetI2CRepeater(on bool) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.repON == on {
		return nil
	}
	val := uint16(0x10)
	if on {
		val = 0x18
	}
	if err := d.writeDemodRegLocked(1, 0x01, val, 1); err != nil {
		return fmt.Errorf("rtl2832u: SetI2CRepeater(%v): %w", on, err)
	}
	d.repON = on
	return nil
}

// I2CWrite issues a bulk write to the tuner over the I2C bridge.
// Matches rtlsdr_i2c_write — addr is the 7-bit tuner I2C address; the
// 8th bit is the read/write flag the chip injects automatically.
func (d *Demod) I2CWrite(i2cAddr uint8, data []byte) error {
	if len(data) == 0 {
		return nil
	}
	if len(data) > 0xFFFF {
		return fmt.Errorf("rtl2832u: I2CWrite length %d out of range", len(data))
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.i2cWriteLocked(i2cAddr, data)
}

func (d *Demod) i2cWriteLocked(i2cAddr uint8, data []byte) error {
	index := uint16(BlockIIC)<<8 | 0x10
	if err := d.t.ControlOut(0, uint16(i2cAddr), index, data, CtrlTimeoutMs); err != nil {
		return fmt.Errorf("rtl2832u: I2CWrite addr=0x%02x: %w", i2cAddr, err)
	}
	return nil
}

// I2CRead bulk-reads n bytes from the tuner. The caller is expected to
// have already issued an I2CWrite with the register pointer if the
// tuner uses register-addressed access (R820T-family does; FC0012
// uses bare reads).
func (d *Demod) I2CRead(i2cAddr uint8, n int) ([]byte, error) {
	if n <= 0 {
		return nil, nil
	}
	if n > 0xFFFF {
		return nil, fmt.Errorf("rtl2832u: I2CRead length %d out of range", n)
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.i2cReadLocked(i2cAddr, n)
}

func (d *Demod) i2cReadLocked(i2cAddr uint8, n int) ([]byte, error) {
	index := uint16(BlockIIC) << 8
	out, err := d.t.ControlIn(0, uint16(i2cAddr), index, n, CtrlTimeoutMs)
	if err != nil {
		return nil, fmt.Errorf("rtl2832u: I2CRead addr=0x%02x: %w", i2cAddr, err)
	}
	return out, nil
}

// I2CWriteReg writes a single byte to a tuner register over the bridge.
// Matches rtlsdr_i2c_write_reg: emits a 2-byte payload (reg, val).
func (d *Demod) I2CWriteReg(i2cAddr, reg, val uint8) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.i2cWriteLocked(i2cAddr, []byte{reg, val})
}

// I2CReadReg reads a single byte from a tuner register: write the
// register pointer, then read 1 byte. Matches rtlsdr_i2c_read_reg.
func (d *Demod) I2CReadReg(i2cAddr, reg uint8) (uint8, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if err := d.i2cWriteLocked(i2cAddr, []byte{reg}); err != nil {
		return 0, err
	}
	out, err := d.i2cReadLocked(i2cAddr, 1)
	if err != nil {
		return 0, err
	}
	if len(out) == 0 {
		return 0, fmt.Errorf("rtl2832u: I2CReadReg(0x%02x, 0x%02x): short read", i2cAddr, reg)
	}
	return out[0], nil
}
