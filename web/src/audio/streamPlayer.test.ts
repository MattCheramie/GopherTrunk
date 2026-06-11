import { describe, it, expect } from "vitest";
import {
  parseWavHeader,
  int16ToFloat32,
  PcmFramer,
  WAV_HEADER_SIZE,
  createAudioContext,
  STREAM_SAMPLE_RATE,
} from "./streamPlayer";

// Build a canonical 44-byte RIFF/WAVE header byte-for-byte the way the
// daemon does in internal/api/handlers_audio_stream.go
// (streamingWAVHeader): 16-bit little-endian mono PCM at the given rate.
// Pinning the layout here is the whole point — the framer's offsets must
// match the server's.
function makeHeader(sampleRate: number): Uint8Array {
  const buf = new Uint8Array(WAV_HEADER_SIZE);
  const dv = new DataView(buf.buffer);
  const ascii = (off: number, s: string) => {
    for (let i = 0; i < s.length; i++) buf[off + i] = s.charCodeAt(i);
  };
  const dataSize = 0x7fffffff;
  ascii(0, "RIFF");
  dv.setUint32(4, dataSize - 36, true);
  ascii(8, "WAVE");
  ascii(12, "fmt ");
  dv.setUint32(16, 16, true);
  dv.setUint16(20, 1, true); // PCM
  dv.setUint16(22, 1, true); // mono
  dv.setUint32(24, sampleRate, true);
  dv.setUint32(28, (sampleRate * 1 * 16) / 8, true);
  dv.setUint16(32, 2, true);
  dv.setUint16(34, 16, true);
  ascii(36, "data");
  dv.setUint32(40, dataSize, true);
  return buf;
}

// Encode int16 samples to a little-endian byte buffer.
function pcmBytes(samples: number[]): Uint8Array {
  const buf = new Uint8Array(samples.length * 2);
  const dv = new DataView(buf.buffer);
  samples.forEach((s, i) => dv.setInt16(i * 2, s, true));
  return buf;
}

function concat(...parts: Uint8Array[]): Uint8Array {
  const total = parts.reduce((n, p) => n + p.length, 0);
  const out = new Uint8Array(total);
  let off = 0;
  for (const p of parts) {
    out.set(p, off);
    off += p.length;
  }
  return out;
}

describe("parseWavHeader", () => {
  it("extracts the format from a canonical 8 kHz header", () => {
    const fmt = parseWavHeader(makeHeader(8000));
    expect(fmt).toEqual({ sampleRate: 8000, channels: 1, bitsPerSample: 16 });
  });

  it("reads the sample rate from the header (not hardcoded)", () => {
    expect(parseWavHeader(makeHeader(16000))?.sampleRate).toBe(16000);
  });

  it("returns null when fewer than 44 bytes are buffered", () => {
    expect(parseWavHeader(makeHeader(8000).subarray(0, 30))).toBeNull();
  });

  it("throws on bad RIFF magic", () => {
    const bad = makeHeader(8000);
    bad[0] = "X".charCodeAt(0);
    expect(() => parseWavHeader(bad)).toThrow(/RIFF/);
  });

  it("throws on bad WAVE magic", () => {
    const bad = makeHeader(8000);
    bad[8] = "X".charCodeAt(0);
    expect(() => parseWavHeader(bad)).toThrow(/WAVE/);
  });
});

describe("int16ToFloat32", () => {
  it("maps boundary values into [-1, 1)", () => {
    const out = int16ToFloat32(pcmBytes([0, -32768, 32767, 1]));
    expect(out[0]).toBe(0);
    expect(out[1]).toBe(-1);
    expect(out[2]).toBeCloseTo(0.99997, 4);
    expect(out[3]).toBeCloseTo(1 / 32768, 8);
  });

  it("decodes little-endian byte order", () => {
    // 0x0100 little-endian = 1.
    const out = int16ToFloat32(new Uint8Array([0x01, 0x00]));
    expect(out[0]).toBeCloseTo(1 / 32768, 8);
  });
});

