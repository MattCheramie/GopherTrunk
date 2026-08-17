import { describe, it, expect, vi, beforeEach } from "vitest";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router-dom";

vi.mock("../api/client", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../api/client")>();
  return {
    ...actual,
    api: { ...actual.api, talkgroups: vi.fn() },
  };
});

vi.mock("../api/write", () => ({
  writes: { updateTalkgroup: vi.fn() },
}));

import { api } from "../api/client";
import { writes } from "../api/write";
import { useShared } from "../store/shared";
import type { TalkgroupDTO } from "../api/types";
import { Talkgroups } from "./Talkgroups";

const tg: TalkgroupDTO = {
  id: 1020545,
  alpha_tag: "OLD-NAME",
  scan: true,
};

function resetStore(canMutate: boolean) {
  useShared.setState({
    serverURL: "http://localhost:8080",
    token: null,
    connected: true,
    wsStatus: "idle",
    // selectCanMutate requires BOTH the daemon's capability and the
    // browser's write-mode toggle.
    writeMode: canMutate,
    mutations: canMutate ? { allow_mutations: true } : null,
    lastError: null,
    events: [],
    activeCalls: [],
    devices: [],
    systems: [],
    talkgroups: [tg],
    health: null,
    audio: null,
    scanner: null,
  });
}

function renderPanel() {
  return render(
    <MemoryRouter>
      <Talkgroups />
    </MemoryRouter>,
  );
}

describe("Talkgroups panel", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    vi.mocked(api.talkgroups).mockResolvedValue([tg]);
  });

  // Naming a talkgroup was impossible from the UI: the PATCH body had no
  // field for it and the panel rendered no input.
  it("patches alpha_tag from the detail modal", async () => {
    resetStore(true);
    vi.mocked(writes.updateTalkgroup).mockResolvedValue({
      ...tg,
      alpha_tag: "TAC-1",
    });
    const user = userEvent.setup();
    renderPanel();

    await user.click(await screen.findByText("OLD-NAME"));
    // role-scoped: the DataTable's column-visibility control also exposes a
    // checkbox named "Alpha tag".
    const input = await screen.findByRole("textbox", { name: "Alpha tag" });
    await user.clear(input);
    await user.type(input, "TAC-1{Enter}"); // commit on Enter, not per keystroke

    await waitFor(() => {
      expect(writes.updateTalkgroup).toHaveBeenCalled();
    });
    const [, id, patch] = vi.mocked(writes.updateTalkgroup).mock.calls[0];
    expect(id).toBe(1020545);
    expect(patch).toMatchObject({ alpha_tag: "TAC-1" });
  });

  it("commits once per edit, not once per keystroke", async () => {
    resetStore(true);
    vi.mocked(writes.updateTalkgroup).mockResolvedValue(tg);
    const user = userEvent.setup();
    renderPanel();

    await user.click(await screen.findByText("OLD-NAME"));
    // role-scoped: the DataTable's column-visibility control also exposes a
    // checkbox named "Alpha tag".
    const input = await screen.findByRole("textbox", { name: "Alpha tag" });
    await user.clear(input);
    await user.type(input, "ABCDE{Enter}");

    await waitFor(() => {
      expect(writes.updateTalkgroup).toHaveBeenCalledTimes(1);
    });
  });

  // Blur is the path an operator hits most: type, then click elsewhere.
  it("commits a name on blur", async () => {
    resetStore(true);
    vi.mocked(writes.updateTalkgroup).mockResolvedValue(tg);
    const user = userEvent.setup();
    renderPanel();

    await user.click(await screen.findByText("OLD-NAME"));
    const input = await screen.findByRole("textbox", { name: "Alpha tag" });
    await user.clear(input);
    await user.type(input, "BLURRED");
    fireEvent.blur(input);

    await waitFor(() => {
      expect(writes.updateTalkgroup).toHaveBeenCalledTimes(1);
    });
    expect(vi.mocked(writes.updateTalkgroup).mock.calls[0][2]).toMatchObject({
      alpha_tag: "BLURRED",
    });
  });

  it("hides the name inputs when write mode is off", async () => {
    resetStore(false);
    const user = userEvent.setup();
    renderPanel();

    await user.click(await screen.findByText("OLD-NAME"));
    expect(screen.queryByRole("textbox", { name: "Alpha tag" })).toBeNull();
    expect(
      await screen.findByText(/Enable write mode in Settings/),
    ).toBeInTheDocument();
  });

  it("offers a CSV export of the names", async () => {
    resetStore(true);
    renderPanel();
    const link = await screen.findByText(/Export names/);
    expect(link.getAttribute("href")).toContain("/api/v1/labels/export");
    expect(link.getAttribute("href")).toContain("kind=talkgroup");
  });
});
