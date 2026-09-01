#!/usr/bin/env python3
"""Procedural license-free ambient bed (pipeline doc §7).

Slow two-chord pad (root+fifth+octave, sine/triangle partials, slow LFOs) with
a faint lowpassed noise floor. Loop-safe: length is a whole number of chord
cycles and all LFOs complete integer periods. Output: 48 kHz stereo WAV at
about -38 dBFS RMS (>=20 dB under VO).

Usage: music.py <out.wav> [seconds=96]
"""
import sys, wave
import numpy as np

SR = 48000


def tri(ph):
    return 2.0 * np.abs(2.0 * (ph % 1.0) - 1.0) - 1.0


def pad(dur):
    t = np.arange(int(dur * SR)) / SR
    n = len(t)
    cycle = 32.0  # chord change period
    # chords: C major (C3 G3 C4 E4) and A minor (A2 E3 A3 C4)
    chords = [[130.81, 196.00, 261.63, 329.63], [110.00, 164.81, 220.00, 261.63]]
    xf = 0.5 * (1 - np.cos(2 * np.pi * t / cycle))  # 0 -> chord A, 1 -> chord B
    out_l = np.zeros(n)
    out_r = np.zeros(n)
    for ci, chord in enumerate(chords):
        w = (1 - xf) if ci == 0 else xf
        for vi, f in enumerate(chord):
            lfo = 0.6 + 0.4 * np.sin(2 * np.pi * t * (vi + 1) / dur * 8 + vi * 1.7)
            det = 1.0 + 0.0004 * (vi - 1.5)
            amp = w * lfo * (0.5 if vi < 2 else 0.3)
            out_l += amp * (0.7 * np.sin(2 * np.pi * f * t) + 0.3 * tri(f * t))
            out_r += amp * (0.7 * np.sin(2 * np.pi * f * det * t + 0.5) + 0.3 * tri(f * det * t + 0.25))
    # faint lowpassed noise (one-pole) for air
    rng = np.random.default_rng(7)
    noise = rng.standard_normal(n)
    lp = np.zeros(n)
    a = 0.015
    acc = 0.0
    for i in range(n):
        acc += a * (noise[i] - acc)
        lp[i] = acc
    swell = 0.4 + 0.6 * 0.5 * (1 - np.cos(2 * np.pi * t / cycle))
    out_l += 0.35 * lp * swell
    out_r += 0.35 * np.roll(lp, 480) * swell
    # loop-safe edges
    edge = int(0.05 * SR)
    env = np.ones(n)
    env[:edge] = np.linspace(0, 1, edge)
    env[-edge:] = np.linspace(1, 0, edge)
    out_l *= env; out_r *= env
    st = np.stack([out_l, out_r], axis=1)
    rms = np.sqrt(np.mean(st ** 2))
    target = 10 ** (-38 / 20)
    st *= target / rms
    return np.clip(st * 32767, -32767, 32767).astype(np.int16)


def main():
    out = sys.argv[1]
    dur = float(sys.argv[2]) if len(sys.argv) > 2 else 96.0
    data = pad(dur)
    with wave.open(out, "wb") as wf:
        wf.setnchannels(2); wf.setsampwidth(2); wf.setframerate(SR)
        wf.writeframes(data.tobytes())
    print(f"bed: {out} {dur:.0f}s rms=-38dBFS")


if __name__ == "__main__":
    main()