describe("PcmFramer", () => {
  it("decodes header + samples delivered in one chunk", () => {
    const f = new PcmFramer();
    f.feed(concat(makeHeader(8000), pcmBytes([100, -100, 200])));
    expect(f.format).toBeNull(); // not parsed until takeSamples
    const got = f.takeSamples();
    expect(f.format?.sampleRate).toBe(8000);
    expect(Array.from(got!)).toEqual([100 / 32768, -100 / 32768, 200 / 32768]);
  });

  it("waits for the header to fully arrive across feeds", () => {
    const f = new PcmFramer();
    const header = makeHeader(8000);
    f.feed(header.subarray(0, 30));
    expect(f.takeSamples()).toBeNull();
    expect(f.format).toBeNull();
    f.feed(header.subarray(30));
    f.feed(pcmBytes([7]));
    const got = f.takeSamples();
    expect(f.format?.sampleRate).toBe(8000);
    expect(Array.from(got!)).toEqual([7 / 32768]);
  });

  it("carries a trailing odd byte to the next feed", () => {
    const f = new PcmFramer();
    const body = pcmBytes([300, 400]); // 4 bytes
    f.feed(concat(makeHeader(8000), body.subarray(0, 3))); // header + 3 bytes
    expect(Array.from(f.takeSamples()!)).toEqual([300 / 32768]); // 1 sample, 1 byte held
    expect(f.takeSamples()).toBeNull(); // odd byte alone is not a sample
    f.feed(body.subarray(3)); // completes the second sample
    expect(Array.from(f.takeSamples()!)).toEqual([400 / 32768]);
  });

  it("reconstructs an int16 split across the chunk boundary", () => {
    const f = new PcmFramer();
    const body = pcmBytes([-12345]);
    f.feed(concat(makeHeader(8000), body.subarray(0, 1))); // low byte only
    expect(f.takeSamples()).toBeNull();
    f.feed(body.subarray(1)); // high byte
    expect(Array.from(f.takeSamples()!)).toEqual([-12345 / 32768]);
  });

  it("byte-by-byte feeds yield the same result as one big feed", () => {
    const stream = concat(
      makeHeader(8000),
      pcmBytes([1, -2, 3, -4, 5, -6, 7, -8]),
    );

    const bulk = new PcmFramer();
    bulk.feed(stream);
    const bulkOut = Array.from(bulk.takeSamples()!);

    const drip = new PcmFramer();
    const dripOut: number[] = [];
    for (const byte of stream) {
      drip.feed(new Uint8Array([byte]));
      const got = drip.takeSamples();
      if (got) dripOut.push(...got);
    }
    expect(dripOut).toEqual(bulkOut);
  });

  it("ignores empty chunks", () => {
    const f = new PcmFramer();
    f.feed(new Uint8Array(0));
    f.feed(concat(makeHeader(8000), pcmBytes([42])));
    f.feed(new Uint8Array(0));
    expect(Array.from(f.takeSamples()!)).toEqual([42 / 32768]);
  });
});

describe("createAudioContext", () => {
  // Pinning the context to the stream's 8 kHz rate is what lets Web Audio skip
  // per-chunk resampling (#598); assert the rate is actually requested.
  it("builds the context at the requested sample rate", () => {
    const seen: (AudioContextOptions | undefined)[] = [];
    class FakeCtx {
      constructor(opts?: AudioContextOptions) {
        seen.push(opts);
      }
    }
    const ctx = createAudioContext(
      FakeCtx as unknown as typeof AudioContext,
      STREAM_SAMPLE_RATE,
    );
    expect(ctx).toBeInstanceOf(FakeCtx);
    expect(seen).toEqual([{ sampleRate: 8000 }]);
  });

  // Older Safari throws when handed an explicit sampleRate; we must still get a
  // usable (hardware-default) context rather than failing playback outright.
  it("falls back to a default context when the rate is rejected", () => {
    let calls = 0;
    class PickyCtx {
      constructor(opts?: AudioContextOptions) {
        calls++;
        if (opts) throw new Error("sampleRate not supported");
      }
    }
    const ctx = createAudioContext(
      PickyCtx as unknown as typeof AudioContext,
      STREAM_SAMPLE_RATE,
    );
    expect(ctx).toBeInstanceOf(PickyCtx);
    expect(calls).toBe(2); // first (with opts) throws, second (no opts) succeeds
  });
});
