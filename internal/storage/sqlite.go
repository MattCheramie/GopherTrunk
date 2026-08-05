package storage

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

// DB is a thin wrapper over *sql.DB that lets the call-log + retention
// helpers share a typed handle. The schema is migrated on Open.
type DB struct {
	sql *sql.DB
}

// Open creates (or opens) a SQLite database at path and applies the
// embedded schema migrations. The path's parent directory is created
// if missing.
//
// `:memory:` and the standard "file:..." DSN forms are passed through
// to the driver — useful for tests.
func Open(path string) (*DB, error) {
	if path == "" {
		return nil, errors.New("storage: db path is required")
	}
	if path != ":memory:" && !looksLikeDSN(path) {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return nil, fmt.Errorf("storage: mkdir: %w", err)
		}
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("storage: open: %w", err)
	}
	// SQLite is single-writer; cap to one connection so the call-log
	// writer doesn't fight the retention sweeper.
	db.SetMaxOpenConns(1)
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("storage: ping: %w", err)
	}
	d := &DB{sql: db}
	if err := d.migrate(); err != nil {
		db.Close()
		return nil, err
	}
	return d, nil
}

// Close releases the connection.
func (d *DB) Close() error { return d.sql.Close() }

// SQL returns the underlying *sql.DB. Exposed so tests and future
// integrations (an /api/v1/calls/history handler, etc.) can run their
// own queries without adding a method here for every shape.
func (d *DB) SQL() *sql.DB { return d.sql }

func looksLikeDSN(s string) bool {
	return len(s) >= 5 && s[:5] == "file:"
}

