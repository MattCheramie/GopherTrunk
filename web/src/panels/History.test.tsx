import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { act, render, screen, waitFor } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";

vi.mock("../api/client", () => ({
  api: { history: vi.fn() },
}));

vi.mock("../api/write", () => ({
  writes: { sweepRetention: vi.fn() },
}));

import { api } from "../api/client";
import { useShared } from "../store/shared";
import type { CallRow } from "../api/types";
import { History } from "./History";

function resetStore() {
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
}

function renderPanel(entry = "/history") {
  return render(
    <MemoryRouter initialEntries={[entry]}>
      <History />
    </MemoryRouter>,
  );
}

const row: CallRow = {
  id: 1,
  system: "250_013",
  protocol: "tetra",
  group_id: 1020545,
  source_id: 1005736,
  frequency_hz: 467912500,
  device_serial: "soapy-0",
  started_at: "2026-08-16T21:07:47Z",
};

describe("History panel", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    resetStore();
  });
  afterEach(() => {
    vi.useRealTimers();
  });

  // Regression: the client used to read the wrong envelope key, so every
  // History query rendered "No calls in the daemon's call log for this
  // filter" no matter what the daemon returned.
  it("renders the rows the daemon returned", async () => {
    vi.mocked(api.history).mockResolvedValue([row]);
    renderPanel();
    await waitFor(() => {
      expect(screen.getByText("1020545")).toBeInTheDocument();
    });
    expect(screen.queryByText(/No calls in the daemon's call log/)).toBeNull();
  });

  it("seeds the filter from ?system= and ?group_id=", async () => {
    vi.mocked(api.history).mockResolvedValue([]);
    renderPanel("/history?system=250_013&group_id=1020545");
    await waitFor(() => {
      expect(api.history).toHaveBeenCalled();
    });
    expect(vi.mocked(api.history).mock.calls[0][1]).toMatchObject({
      system: "250_013",
      group_id: 1020545,
    });
  });

  it("seeds the filter from ?source_id= (the per-radio view)", async () => {
    vi.mocked(api.history).mockResolvedValue([]);
    renderPanel("/history?source_id=1005492");
    await waitFor(() => {
      expect(api.history).toHaveBeenCalled();
    });
    expect(vi.mocked(api.history).mock.calls[0][1]).toMatchObject({
      source_id: 1005492,
    });
  });

  // Regression: the Started column was a raw slice of the daemon's UTC
  // string, so an operator in UTC+3 saw History three hours behind the
  // activity feed and read it as "history stops updating".
  it("renders Started in the browser's local time, not UTC", async () => {
    vi.stubEnv("TZ", "Europe/Moscow"); // UTC+3
    try {
      vi.mocked(api.history).mockResolvedValue([
        { ...row, started_at: "2026-09-10T00:15:18Z" },
      ]);
      renderPanel();
      await waitFor(() => {
        expect(screen.getByText("2026-09-10 03:15:18")).toBeInTheDocument();
      });
      expect(screen.queryByText("2026-09-10 00:15:18")).toBeNull();
    } finally {
      vi.unstubAllEnvs();
    }
  });

  // Regression: History fetched once per filter change and never again, so
  // new calls appeared only on a page reload. A call.end in the live event
  // feed now triggers a (debounced) refetch.
  it("refetches after a call.end arrives in the event feed", async () => {
    vi.useFakeTimers({ shouldAdvanceTime: true });
    try {
      vi.mocked(api.history).mockResolvedValue([row]);
      renderPanel();
      await waitFor(() => {
        expect(api.history).toHaveBeenCalledTimes(1);
      });
      act(() => {
        useShared.getState().appendEvents([
          {
            kind: "call.end",
            timestamp: "2026-09-10T00:15:49Z",
            payload: { group_id: 1020545 },
          },
        ]);
      });
      // Not immediately — the refetch waits for the recorder's call.complete.
      expect(api.history).toHaveBeenCalledTimes(1);
      await act(async () => {
        await vi.advanceTimersByTimeAsync(2_000);
      });
      await waitFor(() => {
        expect(api.history).toHaveBeenCalledTimes(2);
      });
    } finally {
      vi.useRealTimers();
    }
  });

  it("shows the empty state when the filter matches nothing", async () => {
    vi.mocked(api.history).mockResolvedValue([]);
    renderPanel();
    await waitFor(() => {
      expect(
        screen.getByText(/No calls in the daemon's call log/),
      ).toBeInTheDocument();
    });
  });
});
