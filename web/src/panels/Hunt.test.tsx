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

    await userEvent.type(screen.getByPlaceholderText("00000001"), "dongle-7");
    await userEvent.type(screen.getByPlaceholderText("p25"), "p25");
    await userEvent.click(screen.getByRole("button", { name: /start hunt/i }));

    await waitFor(() => expect(writes.huntStart).toHaveBeenCalled());
    const body = vi.mocked(writes.huntStart).mock.calls[0][1];
    expect(body.serial).toBe("dongle-7");
    expect(body.protocol).toBe("p25");
    expect(body.bands).toEqual(["851:869"]); // default band, comma-split
  });
});