const schema = `
CREATE TABLE IF NOT EXISTS schema_version (
    version INTEGER PRIMARY KEY
);

CREATE TABLE IF NOT EXISTS call_log (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    system          TEXT    NOT NULL,
    protocol        TEXT    NOT NULL DEFAULT '',
    group_id        INTEGER NOT NULL,
    source_id       INTEGER NOT NULL DEFAULT 0,
    frequency_hz    INTEGER NOT NULL DEFAULT 0,
    encrypted       INTEGER NOT NULL DEFAULT 0,
    algorithm_id    INTEGER NOT NULL DEFAULT 0,
    key_id          INTEGER NOT NULL DEFAULT 0,
    emergency       INTEGER NOT NULL DEFAULT 0,
    data_call       INTEGER NOT NULL DEFAULT 0,
    individual      INTEGER NOT NULL DEFAULT 0,  -- 1 = unit-to-unit / private call (group_id is a target radio, not a talkgroup)
    timeslot        INTEGER NOT NULL DEFAULT 0,  -- DMR TDMA slot: 0=n/a, 1=TS1, 2=TS2
    priority        INTEGER NOT NULL DEFAULT 0,  -- on-air signalled call priority (service-options low 3 bits)
    device_serial   TEXT    NOT NULL,
    source_alpha    TEXT,  -- resolved alias/name of the source radio (RID); NULL = unresolved
    started_at      INTEGER NOT NULL,  -- unix nanoseconds
    ended_at        INTEGER,
    duration_ms     INTEGER,
    end_reason      TEXT,
    talkgroup_alpha TEXT,
    signal_dbfs     REAL,  -- mean received channel power, dBFS; NULL = unmeasured
    evm_pct         REAL,  -- demod RMS error-vector magnitude, %; NULL = unmeasured
    snr_db          REAL   -- demod estimated symbol SNR, dB; NULL = unmeasured
);

CREATE INDEX IF NOT EXISTS idx_call_log_started ON call_log(started_at);
CREATE INDEX IF NOT EXISTS idx_call_log_system  ON call_log(system, started_at);
CREATE INDEX IF NOT EXISTS idx_call_log_group   ON call_log(group_id, started_at);
CREATE UNIQUE INDEX IF NOT EXISTS idx_call_log_active ON call_log(device_serial, started_at);

CREATE TABLE IF NOT EXISTS location_log (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    system      TEXT    NOT NULL DEFAULT '',
    protocol    TEXT    NOT NULL DEFAULT '',
    radio_id    INTEGER NOT NULL DEFAULT 0,
    talkgroup   INTEGER NOT NULL DEFAULT 0,
    latitude    REAL    NOT NULL,
    longitude   REAL    NOT NULL,
    speed_knots REAL    NOT NULL DEFAULT 0,
    heading_deg REAL    NOT NULL DEFAULT 0,
    reported_at INTEGER NOT NULL  -- unix nanoseconds
);

CREATE INDEX IF NOT EXISTS idx_location_log_time  ON location_log(reported_at);
CREATE INDEX IF NOT EXISTS idx_location_log_radio ON location_log(radio_id, reported_at);

-- Operator-managed conventional channel bookmarks: UI-editable shortlist
-- of "where do I park the SDR" frequencies that complements the YAML
-- scanner.conventional list. Each row carries enough metadata to
-- click-to-tune from the spectrum waterfall and to filter against a
-- channel-group navigator.
CREATE TABLE IF NOT EXISTS bookmarks (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    name        TEXT    NOT NULL,
    freq_hz     INTEGER NOT NULL,
    mode        TEXT    NOT NULL DEFAULT 'FM',
    ctcss_hz    REAL    NOT NULL DEFAULT 0,
    dcs_code    INTEGER NOT NULL DEFAULT 0,
    notes       TEXT    NOT NULL DEFAULT '',
    grouping    TEXT    NOT NULL DEFAULT '',  -- "marine" / "ham-2m" / "utility" — operator-defined
    created_at  INTEGER NOT NULL,             -- unix nanoseconds
    updated_at  INTEGER NOT NULL              -- unix nanoseconds
);

CREATE INDEX IF NOT EXISTS idx_bookmarks_freq  ON bookmarks(freq_hz);
CREATE INDEX IF NOT EXISTS idx_bookmarks_group ON bookmarks(grouping, name);

-- POCSAG pager messages persisted from the decoder pipeline. Each
-- row is one fully-reassembled page (address codeword + N message
-- codewords). The function field is the 2-bit code from the
-- address codeword (A/B/C/D); encoding distinguishes numeric vs.
-- alphanumeric.
CREATE TABLE IF NOT EXISTS pager_log (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    received_at INTEGER NOT NULL,                  -- unix nanoseconds
    protocol    TEXT    NOT NULL DEFAULT 'pocsag', -- "pocsag" | "flex"
    ric         INTEGER NOT NULL,                  -- 21-bit address / capcode
    func        INTEGER NOT NULL,                  -- 0..3 (A..D); 0 for FLEX
    encoding    TEXT    NOT NULL DEFAULT '',        -- "numeric" | "alpha" | "tone"
    body        TEXT    NOT NULL DEFAULT '',
    corrected   INTEGER NOT NULL DEFAULT 0          -- total BCH bit-error count
);

CREATE INDEX IF NOT EXISTS idx_pager_log_time ON pager_log(received_at);
CREATE INDEX IF NOT EXISTS idx_pager_log_ric  ON pager_log(ric, received_at);

-- APRS / AX.25 packets persisted from the decoder pipeline. Each
-- row is one decoded frame: AX.25 envelope (src + dst + path) plus
-- the decoded APRS sub-payload (type tag + extracted fields).
CREATE TABLE IF NOT EXISTS aprs_log (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    received_at INTEGER NOT NULL,            -- unix nanoseconds
    src         TEXT    NOT NULL DEFAULT '', -- "W1AW-9"
    dst         TEXT    NOT NULL DEFAULT '', -- "APRS"
    path        TEXT    NOT NULL DEFAULT '', -- "WIDE1-1,WIDE2-1*"
    type        TEXT    NOT NULL DEFAULT '', -- "position" | "message" | "status" | ...
    body        TEXT    NOT NULL DEFAULT '', -- type-specific summary
    latitude    REAL    NOT NULL DEFAULT 0,
    longitude   REAL    NOT NULL DEFAULT 0,
    raw_info    TEXT    NOT NULL DEFAULT '', -- raw AX.25 info-field bytes (as ASCII)
    fcs_ok      INTEGER NOT NULL DEFAULT 0
);

CREATE INDEX IF NOT EXISTS idx_aprs_log_time ON aprs_log(received_at);
CREATE INDEX IF NOT EXISTS idx_aprs_log_src  ON aprs_log(src, received_at);

-- AIS / vessel messages persisted from the decoder pipeline. One
-- row per decoded message: MMSI + type tag + position (lat/lon/COG/
-- SOG/heading) for position-bearing types + static data (vessel
-- name, callsign, destination, ship type, IMO) for static types.
-- Wire-bit payload preserved as hex on raw_hex for round-tripping
-- into offline decoders.
CREATE TABLE IF NOT EXISTS vessel_log (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    received_at  INTEGER NOT NULL,             -- unix nanoseconds
    mmsi         INTEGER NOT NULL DEFAULT 0,   -- 9-digit Maritime Mobile Service Identity
    type         TEXT    NOT NULL DEFAULT '',  -- "position-a" | "position-b" | "static-voyage" | ...
    body         TEXT    NOT NULL DEFAULT '',  -- type-specific summary
    latitude     REAL    NOT NULL DEFAULT 0,
    longitude    REAL    NOT NULL DEFAULT 0,
    sog          REAL    NOT NULL DEFAULT 0,   -- speed over ground (knots)
    cog          REAL    NOT NULL DEFAULT 0,   -- course over ground (degrees)
    heading      INTEGER NOT NULL DEFAULT 511, -- true heading (511 = not available)
    has_position INTEGER NOT NULL DEFAULT 0,
    vessel_name  TEXT    NOT NULL DEFAULT '',
    callsign     TEXT    NOT NULL DEFAULT '',
    destination  TEXT    NOT NULL DEFAULT '',
    ship_type    INTEGER NOT NULL DEFAULT 0,
    imo          INTEGER NOT NULL DEFAULT 0,
    raw_hex      TEXT    NOT NULL DEFAULT '',
    fcs_ok       INTEGER NOT NULL DEFAULT 0
);

CREATE INDEX IF NOT EXISTS idx_vessel_log_time ON vessel_log(received_at);
CREATE INDEX IF NOT EXISTS idx_vessel_log_mmsi ON vessel_log(mmsi, received_at);

-- LoRa frames persisted from the decoder pipeline. One row per decoded
-- PHY frame: the physical parameters (spreading factor, coding rate,
-- bandwidth), link metrics (RSSI / SNR / carrier offset), the CRC flag and
-- the de-whitened payload (hex), plus any LoRaWAN MAC fields recovered from
-- it (message type, DevAddr, frame counter, port, MIC-valid, decrypted).
CREATE TABLE IF NOT EXISTS lora_log (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    received_at  INTEGER NOT NULL,             -- unix nanoseconds
    frequency_hz INTEGER NOT NULL DEFAULT 0,   -- sub-channel centre frequency
    sf           INTEGER NOT NULL DEFAULT 0,   -- spreading factor 7..12
    cr           INTEGER NOT NULL DEFAULT 0,   -- coding rate 1..4 (4/5..4/8)
    bandwidth_hz INTEGER NOT NULL DEFAULT 0,
    rssi         REAL    NOT NULL DEFAULT 0,
    snr          REAL    NOT NULL DEFAULT 0,
    cfo_hz       REAL    NOT NULL DEFAULT 0,   -- estimated carrier offset
    crc_ok       INTEGER NOT NULL DEFAULT 0,
    payload_hex  TEXT    NOT NULL DEFAULT '',  -- de-whitened PHY payload
    is_lorawan   INTEGER NOT NULL DEFAULT 0,
    mtype        TEXT    NOT NULL DEFAULT '',  -- LoRaWAN message type
    dev_addr     TEXT    NOT NULL DEFAULT '',  -- LoRaWAN device address (hex)
    fcnt         INTEGER NOT NULL DEFAULT 0,   -- LoRaWAN frame counter
    fport        INTEGER NOT NULL DEFAULT -1,  -- LoRaWAN port (-1 = absent)
    mic_ok       INTEGER NOT NULL DEFAULT 0,   -- MIC verified against a key
    decrypted    INTEGER NOT NULL DEFAULT 0,   -- FRMPayload was decrypted
    decoded      TEXT    NOT NULL DEFAULT ''   -- decoded payload summary
);

CREATE INDEX IF NOT EXISTS idx_lora_log_time ON lora_log(received_at);
CREATE INDEX IF NOT EXISTS idx_lora_log_devaddr ON lora_log(dev_addr, received_at);

-- DSC (Digital Selective Calling) sequences persisted from the
-- decoder pipeline. One row per decoded sequence: format
-- (distress / all-ships / individual / ...), category, source +
-- target MMSI, plus distress-specific fields (nature, time,
-- position) when applicable.
CREATE TABLE IF NOT EXISTS dsc_log (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    received_at  INTEGER NOT NULL,             -- unix nanoseconds
    format       TEXT    NOT NULL DEFAULT '',  -- "distress" | "all-ships" | "individual" | ...
    category     TEXT    NOT NULL DEFAULT '',  -- "distress" | "urgency" | "safety" | "routine"
    self_mmsi    INTEGER NOT NULL DEFAULT 0,   -- sender's 9-digit MMSI
    target_mmsi  INTEGER NOT NULL DEFAULT 0,   -- recipient's MMSI (0 for all-ships)
    nature       TEXT    NOT NULL DEFAULT '',  -- distress nature ("fire", "sinking", ...)
    time_utc     TEXT    NOT NULL DEFAULT '',  -- HH:MM, distress only
    latitude     REAL    NOT NULL DEFAULT 0,
    longitude    REAL    NOT NULL DEFAULT 0,
    has_position INTEGER NOT NULL DEFAULT 0,
    body         TEXT    NOT NULL DEFAULT '',  -- type-specific summary
    raw_hex      TEXT    NOT NULL DEFAULT ''   -- hex-encoded 7-bit symbol stream
);

CREATE INDEX IF NOT EXISTS idx_dsc_log_time      ON dsc_log(received_at);
CREATE INDEX IF NOT EXISTS idx_dsc_log_self_mmsi ON dsc_log(self_mmsi, received_at);

-- M17 link-setup frames persisted from the decoder pipeline. One row
-- per reassembled Link Setup Frame: source / destination callsigns,
-- the mode (voice / data / packet), channel-access number, hex META
-- block, and the CRC-valid flag.
CREATE TABLE IF NOT EXISTS m17_log (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    received_at INTEGER NOT NULL,            -- unix nanoseconds
    src         TEXT    NOT NULL DEFAULT '', -- source callsign
    dst         TEXT    NOT NULL DEFAULT '', -- destination / "BROADCAST"
    mode        TEXT    NOT NULL DEFAULT '', -- "voice" | "data" | "packet" | ...
    can         INTEGER NOT NULL DEFAULT 0,  -- channel-access number
    meta        TEXT    NOT NULL DEFAULT '', -- hex-encoded META block
    crc_ok      INTEGER NOT NULL DEFAULT 0,
    body        TEXT    NOT NULL DEFAULT ''
);

CREATE INDEX IF NOT EXISTS idx_m17_log_time ON m17_log(received_at);
CREATE INDEX IF NOT EXISTS idx_m17_log_src  ON m17_log(src, received_at);

-- ADS-B / Mode-S frames persisted from the decoder pipeline. One
-- row per decoded extended-squitter message: ICAO 24-bit address +
-- kind (identification / airborne-pos / velocity / ...) + kind-
-- specific fields (callsign + category for identification; lat/lon
-- + altitude for position; ground speed + track + vertical rate
-- for velocity).
CREATE TABLE IF NOT EXISTS aircraft_log (
    id                INTEGER PRIMARY KEY AUTOINCREMENT,
    received_at       INTEGER NOT NULL,             -- unix nanoseconds
    icao              INTEGER NOT NULL DEFAULT 0,   -- 24-bit ICAO aircraft address
    icao_hex          TEXT    NOT NULL DEFAULT '',  -- "4840D6" form for display
    kind              TEXT    NOT NULL DEFAULT '',  -- "ident" | "airborne-pos" | "velocity" | ...
    body              TEXT    NOT NULL DEFAULT '',  -- kind-specific summary
    crc_valid         INTEGER NOT NULL DEFAULT 0,
    callsign          TEXT    NOT NULL DEFAULT '',
    category          INTEGER NOT NULL DEFAULT 0,
    latitude          REAL    NOT NULL DEFAULT 0,
    longitude         REAL    NOT NULL DEFAULT 0,
    altitude_ft       INTEGER NOT NULL DEFAULT 0,
    has_position      INTEGER NOT NULL DEFAULT 0,
    has_altitude      INTEGER NOT NULL DEFAULT 0,
    ground_speed_kn   INTEGER NOT NULL DEFAULT 0,
    track_deg         REAL    NOT NULL DEFAULT 0,
    vertical_rate_fpm INTEGER NOT NULL DEFAULT 0,
    raw_hex           TEXT    NOT NULL DEFAULT ''
);

CREATE INDEX IF NOT EXISTS idx_aircraft_log_time ON aircraft_log(received_at);
CREATE INDEX IF NOT EXISTS idx_aircraft_log_icao ON aircraft_log(icao, received_at);

-- MDC1200 signaling bursts persisted from the decoder pipeline. One
-- row per decoded burst: the transmitting radio's unit ID + the
-- operation (PTT ID / emergency / status / radio check / ...) decoded
-- from the op/arg pair, plus whether the CRC validated.
CREATE TABLE IF NOT EXISTS mdc1200_log (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    received_at INTEGER NOT NULL,             -- unix nanoseconds
    op          INTEGER NOT NULL DEFAULT 0,   -- operation code (data[0])
    arg         INTEGER NOT NULL DEFAULT 0,   -- operation argument (data[1])
    unit_id     INTEGER NOT NULL DEFAULT 0,   -- transmitting radio's unit ID
    operation   TEXT    NOT NULL DEFAULT '',  -- "PTT ID" | "Emergency" | ... ("" if unknown)
    body        TEXT    NOT NULL DEFAULT '',  -- one-line summary
    raw_hex     TEXT    NOT NULL DEFAULT '',  -- hex of the decoded header bytes
    crc_ok      INTEGER NOT NULL DEFAULT 0    -- 1 when the CRC validated
);

CREATE INDEX IF NOT EXISTS idx_mdc1200_log_time    ON mdc1200_log(received_at);
CREATE INDEX IF NOT EXISTS idx_mdc1200_log_unit_id ON mdc1200_log(unit_id, received_at);
`

