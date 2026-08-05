// Boundary-declick envelope for the live-audio ring-buffer worklet.
//
// The recorded/streamed PCM rarely lands on zero at a PTT edge, so starting or
// stopping playback on a hard sample step injects a broadband click — what the
// user hears as a "DC spike" at PTT-in / PTT-out and at every mid-stream call
// seam. A short amplitude ramp at those transitions removes it, exactly the way
// rdio-scanner fades each clip's head and tail.
//
// The math here is intentionally tiny and is MIRRORED inline in
// ringBufferProcessor.js — that worklet is loaded raw by addModule() and must
// not import anything, so this module exists to host the logic under test.
// Keep the two in sync; fadeEnvelope.test.ts pins the behaviour.

// envelopeTarget picks the gain the envelope should be heading toward this
// render quantum:
//   - 0 while the ring is not draining (primed-silent or post-underrun), so the
//     first real sample after a prime fades UP from zero, not a step.
//   - 0 while draining but within `releaseFrames` of empty, so the tail fades
//     DOWN to zero before the hard underrun cut instead of clicking off.
//   - 1 during healthy steady playback (available deep enough), so mid-call
//     audio is untouched (unity) — no coloration.
export function envelopeTarget(
  draining: boolean,
  available: number,
  releaseFrames: number,
): number {
  if (!draining) return 0;
  if (available <= releaseFrames) return 0;
  return 1;
}

// nextEnvelopeGain steps the current gain toward target by at most `step`
// (1/rampFrames), clamped to [0, 1]. A linear ramp is enough to kill the click
// and is cheap per-sample.
export function nextEnvelopeGain(
  gain: number,
  target: number,
  step: number,
): number {
  if (gain < target) return Math.min(target, gain + step);
  if (gain > target) return Math.max(target, gain - step);
  return gain;
}

// rampStepForFrames converts a desired ramp length in frames to a per-frame
// step. A non-positive length yields an instant (step 1) transition.
export function rampStepForFrames(rampFrames: number): number {
  return rampFrames > 0 ? 1 / rampFrames : 1;
}
