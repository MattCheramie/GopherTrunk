import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";

vi.mock("../api/pagers", () => ({
  fetchPagerMessages: vi.fn(),
  functionLabel: (n: number) =>
    n >= 0 && n <= 3 ? String.fromCharCode(65 + n) : "?",
}));

import { fetchPagerMessages } from "../api/pagers";
import { useShared } from "../store/shared";
import { Pagers } from "./Pagers";

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

describe("Pagers panel", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    resetStore();
  });

  it("renders an empty-state when no messages are present", async () => {
    vi.mocked(fetchPagerMessages).mockResolvedValue([]);
    render(<Pagers />);
    await waitFor(() => {
      expect(screen.getByText(/No pager messages yet/)).toBeInTheDocument();
    });
  });

  it("renders rows with type, RIC, function, and body", async () => {
    vi.mocked(fetchPagerMessages).mockResolvedValue([
      {
        id: 1,
        received_at: "2026-05-26T12:34:56Z",
        protocol: "pocsag",
        ric: 1234567,
        func: 2,
        encoding: "alpha",
        body: "STRUCTURE FIRE",
        corrected: 0,
      },
      {
        id: 2,
        received_at: "2026-05-26T12:35:00Z",
        protocol: "flex",
        ric: 7654321,
        func: 1,
        encoding: "numeric",
        body: "911",
        corrected: 1,
      },
    ]);
    render(<Pagers />);
    await waitFor(() => {
      expect(screen.getByText("STRUCTURE FIRE")).toBeInTheDocument();
      expect(screen.getByText("911")).toBeInTheDocument();
      expect(screen.getByText("1234567")).toBeInTheDocument();
      expect(screen.getByText("C")).toBeInTheDocument(); // function 2 = C
      expect(screen.getByText("B")).toBeInTheDocument(); // function 1 = B
      expect(screen.getByText("POCSAG")).toBeInTheDocument();
      expect(screen.getByText("FLEX")).toBeInTheDocument();
    });
  });

  it("surfaces fetch errors", async () => {
    vi.mocked(fetchPagerMessages).mockRejectedValue(new Error("daemon down"));
    render(<Pagers />);
    await waitFor(() => {
      expect(screen.getByText(/daemon down/)).toBeInTheDocument();
    });
  });
});