func (d *DB) migrate() error {
	_, err := d.sql.Exec(schema)
	if err != nil {
		return fmt.Errorf("storage: migrate: %w", err)
	}
	if err := d.ensureCallLogColumns(); err != nil {
		return err
	}
	if err := d.ensureColumns("pager_log", []columnAdd{
		{"protocol", `ALTER TABLE pager_log ADD COLUMN protocol TEXT NOT NULL DEFAULT 'pocsag'`},
	}); err != nil {
		return err
	}
	// Stamp the current schema version; future migrations check this
	// row before running.
	_, _ = d.sql.Exec(`INSERT OR IGNORE INTO schema_version(version) VALUES (2)`)
	return nil
}

// ensureCallLogColumns adds call_log columns introduced after the
// initial schema. CREATE TABLE IF NOT EXISTS never alters an existing
// table, so a database created by an earlier GopherTrunk keeps the old
// column set; this brings it forward. It is idempotent — columns
// already present (fresh databases, repeat opens) are skipped — so it
// needs no schema-version gate.
func (d *DB) ensureCallLogColumns() error {
	rows, err := d.sql.Query(`PRAGMA table_info(call_log)`)
	if err != nil {
		return fmt.Errorf("storage: inspect call_log: %w", err)
	}
	have := make(map[string]bool)
	for rows.Next() {
		var (
			cid, notnull, pk int
			name, ctype      string
			dflt             sql.NullString
		)
		if err := rows.Scan(&cid, &name, &ctype, &notnull, &dflt, &pk); err != nil {
			rows.Close()
			return fmt.Errorf("storage: inspect call_log: %w", err)
		}
		have[name] = true
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return fmt.Errorf("storage: inspect call_log: %w", err)
	}
	rows.Close()

	adds := []struct{ name, ddl string }{
		{"algorithm_id", `ALTER TABLE call_log ADD COLUMN algorithm_id INTEGER NOT NULL DEFAULT 0`},
		{"key_id", `ALTER TABLE call_log ADD COLUMN key_id INTEGER NOT NULL DEFAULT 0`},
		{"timeslot", `ALTER TABLE call_log ADD COLUMN timeslot INTEGER NOT NULL DEFAULT 0`},
		{"priority", `ALTER TABLE call_log ADD COLUMN priority INTEGER NOT NULL DEFAULT 0`},
		{"source_alpha", `ALTER TABLE call_log ADD COLUMN source_alpha TEXT`},
		// Nullable (no DEFAULT) so a row with no measurement reads NULL —
		// "unset" is distinct from a legitimate 0 dBFS reading.
		{"signal_dbfs", `ALTER TABLE call_log ADD COLUMN signal_dbfs REAL`},
		// Demod quality (EVM % / SNR dB), likewise nullable — NULL when the
		// call's chain fed no demod taps (every protocol but P25 Phase 1).
		{"evm_pct", `ALTER TABLE call_log ADD COLUMN evm_pct REAL`},
		{"snr_db", `ALTER TABLE call_log ADD COLUMN snr_db REAL`},
		{"individual", `ALTER TABLE call_log ADD COLUMN individual INTEGER NOT NULL DEFAULT 0`},
	}
	for _, a := range adds {
		if have[a.name] {
			continue
		}
		if _, err := d.sql.Exec(a.ddl); err != nil {
			return fmt.Errorf("storage: add call_log.%s: %w", a.name, err)
		}
	}
	return nil
}

