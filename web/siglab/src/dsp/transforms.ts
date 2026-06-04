// Client-side spectral transforms for the visualization panels. TensorFlow.js
// provides the numpy-like array/FFT math; it is imported dynamically so the
// large tfjs chunk only loads when a spectral view (PSD / spectrogram) opens.
// Window functions come from a SciPy-like helper (see ./stats.ts).

import type { IQPoint } from "../api/types";
import { hannWindow } from "./stats";

// fftMagnitudes runs an FFT over one windowed complex frame and returns the
// fftshifted magnitude spectrum (DC centred).
async function fftMagnitudes(
  re: Float32Array,
  im: Float32Array,
): Promise<Float32Array> {
  const tf = await import("@tensorflow/tfjs");
  return tf.tidy(() => {
    const real = tf.tensor1d(re);
    const imag = tf.tensor1d(im);
    const spec = tf.spectral.fft(tf.complex(real, imag));
    const mag = tf.abs(spec).dataSync() as Float32Array;
    return fftshift(Float32Array.from(mag));
  });
}

function fftshift(a: Float32Array): Float32Array {
  const n = a.length;
  const half = Math.floor(n / 2);
  const out = new Float32Array(n);
  out.set(a.subarray(half), 0);
  out.set(a.subarray(0, half), n - half);
  return out;
}

function toFloat(points: IQPoint[], start: number, len: number) {
  const re = new Float32Array(len);
  const im = new Float32Array(len);
  for (let i = 0; i < len; i++) {
    const p = points[start + i];
    re[i] = p.i;
    im[i] = p.q;
  }
  return { re, im };
}

export interface PSDResult {
  freqHz: number[];
  psdDb: number[];
}

// welchPSD computes an averaged (Welch) periodogram of the decimated IQ.
export async function welchPSD(
  points: IQPoint[],
  rateHz: number,
  fftSize = 1024,
  overlap = 0.5,
): Promise<PSDResult> {
  const size = Math.min(fftSize, points.length);
  if (size < 8) return { freqHz: [], psdDb: [] };
  const win = hannWindow(size);
  const hop = Math.max(1, Math.floor(size * (1 - overlap)));
  const acc = new Float64Array(size);
  let frames = 0;
  for (let start = 0; start + size <= points.length; start += hop) {
    const { re, im } = toFloat(points, start, size);
    for (let i = 0; i < size; i++) {
      re[i] *= win[i];
      im[i] *= win[i];
    }
    const mag = await fftMagnitudes(re, im);
    for (let i = 0; i < size; i++) acc[i] += mag[i] * mag[i];
    frames++;
  }
  if (frames === 0) return { freqHz: [], psdDb: [] };
  const freqHz: number[] = [];
  const psdDb: number[] = [];
  for (let i = 0; i < size; i++) {
    const f = ((i - size / 2) * rateHz) / size;
    freqHz.push(f);
    psdDb.push(10 * Math.log10(acc[i] / frames + 1e-12));
  }
  return { freqHz, psdDb };
}

export interface Spectrogram {
  z: number[][]; // [frame][freqBin] magnitude in dB
  rateHz: number;
  fftSize: number;
  frames: number;
}

// spectrogram computes a short-time Fourier transform of the decimated IQ.
export async function spectrogram(
  points: IQPoint[],
  rateHz: number,
  fftSize = 256,
  hop = 128,
): Promise<Spectrogram> {
  const size = Math.min(fftSize, points.length);
  if (size < 8) return { z: [], rateHz, fftSize: size, frames: 0 };
  const win = hannWindow(size);
  const z: number[][] = [];
  // Cap the number of frames so a long capture doesn't produce a giant
  // heatmap; the panel re-bins time on the canvas anyway.
  const maxFrames = 600;
  const totalFrames = Math.floor((points.length - size) / hop) + 1;
  const stride = Math.max(1, Math.ceil(totalFrames / maxFrames));
  for (let f = 0; f + size <= points.length; f += hop * stride) {
    const { re, im } = toFloat(points, f, size);
    for (let i = 0; i < size; i++) {
      re[i] *= win[i];
      im[i] *= win[i];
    }
    const mag = await fftMagnitudes(re, im);
    const row = new Array(size);
    for (let i = 0; i < size; i++) row[i] = 20 * Math.log10(mag[i] + 1e-9);
    z.push(row);
  }
  return { z, rateHz, fftSize: size, frames: z.length };
}
