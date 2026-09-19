import { useEffect, useMemo, useRef, useState } from "react";
import { useSearchParams } from "react-router-dom";
import {
  fetchSpectrumDevices,
  defaultSymbolDevice,
  initialDeviceSerial,
  type SpectrumDevice,
} from "../api/spectrum";
import {
  autoProtoFor,
  openSymbolStream,
  type SymbolTarget,
} from "../api/symbols";
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

export type { SymbolTarget } from "../api/symbols";

export interface SymbolQualityState {
  quality: Quality | null;
  status: SignalQualityState["status"];
}

// useSymbolQualityStream opens exactly ONE /diag/symbols subscriber for an
// explicit target and returns a rolling recovered-symbol quality estimate. It
// accumulates a WINDOW_SYMBOLS tail (mirroring the Histogram panel's alignment
// guard) and recomputes on a 1 s cadence. Pass `null` to open no stream (the
// verdict stays null / idle) — e.g. for a system that isn't locked, so an
// idle/hunting system never spins up a full DSP chain. One extra websocket per
// non-null target while mounted, closed on unmount or when the target changes.
// `onGone` fires when the daemon gives up on the serial so the caller can
// re-enumerate.
//
// This is the shared core behind useSignalQuality (which auto-picks the target
// from the control-role SDR) and the per-system Dashboard meters (which build
// one target per locked system).
export function useSymbolQualityStream(
  target: SymbolTarget | null,
  onGone?: () => void,
): SymbolQualityState {
  const cfg = useShared(selectClientConfig);
  const [status, setStatus] = useState<SignalQualityState["status"]>("idle");
  const [quality, setQuality] = useState<Quality | null>(null);
  const dibitsRef = useRef<number[]>([]);
  const softRef = useRef<number[]>([]);
  // Keep the latest onGone without re-subscribing the stream when the caller
  // passes a fresh closure each render.
  const goneRef = useRef(onGone);
  goneRef.current = onGone;

  const serial = target?.serial ?? null;
  const proto = target?.proto ?? "";
  const offset = target?.offset ?? 0;

  useEffect(() => {
    if (!serial) {
      setStatus("idle");
      setQuality(null);
      return;
    }
    dibitsRef.current = [];
    softRef.current = [];
    setQuality(null);
    const stream = openSymbolStream(cfg, {
      serial,
      proto,
      offset,
      onStatus: (s) => setStatus(s === "gone" ? "closed" : s),
      onGone: () => goneRef.current?.(),
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
  }, [cfg, serial, proto, offset]);

  return { quality, status };
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

  // Enumerate SDRs and pick one (control-role default, or ?device=).
  //
  // `epoch` forces a re-enumeration when the open stream gives up on the
  // selected serial. Without it, `selected` was set once and never reconciled:
  // after a daemon restart with different hardware the panel kept asking for a
  // device that no longer existed, forever.
  const [epoch, setEpoch] = useState(0);
  useEffect(() => {
    let cancel = false;
    (async () => {
      try {
        const list = await fetchSpectrumDevices(cfg);
        if (cancel) return;
        setDevices(list);
        // Re-pick when nothing is selected yet, or when the selection is no
        // longer one of the daemon's devices.
        const stale =
          selected != null && !list.some((d) => d.serial === selected);
        if (selected == null || stale) {
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
  }, [cfg, selected, targetDevice, epoch]);

  const device = useMemo(
    () => devices.find((d) => d.serial === selected) ?? null,
    [devices, selected],
  );
  const proto = autoProtoFor(device);
  const offset =
    device?.control_channel_hz && device.center_hz
      ? device.control_channel_hz - device.center_hz
      : 0;

  // Stream symbols on the picked device; re-enumerate when it gives up on the
  // serial (daemon restart with different hardware).
  const { quality, status } = useSymbolQualityStream(
    selected ? { serial: selected, proto, offset } : null,
    () => setEpoch((n) => n + 1),
  );

  return { quality, status, device: selected };
}
