import { describe, it, expect, vi, beforeEach } from "vitest";
import { render as rtlRender, screen, waitFor } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";

// Panels render react-router links / read search params, so every
// render needs a Router context.
function render(ui: Parameters<typeof rtlRender>[0]) {
  return rtlRender(<MemoryRouter>{ui}</MemoryRouter>);
}

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

  // The signal-quality chip renders from the per-system decode-health fields, with
  // the carrier offset in its tooltip. Cross-protocol — here a locked TETRA system.
  it("shows a signal-quality chip when decode health is present", async () => {
    vi.mocked(api.scanner).mockResolvedValue({
      scan_mode: "all",
      systems: [
        {
          name: "Tetra1",
          protocol: "tetra",
          state: "locked",
          locked_freq_hz: 467_912_500,
          decode_quality: "marginal",
          carrier_offset_hz: -958,
          has_decode_health: true,
        },
      ],
      conventional: { enabled: false, channels: [] },
      tg_scan_count: 0,
      tg_total: 0,
    } as unknown as ScannerStatusDTO);

    render(<Scanner />);

    const chip = await screen.findByText(/decode: marginal/i);
    expect(chip).toBeInTheDocument();
    expect(chip.closest("[title]")?.getAttribute("title")).toContain("offset -958 Hz");
  });

  // No chip when the daemon reports no decode health (unlocked / too few frames).
  it("omits the signal chip when decode health is absent", async () => {
    vi.mocked(api.scanner).mockResolvedValue({
      scan_mode: "all",
      systems: [{ name: "Tetra1", protocol: "tetra", state: "hunting" }],
      conventional: { enabled: false, channels: [] },
      tg_scan_count: 0,
      tg_total: 0,
    } as unknown as ScannerStatusDTO);

    render(<Scanner />);

    await waitFor(() => expect(screen.getByText("Tetra1")).toBeInTheDocument());
    expect(screen.queryByText(/decode:/i)).not.toBeInTheDocument();
  });
});
