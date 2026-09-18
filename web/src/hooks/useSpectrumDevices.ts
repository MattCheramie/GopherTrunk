import { useEffect, useState } from "react";
import { fetchSpectrumDevices, type SpectrumDevice } from "../api/spectrum";
import { selectClientConfig, useShared } from "../store/shared";

// useSpectrumDevices polls GET /api/v1/spectrum/devices and returns the full
// SDR pool with per-device tuning + protocol metadata (centre, sample rate,
// symbol_proto, control_channel_hz). Callers that need to map a system to the
// SDR carrying its control channel — the per-system signal meters — use this
// list. On an error the last good list is kept (a transient fetch failure
// shouldn't blank the meters). One lightweight JSON poll while mounted.
export function useSpectrumDevices(intervalMs = 5_000): SpectrumDevice[] {
  const cfg = useShared(selectClientConfig);
  const [devices, setDevices] = useState<SpectrumDevice[]>([]);

  useEffect(() => {
    let cancel = false;
    const poll = async () => {
      try {
        const list = await fetchSpectrumDevices(cfg);
        if (!cancel) setDevices(list);
      } catch {
        // Devices endpoint unreachable — leave the last reading in place.
      }
    };
    poll();
    const t = window.setInterval(poll, intervalMs);
    return () => {
      cancel = true;
      window.clearInterval(t);
    };
  }, [cfg, intervalMs]);

  return devices;
}
