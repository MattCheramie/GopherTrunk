import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { api, audioStreamURL } from "../api/client";
import { createStreamPlayer, type StreamPlayer } from "../audio/streamPlayer";
import { MAX_VOLUME } from "../audio/volume";
import { writes } from "../api/write";
import { selectClientConfig, useShared } from "../store/shared";
import { useActiveCallsPoll } from "../hooks/useActiveCallsPoll";
import { useDataPoll } from "../hooks/useDataPoll";
import { pickFollowSerial } from "../audio/followSelect";
import { prefs } from "../store/prefs";

// AudioPlayer is a floating, dismissable mini-player that streams
// /api/v1/audio/stream through the Web Audio API. iOS and Android both
// require a user gesture before audio starts playing, so the first
// activation goes through a "Tap to enable audio" button.
//
// Issue #598: this used to point a hidden <audio> element at the stream
// URL, which failed silently in Chrome on macOS (an open-ended chunked
// "infinite WAV" is unreliable for the media element). We now fetch the
// stream and play the decoded PCM through an AudioContext — see
// ../audio/streamPlayer.ts — which is robust across browsers and lets us
// send the Authorization header for auth-gated daemons. Failures surface
// as a visible chip rather than a swallowed console.warn.
//
// Mobile niceties:
//   - Media Session API lights up the OS lock-screen / control
//     center with track-style metadata pulled from the active call.
//
// Issue #1351: the player used to read the daemon's cockpit state from
// the shared store, but nothing ever loaded it there, so `recording`
// stayed null for the life of the tab. A null rendered as "Recording
// off", and the first click on REC sent recording_enabled: false — it
// switched the operator's recordings OFF while looking like it turned
// them on. The player now polls GET /api/v1/audio itself, keeps the
// store current from every PATCH response and audio.state event, and
// refuses to toggle a knob whose state it has not seen. It also says
// WHY nothing plays: the daemon reports how much decoded audio has
// reached its live path, and zero means no call has been decoded at
// all — nothing to play and nothing to record.
export function AudioPlayer() {
  const cfg = useShared(selectClientConfig);
  const activeCalls = useShared((s) => s.activeCalls);
  const setAudio = useShared((s) => s.setAudio);
  // AudioPlayer is always mounted but, unlike Active/Dashboard, didn't
  // refresh activeCalls — so its Media Session metadata and (below) the
  // call it follows went stale on every other route. Poll here so the
  // follow logic stays live wherever the operator is.
  useActiveCallsPoll();
  const [enabled, setEnabled] = useState(false);
  const [muted, setMuted] = useState(false);
  const [volume, setVolume] = useState(prefs.audioVolume());
  const [recording, setRecording] = useState<boolean | null>(null);
  const [error, setError] = useState<string | null>(null);
  const playerRef = useRef<StreamPlayer | null>(null);
  // Device serial the running stream is filtered to (null = unfiltered).
  const playerSerialRef = useRef<string | null>(null);
  // Read inside startPlayer without re-creating it (and thus restarting
  // the stream) every time the volume slider moves.
  const volumeRef = useRef(volume);
  volumeRef.current = volume;
  const mutedRef = useRef(muted);
  mutedRef.current = muted;

  // The single call live audio follows. Playing one call at a time (standard
  // scanner behaviour) is what stops concurrent talkgroups from overlapping
  // into one garbled stream — the daemon's /audio/stream supports a ?device=
  // filter. pickFollowSerial HOLDS the current call until its device stops
  // transmitting, then advances to the newest — so a call plays to completion
  // instead of being cut off the moment a newer grant lands ("eating
  // conversations"). null when nothing is up.
  const primarySerial = useMemo(
    () => pickFollowSerial(activeCalls, playerSerialRef.current),
    [activeCalls],
  );

  // Load the daemon's cockpit state and keep it fresh (5 s; the poll
  // only delivers changed snapshots). useDataPoll swallows fetch errors,
  // so a daemon without the audio cockpit wired (503) leaves the store
  // null and the REC button in its "unknown" state rather than guessing.
  useDataPoll({
    fetcher: () => api.audio(cfg),
    onData: setAudio,
    intervalMs: 5_000,
    enabled: !!cfg.baseURL,
    resetKey: cfg.baseURL,
  });

  // Reflect the daemon's current audio state into the local toggles.
  const daemonAudio = useShared((s) => s.audio);
  useEffect(() => {
    if (!daemonAudio) return;
    setMuted(daemonAudio.muted);
    setRecording(daemonAudio.recording_enabled);
  }, [daemonAudio]);

  // The call the stream is following right now, for the status line.
  const current = useMemo(
    () =>
      primarySerial
        ? activeCalls.find((c) => c.device_serial === primarySerial)
        : undefined,
    [activeCalls, primarySerial],
  );
  // What to tell the operator while enabled. A daemon that has decoded
  // no voice since start-up (stream_samples_total === 0) is the usual
  // reason for "no audio and no recordings": there is nothing to play.
  // Daemons older than that field report undefined and get the neutral
  // "waiting" text.
  const status = current
    ? {
        text: `TG ${current.talkgroup?.alpha_tag || current.grant.group_id}`,
        title: `Playing ${current.grant.system} talkgroup ${current.grant.group_id}`,
      }
    : daemonAudio?.stream_samples_total === 0
      ? {
          text: "no decoded audio yet",
          title:
            "The daemon has not decoded any voice since it started, so there is nothing to play or record. Check the Dashboard for a control-channel lock and an attached voice SDR.",
        }
      : {
          text: "waiting for a call",
          title: "Connected to the daemon's live stream; audio plays when a call is decoded.",
        };

  // When the active call list changes, update the Media Session
  // metadata so the lock-screen shows what's playing.
  useEffect(() => {
    if (!("mediaSession" in navigator)) return;
    const top = activeCalls[0];
    if (!top) {
      navigator.mediaSession.metadata = null;
      return;
    }
    navigator.mediaSession.metadata = new MediaMetadata({
      title:
        top.talkgroup?.alpha_tag || `TG ${top.grant.group_id}`,
      artist: top.grant.system,
      album: top.talkgroup?.group ?? "",
    });
  }, [activeCalls]);

  // Tear the player down when the target daemon (URL or token) changes
  // so a stale stream from the previous server doesn't keep playing.
  useEffect(() => {
    return () => {
      playerRef.current?.stop();
      playerRef.current = null;
      playerSerialRef.current = null;
    };
  }, [cfg]);

  // startPlayer points the stream at one call's device serial. On the first
  // call it builds the player (and its single AudioContext, inside the user
  // gesture from enable()); every later call REUSES that player via
  // reconnect() rather than tearing down and rebuilding a context. Building a
  // fresh AudioContext per follow-switch leaked contexts until Chrome's
  // ~6-per-page cap threw and playback went silent (macOS Chrome first).
  // Stable across volume/mute changes (those read refs).
  const startPlayer = useCallback(
    (serial: string | null) => {
      if (!cfg.baseURL) return;
      setError(null);
      const url = audioStreamURL(cfg, serial ? { device: serial } : {});
      if (playerRef.current) {
        playerSerialRef.current = serial;
        playerRef.current.reconnect(url);
        return;
      }
      const player = createStreamPlayer({
        url,
        token: cfg.token,
        onError: (msg) => {
          setError(msg);
          setEnabled(false);
        },
        onStateChange: (state) => {
          setEnabled(state === "playing" || state === "connecting");
        },
      });
      playerRef.current = player;
      playerSerialRef.current = serial;
      player.setVolume(volumeRef.current);
      player.setMuted(mutedRef.current);
      player.start();
    },
    [cfg],
  );

  // enable must stay synchronous: createStreamPlayer().start() resumes
  // the AudioContext inside this gesture so the autoplay policy doesn't
  // reject playback. Open onto the current primary call straight away.
  const enable = () => {
    if (!cfg.baseURL) return;
    startPlayer(primarySerial);
    setEnabled(true);
  };

  const disable = () => {
    playerRef.current?.stop();
    playerRef.current = null;
    playerSerialRef.current = null;
    setEnabled(false);
  };

  // Follow the primary call: when it changes, re-point the running stream
  // at the new device so audio switches cleanly instead of overlapping.
  // Only acts on a real call (non-null) and only when a stream is already
  // running — the initial open happens in enable() inside the user
  // gesture, and a call simply ending lets the current filter fall silent
  // rather than dropping back to the (overlap-prone) unfiltered stream.
  useEffect(() => {
    if (!playerRef.current) return;
    if (!primarySerial) return;
    if (primarySerial === playerSerialRef.current) return;
    startPlayer(primarySerial);
  }, [primarySerial, startPlayer]);

  const onVolume = (v: number) => {
    setVolume(v);
    prefs.setAudioVolume(v);
    playerRef.current?.setVolume(v);
  };

  const toggleMute = async () => {
    const next = !muted;
    setMuted(next);
    playerRef.current?.setMuted(next);
    try {
      setAudio(await writes.setAudio(cfg, { muted: next }));
    } catch {
      /* mutation may be gated; UI reflects local toggle anyway. */
    }
  };

  const toggleRecording = async () => {
    // Never toggle a state we have not seen: with the daemon's answer
    // still pending (or the cockpit not wired) a guess could switch
    // recordings off (#1351). The button is disabled in that state too.
    if (recording === null) return;
    const next = !recording;
    setRecording(next);
    try {
      setAudio(await writes.setAudio(cfg, { recording_enabled: next }));
    } catch {
      /* mutation may be gated. */
    }
  };

  if (!cfg.baseURL) return null;
  return (
    <div className="fixed sm:bottom-3 bottom-16 right-3 z-30 panel p-3 flex items-center gap-2 max-w-[calc(100%-1.5rem)]">
      {!enabled ? (
        <>
          <button
            type="button"
            onClick={enable}
            className="btn-primary text-xs"
            aria-label="Enable audio"
          >
            <span aria-hidden>▶</span>{" "}
            {error ? "Audio failed — tap to retry" : "Tap to enable audio"}
          </button>
          {error && (
            <span
              role="alert"
              title={error}
              className="text-xs text-red-400 max-w-[12rem] truncate"
            >
              ⚠ {error}
            </span>
          )}
        </>
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
          {/* Up to MAX_VOLUME (400 %): above 100 % the player's limiter
              keeps loud calls from clipping (#1261). */}
          <input
            type="range"
            min={0}
            max={MAX_VOLUME}
            step={0.05}
            value={volume}
            onChange={(e) => onVolume(Number(e.target.value))}
            aria-label="Volume"
            aria-valuetext={`${Math.round(volume * 100)}%`}
            title={`Volume ${Math.round(volume * 100)}%`}
            className="w-24 accent-accent"
          />
          <span className="w-10 text-right text-xs tabular-nums text-muted">
            {Math.round(volume * 100)}%
          </span>
          <span
            role="status"
            title={status.title}
            className="hidden sm:inline max-w-[9rem] truncate text-xs text-muted"
          >
            {status.text}
          </span>
          <button
            type="button"
            onClick={toggleRecording}
            disabled={recording === null}
            className={
              recording
                ? "btn-danger text-xs"
                : "btn-ghost text-xs"
            }
            aria-pressed={recording === null ? undefined : recording}
            aria-label={
              recording === null
                ? "Recording state unknown"
                : recording
                  ? "Recording on (click to disable)"
                  : "Recording off (click to enable)"
            }
            title={
              recording === null
                ? "Waiting for the daemon's audio state"
                : recording
                  ? "Recording on: decoded calls are written to recordings.dir"
                  : "Recording is OFF: no call files are being written. Click to turn it on."
            }
          >
            ● REC
          </button>
        </>
      )}
    </div>
  );
}
