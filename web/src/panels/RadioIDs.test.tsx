import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter, Routes, Route } from "react-router-dom";

vi.mock("../api/client", async () => {
  const actual = await vi.importActual<typeof import("../api/client")>(
    "../api/client",
  );
  return {
    ...actual,
    api: {
      rids: vi.fn(),
      rid: vi.fn(),
      ridHistory: vi.fn(),
      systems: vi.fn().mockResolvedValue([]),
      locations: vi.fn().mockResolvedValue([]),
    },
  };
});

vi.mock("../api/write", () => ({
  writes: { updateRID: vi.fn() },
}));

import { api } from "../api/client";
import { writes } from "../api/write";
import { useShared } from "../store/shared";
import { RadioIDs } from "./RadioIDs";
import type { RIDDTO } from "../api/types";

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
    rids: [],
    health: null,
    audio: null,
    scanner: null,
  });
}

function renderPanel() {
  return render(
    <MemoryRouter>
      <RadioIDs />
    </MemoryRouter>,
  );
}

// renderAt mounts RadioIDs under both /rids and /rids/:id (mirroring App.tsx)
// starting at the given URL, so the deep-link behaviour can be exercised.
function renderAt(path: string) {
  return render(
    <MemoryRouter initialEntries={[path]}>
      <Routes>
        <Route path="/rids" element={<RadioIDs />} />
        <Route path="/rids/:id" element={<RadioIDs />} />
      </Routes>
    </MemoryRouter>,
  );
}

