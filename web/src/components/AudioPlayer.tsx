import { useEffect, useMemo, useRef, useState } from "react";
import { api, audioStreamURL } from "../api/client";
import { writes } from "../api/write";
import { selectClientConfig, useShared } from "../store/shared";
import { prefs } from "../store/prefs";
import type { ActiveCallDTO } from "../api/types";
import { formatHz } from "../api/format";

// AudioPlayer is a floating, dismissable mini-player that streams
// /api/v1/audio/stream into a long-lived <audio> element. iOS and
// Android both require a user gesture before audio starts playing,
// so the first activation goes through a "Tap to enable audio"
// button. Once unlocked the player auto-resumes whenever the SPA
// regains focus or the daemon reconnects.
//
// Mobile niceties:
//   - Media Session API lights up the OS lock-screen / control
//     center with track-style metadata pulled from the active call.
//   - The audio tag's `preload=none` + manual `src` swap keeps iOS
//     from auto-starting the stream when the browser is reopened.
function callKey(call: ActiveCallDTO): string {
  return `${call.device_serial}|${call.grant.group_id}|${call.started_at}`;
}

function callLabel(call: ActiveCallDTO): string {
  return call.talkgroup?.alpha_tag ?? `TG ${call.grant.group_id}`;
}

const AUDIO_MIN_SIGNAL_DBFS = -50;

function hasUsableSignal(call: ActiveCallDTO): boolean {
  return call.signal_dbfs == null || call.signal_dbfs >= AUDIO_MIN_SIGNAL_DBFS;
}

function isPlayable(call: ActiveCallDTO): boolean {
  return !call.grant.encrypted && !call.grant.data_call;
}

function isSelectableForAudio(call: ActiveCallDTO): boolean {
  return isPlayable(call) && hasUsableSignal(call);
}

function chooseDefaultCall(calls: ActiveCallDTO[]): ActiveCallDTO | null {
  return calls.find(isSelectableForAudio) ?? calls.find(isPlayable) ?? null;
}

function stopAudioElement(el: HTMLAudioElement | null) {
  if (!el) return;
  el.pause();
  el.removeAttribute("src");
  el.load();
}

