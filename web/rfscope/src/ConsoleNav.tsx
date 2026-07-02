import { useEffect, useState } from "react";

/** The GopherTrunk web consoles that cross-link to one another. `self` is the
 *  current console and is omitted from its own switcher. */
export type Console = "config" | "siglab" | "rfscope" | "cryptolab";

type Advertised = { cryptolab: boolean; siglab: boolean; rfscope: boolean };

// ConsoleNav renders links to the sibling GopherTrunk consoles the daemon
// advertises as reachable. It probes GET /api/v1/runtime once; the probe fails
// closed under a standalone `… serve` (which wires no RuntimeProvider, so the
// route 503s), meaning the links only surface when this SPA is daemon-mounted
// alongside its siblings. The daemon always mounts the operator console at /
// and the Config Builder at /config/; Signal Lab, RF Scope, and Crypto Lab are
// gated on their respective runtime.*_console flags. Kept in sync across the
// sub-apps by convention, mirroring the duplicated --gt-* token blocks.
export function ConsoleNav({ self }: { self: Console }) {
  const [rt, setRt] = useState<Advertised | null>(null);
  useEffect(() => {
    let alive = true;
    fetch("/api/v1/runtime")
      .then((r) => (r.ok ? r.json() : null))
      .then((j) => {
        if (alive && j && typeof j === "object")
          setRt({
            cryptolab: j.cryptolab_console === true,
            siglab: j.siglab_console === true,
            rfscope: j.rfscope_console === true,
          });
      })
      .catch(() => {
        /* no daemon / standalone serve — leave the switcher hidden */
      });
    return () => {
      alive = false;
    };
  }, []);
  if (!rt) return null;
  const cls = "rounded-md px-2 py-1 hover:text-fg";
  return (
    <nav className="flex flex-wrap items-center gap-1 text-xs text-muted">
      <a href="/" title="GopherTrunk console" className={cls}>
        ◀ GopherTrunk
      </a>
      {self !== "config" ? (
        <a href="/config/" title="Config Builder" className={cls}>
          🛠 Config
        </a>
      ) : null}
      {rt.siglab && self !== "siglab" ? (
        <a href="/siglab/" title="Signal Lab" className={cls}>
          ◷ Signal Lab
        </a>
      ) : null}
      {rt.rfscope && self !== "rfscope" ? (
        <a href="/rfscope/" title="RF Scope" className={cls}>
          ⧉ RF Scope
        </a>
      ) : null}
      {rt.cryptolab && self !== "cryptolab" ? (
        <a href="/cryptolab/" title="Crypto Lab" className={cls}>
          🔐 Crypto Lab
        </a>
      ) : null}
    </nav>
  );
}
