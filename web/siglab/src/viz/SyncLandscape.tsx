import { useEffect, useRef } from "react";
import type { SyncLandscape as SyncLandscapeData } from "../api/types";

// SyncLandscape renders the sync-word correlation grid (synclandscape.go) as a
// Plotly heatmap: variant × rotation, coloured by clean-hit count. A single
// bright cell means the demod produces canonical symbols at a real cadence.
export function SyncLandscape({ sync }: { sync: SyncLandscapeData }) {
  const ref = useRef<HTMLDivElement | null>(null);

  useEffect(() => {
    if (!ref.current || !sync.stats?.length) return;
    let cancelled = false;
    (async () => {
      const Plotly = (await import("plotly.js-dist-min")).default;
      if (cancelled || !ref.current) return;
      const variants = Array.from(new Set(sync.stats.map((s) => s.variant)));
      const rotations = Array.from(new Set(sync.stats.map((s) => s.rotation))).sort(
        (a, b) => a - b,
      );
      const z = variants.map((v) =>
        rotations.map((rot) => {
          const cell = sync.stats.find((s) => s.variant === v && s.rotation === rot);
          return cell ? cell.hits : 0;
        }),
      );
      await Plotly.react(
        ref.current,
        [
          {
            z,
            x: rotations.map((r) => `rot ${r}`),
            y: variants,
            type: "heatmap",
            colorscale: "Cividis",
          },
        ],
        {
          margin: { l: 90, r: 10, t: 10, b: 30 },
          paper_bgcolor: "rgba(0,0,0,0)",
          plot_bgcolor: "rgba(0,0,0,0)",
          font: { color: "#94a3b8", size: 10 },
          height: Math.max(160, variants.length * 22 + 60),
        },
        { displayModeBar: false, responsive: true } as Record<string, unknown>,
      );
    })();
    return () => {
      cancelled = true;
    };
  }, [sync]);

  return (
    <div className="card">
      <h3 className="mb-2 text-sm font-semibold">Sync landscape</h3>
      <p className="mb-1 text-xs text-muted">
        winner {sync.winner_variant || "—"} rot {sync.winner_rotation} · {sync.winner_hits}{" "}
        hits · modal spacing {sync.modal_spacing}
      </p>
      <div ref={ref} />
    </div>
  );
}
