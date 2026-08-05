import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { audioStreamURL } from "../api/client";
import { createStreamPlayer, type StreamPlayer } from "../audio/streamPlayer";
import { writes } from "../api/write";
import { selectClientConfig, useShared } from "../store/shared";
import { useActiveCallsPoll } from "../hooks/useActiveCallsPoll";
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
export function AudioPlayer() {
  const cfg = useShared(selectClientConfig);
  const activeCalls = useShared((s) => s.activeCalls);
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
    const top = activeCalls[0];
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
    </div>
  );
}
