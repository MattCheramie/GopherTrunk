import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";

vi.mock("../api/client", () => ({
  api: { scanner: vi.fn() },
}));

vi.mock("../api/write", () => ({
  writes: {
    setScanMode: vi.fn(),
    huntHold: vi.fn(),
    huntResume: vi.fn(),
    huntRetune: vi.fn(),
    convHold: vi.fn(),
    convResume: vi.fn(),
    convDwell: vi.fn(),
    convLockout: vi.fn(),
    convUnlockout: vi.fn(),
    manualTune: vi.fn(),
  },
}));

import { api } from "../api/client";
import type { ScannerStatusDTO } from "../api/types";
import { useShared } from "../store/shared";
import { Scanner } from "./Scanner";

function resetStore() {
  useShared.setState({
    serverURL: "http://localhost:8080",
    token: null,
    writeMode: true,
    mutations: { allow_mutations: true },
    lastError: null,
    scanner: null,
  });
}

describe("Scanner panel", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    resetStore();
  });

  // Regression for the field-reported "reading 'length' of undefined" crash:
  // the daemon can serialize a scanner snapshot with null systems/conventional
  // (no hunt systems configured, conventional scanner absent). The panel's
  // `systems ?? []` / `conventional ?? { … }` guards must keep it from
  // dereferencing those nulls. Without the guards this render throws.
  it("renders without crashing when systems and conventional are null", async () => {
    vi.mocked(api.scanner).mockResolvedValue({
      scan_mode: "all",
      systems: null,
      conventional: null,
      tg_scan_count: 0,
      tg_total: 0,
    } as unknown as ScannerStatusDTO);

    render(<Scanner />);

    // The poll resolves, setScanner replaces the null snapshot, and the panel
    // re-renders through the guarded branches without throwing.
    await waitFor(() =>
      expect(screen.getByText(/talkgroups eligible/i)).toBeInTheDocument(),
    );
    expect(api.scanner).toHaveBeenCalled();
  });

  // The populated path renders the header counts so the test also locks the
  // non-null branch the guards fall through to.
  it("renders the eligible-talkgroup counts from a populated snapshot", async () => {
    vi.mocked(api.scanner).mockResolvedValue({
      scan_mode: "list",
      systems: [],
      conventional: { enabled: false, channels: [] },
      tg_scan_count: 7,
      tg_total: 42,
    });

    render(<Scanner />);

    await waitFor(() =>
      expect(screen.getByText(/7 of 42 talkgroups eligible/i)).toBeInTheDocument(),
    );
  });
});
