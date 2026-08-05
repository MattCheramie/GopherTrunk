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

import { SincResampler } from "./sincResampler";

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
  /** Factory for the AudioContext. Defaults to the browser's native
   *  constructor at the hardware rate. Injectable so tests can count how
   *  many contexts a player builds (the follow-switch leak that caused
   *  Chrome's ~6-context ceiling — and the resulting silence — to trip). */
  contextFactory?: () => AudioContext;
}

export interface StreamPlayer {
  /** Must be invoked synchronously from a user-gesture handler so the
   *  AudioContext resume isn't rejected by the autoplay policy. */
  start(): void;
  /** Re-point the running stream at a new URL (following a new call)
   *  WITHOUT tearing down the AudioContext/worklet. Reusing the one
   *  context is what keeps the WebUI under Chrome's per-page context
   *  limit — building a fresh context per call switch is what leaked to
   *  silence. Safe to call outside a user gesture: the context is already
   *  running from the initial start(). */
  reconnect(url: string): void;
  stop(): void;
  setVolume(v: number): void;
  setMuted(muted: boolean): void;
}

// Lead time before the first scheduled buffer — a jitter cushion so brief
// network hiccups and bursty chunk delivery don't cause audible gaps. 0.25 s
// roughly doubles the tolerance (and the ring's re-prime depth after an
// underrun) vs. the original 0.15 s while staying well under MAX_LEAD_SECONDS,
// so steady-state latency stays low.
const LEAD_SECONDS = 0.25;
// Upper bound on scheduled-ahead audio; past this we realign so a burst
// of buffered frames doesn't pin latency high for the rest of the call.
const MAX_LEAD_SECONDS = 0.6;
// Backoff before a single reconnect attempt when the stream ends on its
// own (e.g. the daemon restarted) while the user still wants audio.
const RECONNECT_DELAY_MS = 1000;
// Capacity of the AudioWorklet ring buffer, in seconds of audio. The producer
// and consumer run at the same rate so the ring hovers near the lead cushion;
// this only bounds the worst case (and the memory the worklet holds).
const MAX_RING_SECONDS = 2;

type AudioContextCtor = typeof AudioContext;

// The daemon streams 8 kHz PCM (vocoder-native for digital; the analog
// default). This is the wire rate, NOT the AudioContext rate: the WAV header
// carries it, PcmFramer reports it, and it seeds LegacySink before the header
// arrives. The playback context runs at the hardware-native rate instead (see
// createAudioContext) and WorkletSink's continuous SincResampler bridges
// 8k->native band-limited, like a recorded .wav.
//
// We used to pin the context to 8 kHz to skip resampling, but on Windows/WASAPI
// an 8 kHz AudioContext is often accepted without error yet renders no output —
// silent playback in every browser (#598 follow-up). Running at the native rate
// and letting the SincResampler (#629) do the 8k->48k upconversion is reliable
// across platforms and keeps the "as good as the file" quality the resampler
// was built for.
export const STREAM_SAMPLE_RATE = 8000;

function resolveAudioContext(): AudioContextCtor | null {
  const w = window as unknown as {
    AudioContext?: AudioContextCtor;
    webkitAudioContext?: AudioContextCtor;
  };
  return w.AudioContext ?? w.webkitAudioContext ?? null;
}

// createAudioContext builds the context at the hardware-native sample rate — no
// explicit sampleRate is requested. Forcing the stream's 8 kHz here made
// Windows/WASAPI silently render nothing (#598 follow-up); WorkletSink's
// SincResampler upconverts 8k->native instead. Pure except for the Ctor it is
// handed, so it is unit-testable.
export function createAudioContext(Ctor: AudioContextCtor): AudioContext {
  return new Ctor();
}

// A PcmSink consumes the decoded Float32 stream and renders it. configure() is
// called once per connection with the stream's sample rate (from the WAV
// header); push() delivers whole-sample chunks; reset() drops any buffered
// audio so a stop/reconnect doesn't replay stale frames.
interface PcmSink {
  configure(streamRate: number): void;
  push(samples: Float32Array): void;
  reset(): void;
}

