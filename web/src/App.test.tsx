import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, waitFor } from "@testing-library/react";
import { HashRouter } from "react-router-dom";

// The real openEventStream synchronously reports "connecting", which
// writes to the store — the feedback edge that, combined with an
// unstable `cfg` reference, drove the issue #290 render loop. The mock
// reproduces that edge so this test genuinely exercises the loop.
vi.mock("./api/events", () => ({
  openEventStream: vi.fn(
    (_cfg: unknown, opts: { onStatus?: (s: string) => void }) => {
      opts.onStatus?.("connecting");
      return { close: vi.fn() };
    },
  ),
}));

vi.mock("./api/client", () => ({
  api: {
    mutations: vi.fn().mockResolvedValue({ allow_mutations: false }),
    health: vi.fn().mockResolvedValue({ status: "ok" }),
  },
  HTTPError: class HTTPError extends Error {
    status: number;
    body: string;
    constructor(status: number, body: string, message: string) {
      super(message);
      this.status = status;
      this.body = body;
    }
  },
}));

// Stub Routes so the routed panels (and their charts) stay unmounted —
// this test is only about App's connection effects.
vi.mock("react-router-dom", async (importOriginal) => {
  const actual = await importOriginal<typeof import("react-router-dom")>();
  return { ...actual, Routes: () => null };
});

vi.mock("./components/TabBar", () => ({ TabBar: () => null }));
vi.mock("./components/AudioPlayer", () => ({ AudioPlayer: () => null }));
vi.mock("./components/InstallPrompt", () => ({ InstallPrompt: () => null }));

import { openEventStream } from "./api/events";
import { useShared } from "./store/shared";
import { App } from "./App";

describe("App connection bootstrap", () => {
  beforeEach(() => {
    vi.mocked(openEventStream).mockClear();
    useShared.setState({
      serverURL: "http://localhost:8080",
      token: null,
      connected: true,
      wsStatus: "idle",
      mutations: null,
      lastError: null,
    });
  });

  it("opens the event stream exactly once — no render loop (issue #290)", async () => {
    render(
      <HashRouter>
        <App />
      </HashRouter>,
    );

    await waitFor(() => expect(openEventStream).toHaveBeenCalled());
    // A regressed `cfg` reference would re-fire the effect on every
    // store write; give that loop a window to manifest.
    await new Promise((resolve) => setTimeout(resolve, 50));
    expect(openEventStream).toHaveBeenCalledTimes(1);
  });
});
