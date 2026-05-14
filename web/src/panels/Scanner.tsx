import { useEffect, useState } from "react";
import { api } from "../api/client";
import { writes } from "../api/write";
import { ConfirmModal } from "../components/ConfirmModal";
import type {
  ConvChannelStatusDTO,
  SystemHuntStatusDTO,
} from "../api/types";
import { selectCanMutate, selectClientConfig, useShared } from "../store/shared";

const POLL_INTERVAL_MS = 3_000;

// Scanner combines the CC-hunter and conventional-FM scanner panels
// from the TUI. Mutations (hold/resume/retune/lockout/manual-tune)
// are gated behind selectCanMutate.
export function Scanner() {
  const cfg = useShared(selectClientConfig);
  const canMutate = useShared(selectCanMutate);
  const setError = useShared((s) => s.setError);
  const scanner = useShared((s) => s.scanner);
  const setScanner = useShared((s) => s.setScanner);

  const [confirm, setConfirm] = useState<null | {
    title: string;
    message: string;
    destructive?: boolean;
    onConfirm: () => Promise<void>;
  }>(null);

  useEffect(() => {
    let cancel = false;
    const refresh = async () => {
      try {
        const data = await api.scanner(cfg);
        if (!cancel) setScanner(data);
      } catch {
        // Keep the previous snapshot.
      }
    };
    refresh();
    const t = window.setInterval(refresh, POLL_INTERVAL_MS);
    return () => {
      cancel = true;
      window.clearInterval(t);
    };
  }, [cfg, setScanner]);

  async function reload() {
    try {
      const data = await api.scanner(cfg);
      setScanner(data);
    } catch {
      /* ignored */
    }
  }

  async function wrap(label: string, fn: () => Promise<unknown>) {
    try {
      await fn();
      await reload();
    } catch (e: unknown) {
      setError(
        e instanceof Error ? `${label} failed: ${e.message}` : `${label} failed`,
      );
      throw e;
    }
  }

  if (!scanner) {
    return (
      <div className="space-y-3">
        <h2 className="text-xl font-semibold">Scanner</h2>
        <p className="text-muted text-sm">Loading scanner status…</p>
      </div>
    );
  }

  return (
    <div className="space-y-4">
      <header className="flex items-center justify-between gap-3 flex-wrap">
        <h2 className="text-xl font-semibold">Scanner cockpit</h2>
        <div className="flex items-center gap-2 text-xs">
          <span className="text-muted uppercase tracking-wider">scan_mode</span>
          <span className="font-mono">{scanner.scan_mode}</span>
          {canMutate && (
            <button
              className="btn-ghost text-xs"
              onClick={() =>
                setConfirm({
                  title: `Switch scan mode to "${scanner.scan_mode === "all" ? "list" : "all"}"?`,
                  message:
                    scanner.scan_mode === "all"
                      ? "list mode only listens for talkgroups marked Scan=true."
                      : "all mode listens for every talkgroup the daemon knows about.",
                  onConfirm: () =>
                    wrap("set_scan_mode", () =>
                      writes.setScanMode(
                        cfg,
                        scanner.scan_mode === "all" ? "list" : "all",
                      ),
                    ),
                })
              }
            >
              flip
            </button>
          )}
        </div>
      </header>

      <p className="text-xs text-muted">
        {scanner.tg_scan_count} of {scanner.tg_total} talkgroups eligible
      </p>

      <Hunt
        systems={scanner.systems}
        canMutate={canMutate}
        onMutate={wrap}
        onConfirm={(c) => setConfirm(c)}
      />

      <Conventional
        conv={scanner.conventional}
        canMutate={canMutate}
        onMutate={wrap}
        onConfirm={(c) => setConfirm(c)}
      />

      {canMutate && <ManualTune onMutate={wrap} />}

      {confirm && (
        <ConfirmModal
          title={confirm.title}
          message={confirm.message}
          destructive={confirm.destructive}
          onConfirm={async () => {
            try {
              await confirm.onConfirm();
            } finally {
              setConfirm(null);
            }
          }}
          onCancel={() => setConfirm(null)}
        />
      )}
    </div>
  );
}

