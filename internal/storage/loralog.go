// LoRa frame log writer — drains KindLoRaFrame events off the shared bus
// and writes one row per decoded PHY frame to the SQLite lora_log table.
// Mirrors m17log.go / dsclog.go.
package storage

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/events"
)

// LoRaFrame is one persisted LoRa frame: the PHY parameters and payload
// plus any LoRaWAN MAC fields recovered from it. A frame with HasCRC=false
// carries CRCOK=true (nothing to check). The LoRaWAN fields are zero/empty
// for non-LoRaWAN traffic (IsLoRaWAN=false).
type LoRaFrame struct {
	ID          int64     `json:"id"`
	ReceivedAt  time.Time `json:"received_at"`
	FrequencyHz uint32    `json:"frequency_hz"` // sub-channel centre frequency
	SF          int       `json:"sf"`           // spreading factor 7..12
	CR          int       `json:"cr"`           // coding rate 1..4 (4/5..4/8)
	BandwidthHz int       `json:"bandwidth_hz"`
	RSSI        float64   `json:"rssi"`
	SNR         float64   `json:"snr"`
	CFOHz       float64   `json:"cfo_hz"`
	CRCOK       bool      `json:"crc_ok"`
	PayloadHex  string    `json:"payload_hex"` // de-whitened PHY payload

	// LoRaWAN MAC fields (populated when IsLoRaWAN is true).
	IsLoRaWAN bool   `json:"is_lorawan"`
	MType     string `json:"mtype"`     // e.g. "unconfirmed-up", "join-request"
	DevAddr   string `json:"dev_addr"`  // hex, network data frames
	FCnt      uint32 `json:"fcnt"`      // frame counter
	FPort     int    `json:"fport"`     // -1 when absent
	MICOK     bool   `json:"mic_ok"`    // MIC verified against a known key
	Decrypted bool   `json:"decrypted"` // FRMPayload was decrypted
	DecodedJS string `json:"decoded"`   // best-effort decoded payload summary
}

// LoRaLog drains KindLoRaFrame events until ctx cancels or the bus closes.
type LoRaLog struct {
	db        *DB
	bus       *events.Bus
	log       *slog.Logger
	sub       *events.Subscription
	runDone   chan struct{}
	closeOnce sync.Once
}

// NewLoRaLog wires the log to the bus. Subscription happens at construction
// so frames published before Run() begins aren't lost.
func NewLoRaLog(db *DB, bus *events.Bus, logger *slog.Logger) (*LoRaLog, error) {
	if db == nil {
		return nil, errors.New("storage/loralog: DB is required")
	}
	if bus == nil {
		return nil, errors.New("storage/loralog: events.Bus is required")
	}
	if logger == nil {
		logger = slog.Default()
	}
	l := &LoRaLog{db: db, bus: bus, log: logger, runDone: make(chan struct{})}
	l.sub = bus.Subscribe()
	return l, nil
}

// Run drains KindLoRaFrame events until ctx cancels or the bus closes.
func (l *LoRaLog) Run(ctx context.Context) error {
	defer close(l.runDone)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case ev, ok := <-l.sub.C:
			if !ok {
				return nil
			}
			if ev.Kind != events.KindLoRaFrame {
				continue
			}
			f, ok := ev.Payload.(LoRaFrame)
			if !ok {
				continue
			}
			if err := l.insert(f); err != nil {
				l.log.Error("loralog: insert failed", "err", err)
			}
		}
	}
}

func b2i(b bool) int {
	if b {
		return 1
	}
	return 0
}

func (l *LoRaLog) insert(f LoRaFrame) error {
	at := f.ReceivedAt
	if at.IsZero() {
		at = time.Now()
	}
	_, err := l.db.SQL().Exec(
		`INSERT INTO lora_log
		 (received_at, frequency_hz, sf, cr, bandwidth_hz, rssi, snr, cfo_hz,
		  crc_ok, payload_hex, is_lorawan, mtype, dev_addr, fcnt, fport,
		  mic_ok, decrypted, decoded)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		at.UnixNano(), f.FrequencyHz, f.SF, f.CR, f.BandwidthHz, f.RSSI, f.SNR,
		f.CFOHz, b2i(f.CRCOK), f.PayloadHex, b2i(f.IsLoRaWAN), f.MType,
		f.DevAddr, f.FCnt, f.FPort, b2i(f.MICOK), b2i(f.Decrypted), f.DecodedJS,
	)
	return err
}

// Recent returns the most recent frames, newest first, capped at limit.
// limit ≤ 0 picks 200; limit > 5000 caps at 5000.
func (l *LoRaLog) Recent(limit int) ([]LoRaFrame, error) {
	if limit <= 0 {
		limit = 200
	}
	if limit > 5000 {
		limit = 5000
	}
	rows, err := l.db.SQL().Query(
		`SELECT id, received_at, frequency_hz, sf, cr, bandwidth_hz, rssi, snr,
		        cfo_hz, crc_ok, payload_hex, is_lorawan, mtype, dev_addr, fcnt,
		        fport, mic_ok, decrypted, decoded
		 FROM lora_log ORDER BY received_at DESC LIMIT ?`, limit)
	if err != nil {
		return nil, fmt.Errorf("storage/loralog: query: %w", err)
	}
	defer rows.Close()
	var out []LoRaFrame
	for rows.Next() {
		var (
			f                             LoRaFrame
			ns                            int64
			crcOK, isLW, micOK, decrypted int
		)
		if err := rows.Scan(&f.ID, &ns, &f.FrequencyHz, &f.SF, &f.CR,
			&f.BandwidthHz, &f.RSSI, &f.SNR, &f.CFOHz, &crcOK, &f.PayloadHex,
			&isLW, &f.MType, &f.DevAddr, &f.FCnt, &f.FPort, &micOK,
			&decrypted, &f.DecodedJS); err != nil {
			return nil, fmt.Errorf("storage/loralog: scan: %w", err)
		}
		f.ReceivedAt = time.Unix(0, ns)
		f.CRCOK = crcOK != 0
		f.IsLoRaWAN = isLW != 0
		f.MICOK = micOK != 0
		f.Decrypted = decrypted != 0
		out = append(out, f)
	}
	return out, rows.Err()
}

// Close releases the bus subscription and waits for Run to drain.
func (l *LoRaLog) Close() error {
	l.closeOnce.Do(func() {
		l.sub.Close()
		select {
		case <-l.runDone:
		case <-time.After(time.Second):
		}
	})
	return nil
}