describe("RadioIDs panel", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    resetStore();
  });

  it("renders empty-state guidance when neither RIDDB nor live tracker has data", async () => {
    vi.mocked(api.rids).mockResolvedValue([]);
    renderPanel();
    await waitFor(() => {
      expect(screen.getByText(/No radio IDs observed yet/)).toBeInTheDocument();
    });
  });

  it("renders configured and live-only rows distinctly", async () => {
    const rows: RIDDTO[] = [
      {
        id: 100,
        alias: "CHIEF",
        owner: "Cmdr",
        configured: true,
        watch: true,
        talker_alias: "CHIEF-ENG",
        last_talkgroup: 50,
        call_count: 3,
      },
      {
        id: 300,
        configured: false,
        last_talkgroup: 50,
        call_count: 1,
      },
    ];
    vi.mocked(api.rids).mockResolvedValue(rows);
    renderPanel();

    await waitFor(() => {
      expect(screen.getByText("CHIEF")).toBeInTheDocument();
    });
    // Configured row shows the alias and a "cfg" pill.
    const chiefRow = screen.getByText("CHIEF").closest("tr");
    expect(chiefRow).not.toBeNull();
    expect(chiefRow!.textContent).toMatch(/cfg/);
    expect(chiefRow!.textContent).toMatch(/CHIEF-ENG/);
    // Live-only row has no alias and no "cfg" pill.
    const liveRow = screen.getByText("300").closest("tr");
    expect(liveRow).not.toBeNull();
    expect(liveRow!.textContent).not.toMatch(/cfg/);
  });

  it("filters by id, alias, talker alias, owner", async () => {
    const rows: RIDDTO[] = [
      { id: 100, alias: "CHIEF", configured: true, watch: true },
      { id: 200, alias: "PATROL", configured: true, watch: true },
      { id: 300, talker_alias: "ENG-12", configured: false },
    ];
    vi.mocked(api.rids).mockResolvedValue(rows);
    renderPanel();
    await waitFor(() => {
      expect(screen.getByText("CHIEF")).toBeInTheDocument();
    });

    const search = screen.getByPlaceholderText(/Search by id, alias/);
    await userEvent.type(search, "eng");
    // Only the live row whose talker_alias matches survives.
    expect(screen.queryByText("CHIEF")).not.toBeInTheDocument();
    expect(screen.queryByText("PATROL")).not.toBeInTheDocument();
    expect(screen.getByText("300")).toBeInTheDocument();
  });

  it("filters by system via the dropdown and re-fetches scoped", async () => {
    vi.mocked(api.rids).mockResolvedValue([
      { id: 100, alias: "CHIEF", configured: true, watch: true, system: "Metro" },
    ]);
    vi.mocked(api.systems).mockResolvedValue([
      { name: "Metro", protocol: "tetra" },
      { name: "Harbor", protocol: "dmr-tier2" },
    ] as never);

    renderPanel();
    // First load is unscoped.
    await waitFor(() => {
      expect(api.rids).toHaveBeenCalledWith(expect.anything(), {});
    });
    // Dropdown is populated from api.systems.
    const select = await screen.findByLabelText("Filter by system");
    await waitFor(() => {
      expect(screen.getByRole("option", { name: "Harbor" })).toBeInTheDocument();
    });

    await userEvent.selectOptions(select, "Harbor");
    // Selecting a system re-fetches scoped to it.
    await waitFor(() => {
      expect(api.rids).toHaveBeenCalledWith(expect.anything(), {
        system: "Harbor",
      });
    });
  });

  it("opens a detail modal and fetches recent calls when a row is clicked", async () => {
    vi.mocked(api.rids).mockResolvedValue([
      {
        id: 4242,
        alias: "ALPHA",
        configured: true,
        watch: true,
        call_count: 2,
        last_talkgroup: 99,
      },
    ]);
    vi.mocked(api.ridHistory).mockResolvedValue([
      {
        id: 1,
        system: "Metro",
        protocol: "p25",
        group_id: 99,
        frequency_hz: 851_000_000,
        started_at: "2026-05-26T12:00:00Z",
        talkgroup_alpha: "FIRE-DISP",
      },
      {
        id: 2,
        system: "Metro",
        protocol: "p25",
        group_id: 99,
        frequency_hz: 851_000_000,
        started_at: "2026-05-26T12:01:00Z",
      },
    ]);
    renderPanel();
    await waitFor(() => {
      expect(screen.getByText("ALPHA")).toBeInTheDocument();
    });

    await userEvent.click(screen.getByText("ALPHA"));

    await waitFor(() => {
      expect(api.ridHistory).toHaveBeenCalledWith(
        expect.anything(),
        4242,
        expect.objectContaining({ limit: 50 }),
      );
    });
    // Modal renders the alpha tag of one of the call rows.
    await waitFor(() => {
      expect(screen.getByText(/FIRE-DISP/)).toBeInTheDocument();
    });
  });

  // Regression: a /rids/:id deep link (e.g. a source-radio click in CC
  // Activity or call history) must open that radio's detail modal. The route
  // used to be unregistered, so such links fell through to the dashboard.
  it("opens the detail modal from a /rids/:id deep link (row already loaded)", async () => {
    const rows: RIDDTO[] = [{ id: 777, alias: "DEEPLINK", configured: true }];
    // Seed the store so the row is present on the first render (the effect
    // resolves it from the list rather than racing the poll into a fetch).
    useShared.setState({ rids: rows });
    vi.mocked(api.rids).mockResolvedValue(rows);
    vi.mocked(api.ridHistory).mockResolvedValue([]);
    renderAt("/rids/777");

    // The modal subtitle "RID 777" appears without any click.
    await waitFor(() => {
      expect(screen.getByText(/RID 777/)).toBeInTheDocument();
    });
    // It resolved from the loaded list, so no single-RID fetch was needed.
    expect(api.rid).not.toHaveBeenCalled();
  });

  it("fetches a /rids/:id deep link that is not in the current list", async () => {
    vi.mocked(api.rids).mockResolvedValue([]); // filtered/empty list
    vi.mocked(api.ridHistory).mockResolvedValue([]);
    vi.mocked(api.rid).mockResolvedValue({
      id: 999,
      alias: "OFFLIST",
      configured: false,
    });
    renderAt("/rids/999");

    await waitFor(() => {
      expect(api.rid).toHaveBeenCalledWith(expect.anything(), 999);
    });
    await waitFor(() => {
      expect(screen.getByText("OFFLIST")).toBeInTheDocument();
    });
  });

  // The operator's case: a radio seen on the air and in no rid_alias_file.
  // Naming it used to be impossible — the panel showed a dead end telling
  // them to edit a file and reload the daemon, and the PATCH 404'd anyway.
  it("lets an operator name a radio that is in no alias file", async () => {
    useShared.setState({ writeMode: true, mutations: { allow_mutations: true } });
    const rows: RIDDTO[] = [{ id: 1005736, configured: false, call_count: 2012 }];
    vi.mocked(api.rids).mockResolvedValue(rows);
    vi.mocked(api.ridHistory).mockResolvedValue([]);
    vi.mocked(writes.updateRID).mockResolvedValue({
      id: 1005736,
      alias: "CPL-SMITH",
    });
    const user = userEvent.setup();
    renderPanel();

    await user.click(await screen.findByText("1005736"));
    const input = await screen.findByRole("textbox", { name: "Name" });
    await user.type(input, "CPL-SMITH{Enter}");

    await waitFor(() => {
      expect(writes.updateRID).toHaveBeenCalled();
    });
    const [, id, patch] = vi.mocked(writes.updateRID).mock.calls[0];
    expect(id).toBe(1005736);
    expect(patch).toMatchObject({ alias: "CPL-SMITH" });
    // The old dead-end copy told operators to edit a file and reload.
    expect(screen.queryByText(/Add it to the file/)).toBeNull();
  });

  it("offers a CSV export of the names", async () => {
    vi.mocked(api.rids).mockResolvedValue([]);
    renderPanel();
    const link = await screen.findByText(/Export names/);
    expect(link.getAttribute("href")).toContain("/api/v1/labels/export");
    expect(link.getAttribute("href")).toContain("kind=rid");
  });
});
