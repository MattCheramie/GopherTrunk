// Live-audio player for the WebUI. Streams the daemon's continuous WAV
// body (GET /api/v1/audio/stream) and plays it through the Web Audio
// API instead of a hidden <audio> element.
//
// Issue #598: the <audio src=stream> approach failed silently in Chrome
// on macOS — an open-ended chunked "infinite WAV" is unreliable for the
// media element, and the failure was swallowed in a console.warn. A
// fetch() reader + AudioContext is deterministic across Chrome / Firefox
// / Safari, sidesteps the whole <audio>/Range/206/Content-Length class
// of bugs, works the same whether or not a reverse proxy sits in front,
// and (unlike <audio>) can send the Authorization: Bearer header for
// auth-gated daemons.
//
// The wire format is a 44-byte canonical RIFF/WAVE header (16-bit
// little-endian mono PCM, sample rate in header bytes 24-28) followed by
// concatenated raw int16 samples — see streamingWAVHeader in
// internal/api/handlers_audio_stream.go. The byte-framing logic below
// must match that layout.

// ---------------------------------------------------------------------------
// Pure, testable framing helpers (no DOM / AudioContext).
// ---------------------------------------------------------------------------

export interface WavFormat {
  sampleRate: number;
  channels: number;
  bitsPerSample: number;
}

/** Size of the canonical RIFF/WAVE header the daemon emits. */
export const WAV_HEADER_SIZE = 44;

// parseWavHeader validates the RIFF/WAVE magic and extracts the audio
// format from the first 44 bytes. Returns null when fewer than 44 bytes
// are buffered (the caller should wait for more), and throws when the
// magic is wrong so a corrupt stream surfaces as a visible error rather
// than playing noise.
export function parseWavHeader(bytes: Uint8Array): WavFormat | null {
  if (bytes.length < WAV_HEADER_SIZE) return null;
  const ascii = (off: number, len: number) =>
    String.fromCharCode(...bytes.subarray(off, off + len));
  if (ascii(0, 4) !== "RIFF") {
    throw new Error(`bad WAV header: expected RIFF, got "${ascii(0, 4)}"`);
  }
  if (ascii(8, 4) !== "WAVE") {
    throw new Error(`bad WAV header: expected WAVE, got "${ascii(8, 4)}"`);
  }
  const dv = new DataView(bytes.buffer, bytes.byteOffset, WAV_HEADER_SIZE);
  return {
    channels: dv.getUint16(22, true),
    sampleRate: dv.getUint32(24, true),
    bitsPerSample: dv.getUint16(34, true),
  };
}

// int16ToFloat32 converts a little-endian int16 PCM byte buffer to
// Float32 samples in [-1, 1). The input length must be even; callers use
// PcmFramer to guarantee that.
export function int16ToFloat32(bytes: Uint8Array): Float32Array {
  const count = bytes.length >> 1;
  const out = new Float32Array(count);
  const dv = new DataView(bytes.buffer, bytes.byteOffset, count * 2);
  for (let i = 0; i < count; i++) {
    out[i] = dv.getInt16(i * 2, true) / 32768;
  }
  return out;
}

// PcmFramer reassembles the WAV stream from arbitrarily-sized network
// reads. A chunk boundary can fall anywhere — mid-header, or between the
// two bytes of an int16 sample — so the framer accumulates bytes,
// consumes the 44-byte header exactly once, and only ever yields whole
// samples, carrying a trailing odd byte forward to the next chunk.
export class PcmFramer {
  private buf = new Uint8Array(0);
  private fmt: WavFormat | null = null;

  /** The parsed WAV format, or null until the 44-byte header arrives. */
  get format(): WavFormat | null {
    return this.fmt;
  }

  /** Append a network chunk. Cheap; decoding happens in takeSamples. */
  feed(chunk: Uint8Array): void {
    if (chunk.length === 0) return;
    const next = new Uint8Array(this.buf.length + chunk.length);
    next.set(this.buf, 0);
    next.set(chunk, this.buf.length);
    this.buf = next;
  }