// WorkletSink feeds the ring-buffer AudioWorklet (issue #629). It resamples the
// 8 kHz stream up to the native context rate with one continuous SincResampler
// (so chunk boundaries don't glitch and the 8k->48k conversion is band-limited
// like a recorded .wav, not the harsh linear interpolation of the old resampler)
// and hands the samples to the audio thread via a transferred buffer. The worklet
// emits silence on underrun, so network jitter no longer skips/realigns the
// playback cursor the way the legacy scheduler did.
export class WorkletSink implements PcmSink {
  private resampler: SincResampler | null = null;

  constructor(
    private readonly node: AudioWorkletNode,
    private readonly contextRate: number,
  ) {}

  configure(streamRate: number): void {
    this.resampler = new SincResampler(streamRate, this.contextRate);
    this.node.port.postMessage({ type: "reset" });
  }

  reset(): void {
    this.node.port.postMessage({ type: "reset" });
  }

  push(samples: Float32Array): void {
    if (samples.length === 0) return;
    const out = this.resampler ? this.resampler.process(samples) : samples;
    if (out.length === 0) return;
    // Transfer ownership to the audio thread — the run loop never reuses the
    // buffer, and each takeSamples() returns a fresh one, so this is safe.
    this.node.port.postMessage({ type: "push", samples: out }, [out.buffer]);
  }
}

// LegacySink is the original per-chunk AudioBufferSource scheduler, kept as a
// fallback for browsers without AudioWorklet (or when the worklet module can't
// load). It hands each buffer to the context at the stream rate and lets the
// context resample per-chunk to the native device rate — lower quality than the
// WorkletSink's continuous SincResampler, but audible (and far better than the
// silence the old 8 kHz context pin produced on Windows).
class LegacySink implements PcmSink {
  private nextStartTime = 0;
  private streamRate = STREAM_SAMPLE_RATE;

  constructor(
    private readonly ctx: AudioContext,
    private readonly gain: GainNode,
  ) {}

  configure(streamRate: number): void {
    this.streamRate = streamRate;
    this.nextStartTime = 0;
  }

  reset(): void {
    this.nextStartTime = 0;
  }

  push(samples: Float32Array): void {
    const { ctx, gain } = this;
    if (samples.length === 0) return;
    const buffer = ctx.createBuffer(1, samples.length, this.streamRate);
    buffer.getChannelData(0).set(samples);
    const src = ctx.createBufferSource();
    src.buffer = buffer;
    src.connect(gain);

    const now = ctx.currentTime;
    if (this.nextStartTime < now + 0.001) {
      // Underrun (or first buffer): we've fallen behind real time. Resync
      // forward with a fresh lead cushion rather than scheduling in the
      // past (which the API clamps to "now" and stacks).
      this.nextStartTime = now + LEAD_SECONDS;
    } else if (this.nextStartTime > now + MAX_LEAD_SECONDS) {
      // Too far ahead (a burst drained in one tick): pull the cursor back
      // so latency doesn't grow unbounded for the rest of the call.
      this.nextStartTime = now + MAX_LEAD_SECONDS;
    }
    src.start(this.nextStartTime);
    this.nextStartTime += buffer.duration;
  }
}

