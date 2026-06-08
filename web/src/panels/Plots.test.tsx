import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor, fireEvent } from "@testing-library/react";
import { MemoryRouter, Route, Routes } from "react-router-dom";

vi.mock("../api/spectrum", () => ({
  fetchSpectrumDevices: vi.fn(),
}));

vi.mock("../api/symbols", () => ({
  openSymbolStream: vi.fn(),
}));

vi.mock("../api/diag", () => ({
  openIQStream: vi.fn(),
}));

// Plots imports the chart-backed scopes (Tuning/Histogram), which register
// Chart.js components at module load — stub the chart libs.
vi.mock("react-chartjs-2", () => ({ Line: () => null, Bar: () => null }));
vi.mock("chart.js", () => {
  const noop = class {};
  return {
    Chart: { register: () => {} },
    BarElement: noop,
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

import { fetchSpectrumDevices } from "../api/spectrum";
import { openSymbolStream } from "../api/symbols";
import { openIQStream } from "../api/diag";
import { useShared } from "../store/shared";
import { Plots } from "./Plots";

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

function renderAt(path: string) {
  return render(
    <MemoryRouter initialEntries={[path]}>
      <Routes>
        <Route path="/plots" element={<Plots />} />
        <Route path="/plots/:tab" element={<Plots />} />
      </Routes>
    </MemoryRouter>,
  );
}

describe("Plots hub", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    resetStore();
    window.localStorage.clear();
    vi.mocked(fetchSpectrumDevices).mockResolvedValue([]);
    vi.mocked(openSymbolStream).mockReturnValue({ close: vi.fn() });
    vi.mocked(openIQStream).mockReturnValue({ close: vi.fn() });
  });

  it("shows all scope sub-tabs and defaults to the constellation", async () => {
    renderAt("/plots");
    const tablist = await screen.findByRole("tablist", { name: "Plots" });
    expect(tablist).toBeInTheDocument();
    for (const label of [
      "Constellation",
      "Symbol scope",
      "Eye diagram",
      "Tuning",
      "Histogram",
    ]) {
      expect(screen.getByRole("tab", { name: label })).toBeInTheDocument();
    }
    // The default sub-tab is the Constellation (its heading renders).
    expect(
      screen.getByRole("heading", { name: "Constellation" }),
    ).toBeInTheDocument();
  });

  it("renders the sub-tab named in the URL", async () => {
    renderAt("/plots/eye");
    await waitFor(() => {
      expect(
        screen.getByRole("heading", { name: "Eye diagram" }),
      ).toBeInTheDocument();
    });
  });

  it("switches sub-tab on click", async () => {
    renderAt("/plots");
    fireEvent.click(screen.getByRole("tab", { name: "Histogram" }));
    await waitFor(() => {
      expect(
        screen.getByRole("heading", { name: "Symbol histogram" }),
      ).toBeInTheDocument();
    });
  });
});