interface ConfirmRequest {
  title: string;
  message: string;
  destructive?: boolean;
  onConfirm: () => Promise<void>;
}

function Hunt({
  systems,
  canMutate,
  onMutate,
  onConfirm,
}: {
  systems: SystemHuntStatusDTO[];
  canMutate: boolean;
  onMutate: (label: string, fn: () => Promise<unknown>) => Promise<void>;
  onConfirm: (c: ConfirmRequest) => void;
}) {
  const cfg = useShared(selectClientConfig);

  return (
    <section className="panel p-4">
      <h3 className="panel-title mb-3">Trunked-system hunter</h3>
      {systems.length === 0 ? (
        <p className="text-muted text-sm">No trunked systems configured.</p>
      ) : (
        <ul className="divide-y divide-panel">
          {systems.map((sys) => (
            <li key={sys.name} className="py-3 space-y-1">
              <div className="flex items-baseline gap-2 flex-wrap">
                <span className="font-medium">{sys.name}</span>
                <span className="font-mono text-xs text-accent">
                  {sys.protocol}
                </span>
                <StatePill state={sys.state} />
                {sys.locked_freq_hz != null && (
                  <span className="font-mono text-xs text-muted">
                    lock {formatHz(sys.locked_freq_hz)}
                  </span>
                )}
              </div>
              <div className="text-xs text-muted font-mono">
                {sys.attempt_index != null && sys.total_candidates != null && (
                  <>
                    try {sys.attempt_index + 1}/{sys.total_candidates}
                  </>
                )}
                {sys.attempted_freq_hz != null && (
                  <> · attempting {formatHz(sys.attempted_freq_hz)}</>
                )}
                {sys.backoff_ms != null && sys.backoff_ms > 0 && (
                  <> · backoff {sys.backoff_ms}ms</>
                )}
                {sys.last_grant_at && (
                  <> · last grant {timeOnly(sys.last_grant_at)}</>
                )}
              </div>
              {canMutate && (
                <div className="flex flex-wrap gap-2 pt-1">
                  <button
                    className="btn-ghost text-xs"
                    onClick={() =>
                      onConfirm({
                        title: `Hold ${sys.name}?`,
                        message:
                          "Pause the CC hunter on this system — it stays on the current frequency without retuning.",
                        onConfirm: () =>
                          onMutate("hunt_hold", () =>
                            writes.huntHold(cfg, sys.name),
                          ),
                      })
                    }
                  >
                    Hold
                  </button>
                  <button
                    className="btn-ghost text-xs"
                    onClick={() =>
                      onMutate("hunt_resume", () =>
                        writes.huntResume(cfg, sys.name),
                      )
                    }
                  >
                    Resume
                  </button>
                  <button
                    className="btn-ghost text-xs"
                    onClick={() =>
                      onConfirm({
                        title: `Retune ${sys.name}?`,
                        message:
                          "Force the hunter to drop the current lock and start over from the candidate list.",
                        destructive: true,
                        onConfirm: () =>
                          onMutate("hunt_retune", () =>
                            writes.huntRetune(cfg, sys.name),
                          ),
                      })
                    }
                  >
                    Retune
                  </button>
                </div>
              )}
            </li>
          ))}
        </ul>
      )}
    </section>
  );
}

