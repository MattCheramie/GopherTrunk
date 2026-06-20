import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

vi.mock("../api/client", () => ({
  api: { hunt: vi.fn() },
}));

vi.mock("../api/write", () => ({
  writes: {
    huntStart: vi.fn(),
    huntStop: vi.fn(),
  },
}));

import { api } from "../api/client";
import { writes } from "../api/write";
import { useShared } from "../store/shared";
import { Hunt } from "./Hunt";

function resetStore() {
  useShared.setState({
    serverURL: "http://localhost:8080",
    token: null,
    writeMode: true,
    mutations: { allow_mutations: true },
    lastError: null,
  });
}

describe("Hunt panel", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    resetStore();
    vi.mocked(api.hunt).mockResolvedValue({
      run_id: 0,
      state: "idle",
      running: false,
      sites: 0,
      talkgroups: 0,
    });
    vi.mocked(writes.huntStart).mockResolvedValue({ run_id: 1 });
  });

  it("forwards the serial and protocol inputs on start", async () => {
    render(<Hunt />);
    // Wait for the initial status poll to settle.
    await waitFor(() => expect(api.hunt).toHaveBeenCalled());

    // Blind sweep is collapsed by default — open it first.
    await userEvent.click(screen.getByRole("button", { name: /blind sweep/i }));
    await userEvent.type(screen.getByPlaceholderText("00000001"), "dongle-7");
    await userEvent.type(screen.getByPlaceholderText("p25"), "p25");
    await userEvent.click(screen.getByRole("button", { name: /start hunt/i }));

    await waitFor(() => expect(writes.huntStart).toHaveBeenCalled());
    const body = vi.mocked(writes.huntStart).mock.calls[0][1];
    expect(body.serial).toBe("dongle-7");
    expect(body.protocol).toBe("p25");
    expect(body.bands).toEqual(["851:869"]); // default band, comma-split
  });

  it("parses a control channel with the chosen protocol and dwell", async () => {
    render(<Hunt />);
    await waitFor(() => expect(api.hunt).toHaveBeenCalled());

    // The only <select> in the panel is the Parse-CC protocol picker.
    await userEvent.selectOptions(screen.getByRole("combobox"), "dmr");
    await userEvent.type(screen.getByPlaceholderText("851.0125"), "853.5125");
    const dwell = screen.getByLabelText(/listen for/i);
    await userEvent.clear(dwell);
    await userEvent.type(dwell, "45");
    await userEvent.click(screen.getByRole("button", { name: /parse control channel/i }));

    await waitFor(() => expect(writes.huntStart).toHaveBeenCalled());
    const body = vi.mocked(writes.huntStart).mock.calls[0][1];
    expect(body.candidates).toEqual([853.5125]);
    expect(body.no_sweep).toBe(true);
    expect(body.protocol).toBe("dmr");
    expect(body.dwell_seconds).toBe(45);
  });

  it("renders per-capture reports and discovered sites from the status", async () => {
    vi.mocked(api.hunt).mockResolvedValue({
      run_id: 1,
      state: "done",
      running: false,
      sites: 1,
      talkgroups: 3,
      system_name: "County P25",
      system: {
        name: "County P25",
        sites: [
          {
            rfss: 1,
            site_id: 5,
            site_name: "Downtown",
            control_channels: [{ frequency_hz: 851_012_500, is_control: true }],
          },
        ],
      },
      reports: [
        {
          protocol: "p25",
          control_hz: 851_012_500,
          locked: true,
          confidence: 0.92,
          talkgroups: 3,
        },
        { protocol: "dmr", skipped: true, skip_reason: "no control" },
      ],
    });

    render(<Hunt />);
    await waitFor(() => expect(api.hunt).toHaveBeenCalled());

    // Captures table.
    await waitFor(() => expect(screen.getByText(/Captures \(2\)/)).toBeInTheDocument());
    expect(screen.getByText(/locked · conf 92%/)).toBeInTheDocument();
    expect(screen.getByText(/skipped — no control/)).toBeInTheDocument();

    // Discovered sites table.
    expect(screen.getByText(/Sites \(1\)/)).toBeInTheDocument();
    expect(screen.getByText("Downtown")).toBeInTheDocument();
    expect(screen.getByText(/851\.0125\*/)).toBeInTheDocument();
  });
});
