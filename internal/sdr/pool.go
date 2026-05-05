package sdr

import (
	"errors"
	"fmt"
	"log/slog"
	"sync"
)

// PoolEntry tracks a single discovered-and-opened device along with its role.
type PoolEntry struct {
	Driver Driver
	Device Device
	Info   Info
	Role   Role
}

// Pool holds a fleet of opened SDR devices and assigns roles.
type Pool struct {
	mu      sync.RWMutex
	entries []*PoolEntry
	log     *slog.Logger
}

func NewPool(logger *slog.Logger) *Pool {
	if logger == nil {
		logger = slog.Default()
	}
	return &Pool{log: logger}
}

// Hint guides role assignment when opening devices. Match by serial first;
// fall back to first-found.
type Hint struct {
	Serial string
	Role   Role
}

// Open enumerates every registered driver, opens devices that match the
// supplied hints (or simply all of them when hints is empty), and assigns
// roles. The first opened device gets RoleControl unless a hint says
// otherwise; subsequent devices get RoleVoice.
func (p *Pool) Open(hints []Hint) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if len(p.entries) > 0 {
		return errors.New("pool already populated; close first")
	}

	type discovered struct {
		drv  Driver
		info Info
	}
	var all []discovered
	for _, drv := range Drivers() {
		infos, err := drv.Enumerate()
		if err != nil {
			p.log.Warn("driver enumerate failed", "driver", drv.Name(), "err", err)
			continue
		}
		for _, info := range infos {
			all = append(all, discovered{drv, info})
		}
	}
	if len(all) == 0 {
		return errors.New("no SDR devices discovered")
	}

	hintBySerial := map[string]Hint{}
	controlClaimed := false
	for _, h := range hints {
		if h.Serial != "" {
			hintBySerial[h.Serial] = h
			if h.Role == RoleControl {
				controlClaimed = true
			}
		}
	}

	for _, d := range all {
		role := RoleAuto
		if h, ok := hintBySerial[d.info.Serial]; ok {
			role = h.Role
		}
		if role == RoleAuto {
			if !controlClaimed {
				role = RoleControl
				controlClaimed = true
			} else {
				role = RoleVoice
			}
		}
		dev, err := d.drv.Open(d.info.Index)
		if err != nil {
			p.log.Error("open device failed", "driver", d.drv.Name(), "index", d.info.Index, "err", err)
			continue
		}
		p.entries = append(p.entries, &PoolEntry{Driver: d.drv, Device: dev, Info: d.info, Role: role})
		p.log.Info("device opened", "driver", d.drv.Name(), "serial", d.info.Serial, "role", role.String())
	}
	if len(p.entries) == 0 {
		return errors.New("no SDR devices opened")
	}
	return nil
}

func (p *Pool) Entries() []*PoolEntry {
	p.mu.RLock()
	defer p.mu.RUnlock()
	out := make([]*PoolEntry, len(p.entries))
	copy(out, p.entries)
	return out
}

// FirstByRole returns the first device with the given role, or nil.
func (p *Pool) FirstByRole(r Role) *PoolEntry {
	p.mu.RLock()
	defer p.mu.RUnlock()
	for _, e := range p.entries {
		if e.Role == r {
			return e
		}
	}
	return nil
}

// AllByRole returns every device with the given role.
func (p *Pool) AllByRole(r Role) []*PoolEntry {
	p.mu.RLock()
	defer p.mu.RUnlock()
	var out []*PoolEntry
	for _, e := range p.entries {
		if e.Role == r {
			out = append(out, e)
		}
	}
	return out
}

// FindBySerial returns the entry whose info.Serial matches, or nil.
// Used by the demod-pipeline composer to look up a Voice device that
// the engine has just bound to a call.
func (p *Pool) FindBySerial(serial string) *PoolEntry {
	p.mu.RLock()
	defer p.mu.RUnlock()
	for _, e := range p.entries {
		if e.Info.Serial == serial {
			return e
		}
	}
	return nil
}

func (p *Pool) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	var errs []error
	for _, e := range p.entries {
		if err := e.Device.Close(); err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", e.Info.Serial, err))
		}
	}
	p.entries = nil
	return errors.Join(errs...)
}
