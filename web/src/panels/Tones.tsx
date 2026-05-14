import { useMemo, useState } from "react";
import { writes } from "../api/write";
import { Column, DataTable } from "../components/DataTable";
import { ConfirmModal } from "../components/ConfirmModal";
import type { EventDTO, ToneAlertDTO } from "../api/types";
import { selectCanMutate, selectClientConfig, useShared } from "../store/shared";

interface ToneRow {
  alert: ToneAlertDTO;
  receivedAt: string;
}

// Tones renders a filtered view of the live event stream for
// tone.alert events. Mirrors internal/tui/panels/tones.go. The
// reset button is gated by selectCanMutate so it's invisible until
// the user has both opted into write mode AND the daemon allows it.
export function Tones() {
  const cfg = useShared(selectClientConfig);
  const canMutate = useShared(selectCanMutate);
  const setError = useShared((s) => s.setError);
  const events = useShared((s) => s.events);

  const [confirmReset, setConfirmReset] = useState<string | null>(null);
  const [selected, setSelected] = useState<ToneRow | null>(null);

  const toneRows = useMemo<ToneRow[]>(() => {
    return events
      .filter((e) => e.kind === "tone.alert")
      .map((e) => parseToneAlert(e))
      .filter((t): t is ToneRow => t !== null)
      .reverse(); // newest first
  }, [events]);

  const columns: Column<ToneRow>[] = useMemo(
    () => [
      {
        key: "time",
        header: "Matched",
        render: (r) => (
          <span className="font-mono text-xs text-muted whitespace-nowrap">
            {r.alert.matched_at.replace("T", " ").replace(/\..*$/, "")}
          </span>
        ),
        sort: (a, b) => a.alert.matched_at.localeCompare(b.alert.matched_at),
      },
      {
        key: "profile",
        header: "Profile",
        render: (r) => (
          <span className="font-medium">{r.alert.profile}</span>
        ),
        sort: (a, b) => a.alert.profile.localeCompare(b.alert.profile),
      },
      {
        key: "tag",
        header: "Alpha tag",
        render: (r) =>
          r.alert.alpha_tag ? (
            <span className="text-sm">{r.alert.alpha_tag}</span>
          ) : (
            <span className="text-muted">—</span>
          ),
        sort: (a, b) =>
          (a.alert.alpha_tag ?? "").localeCompare(b.alert.alpha_tag ?? ""),
        className: "hidden md:table-cell",
        headerClassName: "hidden md:table-cell",
      },
      {
        key: "device",
        header: "Device",
        render: (r) => (
          <span className="font-mono text-xs">{r.alert.device_serial}</span>
        ),
        sort: (a, b) =>
          a.alert.device_serial.localeCompare(b.alert.device_serial),
      },
      {
        key: "freqs",
        header: "Frequencies (Hz)",
        render: (r) => (
          <span className="font-mono text-xs text-muted">
            {r.alert.frequencies_hz.map((f) => f.toFixed(1)).join(" · ")}
          </span>
        ),
      },
    ],
    [],
  );

  async function resetDevice(serial: string) {
    try {
      await writes.toneReset(cfg, serial);
    } catch (e: unknown) {
      setError(
        e instanceof Error ? e.message : "tone-reset request failed",
      );
      throw e;
    }
    setConfirmReset(null);
  }

  const deviceSerials = useMemo(() => {
    const seen = new Set<string>();
    for (const r of toneRows) seen.add(r.alert.device_serial);
    return Array.from(seen).sort();
  }, [toneRows]);

  return (
    <div className="space-y-3">
      <header className="flex items-center justify-between gap-3">
        <h2 className="text-xl font-semibold">Tone alerts</h2>
        <span className="text-xs text-muted">{toneRows.length} captured</span>
      </header>

      {canMutate && deviceSerials.length > 0 && (
        <div className="panel p-3 flex flex-wrap items-center gap-2">
          <span className="text-xs text-muted uppercase tracking-wider">
            Reset detector
          </span>
          {deviceSerials.map((serial) => (
            <button
              key={serial}
              className="btn-ghost text-xs"
              onClick={() => setConfirmReset(serial)}
            >
              {serial}
            </button>
          ))}
        </div>
      )}

      <DataTable
        rows={toneRows}
        columns={columns}
        rowKey={(r, i) => `${r.alert.matched_at}-${r.alert.device_serial}-${i}`}
        defaultSortKey="time"
        defaultSortDirection="desc"
        onRowClick={(r) => setSelected(r)}
        emptyMessage="No tone alerts yet. They arrive live over the WebSocket as tone-out detectors fire."
      />

      {selected && (
        <ToneDetail row={selected} onClose={() => setSelected(null)} />
      )}

      {confirmReset && (
        <ConfirmModal
          title={`Reset ${confirmReset}?`}
          message={`Clear the tone-out match progress on device ${confirmReset}. Subsequent detections start from scratch.`}
          confirmLabel="Reset"
          onConfirm={() => resetDevice(confirmReset)}
          onCancel={() => setConfirmReset(null)}
        />
      )}
    </div>
  );
}

