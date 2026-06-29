import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, waitFor } from "@testing-library/react";

vi.mock("../api/client", () => ({
  api: { activeCalls: vi.fn() },
}));

import { api } from "../api/client";
import { useShared } from "../store/shared";
import { useActiveCallsPoll } from "./useActiveCallsPoll";
import type { ActiveCallDTO } from "../api/types";

function Probe() {
  useActiveCallsPoll();
  return null;
}

function call(serial: string, freqHz: number): ActiveCallDTO {
  return {
    grant: { group_id: 1, system: "Sys", protocol: "p25", frequency_hz: freqHz },
    device_serial: serial,
    started_at: "2026-01-01T00:00:00Z",
  } as ActiveCallDTO;
}

describe("useActiveCallsPoll", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    useShared.setState({
      serverURL: "http://localhost:8080",
      token: null,
      activeCalls: [],
    });
  });

  // Regression for the frozen-spectrum-panel bug: a panel that follows
  // live calls must refresh activeCalls itself, since the Active/Dashboard
  // pollers only tick on their own routes. Without the poll the store
  // keeps a stale snapshot and the view freezes on a single freq.
  it("refreshes the shared-store activeCalls from the API on mount", async () => {
    vi.mocked(api.activeCalls).mockResolvedValue([call("rtl-1", 851_000_000)]);

    render(<Probe />);

    await waitFor(() => {
      expect(useShared.getState().activeCalls).toHaveLength(1);
    });
    expect(api.activeCalls).toHaveBeenCalled();
  });

  it("clears a stale active call once the daemon reports none", async () => {
    // Store starts holding a call that has since ended on the daemon.
    useShared.setState({ activeCalls: [call("rtl-1", 851_000_000)] });
    vi.mocked(api.activeCalls).mockResolvedValue([]);

    render(<Probe />);

    await waitFor(() => {
      expect(useShared.getState().activeCalls).toEqual([]);
    });
  });
});