  // takeSamples drains every whole int16 sample currently buffered (after
  // the header) and returns it as Float32. Returns null when the header
  // hasn't fully arrived yet or there isn't at least one whole sample.
  takeSamples(): Float32Array | null {
    if (this.fmt === null) {
      const fmt = parseWavHeader(this.buf);
      if (fmt === null) return null; // header still incomplete
      this.fmt = fmt;
      this.buf = this.buf.subarray(WAV_HEADER_SIZE);
    }
    const whole = this.buf.length - (this.buf.length % 2);
    if (whole < 2) return null; // no complete sample yet
    const samples = int16ToFloat32(this.buf.subarray(0, whole));
    this.buf = this.buf.subarray(whole); // keep the trailing odd byte (if any)
    return samples;
  }
}

// ---------------------------------------------------------------------------
// AudioContext-backed streaming controller.
// ---------------------------------------------------------------------------

export type PlayerState =
  | "idle"
  | "connecting"
  | "playing"
  | "stopped"
  | "error";

export interface StreamPlayerOptions {
  url: string;
  token: string | null;
  onError: (msg: string) => void;
  onStateChange: (state: PlayerState) => void;
}

export interface StreamPlayer {
  /** Must be invoked synchronously from a user-gesture handler so the
   *  AudioContext resume isn't rejected by the autoplay policy. */
  start(): void;
  stop(): void;
  setVolume(v: number): void;
  setMuted(muted: boolean): void;
}

// Lead time before the first scheduled buffer — a small jitter cushion so
// brief network hiccups don't cause audible gaps.
const LEAD_SECONDS = 0.15;
// Upper bound on scheduled-ahead audio; past this we realign so a burst
// of buffered frames doesn't pin latency high for the rest of the call.
const MAX_LEAD_SECONDS = 0.6;
// Backoff before a single reconnect attempt when the stream ends on its
// own (e.g. the daemon restarted) while the user still wants audio.
const RECONNECT_DELAY_MS = 1000;

type AudioContextCtor = typeof AudioContext;

// The daemon streams 8 kHz PCM (vocoder-native for digital; the analog
// default). Creating the AudioContext at this rate makes Web Audio do ZERO
// per-chunk resampling — every scheduled buffer matches the context rate and
// plays back-to-back seamlessly, and the OS does one continuous resample to the
// output device, exactly like a recorded .wav. Without it, every small chunk is
// resampled 8k->48k independently with no interpolation state across chunk
// boundaries, the artifacts that make live sound worse than the file (#598).
export const STREAM_SAMPLE_RATE = 8000;

function resolveAudioContext(): AudioContextCtor | null {
  const w = window as unknown as {
    AudioContext?: AudioContextCtor;
    webkitAudioContext?: AudioContextCtor;
  };
  return w.AudioContext ?? w.webkitAudioContext ?? null;
}

// createAudioContext pins the context to the stream rate, falling back to the
// hardware-default context if the browser rejects an explicit sampleRate (older
// Safari throws). Pure except for the Ctor it is handed, so it is unit-testable.
export function createAudioContext(
  Ctor: AudioContextCtor,
  rate: number,
): AudioContext {
  try {
    return new Ctor({ sampleRate: rate });
  } catch {
    return new Ctor();
  }
}

