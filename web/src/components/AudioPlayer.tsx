import { useEffect, useRef, useState } from "react";
import { audioStreamURL } from "../api/client";
import { writes } from "../api/write";
import { selectClientConfig, useShared } from "../store/shared";
import { prefs } from "../store/prefs";

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
export function AudioPlayer() {
  const cfg = useShared(selectClientConfig);
  const activeCalls = useShared((s) => s.activeCalls);
  const [enabled, setEnabled] = useState(false);
  const [muted, setMuted] = useState(false);
  const [volume, setVolume] = useState(prefs.audioVolume());
  const [recording, setRecording] = useState<boolean | null>(null);
  const audioRef = useRef<HTMLAudioElement | null>(null);

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

  const enable = async () => {
    const el = audioRef.current;
    if (!el || !cfg.baseURL) return;
    el.src = audioStreamURL(cfg);
    el.volume = volume;
    el.muted = false;
    try {
      await el.play();
      setEnabled(true);
    } catch {
      // Autoplay blocked. The next user gesture will retry.
      setEnabled(false);
    }
  };

  const disable = () => {
    const el = audioRef.current;
    if (el) {
      el.pause();
      el.removeAttribute("src");
      el.load();
    }
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
      {!enabled ? (
        <button
          type="button"
          onClick={enable}
          className="btn-primary text-xs"
          aria-label="Enable audio"
        >
          <span aria-hidden>▶</span> Tap to enable audio
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
