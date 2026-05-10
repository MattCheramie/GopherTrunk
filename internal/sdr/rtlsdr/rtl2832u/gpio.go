package rtl2832u

import "fmt"

// SetGPIOOutput configures the given GPIO pin (0..7 on the chip's
// system block) as an output. Bias-tee on RTL-SDR.com v3+ dongles
// lives on GPIO 0; clones map it elsewhere.
func (d *Demod) SetGPIOOutput(gpio uint8) error {
	if gpio > 7 {
		return fmt.Errorf("rtl2832u: GPIO pin %d out of range", gpio)
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.setGPIOOutputLocked(gpio)
}

func (d *Demod) setGPIOOutputLocked(gpio uint8) error {
	// Read-modify-write on GPD (direction) and GPOE (output enable).
	gpd, err := d.readBlockRegLocked(BlockSys, SysGPD, 1)
	if err != nil {
		return err
	}
	if err := d.writeBlockRegLocked(BlockSys, SysGPD, uint16(gpd[0]&^(1<<gpio)), 1); err != nil {
		return err
	}
	gpoe, err := d.readBlockRegLocked(BlockSys, SysGPOE, 1)
	if err != nil {
		return err
	}
	return d.writeBlockRegLocked(BlockSys, SysGPOE, uint16(gpoe[0]|(1<<gpio)), 1)
}

// SetGPIOBit drives the given GPIO pin high or low. The pin must
// already be configured as an output via SetGPIOOutput.
func (d *Demod) SetGPIOBit(gpio uint8, high bool) error {
	if gpio > 7 {
		return fmt.Errorf("rtl2832u: GPIO pin %d out of range", gpio)
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.setGPIOBitLocked(gpio, high)
}

func (d *Demod) setGPIOBitLocked(gpio uint8, high bool) error {
	gpo, err := d.readBlockRegLocked(BlockSys, SysGPO, 1)
	if err != nil {
		return err
	}
	v := gpo[0]
	if high {
		v |= 1 << gpio
	} else {
		v &^= 1 << gpio
	}
	return d.writeBlockRegLocked(BlockSys, SysGPO, uint16(v), 1)
}

// SetBiasTee turns the 5 V LNA-power output on or off. The actual GPIO
// pin varies per dongle (the RTL-SDR.com v3+ uses pin 0; NESDR uses
// pin 4 on some revisions); the driver layer (PR-06) plumbs the right
// pin in based on Info.Product. Provided as a convenience here so the
// tuner code can flip it without rederiving the read-modify-write.
func (d *Demod) SetBiasTee(gpio uint8, on bool) error {
	if err := d.SetGPIOOutput(gpio); err != nil {
		return fmt.Errorf("rtl2832u: SetBiasTee: configure output: %w", err)
	}
	return d.SetGPIOBit(gpio, on)
}
