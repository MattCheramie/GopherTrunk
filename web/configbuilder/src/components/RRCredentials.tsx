import { useState } from "react";
import { TextField } from "./fields";
import { useSection } from "../sections/useSection";
import { api } from "../api/client";
import type { RadioReferenceConfig } from "../api/types";

// RRCredentials is the shared RadioReference sign-in block: just a
// username + password (the SOAP app key is built into the binary at link
// time, so operators never see or supply it) plus a "Verify subscription"
// button. It reads/writes the draft's RadioReference section directly, so
// whatever is typed here is sent on every RR browse/import call and is
// persisted to radioreference.* when the config is saved.
//
// Used both in the dedicated RadioReference Login section and inline at the
// top of the "Add from RadioReference" browse modal, so credentials can be
// entered right where they're needed.
export function RRCredentials() {
  const [v, set] = useSection("RadioReference");
  const cfg = v as RadioReferenceConfig;
  const [verify, setVerify] = useState<{ busy: boolean; ok?: boolean; msg?: string }>({
    busy: false,
  });

  // runVerify sends the edited username/password (the built-in app key is
  // applied server-side) to the server, which calls RadioReference and
  // reports whether the subscription is premium.
  const runVerify = async () => {
    setVerify({ busy: true });
    try {
      const r = await api.rrVerify({ user: cfg.Username, pass: cfg.Password });
      if (!r.ok) {
        setVerify({ busy: false, ok: false, msg: r.fault || "Could not verify the account." });
      } else if (r.premium) {
        setVerify({
          busy: false,
          ok: true,
          msg: `Premium subscription active${r.expires ? ` until ${r.expires}` : ""}${r.username ? ` (${r.username})` : ""}.`,
        });
      } else {
        setVerify({
          busy: false,
          ok: false,
          msg: `Logged in${r.username ? ` as ${r.username}` : ""}, but no active premium subscription was found.`,
        });
      }
    } catch (e) {
      setVerify({ busy: false, ok: false, msg: (e as Error).message });
    }
  };

  return (
    <div className="space-y-3">
      <TextField
        label="Username"
        value={cfg.Username}
        onChange={(x) => set({ ...cfg, Username: x })}
        placeholder="your RadioReference.com username"
      />
      <TextField
        label="Password"
        type="password"
        value={cfg.Password}
        onChange={(x) => set({ ...cfg, Password: x })}
        placeholder="your RadioReference.com password"
      />
      <p className="help">
        GopherTrunk ships with a built-in RadioReference app key — just sign in
        with your RadioReference.com username and password. A premium
        subscription is required for trunked-system browse/import.
      </p>
      <div className="flex flex-wrap items-center gap-3">
        <button className="btn" disabled={verify.busy} onClick={runVerify}>
          {verify.busy ? "Verifying…" : "Verify subscription"}
        </button>
        {verify.msg ? (
          <span className={verify.ok ? "text-sm text-ok" : "text-sm text-err"}>
            {verify.ok ? "✓ " : "✗ "}
            {verify.msg}
          </span>
        ) : null}
      </div>
    </div>
  );
}