// columnAdd pairs a column name with the ALTER TABLE that introduces
// it, for ensureColumns.
type columnAdd struct{ name, ddl string }

// ensureColumns brings an existing table forward with columns added
// after its initial schema. CREATE TABLE IF NOT EXISTS never alters an
// existing table, so a database created by an earlier GopherTrunk
// keeps the old column set; this adds any missing columns. Idempotent
// — present columns are skipped — so it needs no schema-version gate.
func (d *DB) ensureColumns(table string, adds []columnAdd) error {
	rows, err := d.sql.Query(`PRAGMA table_info(` + table + `)`)
	if err != nil {
		return fmt.Errorf("storage: inspect %s: %w", table, err)
	}
	have := make(map[string]bool)
	for rows.Next() {
		var (
			cid, notnull, pk int
			name, ctype      string
			dflt             sql.NullString
		)
		if err := rows.Scan(&cid, &name, &ctype, &notnull, &dflt, &pk); err != nil {
			rows.Close()
			return fmt.Errorf("storage: inspect %s: %w", table, err)
		}
		have[name] = true
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return fmt.Errorf("storage: inspect %s: %w", table, err)
	}
	rows.Close()
	for _, a := range adds {
		if have[a.name] {
			continue
		}
		if _, err := d.sql.Exec(a.ddl); err != nil {
			return fmt.Errorf("storage: add %s.%s: %w", table, a.name, err)
		}
	}
	return nil
}
