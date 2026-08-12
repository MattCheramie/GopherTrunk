import { useNavigate, useParams } from "react-router-dom";
import { Constellation } from "./Constellation";
import { SymbolScope } from "./SymbolScope";
import { EyeDiagram } from "./EyeDiagram";
import { Mixer } from "./Mixer";
import { Tuning } from "./Tuning";
import { Histogram } from "./Histogram";
import { Badge } from "../components/ui/Badge";
import { useSignalQuality } from "../hooks/useSignalQuality";
import { qualityVerdict } from "../lib/signalQuality";

// Plots hub — a single tabbed home for the signal-plot family, mirroring
// OP25's Plots tabs. Each sub-tab is the standalone panel rendered inline;
// the standalone routes (/constellation, /symbols, …) are kept for
// deep-linking, and the sub-tab is reflected in the URL (/plots/<tab>) so
// a chosen view is shareable and survives a refresh.
//
// Spectrum stays its own top-level tab (it's the heavier wideband view);
// this hub gathers the per-channel symbol-domain scopes.

const TABS = [
  { key: "constellation", label: "Constellation", el: <Constellation /> },
  { key: "symbols", label: "Symbol scope", el: <SymbolScope /> },
  { key: "eye", label: "Eye diagram", el: <EyeDiagram /> },
  { key: "mixer", label: "Mixer", el: <Mixer /> },
  { key: "tuning", label: "Tuning", el: <Tuning /> },
  { key: "histogram", label: "Histogram", el: <Histogram /> },
] as const;

const DEFAULT_TAB = "constellation";

export function Plots() {
  const { tab } = useParams<{ tab?: string }>();
  const navigate = useNavigate();
  const active = TABS.find((t) => t.key === tab) ?? TABS[0];

  return (
    <div className="space-y-3">
      <QualityVerdict />
      <div
        className="flex gap-1 overflow-x-auto border-b border-panel pb-1"
        role="tablist"
        aria-label="Plots"
      >
        {TABS.map((t) => {
          const selected = t.key === active.key;
          return (
            <button
              key={t.key}
              role="tab"
              aria-selected={selected}
              onClick={() => navigate(`/plots/${t.key}`)}
              className={
                "px-3 py-1 rounded text-xs whitespace-nowrap " +
                (selected
                  ? "bg-panel text-fg font-semibold"
                  : "text-muted hover:text-fg hover:bg-panel")
              }
            >
              {t.label}
            </button>
          );
        })}
      </div>

      {/* Remount on tab change (key) so each panel's stream subscription is
          opened/closed cleanly rather than lingering behind a hidden tab. */}
      <div key={active.key}>{active.el}</div>
    </div>
  );
}

// QualityVerdict is the at-a-glance signal-health banner atop the Plots hub:
// one clean/marginal/poor pill plus the SNR / balance numbers, from a single
// symbol stream on the control SDR — so an operator sees whether the signal is
// healthy without opening the Histogram or Tuning tab. The scopes below explain
// *why*.
function QualityVerdict() {
  const { quality, status } = useSignalQuality();
  const verdict = qualityVerdict(quality);
  const tone =
    verdict === "clean"
      ? "ok"
      : verdict === "marginal"
        ? "warn"
        : verdict === "poor"
          ? "err"
          : "neutral";
  return (
    <div className="panel flex flex-wrap items-center gap-3 px-3 py-2 text-sm">
      <span className="text-muted text-xs uppercase tracking-wider">Signal</span>
      <Badge tone={tone}>
        {verdict === "unknown" ? "acquiring…" : verdict}
      </Badge>
      {quality?.snrDb != null && (
        <span className="font-mono text-xs text-muted">
          SNR {quality.snrDb.toFixed(1)} dB
        </span>
      )}
      {quality && (
        <span className="font-mono text-xs text-muted">
          balance ±{quality.balanceDev.toFixed(1)}%
        </span>
      )}
      {quality && quality.total > 0 && (
        <span className="font-mono text-xs text-muted">
          {quality.total} symbols
        </span>
      )}
      <span className="ml-auto text-xs text-muted">
        {status === "open"
          ? "live"
          : status === "connecting"
            ? "connecting…"
            : status === "closed"
              ? "offline"
              : "idle"}
      </span>
    </div>
  );
}

export { DEFAULT_TAB };
