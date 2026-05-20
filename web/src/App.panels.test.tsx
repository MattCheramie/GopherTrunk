import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";

// Regression coverage for issue #290. The existing App.test.tsx mocks
// `Routes` to null, so no panel is ever mounted by it. This file mounts
// every routed panel for real, with the daemon connected, and asserts
// none of them loops the renderer into React #185 (which the
// ErrorBoundary would surface as the "Something went wrong" fallback).

vi.mock("./api/events", () => ({
  openEventStream: vi.fn(
    (_cfg: unknown, opts: { onStatus?: (s: string) => void }) => {
      opts.onStatus?.("connecting");
      return { close: vi.fn() };
    },
  ),
}));

vi.mock("./api/client", () => {
  // Defined inside the factory: vi.mock is hoisted above module scope.
  const ok = (value: unknown) => vi.fn().mockResolvedValue(value);
  return {
    api: {
      health: ok({ status: "ok", pool_attached_count: 0, active_calls: 0 }),
      version: ok({ version: "test" }),
      mutations: ok({ allow_mutations: false }),
      runtime: ok({}),
      systems: ok([]),
      talkgroups: ok([]),
      activeCalls: ok([]),
      history: ok([]),
      devices: ok([]),
      scanner: ok({
        scan_mode: "idle",
        systems: [],
        conventional: { enabled: false, channels: [] },
        tg_scan_count: 0,
        tg_total: 0,
      }),
      audio: ok({
        backend_enabled: false,
        sample_rate: 0,
        muted: false,
        recording_enabled: false,
      }),
      metricsText: ok(""),
    },
    HTTPError: class HTTPError extends Error {
      status = 0;
      body = "";
    },
    request: vi.fn(),
    audioStreamURL: () => "http://test/stream",
  };
});

// Metrics renders a Chart.js line chart; stub the chart libs so the
// panel mounts under jsdom without a real <canvas> backend.
vi.mock("react-chartjs-2", () => ({ Line: () => null }));
vi.mock("chart.js", () => {
  const noop = class {};
  return {
    Chart: { register: () => {} },
    CategoryScale: noop,
    Filler: noop,
    Legend: noop,
    LinearScale: noop,
    LineElement: noop,
    PointElement: noop,
    Title: noop,
    Tooltip: noop,
  };
});

import { openEventStream } from "./api/events";
import { useShared } from "./store/shared";
import { App } from "./App";
import { ErrorBoundary } from "./components/ErrorBoundary";

const ROUTES = [
  "/dashboard",
  "/active",
  "/scanner",
  "/systems",
  "/talkgroups",
  "/history",
  "/events",
  "/tones",
  "/metrics",
  "/devices",
  "/settings",
  "/import",
];

describe("App panel mounting (issue #290 regression)", () => {
  beforeEach(() => {
    vi.mocked(openEventStream).mockClear();
    useShared.setState({
      serverURL: "http://localhost:8080",
      token: null,
      connected: true,
      wsStatus: "idle",
      mutations: null,
      lastError: null,
      events: [],
      activeCalls: [],
      devices: [],
      systems: [],
      talkgroups: [],
      health: null,
      audio: null,
      scanner: null,
    });
  });

  it.each(ROUTES)(
    "mounts %s connected without tripping the error boundary",
    async (route) => {
      render(
        <ErrorBoundary>
          <MemoryRouter initialEntries={[route]}>
            <App />
          </MemoryRouter>
        </ErrorBoundary>,
      );

      // Let the panels' initial polling promises settle — a render loop
      // trips React #185 and surfaces the ErrorBoundary fallback.
      await waitFor(() => expect(openEventStream).toHaveBeenCalled());
      await new Promise((resolve) => setTimeout(resolve, 20));

      expect(screen.queryByText(/Something went wrong/i)).toBeNull();
      // The WebSocket effect must still fire exactly once per connect.
      expect(openEventStream).toHaveBeenCalledTimes(1);
    },
  );
});
