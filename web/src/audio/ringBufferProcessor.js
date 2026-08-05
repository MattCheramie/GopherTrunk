// AudioWorklet ring-buffer player for the live-audio stream (issue #629).
//
// The main thread resamples each network chunk to the context rate (see
// resampler.ts) and posts the Float32 samples here via port.postMessage with a
// transferred ArrayBuffer. This processor owns a fixed-size ring buffer and, on
// every render quantum, copies samples out to the output — and crucially writes
// SILENCE on underrun instead of stalling or repositioning the playback cursor.
// That replaces the old per-chunk AudioBufferSource scheduler, whose
// nextStartTime skip/realign produced audible glitches under network jitter.
//
// Plain JS with no imports/exports so addModule() can load it as a classic
// AudioWorklet module untouched by the bundler. `sampleRate` is the global
// AudioWorkletGlobalScope context rate.

class RingBufferProcessor extends AudioWorkletProcessor {
  constructor(options) {
    super();
    const opts = (options && options.processorOptions) || {};
    // Upper bound on buffered audio: caps end-to-end latency and the memory the
    // ring holds. Producer and consumer run at the same (context) rate, so the
    // ring hovers near the prime level rather than filling.
    this.capacity = Math.max(1, opts.capacityFrames || Math.ceil(sampleRate * 2));
    // Jitter cushion: stay silent until this many frames are buffered, then
    // drain. Re-primed after every underrun so playback resumes cleanly.
    this.prime = Math.min(this.capacity, Math.max(1, opts.primeFrames || Math.ceil(sampleRate * 0.25)));
    this.ring = new Float32Array(this.capacity);
    this.read = 0;
    this.write = 0;
    this.available = 0; // frames currently buffered
    this.draining = false; // false until primed; flips back on underrun

    // Boundary-declick envelope. A short amplitude ramp at prime-start, mid-
    // stream call seams, and the pre-underrun tail removes the broadband click
    // ("DC spike") of starting/stopping on a non-zero PCM sample — the same
    // per-clip fade rdio-scanner applies. Logic mirrors ../audio/fadeEnvelope.ts
    // (kept inline: this worklet is loaded raw by addModule and can't import);
    // fadeEnvelope.test.ts pins it. ~5 ms ramp, and the tail fade begins once
    // the ring is within `release` frames of empty. Release is capped below the
    // prime depth so steady playback (ring hovers near prime) stays at unity.
    const ramp = Math.max(1, Math.round(sampleRate * 0.005));
    this.envStep = 1 / ramp;
    this.envRelease = Math.min(ramp, Math.max(1, Math.floor(this.prime / 2)));
    this.env = 0; // current envelope gain, ramps in [0, 1]

    this.port.onmessage = (event) => {
      const msg = event.data;
      if (!msg) return;
      if (msg.type === "reset") {
        this.read = 0;
        this.write = 0;
        this.available = 0;
        this.draining = false;
        this.env = 0; // re-fade in on the next call rather than stepping in
        return;
      }
      if (msg.type === "push" && msg.samples) {
        this.enqueue(msg.samples);
      }
    };
  }

  enqueue(samples) {
    for (let i = 0; i < samples.length; i++) {
      this.ring[this.write] = samples[i];
      this.write = (this.write + 1) % this.capacity;
      if (this.available < this.capacity) {
        this.available++;
      } else {
        // Overflow (producer outran consumer): drop the oldest frame so latency
        // stays bounded rather than letting the cursor lap itself.
        this.read = (this.read + 1) % this.capacity;
      }
    }
  }

  process(_inputs, outputs) {
    const channel = outputs[0] && outputs[0][0];
    if (!channel) return true;

    if (!this.draining && this.available >= this.prime) {
      this.draining = true;
    }
    // Note: still run the sample loop even when not draining so the envelope
    // can finish ramping DOWN through this quantum (a fade-out that started as
    // the ring emptied completes into the silence instead of clicking off).

    for (let i = 0; i < channel.length; i++) {
      // Envelope target: 0 while silent or within `envRelease` of empty (fade
      // the tail before the hard cut), 1 during healthy playback. Mirrors
      // fadeEnvelope.ts envelopeTarget().
      let target;
      if (!this.draining) {
        target = 0;
      } else if (this.available <= this.envRelease) {
        target = 0;
      } else {
        target = 1;
      }
      // Step the envelope toward the target (fadeEnvelope.ts nextEnvelopeGain).
      if (this.env < target) {
        this.env = Math.min(target, this.env + this.envStep);
      } else if (this.env > target) {
        this.env = Math.max(target, this.env - this.envStep);
      }

      let sample = 0;
      if (this.draining && this.available > 0) {
        sample = this.ring[this.read];
        this.read = (this.read + 1) % this.capacity;
        this.available--;
      } else if (this.draining) {
        // Underrun: out of buffered frames. Emit (faded) silence and re-prime
        // before draining again.
        this.draining = false;
      }
      channel[i] = sample * this.env;
    }
    return true;
  }
}

registerProcessor("ringbuffer-player", RingBufferProcessor);
