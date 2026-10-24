// Volume range of the live-audio player, shared by the player, the stored
// preference and the slider.

// MAX_VOLUME is the top of the live player's volume range: 4 = 400 %, +12 dB
// over the level the daemon sends (#1261: "the dashboard audio is quiet
// compared with other pages"). Above 1 the gain can push loud calls past full
// scale, so the player runs a peak limiter after its gain stage (limiterFor in
// streamPlayer.ts). The limiter acts only above -1 dBFS, which the daemon's
// int16 PCM can reach at volume <= 1 only in its loudest 1 dB.
export const MAX_VOLUME = 4;

// clampVolume bounds a volume to [0, MAX_VOLUME]; NaN reads as 0.
export function clampVolume(v: number): number {
  if (!Number.isFinite(v)) return v === Infinity ? MAX_VOLUME : 0;
  return Math.max(0, Math.min(MAX_VOLUME, v));
}
