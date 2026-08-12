import { useEffect, useMemo, useRef, useState } from "react";
import { useSearchParams } from "react-router-dom";
import {
  fetchSpectrumDevices,
  defaultSymbolDevice,
  initialDeviceSerial,
  type SpectrumDevice,
} from "../api/spectrum";
import { demodModeToProto, openSymbolStream } from "../api/symbols";
import {
  computeQuality,
  WINDOW_SYMBOLS,
  type Quality,
} from "../lib/signalQuality";
import { selectClientConfig, useShared } from "../store/shared";

export interface SignalQualityState {
  quality: Quality | null;
  status: "connecting" | "open" | "closed" | "idle";
  device: string | null;
}

// useSignalQuality opens a single /diag/symbols subscriber on the control-role
// SDR (or the ?device= target) and returns a rolling recovered-symbol quality
// estimate — the data behind the Plots-hub verdict. It rests the view on the
// device's configured control channel so it reports control-channel health, and
// accumulates a WINDOW_SYMBOLS tail (mirroring the Histogram panel's alignment
// guard), recomputing on a 1 s cadence. One extra websocket while mounted;
// closed on unmount.
export function useSignalQuality(): SignalQualityState {
  const cfg = useShared(selectClientConfig);
  const targetDevice = useSearchParams()[0].get("device");
  const [devices, setDevices] = useState<SpectrumDevice[]>([]);
  const [selected, setSelected] = useState<string | null>(null);
  const [status, setStatus] = useState<SignalQualityState["status"]>("idle");
  const [quality, setQuality] = useState<Quality | null>(null);
  const dibitsRef = useRef<number[]>([]);
  const softRef = useRef<number[]>([]);

  // Enumerate SDRs once and pick one (control-role default, or ?device=).
  useEffect(() => {
    let cancel = false;
    (async () => {
      try {
        const list = await fetchSpectrumDevices(cfg);
        if (cancel) return;
        setDevices(list);
        if (selected == null) {
          setSelected(
            initialDeviceSerial(list, targetDevice, defaultSymbolDevice),
          );
        }
      } catch {
        // No devices reachable — the verdict stays "unknown".
      }
    })();
    return () => {
      cancel = true;
    };
  }, [cfg, selected, targetDevice]);

  const device = useMemo(
    () => devices.find((d) => d.serial === selected) ?? null,
    [devices, selected],
  );
  const proto = demodModeToProto(device?.p25_modulation);
  const offset =
    device?.control_channel_hz && device.center_hz
      ? device.control_channel_hz - device.center_hz
      : 0;

  // Stream symbols + accumulate a rolling window; recompute quality on a timer.
  useEffect(() => {
    if (!selected) return;
    dibitsRef.current = [];
    softRef.current = [];
    setQuality(null);
    const stream = openSymbolStream(cfg, {
      serial: selected,
      proto,
      offset,
      onStatus: setStatus,
      onFrame: (f) => {
        const sb = softRef.current.concat(f.soft ?? []);
        const db = dibitsRef.current.concat(f.dibits ?? []);
        // Keep soft aligned index-for-index with dibits; drop it on a desync
        // so computeQuality's soft-SNR path only runs on matched tracks.
        softRef.current =
          sb.length === db.length ? sb.slice(Math.max(0, sb.length - WINDOW_SYMBOLS)) : [];
        dibitsRef.current = db.slice(Math.max(0, db.length - WINDOW_SYMBOLS));
      },
    });
    const t = window.setInterval(() => {
      setQuality(computeQuality(dibitsRef.current, softRef.current));
    }, 1000);
    return () => {
      stream.close();
      window.clearInterval(t);
    };
  }, [cfg, selected, proto, offset]);

  return { quality, status, device: selected };
}