function ToneDetail({ row, onClose }: { row: ToneRow; onClose: () => void }) {
  return (
    <div
      role="dialog"
      aria-modal="true"
      className="fixed inset-0 z-50 flex items-end sm:items-stretch sm:justify-end bg-black/50 backdrop-blur-sm"
      onClick={onClose}
    >
      <div
        className="panel w-full sm:max-w-md sm:h-full bg-bg p-5 overflow-auto rounded-t-lg sm:rounded-none sm:rounded-l-lg"
        onClick={(e) => e.stopPropagation()}
      >
        <header className="flex items-start gap-3 mb-4">
          <div className="flex-1">
            <h3 className="text-lg font-semibold">{row.alert.profile}</h3>
            <p className="text-xs text-muted">
              {row.alert.matched_at.replace("T", " ").replace(/\..*$/, "")}
            </p>
          </div>
          <button
            className="btn-ghost !min-h-0 !p-1.5 text-xs"
            onClick={onClose}
            aria-label="Close"
          >
            ✕
          </button>
        </header>
        <div className="space-y-3">
          <Field label="Device" value={row.alert.device_serial} mono />
          <Field label="Alpha tag" value={row.alert.alpha_tag} />
          <Field label="System" value={row.alert.system} />
          <Field label="Group ID" value={row.alert.group_id ?? null} mono />
          <Field
            label="Frequencies (Hz)"
            mono
            value={row.alert.frequencies_hz.map((f) => f.toFixed(2)).join(" · ")}
          />
        </div>
      </div>
    </div>
  );
}

function Field({
  label,
  value,
  mono,
}: {
  label: string;
  value: React.ReactNode;
  mono?: boolean;
}) {
  return (
    <div>
      <p className="text-xs uppercase tracking-wider text-muted">{label}</p>
      <p className={`text-sm mt-0.5 ${mono ? "font-mono" : ""}`}>
        {value === null || value === undefined || value === "" ? (
          <span className="text-muted">—</span>
        ) : (
          value
        )}
      </p>
    </div>
  );
}

function parseToneAlert(ev: EventDTO): ToneRow | null {
  if (!ev.payload || typeof ev.payload !== "object") return null;
  const p = ev.payload as Partial<ToneAlertDTO>;
  if (!p.profile || !p.device_serial || !Array.isArray(p.frequencies_hz)) {
    return null;
  }
  return {
    alert: {
      profile: p.profile,
      alpha_tag: p.alpha_tag,
      system: p.system,
      group_id: p.group_id,
      device_serial: p.device_serial,
      matched_at: p.matched_at ?? ev.timestamp,
      frequencies_hz: p.frequencies_hz,
    },
    receivedAt: ev.timestamp,
  };
}
