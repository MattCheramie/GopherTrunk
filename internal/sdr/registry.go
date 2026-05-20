package sdr

import (
	"fmt"
	"sort"
	"sync"
)

var (
	registryMu sync.RWMutex
	registry   = map[string]Driver{}
)

func Register(d Driver) {
	registryMu.Lock()
	defer registryMu.Unlock()
	registry[d.Name()] = d
}

func Drivers() []Driver {
	registryMu.RLock()
	defer registryMu.RUnlock()
	out := make([]Driver, 0, len(registry))
	for _, d := range registry {
		out = append(out, d)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name() < out[j].Name() })
	return out
}

func DriverByName(name string) (Driver, error) {
	registryMu.RLock()
	defer registryMu.RUnlock()
	d, ok := registry[name]
	if !ok {
		return nil, fmt.Errorf("sdr: unknown driver %q", name)
	}
	return d, nil
}

// EnumerateAll asks every registered driver to list its devices. It
// returns the combined device list plus one error per driver that
// failed to enumerate, so callers can surface the failure instead of
// silently reporting an empty list.
func EnumerateAll() ([]Info, []error) {
	var out []Info
	var errs []error
	for _, d := range Drivers() {
		infos, err := d.Enumerate()
		if err != nil {
			errs = append(errs, err)
			continue
		}
		out = append(out, infos...)
	}
	return out, errs
}
