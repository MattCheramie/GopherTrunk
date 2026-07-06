// High-quality band-limited resampler for the live-audio worklet path.
//
// The daemon streams PCM at the call's sample rate (8 kHz for digital and the
// analog default). The AudioWorklet that plays it runs at the AudioContext rate,
// which is the hardware-native rate (~44.1/48 kHz) — we no longer pin the context
// to 8 kHz because that silently rendered nothing on Windows/WASAPI. So the
// common path now resamples 8k -> 48k, and the cheap LinearResampler does it with
// linear interpolation: weak anti-imaging that leaks audible aliases above the
// 4 kHz input Nyquist, the "harsh / tinny" character that makes live sound worse
// than the recorded .wav (the file is resampled by the browser's native sinc path).
//
// SincResampler closes that gap with a windowed-sinc polyphase kernel — the
// same class of filter the browser uses for a recorded buffer — while keeping
// the two properties issue #629 depends on:
//   1. Continuous state across network chunks: one instance spans the whole
//      stream, carrying a history tail and a fractional read cursor so feeding
//      [a,b] then [c,d] yields exactly what feeding [a,b,c,d] at once would.
//   2. Zero-work passthrough when input and output rates match (e.g. a device
//      whose native rate happens to be 8 kHz), returning the input unchanged.
//
// It is a drop-in for LinearResampler: same constructor, `passthrough` getter,
// and process() contract.

// Zero-crossings of the sinc retained on each side of the interpolation point.
// 16 (a 32-tap effective window) gives a clean stopband for telephone-band
// speech without being expensive at ~48 kHz mono.
const HALF_WIDTH = 16;
// Sub-sample resolution of the precomputed polyphase table. The fractional
// position is quantized to the nearest of these phases, so 256 keeps the
// quantization error well below the int16 noise floor.
const PHASES = 256;
// Pull the cutoff slightly below the lower Nyquist so the transition band has
// room and the kernel doesn't ring against the very edge.
const ROLLOFF = 0.9;

// blackman returns the Blackman window weight at position x in [0, 1].
// Endpoints are 0; centre is 1. ~-58 dB peak sidelobe — ample for voice.
function blackman(x: number): number {
  return 0.42 - 0.5 * Math.cos(2 * Math.PI * x) + 0.08 * Math.cos(4 * Math.PI * x);
}

// sinc returns the normalized sinc sin(pi*x)/(pi*x), with the removable
// singularity at x = 0 handled explicitly.
function sinc(x: number): number {
  if (x === 0) return 1;
  const px = Math.PI * x;
  return Math.sin(px) / px;
}

export class SincResampler {
  // Input samples consumed per output sample (inputRate / outputRate).
  private readonly ratio: number;
  // kernel[phase] is a length-(2*HALF_WIDTH) row of filter taps for that
  // fractional offset, ordered from the leftmost neighbour to the rightmost.
  private readonly kernel: Float32Array[];
  // The last 2*HALF_WIDTH input samples seen, zero-filled at stream start so
  // the first outputs blend in silence — identical whether fed split or whole.
  private readonly history: Float32Array;
  // Position of the next output in input-sample coordinates, relative to the
  // start of the current chunk. Index -1 is history[last], -2 the one before,
  // etc. Starts at 0 so the first output is centred on the first real sample.
  private t = 0;

  constructor(inputRate: number, outputRate: number) {
    if (inputRate <= 0 || outputRate <= 0) {
      throw new Error("SincResampler: sample rates must be positive");
    }
    this.ratio = inputRate / outputRate;
    // Cutoff in input-sample cycles. When downsampling (outputRate < inputRate)
    // the band limit is the output Nyquist, so scale the cutoff down by the
    // rate ratio; when upsampling it stays at the input Nyquist.
    const cutoff = ROLLOFF * Math.min(1, outputRate / inputRate);
    this.kernel = SincResampler.buildKernel(cutoff);
    this.history = new Float32Array(2 * HALF_WIDTH);
  }

  // buildKernel precomputes one windowed-sinc tap row per fractional phase.
  // Each row is normalized to unity DC gain so a constant input is reproduced
  // exactly (no amplitude drift, and the DC-preservation test passes).
  private static buildKernel(cutoff: number): Float32Array[] {
    const rows: Float32Array[] = new Array(PHASES);
    const taps = 2 * HALF_WIDTH;
    for (let p = 0; p < PHASES; p++) {
      const frac = p / PHASES; // sub-sample offset of the output, in [0, 1)
      const row = new Float32Array(taps);
      let sum = 0;
      for (let k = 0; k < taps; k++) {
        // Tap k covers input neighbour (k - HALF_WIDTH + 1); its distance from
        // the output point is that index minus the fractional offset.
        const x = k - HALF_WIDTH + 1 - frac;
        const w = blackman((x + HALF_WIDTH) / taps);
        const h = cutoff * sinc(cutoff * x) * w;
        row[k] = h;
        sum += h;
      }
      const norm = sum !== 0 ? 1 / sum : 1;
      for (let k = 0; k < taps; k++) row[k] *= norm;
      rows[p] = row;
    }
    return rows;
  }

  /** True when input and output rates match and process() is a no-op. */
  get passthrough(): boolean {
    return this.ratio === 1;
  }

  // process resamples one chunk, emitting every output sample whose full kernel
  // support lies within the samples seen so far. Any output whose right edge
  // would read past this chunk's end is deferred to the next call, and the
  // chunk's tail is folded into the history so consecutive chunks join
  // seamlessly. Returns the input unchanged when passthrough.
  process(input: Float32Array): Float32Array {
    if (input.length === 0) return input;
    if (this.ratio === 1) return input;

    const { history, kernel } = this;
    const hist = history.length; // == 2 * HALF_WIDTH
    const len = input.length;
    // Read from the logical stream [history ... input]; index 0 is the first
    // history sample, `hist` the first input sample. A sample at history-frame
    // `f` (the input-sample coordinate, where -1 is history's last) maps to
    // logical index f + hist.
    const sampleAt = (logical: number): number =>
      logical < hist ? history[logical] : input[logical - hist];

    const out: number[] = [];
    let t = this.t;
    // The kernel centred at frame `t` (its right neighbour is ceil-side) reads
    // neighbours [floor(t) - HALF_WIDTH + 1 ... floor(t) + HALF_WIDTH]. We can
    // emit while the rightmost neighbour still exists, i.e. floor(t)+HALF_WIDTH
    // is within the input chunk: floor(t) <= len - 1 - HALF_WIDTH.
    while (Math.floor(t) <= len - 1 - HALF_WIDTH) {
      const i = Math.floor(t);
      const frac = t - i;
      const row = kernel[(frac * PHASES) | 0];
      // Leftmost neighbour index (in input-sample coordinates) is i-HALF_WIDTH+1;
      // its logical index adds `hist`.
      const base = i - HALF_WIDTH + 1 + hist;
      let acc = 0;
      for (let k = 0; k < row.length; k++) acc += row[k] * sampleAt(base + k);
      out.push(acc);
      t += this.ratio;
    }

    // Fold the chunk tail into history: the new history is the last `hist`
    // samples of [history ... input]. Shift the cursor by len so it stays
    // continuous across the boundary (next chunk's frame -1 is this tail's end).
    if (len >= hist) {
      history.set(input.subarray(len - hist));
    } else {
      history.copyWithin(0, len); // drop the `len` oldest history samples
      history.set(input, hist - len);
    }
    this.t = t - len;
    return Float32Array.from(out);
  }
}
