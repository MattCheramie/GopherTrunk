// Follow-selection policy for the live-audio mini-player.
//
// The WebUI plays one call at a time (standard scanner behaviour) and the
// daemon's /audio/stream is filtered to a single device serial. The question
// this module answers is *which* call to follow as the active-call set churns.
//
// The naive "always follow the newest started_at" rule tears the in-progress
// call off the instant any newer grant appears — and because the live stream
// has no backfill, the abandoned call's tail is simply lost ("eating
// conversations"). Instead we HOLD the current call: keep following its device
// until that device drops out of the active set (the call ended), then advance
// to whatever is live. Each call plays to completion, then we move on.

interface FollowCall {
  device_serial: string;
  started_at: string;
}

// pickFollowSerial returns the device serial the player should be following.
//   - null when nothing is up (stream falls silent rather than dropping to the
//     overlap-prone unfiltered stream).
//   - the current serial when its call is still active (hold to completion).
//   - otherwise the newest active call's serial (advance once the held call
//     ends, or on the very first call when nothing is held yet).
// Calls without a device serial are ignored — an announced-but-unfollowed call
// has no tuner and thus no audio to stream.
export function pickFollowSerial(
  activeCalls: readonly FollowCall[],
  current: string | null,
): string | null {
  const withSerial = activeCalls.filter((c) => c.device_serial);
  if (withSerial.length === 0) return null;
  // Hold the current call while its device is still transmitting.
  if (current && withSerial.some((c) => c.device_serial === current)) {
    return current;
  }
  // Otherwise advance to the most recently started active call.
  const newest = withSerial.reduce((a, b) =>
    b.started_at > a.started_at ? b : a,
  );
  return newest.device_serial;
}
