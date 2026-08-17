/**
 * A WebSocket that reconnects with exponential backoff, and gives up when the
 * endpoint says the thing it is asking for does not exist.
 *
 * Four diagnostic clients (symbols, spectrum, diag/iq, mixer) each carried a
 * byte-for-byte copy of this logic, with the same two defects.
 *
 * 1. `backoff` was reset inside `onopen`. That looks right, but the daemon used
 *    to complete the WebSocket handshake BEFORE resolving the requested SDR and
 *    then close with 1011 for an unknown one — so `onopen` fired on every
 *    attempt, the backoff reset to 500 ms every time, and `MAX_BACKOFF` was
 *    dead code. An operator who left a browser tab open across a daemon restart
 *    that changed SDRs got a reconnect two to four times a second, forever,
 *    across six mounted panels. The server now answers 404 for that case, and
 *    this helper additionally treats "opened but never delivered a frame" as a
 *    failed attempt, so the backoff grows even against a server that behaves
 *    the old way.
 * 2. `onDown` was assigned to BOTH `onerror` and `onclose` with no guard. A
 *    failure that fires both scheduled two reconnect timers while only the last
 *    handle was remembered, so timers leaked and the rate compounded.
 *
 * The give-up path matters as much as the backoff. When a socket keeps failing
 * without ever delivering data, `onGone` fires so the caller can re-enumerate
 * devices and pick a live one instead of retrying a serial that is never coming
 * back.
 */

export type SocketStatus = "connecting" | "open" | "closed" | "gone";

export const INITIAL_BACKOFF = 500;
export const MAX_BACKOFF = 30_000;

/**
 * How long a socket must stay open before the attempt counts as a success. A
 * handshake that closes immediately is a failure however clean its status code:
 * the point is whether the stream actually works, not whether TCP connected.
 */
export const OPEN_GRACE_MS = 2_000;

/**
 * Consecutive failed attempts before the stream is declared gone. Enough to
 * ride out a daemon restart, few enough that a stale device is noticed quickly.
 */
export const GONE_AFTER_FAILURES = 6;

export interface ReconnectingSocketOptions {
  /** Builds the URL for each attempt, so a caller can vary query params. */
  url: () => string;
  /** Called for every text frame received. */
  onMessage: (data: string) => void;
  /** Status transitions, including the terminal "gone". */
  onStatus?: (s: SocketStatus) => void;
  /**
   * Called once when the stream is declared gone. The caller should stop
   * relying on this serial — typically by re-enumerating devices.
   */
  onGone?: () => void;
}

export interface ReconnectingSocket {
  close(): void;
}

export function openReconnectingSocket(
  opts: ReconnectingSocketOptions,
): ReconnectingSocket {
  let closed = false;
  let ws: WebSocket | null = null;
  let backoff = INITIAL_BACKOFF;
  let failures = 0;
  let reconnectTimer: number | undefined;

  const setStatus = (s: SocketStatus) => {
    if (!closed) opts.onStatus?.(s);
  };

  const jittered = (base: number) => base / 2 + Math.random() * (base / 2);

  const connect = () => {
    if (closed) return;
    setStatus("connecting");

    let openedAt = 0;
    let gotMessage = false;
    // One attempt may fire onerror AND onclose; only the first counts.
    let settled = false;

    const socket = new WebSocket(opts.url());
    ws = socket;

    socket.onopen = () => {
      openedAt = Date.now();
      setStatus("open");
      // NOTE: backoff is deliberately NOT reset here. A handshake proves
      // nothing on its own — see the header comment.
    };

    socket.onmessage = (ev) => {
      if (closed) return;
      if (!gotMessage) {
        // First real frame: the stream works. Now the backoff may reset.
        gotMessage = true;
        backoff = INITIAL_BACKOFF;
        failures = 0;
      }
      opts.onMessage(ev.data as string);
    };

    const onDown = () => {
      if (settled) return;
      settled = true;
      socket.onopen = null;
      socket.onmessage = null;
      socket.onerror = null;
      socket.onclose = null;
      if (ws === socket) ws = null;
      if (closed) return;

      // An attempt counts as successful only if it delivered a frame, or stayed
      // open long enough that a silent-but-healthy stream is not punished.
      const heldOpen = openedAt !== 0 && Date.now() - openedAt >= OPEN_GRACE_MS;
      if (gotMessage || heldOpen) {
        backoff = INITIAL_BACKOFF;
        failures = 0;
      } else {
        failures += 1;
      }

      if (failures >= GONE_AFTER_FAILURES) {
        setStatus("gone");
        opts.onGone?.();
        closed = true;
        return;
      }

      setStatus("closed");
      const wait = jittered(backoff);
      backoff = Math.min(backoff * 2, MAX_BACKOFF);
      if (reconnectTimer !== undefined) window.clearTimeout(reconnectTimer);
      reconnectTimer = window.setTimeout(connect, wait);
    };

    socket.onerror = onDown;
    socket.onclose = onDown;
  };

  connect();

  return {
    close() {
      closed = true;
      setStatus("closed");
      if (reconnectTimer !== undefined) {
        window.clearTimeout(reconnectTimer);
        reconnectTimer = undefined;
      }
      if (ws) {
        try {
          ws.close();
        } catch {
          // ignore
        }
      }
    },
  };
}
