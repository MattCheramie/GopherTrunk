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

    this.port.onmessage = (event) => {
      const msg = event.data;
      if (!msg) return;
      if (msg.type === "reset") {
        this.read = 0;
        this.write = 0;
        this.available = 0;
        this.draining = false;
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
    if (!this.draining) {
      channel.fill(0);
      return true;
    }

    for (let i = 0; i < channel.length; i++) {
      if (this.available > 0) {
        channel[i] = this.ring[this.read];
        this.read = (this.read + 1) % this.capacity;
        this.available--;
      } else {
        // Underrun: emit silence and re-prime before draining again.
        channel[i] = 0;
        this.draining = false;
      }
    }
    return true;
  }
}

registerProcessor("ringbuffer-player", RingBufferProcessor);
