import { useEffect, useState } from "react";

interface Props {
  stale: boolean;
  lastUpdated: number | null;
}

// StaleIndicator turns the silent "keep the old snapshot on a failed
// poll" behavior into something the operator can see: a small chip that
// distinguishes "the data on screen is current" from "the daemon went
// quiet and you're looking at the last good snapshot from N s ago".
export function StaleIndicator({ stale, lastUpdated }: Props) {
  // Re-render once a second while stale so the age counts up.
  const [, force] = useState(0);
  useEffect(() => {
    if (!stale) return;
    const t = window.setInterval(() => force((n) => n + 1), 1_000);
    return () => window.clearInterval(t);
  }, [stale]);

  if (!stale) return null;

  const age =
    lastUpdated != null
      ? `${Math.max(0, Math.round((Date.now() - lastUpdated) / 1000))}s ago`
      : "unknown";

  return (
    <span className="pill-warn" title={`Last good update ${age}`}>
      reconnecting · {age}
    </span>
  );
}