function Conventional({
  conv,
  canMutate,
  onMutate,
  onConfirm,
}: {
  conv: { enabled: boolean; state?: string; device_serial?: string; cursor_index?: number; channels: ConvChannelStatusDTO[] };
  canMutate: boolean;
  onMutate: (label: string, fn: () => Promise<unknown>) => Promise<void>;
  onConfirm: (c: ConfirmRequest) => void;
}) {
  const cfg = useShared(selectClientConfig);
  if (!conv.enabled) {
    return (
      <section className="panel p-4">
        <h3 className="panel-title mb-2">Conventional FM scanner</h3>
        <p className="text-muted text-sm">disabled (no scanner channels configured).</p>
      </section>
    );
  }

  return (
    <section className="panel p-4 space-y-3">
      <header className="flex items-center justify-between gap-3">
        <h3 className="panel-title">Conventional FM scanner</h3>
        <div className="flex items-center gap-2 text-xs">
          <span className="font-mono">{conv.state ?? "—"}</span>
          {conv.device_serial && (
            <span className="text-muted">on {conv.device_serial}</span>
          )}
          {canMutate && (
            <>
              <button
                className="btn-ghost text-xs"
                onClick={() =>
                  onMutate("conv_hold", () => writes.convHold(cfg))
                }
              >
                Hold
              </button>
              <button
                className="btn-ghost text-xs"
                onClick={() =>
                  onMutate("conv_resume", () => writes.convResume(cfg))
                }
              >
                Resume
              </button>
            </>
          )}
        </div>
      </header>

      {conv.channels.length === 0 ? (
        <p className="text-muted text-sm">No channels in the conv scanner list.</p>
      ) : (
        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            <thead className="text-xs uppercase tracking-wider text-muted">
              <tr>
                <th className="text-left px-2 py-1">#</th>
                <th className="text-left px-2 py-1">Label</th>
                <th className="text-left px-2 py-1">Freq</th>
                <th className="text-left px-2 py-1">Mode</th>
                <th className="text-left px-2 py-1">State</th>
                {canMutate && <th className="text-left px-2 py-1">Actions</th>}
              </tr>
            </thead>
            <tbody className="divide-y divide-panel">
              {conv.channels.map((ch) => (
                <tr
                  key={ch.index}
                  className={ch.active ? "bg-accent/10" : undefined}
                >
                  <td className="px-2 py-1 font-mono text-xs">{ch.index}</td>
                  <td className="px-2 py-1">{ch.label}</td>
                  <td className="px-2 py-1 font-mono text-xs">
                    {formatHz(ch.frequency_hz)}
                  </td>
                  <td className="px-2 py-1 font-mono text-xs">{ch.mode}</td>
                  <td className="px-2 py-1">
                    {ch.locked_out ? (
                      <span className="pill-err">locked</span>
                    ) : ch.active ? (
                      <span className="pill-ok">active</span>
                    ) : (
                      <span className="text-muted text-xs">idle</span>
                    )}
                  </td>
                  {canMutate && (
                    <td className="px-2 py-1">
                      <div className="flex flex-wrap gap-1">
                        <button
                          className="btn-ghost text-xs !px-2 !py-1"
                          onClick={() =>
                            onMutate("conv_dwell", () =>
                              writes.convDwell(cfg, ch.index),
                            )
                          }
                        >
                          Dwell
                        </button>
                        {ch.locked_out ? (
                          <button
                            className="btn-ghost text-xs !px-2 !py-1"
                            onClick={() =>
                              onMutate("conv_unlockout", () =>
                                writes.convUnlockout(cfg, ch.index),
                              )
                            }
                          >
                            Unlock
                          </button>
                        ) : (
                          <button
                            className="btn-ghost text-xs !px-2 !py-1"
                            onClick={() =>
                              onConfirm({
                                title: `Lock out ${ch.label}?`,
                                message: `Stop scanning ${formatHz(ch.frequency_hz)} until the lockout is cleared.`,
                                destructive: true,
                                onConfirm: () =>
                                  onMutate("conv_lockout", () =>
                                    writes.convLockout(cfg, ch.index),
                                  ),
                              })
                            }
                          >
                            Lock
                          </button>
                        )}
                      </div>
                    </td>
                  )}
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </section>
  );
}

function ManualTune({
  onMutate,
}: {
  onMutate: (label: string, fn: () => Promise<unknown>) => Promise<void>;
}) {
  const cfg = useShared(selectClientConfig);
  const [freq, setFreq] = useState("");
  const [label, setLabel] = useState("");
  const [mode, setMode] = useState<"fm" | "nfm">("fm");
  const [squelch, setSquelch] = useState("");
  const [hang, setHang] = useState("");
  const [busy, setBusy] = useState(false);

  async function submit(e: React.FormEvent) {
    e.preventDefault();
    const f = parseFreqInput(freq);
    if (f == null) return;
    setBusy(true);
    try {
      const body: Parameters<typeof writes.manualTune>[1] = {
        frequency_hz: f,
        mode,
      };
      if (label.trim()) body.label = label.trim();
      const s = parseFloat(squelch);
      if (Number.isFinite(s)) body.squelch_dbfs = s;
      const h = parseInt(hang, 10);
      if (Number.isFinite(h)) body.hangtime_ms = h;
      await onMutate("manual_tune", () => writes.manualTune(cfg, body));
      setFreq("");
      setLabel("");
      setSquelch("");
      setHang("");
    } finally {
      setBusy(false);
    }
  }

  return (
    <section className="panel p-4">
      <h3 className="panel-title mb-3">Manual VFO tune</h3>
      <form
        onSubmit={submit}
        className="grid grid-cols-2 sm:grid-cols-6 gap-2 items-end"
      >
        <label className="col-span-2 sm:col-span-2 text-xs space-y-1">
          <span className="text-muted uppercase tracking-wider">Frequency</span>
          <input
            type="text"
            className="input w-full"
            placeholder="154.250 MHz"
            value={freq}
            onChange={(e) => setFreq(e.target.value)}
            required
          />
        </label>
        <label className="col-span-2 text-xs space-y-1">
          <span className="text-muted uppercase tracking-wider">Label</span>
          <input
            type="text"
            className="input w-full"
            placeholder="optional"
            value={label}
            onChange={(e) => setLabel(e.target.value)}
          />
        </label>
        <label className="text-xs space-y-1">
          <span className="text-muted uppercase tracking-wider">Mode</span>
          <select
            className="input w-full"
            value={mode}
            onChange={(e) => setMode(e.target.value as "fm" | "nfm")}
          >
            <option value="fm">FM</option>
            <option value="nfm">NFM</option>
          </select>
        </label>
        <label className="text-xs space-y-1">
          <span className="text-muted uppercase tracking-wider">Squelch dBFS</span>
          <input
            type="number"
            className="input w-full"
            placeholder="-40"
            value={squelch}
            onChange={(e) => setSquelch(e.target.value)}
          />
        </label>
        <label className="text-xs space-y-1">
          <span className="text-muted uppercase tracking-wider">Hang ms</span>
          <input
            type="number"
            className="input w-full"
            placeholder="800"
            value={hang}
            onChange={(e) => setHang(e.target.value)}
          />
        </label>
        <button
          type="submit"
          className="btn-primary col-span-2 sm:col-span-1"
          disabled={busy || !freq.trim()}
        >
          {busy ? "Adding…" : "Add"}
        </button>
      </form>
      <p className="text-xs text-muted mt-2">
        Accepts MHz / kHz / Hz with the unit (e.g. "154.250 MHz") or a
        bare integer in Hz.
      </p>
    </section>
  );
}

function StatePill({ state }: { state: string }) {
  const map: Record<string, string> = {
    locked: "pill-ok",
    hunting: "pill-warn",
    held: "pill",
    failed: "pill-err",
    idle: "pill",
  };
  const cls = map[state] ?? "pill";
  return <span className={cls}>{state}</span>;
}

function timeOnly(ts: string): string {
  return ts.replace("T", " ").replace(/\..*$/, "");
}

function formatHz(hz: number): string {
  if (!Number.isFinite(hz)) return "—";
  if (hz >= 1_000_000) return `${(hz / 1_000_000).toFixed(4)} MHz`;
  if (hz >= 1_000) return `${(hz / 1_000).toFixed(3)} kHz`;
  return `${hz} Hz`;
}

export function parseFreqInput(s: string): number | null {
  const trimmed = s.trim();
  if (!trimmed) return null;
  const m = trimmed.match(/^(-?\d*\.?\d+)\s*(mhz|khz|hz)?$/i);
  if (!m) return null;
  const val = parseFloat(m[1]);
  if (!Number.isFinite(val)) return null;
  const unit = (m[2] ?? "hz").toLowerCase();
  switch (unit) {
    case "mhz":
      return Math.round(val * 1_000_000);
    case "khz":
      return Math.round(val * 1_000);
    default:
      return Math.round(val);
  }
}
