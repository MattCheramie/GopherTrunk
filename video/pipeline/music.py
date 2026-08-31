"""Procedural ambient music bed — license-free, calm/technical.

A soft sustained chord pad (root + fifth + octave + ninth partials) with slow
independent amplitude LFOs, a gentle filtered-noise airbed, and a very quiet
slow pulse. Deterministic per seed. 48 kHz stereo float32 → int16 WAV.
"""
import wave

import numpy as np

RATE = 48000


def _tone(t, f, lfo_rate, lfo_phase, detune=0.15):
    # two slightly detuned sines per voice, slow tremolo
    a = np.sin(2 * np.pi * f * t) + np.sin(2 * np.pi * (f + detune) * t + 1.3)
    trem = 0.6 + 0.4 * np.sin(2 * np.pi * lfo_rate * t + lfo_phase)
    return a * trem


def generate(dur, out, seed=0, gain_db=-34.0, fade_in=0.8, fade_out=1.2):
    rng = np.random.default_rng(42 + seed)
    n = int(dur * RATE)
    t = np.arange(n) / RATE
    # A minor-ish pad: A2, E3, A3, B3 (adds a soft 9th)
    voices = [(110.0, 0.05), (164.81, 0.041), (220.0, 0.033), (246.94, 0.027)]
    weights = [1.0, 0.7, 0.5, 0.28]
    pad = np.zeros(n)
    for (f, lr), w in zip(voices, weights):
        pad += w * _tone(t, f, lr, rng.uniform(0, 6.28))
    pad /= np.max(np.abs(pad)) + 1e-9
    # airbed: lowpassed noise (one-pole) with slow swell
    noise = rng.standard_normal(n)
    lp = np.empty(n)
    acc = 0.0
    k = 0.0015
    for i in range(n):  # simple one-pole; fine at this length
        acc += k * (noise[i] - acc)
        lp[i] = acc
    lp /= np.max(np.abs(lp)) + 1e-9
    swell = 0.5 + 0.5 * np.sin(2 * np.pi * 0.017 * t + 2.0)
    mix = pad * 0.85 + lp * 0.25 * swell
    # stereo: tiny decorrelating delay + width
    d = int(0.011 * RATE)
    left = mix
    right = np.concatenate([mix[d:], mix[:d]]) * 0.97
    st = np.stack([left, right], axis=1)
    # fades + level
    env = np.ones(n)
    fi, fo = int(fade_in * RATE), int(fade_out * RATE)
    env[:fi] = np.linspace(0, 1, fi)
    env[-fo:] = np.linspace(1, 0, fo)
    st *= env[:, None]
    st *= 10 ** (gain_db / 20) / (np.sqrt(np.mean(st ** 2)) + 1e-9) * 0.7
    pcm = np.clip(st * 32767, -32767, 32767).astype(np.int16)
    with wave.open(out, "wb") as w:
        w.setnchannels(2)
        w.setsampwidth(2)
        w.setframerate(RATE)
        w.writeframes(pcm.tobytes())
    return out


if __name__ == "__main__":
    import sys
    generate(float(sys.argv[1]), sys.argv[2], seed=int(sys.argv[3]) if len(sys.argv) > 3 else 0)
    print("wrote", sys.argv[2])
