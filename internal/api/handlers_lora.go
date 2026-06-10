package api

import (
	"net/http"
	"strconv"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/storage"
)

// LoRaProvider is the read surface the lora-log endpoint consumes. The
// daemon implements it on top of storage.LoRaLog; tests substitute a fake.
type LoRaProvider interface {
	RecentLoRaFrames(limit int) ([]storage.LoRaFrame, error)
}

// LoRaFrameDTO is the JSON wire shape for the lora-log endpoint.
type LoRaFrameDTO struct {
	ID          int64     `json:"id"`
	ReceivedAt  time.Time `json:"received_at"`
	FrequencyHz uint32    `json:"frequency_hz"`
	SF          int       `json:"sf"`
	CR          int       `json:"cr"`
	BandwidthHz int       `json:"bandwidth_hz"`
	RSSI        float64   `json:"rssi"`
	SNR         float64   `json:"snr"`
	CFOHz       float64   `json:"cfo_hz"`
	CRCOK       bool      `json:"crc_ok"`
	PayloadHex  string    `json:"payload_hex,omitempty"`

	IsLoRaWAN bool   `json:"is_lorawan"`
	MType     string `json:"mtype,omitempty"`
	DevAddr   string `json:"dev_addr,omitempty"`
	FCnt      uint32 `json:"fcnt"`
	FPort     int    `json:"fport"`
	MICOK     bool   `json:"mic_ok"`
	Decrypted bool   `json:"decrypted"`
	Decoded   string `json:"decoded,omitempty"`
}

func loRaFrameToDTO(f storage.LoRaFrame) LoRaFrameDTO {
	return LoRaFrameDTO{
		ID:          f.ID,
		ReceivedAt:  f.ReceivedAt,
		FrequencyHz: f.FrequencyHz,
		SF:          f.SF,
		CR:          f.CR,
		BandwidthHz: f.BandwidthHz,
		RSSI:        f.RSSI,
		SNR:         f.SNR,
		CFOHz:       f.CFOHz,
		CRCOK:       f.CRCOK,
		PayloadHex:  f.PayloadHex,
		IsLoRaWAN:   f.IsLoRaWAN,
		MType:       f.MType,
		DevAddr:     f.DevAddr,
		FCnt:        f.FCnt,
		FPort:       f.FPort,
		MICOK:       f.MICOK,
		Decrypted:   f.Decrypted,
		Decoded:     f.DecodedJS,
	}
}

// handleLoRaFrames answers GET /api/v1/lora/frames. Optional ?limit=
// (default 200, max 5000). 503 when the storage layer isn't wired (daemon
// started without storage.path).
func (s *Server) handleLoRaFrames(w http.ResponseWriter, r *http.Request) {
	if s.lora == nil {
		s.writeError(w, http.StatusServiceUnavailable, "lora subsystem not enabled (set storage.path in config to persist and view decoded frames)")
		return
	}
	limit := 200
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			limit = n
		}
	}
	rows, err := s.lora.RecentLoRaFrames(limit)
	if err != nil {
		s.log.Error("api: lora frames", "err", err)
		s.writeError(w, http.StatusInternalServerError, "query failed")
		return
	}
	out := make([]LoRaFrameDTO, 0, len(rows))
	for _, row := range rows {
		out = append(out, loRaFrameToDTO(row))
	}
	writeJSON(w, http.StatusOK, out)
}
