package lorawan

import "sync"

// SessionKeys are the per-device LoRaWAN session keys an operator supplies.
type SessionKeys struct {
	NwkSKey [16]byte
	AppSKey [16]byte
}

// KeyStore maps a DevAddr to its session keys. It is safe for concurrent
// use. An empty store decodes MAC headers but verifies no MICs and decrypts
// nothing.
type KeyStore struct {
	mu       sync.RWMutex
	byDevice map[uint32]SessionKeys
}

// NewKeyStore returns an empty store.
func NewKeyStore() *KeyStore {
	return &KeyStore{byDevice: make(map[uint32]SessionKeys)}
}

// Add registers session keys for a device address.
func (ks *KeyStore) Add(devAddr uint32, keys SessionKeys) {
	ks.mu.Lock()
	defer ks.mu.Unlock()
	ks.byDevice[devAddr] = keys
}

// Lookup returns the keys for a device address.
func (ks *KeyStore) Lookup(devAddr uint32) (SessionKeys, bool) {
	ks.mu.RLock()
	defer ks.mu.RUnlock()
	k, ok := ks.byDevice[devAddr]
	return k, ok
}

// Decoded is the result of decoding a LoRaWAN PHY payload: the parsed
// message plus any key-dependent results.
type Decoded struct {
	*Message
	HasKeys   bool   // session keys were found for this DevAddr
	MICOK     bool   // MIC verified against the network session key
	Decrypted bool   // FRMPayload was decrypted
	Plaintext []byte // decrypted FRMPayload (nil when not decrypted)
}

// Decode parses a LoRa PHY payload as LoRaWAN and, when keys are known for
// the device, verifies the MIC and decrypts the frame payload. It returns
// an error only when the bytes are not a well-formed LoRaWAN frame.
func (ks *KeyStore) Decode(phy []byte) (*Decoded, error) {
	msg, err := Parse(phy)
	if err != nil {
		return nil, err
	}
	d := &Decoded{Message: msg}
	if !msg.MType.IsData() {
		return d, nil
	}
	keys, ok := ks.Lookup(msg.DevAddr)
	if !ok {
		return d, nil
	}
	d.HasKeys = true
	d.MICOK = msg.VerifyMIC(keys.NwkSKey)
	if len(msg.FRMPayload) > 0 {
		key := keys.AppSKey
		if msg.FPort == 0 {
			key = keys.NwkSKey // FPort 0 → MAC commands, encrypted with NwkSKey
		}
		d.Plaintext = msg.DecryptFRMPayload(key)
		d.Decrypted = true
	}
	return d, nil
}
