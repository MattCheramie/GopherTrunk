import { useEffect, useState } from "react";
import { api, HTTPError } from "../api/client";
import { useShared } from "../store/shared";

// First-run screen: enter a server URL + optional bearer token. The
// SPA validates the credentials by probing /api/v1/health before
// storing them, so typos / unreachable hosts fail fast with a
// human-readable banner.
//
// Quick-link entry: the URL hash can carry `#server=...&token=...`
// so an operator can hand out a one-click bookmark.
export function ConnectScreen() {
  const setCredentials = useShared((s) => s.setCredentials);
  const setConnected = useShared((s) => s.setConnected);
  const rememberDefault = useShared((s) => s.rememberToken);

  const [url, setUrl] = useState(useShared.getState().serverURL ?? "");
  const [token, setToken] = useState(useShared.getState().token ?? "");
  const [remember, setRemember] = useState(rememberDefault);
  const [busy, setBusy] = useState(false);
  const [err, setErr] = useState<string | null>(null);

  useEffect(() => {
    // Hash bootstrap. Strip immediately so the token isn't visible
    // in the URL after the first render.
    if (!window.location.hash.includes("=")) return;
    const params = new URLSearchParams(window.location.hash.slice(1));
    const hashURL = params.get("server");
    const hashToken = params.get("token");
    if (hashURL) setUrl(hashURL);
    if (hashToken) setToken(hashToken);
    if (hashURL || hashToken) {
      // Wipe the hash so screenshots/bookmarks don't leak the token.
      history.replaceState(null, "", window.location.pathname);
    }
  }, []);

  const submit = async (e: React.FormEvent) => {
    e.preventDefault();
    setErr(null);
    const trimmed = url.trim().replace(/\/+$/, "");
    if (!/^https?:\/\//i.test(trimmed)) {
      setErr("Server URL must start with http:// or https://");
      return;
    }
    setBusy(true);
    try {
      await api.health({ baseURL: trimmed, token: token || null });
      setCredentials(trimmed, token || null, remember);
      setConnected(true);
    } catch (e) {
      if (e instanceof HTTPError) {
        setErr(`Daemon refused the request (${e.status}). ${e.body || ""}`);
      } else if (e instanceof Error) {
        setErr(`Could not reach ${trimmed}: ${e.message}`);
      } else {
        setErr("Connection failed for an unknown reason.");
      }
    } finally {
      setBusy(false);
    }
  };

  return (
    <div className="min-h-full grid place-items-center p-4 pt-safe-top pb-safe-bottom">
      <form
        onSubmit={submit}
        className="panel w-full max-w-md p-6 space-y-5"
        autoComplete="off"
      >
        <header className="space-y-1">
          <h1 className="text-2xl font-semibold tracking-tight">
            GopherTrunk
          </h1>
          <p className="text-sm text-muted">
            Point this app at your daemon's HTTP API. Same machine,
            another box on the network, or a Raspberry Pi running
            headless — all the same.
          </p>
        </header>

        <label className="block space-y-1">
          <span className="text-sm font-medium">Server URL</span>
          <input
            type="url"
            className="input w-full"
            placeholder="http://192.168.1.42:8080"
            value={url}
            onChange={(e) => setUrl(e.target.value)}
            required
            autoCapitalize="off"
            autoCorrect="off"
            spellCheck={false}
            inputMode="url"
          />
        </label>

        <label className="block space-y-1">
          <span className="text-sm font-medium">
            Bearer token{" "}
            <span className="text-muted font-normal">
              (only when api.auth.mode is "required")
            </span>
          </span>
          <input
            type="password"
            className="input w-full"
            placeholder="leave blank if auth is off"
            value={token}
            onChange={(e) => setToken(e.target.value)}
            autoCapitalize="off"
            autoCorrect="off"
            spellCheck={false}
          />
        </label>

        <label className="flex items-center gap-2 text-sm">
          <input
            type="checkbox"
            className="h-5 w-5"
            checked={remember}
            onChange={(e) => setRemember(e.target.checked)}
          />
          <span>Remember on this device</span>
        </label>

        {err && (
          <div className="rounded-md border border-err/40 bg-err/10 px-3 py-2 text-sm text-err">
            {err}
          </div>
        )}

        <button type="submit" disabled={busy} className="btn-primary w-full">
          {busy ? "Connecting…" : "Connect"}
        </button>

        <p className="text-xs text-muted">
          The token (if any) stays in your browser. The page never
          phones home; every request is direct to the daemon.
        </p>
      </form>
    </div>
  );
}