// createStreamPlayer wires a fetch reader to a GainNode → destination
// graph. One instance owns one AudioContext (reused across reconnects)
// and at most one in-flight fetch.
export function createStreamPlayer(
  opts: StreamPlayerOptions,
): StreamPlayer {
  let ctx: AudioContext | null = null;
  let gain: GainNode | null = null;
  let abort: AbortController | null = null;
  let nextStartTime = 0;
  let volume = 1;
  let muted = false;
  let userStopped = false;
  let reconnectTimer: number | null = null;

  const applyGain = () => {
    if (gain && ctx) {
      gain.gain.setTargetAtTime(muted ? 0 : volume, ctx.currentTime, 0.01);
    }
  };

  const ensureGraph = (): boolean => {
    if (ctx === null) {
      const Ctor = resolveAudioContext();
      if (Ctor === null) {
        opts.onError("Web Audio is not supported in this browser");
        opts.onStateChange("error");
        return false;
      }
      ctx = createAudioContext(Ctor, STREAM_SAMPLE_RATE);
    }
    if (gain === null) {
      gain = ctx.createGain();
      gain.connect(ctx.destination);
    }
    applyGain();
    return true;
  };

  const schedule = (samples: Float32Array, sampleRate: number) => {
    if (ctx === null || gain === null || samples.length === 0) return;
    const buffer = ctx.createBuffer(1, samples.length, sampleRate);
    buffer.getChannelData(0).set(samples);
    const src = ctx.createBufferSource();
    src.buffer = buffer;
    src.connect(gain);

    const now = ctx.currentTime;
    if (nextStartTime < now + 0.001) {
      // Underrun (or first buffer): we've fallen behind real time. Resync
      // forward with a fresh lead cushion rather than scheduling in the
      // past (which the API clamps to "now" and stacks).
      nextStartTime = now + LEAD_SECONDS;
    } else if (nextStartTime > now + MAX_LEAD_SECONDS) {
      // Too far ahead (a burst drained in one tick): pull the cursor back
      // so latency doesn't grow unbounded for the rest of the call.
      nextStartTime = now + MAX_LEAD_SECONDS;
    }
    src.start(nextStartTime);
    nextStartTime += buffer.duration;
  };

  const run = async () => {
    abort = new AbortController();
    opts.onStateChange("connecting");
    let res: Response;
    try {
      res = await fetch(opts.url, {
        headers: opts.token
          ? { Authorization: `Bearer ${opts.token}` }
          : {},
        credentials: "include",
        signal: abort.signal,
      });
    } catch (err) {
      if (isAbort(err)) return;
      fail(`stream request failed: ${describe(err)}`);
      return;
    }
    if (!res.ok) {
      fail(`stream HTTP ${res.status}`);
      return;
    }
    if (!res.body) {
      fail("stream response had no body");
      return;
    }

    const framer = new PcmFramer();
    const reader = res.body.getReader();
    nextStartTime = 0;
    let started = false;
    try {
      for (;;) {
        const { value, done } = await reader.read();
        if (done) break;
        if (!value) continue;
        framer.feed(value);
        const fmt = framer.format;
        for (;;) {
          const samples = framer.takeSamples();
          if (samples === null) break;
          const rate = framer.format?.sampleRate ?? fmt?.sampleRate ?? 8000;
          schedule(samples, rate);
          if (!started) {
            started = true;
            opts.onStateChange("playing");
          }
        }
      }
    } catch (err) {
      if (isAbort(err)) return;
      fail(`stream read failed: ${describe(err)}`);
      return;
    }

    // The stream ended on its own. If the user still wants audio (didn't
    // press stop), try a single reconnect after a short backoff to ride
    // out a daemon restart.
    opts.onStateChange("stopped");
    if (!userStopped) {
      reconnectTimer = window.setTimeout(() => {
        reconnectTimer = null;
        if (!userStopped) void run();
      }, RECONNECT_DELAY_MS);
    }
  };

  const fail = (msg: string) => {
    opts.onError(msg);
    opts.onStateChange("error");
  };

  return {
    start() {
      userStopped = false;
      if (reconnectTimer !== null) {
        window.clearTimeout(reconnectTimer);
        reconnectTimer = null;
      }
      if (!ensureGraph() || ctx === null) return;
      // Resume synchronously within the gesture so the autoplay policy
      // doesn't reject playback. fetch() is awaited inside run().
      void ctx.resume();
      void run();
    },
    stop() {
      userStopped = true;
      if (reconnectTimer !== null) {
        window.clearTimeout(reconnectTimer);
        reconnectTimer = null;
      }
      abort?.abort();
      abort = null;
      void ctx?.suspend();
      opts.onStateChange("stopped");
    },
    setVolume(v: number) {
      volume = Math.max(0, Math.min(1, v));
      applyGain();
    },
    setMuted(m: boolean) {
      muted = m;
      applyGain();
    },
  };
}

function isAbort(err: unknown): boolean {
  return err instanceof DOMException && err.name === "AbortError";
}

function describe(err: unknown): string {
  if (err instanceof Error) return err.message;
  return String(err);
}
