// Continuous linear-interpolation resampler for the live-audio path.
//
// The daemon streams PCM at the call's sample rate (8 kHz for digital and the
// analog default; occasionally another rate for analog). The AudioWorklet that
// plays it runs at the AudioContext rate — normally the 8 kHz we pin, but the
// hardware rate (~48 kHz) when a browser rejects the explicit rate. When those
// differ we must resample, and doing it per network chunk with no shared state
// reintroduces the exact boundary artifacts issue #598 set out to remove. One
// LinearResampler instance therefore spans the whole stream: it carries the
// fractional read cursor and the last input sample across calls so consecutive
// chunks join seamlessly. When the rates match it is a zero-work passthrough.

export class LinearResampler {
  // Input samples consumed per output sample (inputRate / outputRate).
  private readonly ratio: number;
  // Position of the next output, in input-sample coordinates relative to the
  // current chunk, where index -1 is `prev` (the previous chunk's last sample).
  // Starts at 0 so the first chunk never reads the (nonexistent) `prev`.
  private t = 0;
  private prev = 0;

  constructor(inputRate: number, outputRate: number) {
    if (inputRate <= 0 || outputRate <= 0) {
      throw new Error("LinearResampler: sample rates must be positive");
    }
    this.ratio = inputRate / outputRate;
  }

  /** True when input and output rates match and process() is a no-op. */
  get passthrough(): boolean {
    return this.ratio === 1;
  }

  // process resamples one chunk, emitting every output sample whose
  // interpolation window lies wholly within the samples seen so far. Any
  // remaining fractional position and the chunk's last sample are carried into
  // the next call, so feeding [a,b] then [c,d] yields exactly what feeding
  // [a,b,c,d] at once would. Returns the input unchanged when passthrough.
  process(input: Float32Array): Float32Array {
    if (input.length === 0) return input;
    if (this.ratio === 1) return input;

    const len = input.length;
    const out: number[] = [];
    let t = this.t;
    // Produce while the upper interpolation neighbour input[i+1] exists, i.e.
    // while t < len - 1. Samples landing exactly on the final boundary are
    // deferred to the next chunk (where this sample becomes `prev`).
    while (t < len - 1) {
      const i = Math.floor(t);
      const frac = t - i;
      const a = i < 0 ? this.prev : input[i];
      const b = input[i + 1];
      out.push(a + (b - a) * frac);
      t += this.ratio;
    }
    // Carry: the next chunk's index -1 is this chunk's last sample, so shift the
    // cursor by len to keep it continuous across the boundary.
    this.prev = input[len - 1];
    this.t = t - len;
    return Float32Array.from(out);
  }
}