// createStreamPlayer wires a fetch reader to a PcmSink → GainNode → destination
// graph. One instance owns one AudioContext (reused across reconnects), one
// sink, and at most one in-flight fetch.
export function createStreamPlayer(
  opts: StreamPlayerOptions,
): StreamPlayer {
  let ctx: AudioContext | null = null;
  let gain: GainNode | null = null;
  let abort: AbortController | null = null;
  let sinkPromise: Promise<PcmSink> | null = null;
  let volume = 1;
  let muted = false;
  let userStopped = false;
  let reconnectTimer: number | null = null;
  // The stream URL is mutable so following a new call can re-point the fetch
  // (see reconnect) while keeping the same AudioContext/worklet alive.
  let url = opts.url;

  const applyGain = () => {
    if (gain && ctx) {
      gain.gain.setTargetAtTime(muted ? 0 : volume, ctx.currentTime, 0.01);
    }
  };

  const ensureGraph = (): boolean => {
    if (ctx === null) {
      if (opts.contextFactory) {
        ctx = opts.contextFactory();
      } else {
        const Ctor = resolveAudioContext();
        if (Ctor === null) {
          opts.onError("Web Audio is not supported in this browser");
          opts.onStateChange("error");
          return false;
        }
        ctx = createAudioContext(Ctor);
      }
    }
    if (gain === null) {
      gain = ctx.createGain();
      gain.connect(ctx.destination);
    }
    applyGain();
    return true;
  };

  // getSink lazily loads the AudioWorklet module and builds the ring-buffer
  // sink, caching it for reuse across reconnects. If worklets are unavailable
  // or the module fails to load (old browser, a CSP that blocks worklets, or a
  // proxy that mangles the asset) it falls back to per-chunk buffer scheduling
  // so audio still plays. Either outcome is cached.
  const getSink = (): Promise<PcmSink> => {
    if (sinkPromise) return sinkPromise;
    sinkPromise = (async (): Promise<PcmSink> => {
      const audioCtx = ctx!;
      const out = gain!;
      if (audioCtx.audioWorklet) {
        try {
          await audioCtx.audioWorklet.addModule(
            new URL("./ringBufferProcessor.js", import.meta.url),
          );
          const node = new AudioWorkletNode(audioCtx, "ringbuffer-player", {
            processorOptions: {
              capacityFrames: Math.ceil(audioCtx.sampleRate * MAX_RING_SECONDS),
              primeFrames: Math.ceil(audioCtx.sampleRate * LEAD_SECONDS),
            },
          });
          node.connect(out);
          return new WorkletSink(node, audioCtx.sampleRate);
        } catch (err) {
          console.warn(
            "GopherTrunk: AudioWorklet unavailable, using buffer scheduling:",
            err,
          );
        }
      }
      return new LegacySink(audioCtx, out);
    })();
    return sinkPromise;
  };

  const run = async () => {
    abort = new AbortController();
    opts.onStateChange("connecting");
    let res: Response;
    try {
      res = await fetch(url, {
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
    const sink = await getSink();
    sink.reset();
    let started = false;
    let configured = false;
    try {
      for (;;) {
        const { value, done } = await reader.read();
        if (done) break;
        if (!value) continue;
        framer.feed(value);
        // Configure the sink's resampler as soon as the header's rate is known.
        if (!configured && framer.format) {
          sink.configure(framer.format.sampleRate);
          configured = true;
        }
        for (;;) {
          const samples = framer.takeSamples();
          if (samples === null) break;
          sink.push(samples);
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
      // Warm the worklet module now so it loads alongside the fetch rather
      // than serially after it; run() awaits the same cached promise.
      void getSink();
      void run();
    },
    reconnect(newURL: string) {
      // Follow a new call by swapping the fetch URL and re-running on the
      // SAME context/gain/worklet. Building a fresh AudioContext per switch
      // (the old behaviour) leaked contexts until Chrome's ~6-per-page cap
      // threw and playback went silent. Here nothing is torn down: abort the
      // in-flight read, drop its buffered tail, and reopen.
      userStopped = false;
      if (reconnectTimer !== null) {
        window.clearTimeout(reconnectTimer);
        reconnectTimer = null;
      }
      if (url === newURL && abort) return; // already following this call
      url = newURL;
      // No context yet (reconnect raced ahead of the initial start): just
      // remember the URL; start() will open it inside the user gesture.
      if (ctx === null) return;
      abort?.abort();
      abort = null;
      // Keep the context live and drop the previous call's buffered audio so
      // the new call doesn't start behind stale frames. The worklet's fade
      // envelope smooths the seam.
      void ctx?.resume();
      void sinkPromise?.then((sink) => sink.reset());
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
      // Close (not just suspend) and drop the graph so no AudioContext
      // lingers past teardown — suspend-only accumulation past Chrome's
      // per-page limit was the silence bug. start() rebuilds from scratch.
      void sinkPromise?.then((sink) => sink.reset());
      void ctx?.close();
      ctx = null;
      gain = null;
      sinkPromise = null;
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