export function AudioPlayer() {
  const cfg = useShared(selectClientConfig);
  const activeCalls = useShared((s) => s.activeCalls);
  const setActiveCalls = useShared((s) => s.setActiveCalls);
  const [enabled, setEnabled] = useState(false);
  const [muted, setMuted] = useState(false);
  const [volume, setVolume] = useState(prefs.audioVolume());
  const [recording, setRecording] = useState<boolean | null>(null);
  const [selectedKey, setSelectedKey] = useState<string | null>(null);
  const [manualSelection, setManualSelection] = useState(false);
  const audioRef = useRef<HTMLAudioElement | null>(null);
  const playAttemptRef = useRef(0);
  const streamURLRef = useRef<string | null>(null);
  const activePollInFlightRef = useRef(false);

  // Keep active calls fresh while the global player has a reason to
  // make call-selection decisions outside the Dashboard/Active panels.
  // Guard against overlapping requests: slow radios or browsers should
  // not stack /calls/active polls.
  useEffect(() => {
    if (!cfg.baseURL || !enabled) return;
    let cancel = false;
    const refresh = async () => {
      if (activePollInFlightRef.current) return;
      activePollInFlightRef.current = true;
      try {
        const data = await api.activeCalls(cfg);
        if (!cancel) setActiveCalls(data);
      } catch {
        // Leave the previous snapshot alone; other UI surfaces report
        // request failures.
      } finally {
        activePollInFlightRef.current = false;
      }
    };
    refresh();
    const t = window.setInterval(refresh, 1_000);
    return () => {
      cancel = true;
      window.clearInterval(t);
    };
  }, [cfg, enabled, setActiveCalls]);

  const audioChoices = useMemo(() => {
    const strong = activeCalls.filter(isSelectableForAudio);
    if (strong.length > 0) return strong;
    // Escape hatch: when every clear call is weak, still expose those
    // options so the operator can force one instead of getting an empty
    // selector. Encrypted/data calls stay hidden from the listening list.
    return activeCalls.filter(isPlayable);
  }, [activeCalls]);
  const hiddenWeakPlayableCount = useMemo(
    () => activeCalls.filter((c) => isPlayable(c) && !hasUsableSignal(c)).length,
    [activeCalls],
  );
  const selectedCall = useMemo(
    () => activeCalls.find((c) => callKey(c) === selectedKey) ?? null,
    [activeCalls, selectedKey],
  );

  // Select exactly one call for playback. Manual choice sticks while
  // that call is alive and playable; if it ends, becomes encrypted, or
  // turns into a data call, switch to the next playable call.
  useEffect(() => {
    if (selectedCall && isPlayable(selectedCall)) return;
    if (manualSelection) setManualSelection(false);
    const next = chooseDefaultCall(activeCalls);
    const nextKey = next ? callKey(next) : null;
    setSelectedKey((current) => (current === nextKey ? current : nextKey));
  }, [activeCalls, manualSelection, selectedCall]);

  // Reflect the daemon's current audio state into the local toggles.
  const daemonAudio = useShared((s) => s.audio);
  useEffect(() => {
    if (!daemonAudio) return;
    setMuted(daemonAudio.muted);
    setRecording(daemonAudio.recording_enabled);
  }, [daemonAudio]);

  // When the active call list changes, update the Media Session
  // metadata so the lock-screen shows what's playing.
  useEffect(() => {
    if (!("mediaSession" in navigator)) return;
    const top = selectedCall;
    if (!top) {
      navigator.mediaSession.metadata = null;
      return;
    }
    navigator.mediaSession.metadata = new MediaMetadata({
      title:
        top.talkgroup?.alpha_tag ?? `TG ${top.grant.group_id}`,
      artist: top.grant.system,
      album: top.talkgroup?.group ?? "",
    });
  }, [selectedCall]);

  const streamURL = selectedCall && isPlayable(selectedCall)
    ? audioStreamURL(cfg, {
        device: selectedCall.device_serial,
        talkgroup: selectedCall.grant.group_id,
      })
    : null;

  const playStream = async () => {
    const el = audioRef.current;
    if (!el || !streamURL) return false;
    const urlChanged = streamURLRef.current !== streamURL;
    const attempt = urlChanged ? ++playAttemptRef.current : playAttemptRef.current;
    const url = streamURL;
    streamURLRef.current = url;
    el.src = url;
    el.volume = volume;
    el.muted = muted;
    el.load();
    try {
      await el.play();
      return attempt === playAttemptRef.current && streamURLRef.current === url;
    } catch {
      return false;
    }
  };

  useEffect(() => {
    if (!enabled) return;
    const el = audioRef.current;
    if (!streamURL) {
      playAttemptRef.current += 1;
      streamURLRef.current = null;
      stopAudioElement(el);
      return;
    }
    if (streamURLRef.current === streamURL) return;
    void playStream();
  }, [enabled, streamURL]);

  const selectedCallPlayable = selectedCall != null && isPlayable(selectedCall);

  const enable = async () => {
    if (!cfg.baseURL) return;
    if (!selectedCall || !isPlayable(selectedCall)) return;
    // Chrome/Safari may keep play() pending on an open-ended WAV stream
    // until enough PCM arrives. The user gesture already happened, so show
    // the enabled controls immediately and only roll back on a real reject.
    const attemptedURL = streamURL;
    const attempt = ++playAttemptRef.current;
    streamURLRef.current = streamURL;
    setEnabled(true);
    if (!(await playStream())) {
      // Autoplay blocked or the stream failed. The next user gesture retries.
      // Do not let an obsolete play() rejection from a prior src disable a
      // newer stream that was selected while the promise was pending.
      if (attempt === playAttemptRef.current && attemptedURL === streamURLRef.current) {
        setEnabled(false);
      }
    }
  };

  const disable = () => {
    playAttemptRef.current += 1;
    streamURLRef.current = null;
    stopAudioElement(audioRef.current);
    setEnabled(false);
  };

  const onVolume = (v: number) => {
    setVolume(v);
    prefs.setAudioVolume(v);
    if (audioRef.current) audioRef.current.volume = v;
  };

  const toggleMute = async () => {
    const next = !muted;
    setMuted(next);
    if (audioRef.current) audioRef.current.muted = next;
    try {
      await writes.setAudio(cfg, { muted: next });
    } catch {
      /* mutation may be gated; UI reflects local toggle anyway. */
    }
  };

  const toggleRecording = async () => {
    const next = !(recording ?? true);
    setRecording(next);
    try {
      await writes.setAudio(cfg, { recording_enabled: next });
    } catch {
      /* mutation may be gated. */
    }
  };

  if (!cfg.baseURL) return null;
  return (
    <div className="fixed sm:bottom-3 bottom-16 right-3 z-30 panel p-3 flex items-center gap-2 max-w-[calc(100%-1.5rem)]">
      {audioChoices.length > 0 && (
        <select
          value={selectedKey ?? ""}
          onChange={(e) => {
            setSelectedKey(e.target.value || null);
            setManualSelection(!!e.target.value);
          }}
          aria-label="Audio call"
          className="bg-bg border border-panel rounded px-2 py-1 text-xs max-w-44"
        >
          {audioChoices.map((c) => (
            <option key={callKey(c)} value={callKey(c)}>
              {callLabel(c)} · {formatHz(c.grant.frequency_hz)}
              {c.signal_dbfs == null ? "" : ` · ${c.signal_dbfs.toFixed(1)} dBFS`}
              {!hasUsableSignal(c) ? " · weak" : ""}
            </option>
          ))}
        </select>
      )}
      <span className="text-xs text-muted max-w-36 truncate">
        {selectedCall
          ? isSelectableForAudio(selectedCall)
            ? callLabel(selectedCall)
            : !isPlayable(selectedCall)
              ? `${callLabel(selectedCall)} has no clear audio`
              : `${callLabel(selectedCall)} is weak`
          : activeCalls.length > 0
            ? hiddenWeakPlayableCount > 0
              ? `No strong clear audio (${hiddenWeakPlayableCount} weak)`
              : "No clear audio"
            : "No active audio"}
      </span>
      {!enabled ? (
        <button
          type="button"
          onClick={enable}
          className="btn-primary text-xs disabled:opacity-50 disabled:cursor-not-allowed"
          aria-label="Enable audio"
          disabled={!selectedCallPlayable}
        >
          <span aria-hidden>▶</span>{" "}
          {selectedCallPlayable
            ? isSelectableForAudio(selectedCall)
              ? "Tap to enable audio"
              : "Tap to enable weak audio"
            : "No clear audio"}
        </button>
      ) : (
        <>
          <button
            type="button"
            onClick={disable}
            className="btn-ghost text-xs"
            aria-label="Stop audio"
          >
            <span aria-hidden>■</span>
          </button>
          <button
            type="button"
            onClick={toggleMute}
            className="btn-ghost text-xs"
            aria-label={muted ? "Unmute" : "Mute"}
          >
            {muted ? "🔇" : "🔈"}
          </button>
          <input
            type="range"
            min={0}
            max={1}
            step={0.05}
            value={volume}
            onChange={(e) => onVolume(Number(e.target.value))}
            aria-label="Volume"
            className="w-24 accent-accent"
          />
          <button
            type="button"
            onClick={toggleRecording}
            className={
              recording
                ? "btn-danger text-xs"
                : "btn-ghost text-xs"
            }
            aria-pressed={!!recording}
            aria-label={
              recording ? "Recording on (click to disable)" : "Recording off"
            }
          >
            ● REC
          </button>
        </>
      )}
      <audio
        ref={audioRef}
        preload="none"
        playsInline
        // Keep the player on-screen but visually empty; controls are
        // bespoke above so the lockscreen MediaSession is the only
        // public surface.
        className="hidden"
      />
    </div>
  );
}

