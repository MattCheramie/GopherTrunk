import { useEffect, useRef, useState } from "react";
import type { ClientConfig } from "../api/client";

// RecordingPlayer plays back one finished call's recording.
//
// It deliberately does NOT point a bare <audio src> at the API URL. That is the
// same failure #598 documents for the live stream: a media element pointed at
// the daemon endpoint can fail silently in Chrome (the operator sees a dead
// "0:00 / 0:00" player and no playback), and — on an auth-gated daemon — a bare
// <audio>/<a download> cannot attach the Authorization header, so both 401 with
// no visible error. Instead we fetch the finite recording through fetch() (which
// carries the bearer token), materialise it as an object URL, and hand THAT to
// the <audio> element. The whole file is small (seconds of 8 kHz mono PCM), so a
// one-shot fetch is cheap and sidesteps every range/streaming quirk; a failed
// fetch surfaces as a visible message rather than a silent dead player.
export function RecordingPlayer({
  cfg,
  callId,
}: {
  cfg: ClientConfig;
  callId: number;
}) {
  const [objURL, setObjURL] = useState<string | null>(null);
  const [ext, setExt] = useState<"wav" | "mp3">("wav");
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);
  // Track the current object URL so the cleanup revokes exactly what is live,
  // even across a callId change that starts a second fetch.
  const urlRef = useRef<string | null>(null);

  useEffect(() => {
    let cancelled = false;
    setLoading(true);
    setError(null);
    const controller = new AbortController();
    const headers: Record<string, string> = {};
    if (cfg.token) headers["Authorization"] = `Bearer ${cfg.token}`;

    fetch(`${cfg.baseURL}/api/v1/calls/${callId}/audio`, {
      headers,
      credentials: "include",
      signal: controller.signal,
    })
      .then(async (res) => {
        if (!res.ok) {
          throw new Error(
            res.status === 404
              ? "recording is unavailable (it may have been swept by retention)"
              : `recording request failed (${res.status})`,
          );
        }
        const blob = await res.blob();
        if (cancelled) return;
        const url = URL.createObjectURL(blob);
        urlRef.current = url;
        setExt(blob.type === "audio/mpeg" ? "mp3" : "wav");
        setObjURL(url);
        setLoading(false);
      })
      .catch((e: unknown) => {
        if (cancelled || (e instanceof DOMException && e.name === "AbortError"))
          return;
        setError(e instanceof Error ? e.message : "recording failed to load");
        setLoading(false);
      });

    return () => {
      cancelled = true;
      controller.abort();
      if (urlRef.current) {
        URL.revokeObjectURL(urlRef.current);
        urlRef.current = null;
      }
    };
  }, [cfg, callId]);

  if (error) {
    return (
      <p className="text-xs text-err" role="alert">
        {error}
      </p>
    );
  }
  if (loading || !objURL) {
    return <p className="text-xs text-muted">Loading recording…</p>;
  }
  return (
    <>
      <audio controls preload="metadata" className="w-full" src={objURL} />
      <a
        className="inline-block text-xs text-accent hover:underline mt-1"
        href={objURL}
        download={`call-${callId}.${ext}`}
      >
        Download recording
      </a>
    </>
  );
}
