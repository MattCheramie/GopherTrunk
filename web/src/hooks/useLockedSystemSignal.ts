import { useEffect, useState } from "react";
import { api } from "../api/client";
import { selectClientConfig, useShared } from "../store/shared";
import type { SystemHuntStatusDTO } from "../api/types";

// useLockedSystemSignal polls GET /api/v1/scanner and returns the primary
// trunked system's decode-health status — the backend clean/marginal/poor
// verdict (frame-error rate), carrier offset, and front-end level. It is the
// DECODE-side companion to useSignalQuality's SYMBOL-side stream: together they
// let the Dashboard and Plots hub show both axes so the same carrier reads
// consistently everywhere. Prefers a locked system that already has decode
// health; otherwise the first locked system, otherwise the first system.
export function useLockedSystemSignal(
  intervalMs = 3_000,
): SystemHuntStatusDTO | null {
  const cfg = useShared(selectClientConfig);
  const [sys, setSys] = useState<SystemHuntStatusDTO | null>(null);

  useEffect(() => {
    let cancel = false;
    const poll = async () => {
      try {
        const s = await api.scanner(cfg);
        if (cancel) return;
        const systems = s.systems ?? [];
        const pick =
          systems.find((x) => x.state === "locked" && x.has_decode_health) ??
          systems.find((x) => x.state === "locked") ??
          systems[0] ??
          null;
        setSys(pick);
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

  return sys;
}
