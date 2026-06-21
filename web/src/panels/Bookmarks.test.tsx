import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

vi.mock("../api/bookmarks", () => ({
  bookmarks: {
    list: vi.fn(),
    create: vi.fn(),
    update: vi.fn(),
    remove: vi.fn(),
  },
}));

import { bookmarks } from "../api/bookmarks";
import { useShared } from "../store/shared";
import { Bookmarks } from "./Bookmarks";

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

describe("Bookmarks panel", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    resetStore();
  });

  it("renders empty-state when no bookmarks exist", async () => {
    vi.mocked(bookmarks.list).mockResolvedValue([]);
    render(<Bookmarks />);
    await waitFor(() => {
      expect(screen.getByText(/No bookmarks yet/)).toBeInTheDocument();
    });
  });

  it("lists bookmarks grouped by group", async () => {
    vi.mocked(bookmarks.list).mockResolvedValue([
      {
        id: 1,
        name: "Ch 16",
        freq_hz: 156_800_000,
        mode: "FM",
        group: "marine",
        created_at: "2026-01-01T00:00:00Z",
        updated_at: "2026-01-01T00:00:00Z",
      },
      {
        id: 2,
        name: "NOAA WX1",
        freq_hz: 162_550_000,
        mode: "FM",
        group: "weather",
        created_at: "2026-01-01T00:00:00Z",
        updated_at: "2026-01-01T00:00:00Z",
      },
    ]);
    render(<Bookmarks />);
    await waitFor(() => {
      expect(screen.getByText("Ch 16")).toBeInTheDocument();
      expect(screen.getByText("NOAA WX1")).toBeInTheDocument();
      expect(screen.getByText("marine")).toBeInTheDocument();
      expect(screen.getByText("weather")).toBeInTheDocument();
    });
  });

  it("submits a new bookmark through the create endpoint", async () => {
    vi.mocked(bookmarks.list).mockResolvedValue([]);
    vi.mocked(bookmarks.create).mockResolvedValue({
      id: 99,
      name: "Test",
      freq_hz: 156_800_000,
      mode: "FM",
      created_at: "2026-01-01T00:00:00Z",
      updated_at: "2026-01-01T00:00:00Z",
    });
    render(<Bookmarks />);

    // Wait for initial empty-state render.
    await waitFor(() => screen.getByText(/No bookmarks yet/));

    const nameInput = screen.getByPlaceholderText(/Marine Ch 16/);
    const freqInput = screen.getByPlaceholderText(/156800000/);

    await userEvent.type(nameInput, "Test");
    await userEvent.type(freqInput, "156800000");
    await userEvent.click(screen.getByRole("button", { name: /Add bookmark/ }));

    await waitFor(() => {
      expect(bookmarks.create).toHaveBeenCalledTimes(1);
    });
    const call = vi.mocked(bookmarks.create).mock.calls[0]?.[1];
    expect(call?.name).toBe("Test");
    expect(call?.freq_hz).toBe(156_800_000);
  });

  it("shows the create-side error from the daemon", async () => {
    vi.mocked(bookmarks.list).mockResolvedValue([]);
    vi.mocked(bookmarks.create).mockRejectedValue(new Error("name required"));
    render(<Bookmarks />);

    await waitFor(() => screen.getByText(/No bookmarks yet/));

    await userEvent.type(screen.getByPlaceholderText(/Marine Ch 16/), "x");
    await userEvent.type(screen.getByPlaceholderText(/156800000/), "100");
    await userEvent.click(screen.getByRole("button", { name: /Add bookmark/ }));

    await waitFor(() => {
      expect(screen.getByText(/name required/)).toBeInTheDocument();
    });
  });

  it("validates the name field and focuses it on submit", async () => {
    vi.mocked(bookmarks.list).mockResolvedValue([]);
    render(<Bookmarks />);
    await waitFor(() => screen.getByText(/No bookmarks yet/));

    // Submit with an empty name → field error + focus, no create call.
    await userEvent.click(screen.getByRole("button", { name: /Add bookmark/ }));

    expect(await screen.findByText("Name is required.")).toBeInTheDocument();
    expect(screen.getByPlaceholderText(/Marine Ch 16/)).toHaveFocus();
    expect(bookmarks.create).not.toHaveBeenCalled();
  });
});
