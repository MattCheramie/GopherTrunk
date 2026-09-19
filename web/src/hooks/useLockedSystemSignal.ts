import { useEffect, useState } from "react";
import { api } from "../api/client";
import { selectClientConfig, useShared } from "../store/shared";
import type { SystemHuntStatusDTO } from "../api/types";

// useScannerSystems polls GET /api/v1/scanner and returns EVERY trunked
// system's hunt status — name, protocol, state, locked frequency, and the
// backend's per-system decode health (clean/marginal/poor), carrier offset and
// front-end level. This is the source for the per-system signal meters: one
// system per row, not one collapsed reading. On an error the last good list is
// kept. One lightweight JSON poll while mounted.
export function useScannerSystems(
  intervalMs = 3_000,
): SystemHuntStatusDTO[] {
  const cfg = useShared(selectClientConfig);
  const [systems, setSystems] = useState<SystemHuntStatusDTO[]>([]);

  useEffect(() => {
    let cancel = false;
    const poll = async () => {
      try {
        const s = await api.scanner(cfg);
        if (cancel) return;
        setSystems(s.systems ?? []);
      } catch {
        // Scanner endpoint unreachable — leave the last reading in place.
      }
    };
    poll();
    const t = window.setInterval(poll, intervalMs);
    return () => {
      cancel = true;
      window.clearInterval(t);
    };
  }, [cfg, intervalMs]);

  return systems;
}

// useLockedSystemSignal returns the primary trunked system's decode-health
// status — the backend clean/marginal/poor verdict (frame-error rate), carrier
// offset, and front-end level. It is the DECODE-side companion to
// useSignalQuality's SYMBOL-side stream: together they let a single-summary view
// (the Plots hub) show both axes so the same carrier reads consistently.
// Prefers a locked system that already has decode health; otherwise the first
// locked system, otherwise the first system.
//
// For a view that shows one meter PER system (the Dashboard), use
// useScannerSystems instead of this reducer.
export function useLockedSystemSignal(
  intervalMs = 3_000,
): SystemHuntStatusDTO | null {
  const systems = useScannerSystems(intervalMs);
  return (
    systems.find((x) => x.state === "locked" && x.has_decode_health) ??
    systems.find((x) => x.state === "locked") ??
    systems[0] ??
    null
  );
}
