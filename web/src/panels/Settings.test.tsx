import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

vi.mock("../api/client", () => ({
  api: { runtime: vi.fn() },
  HTTPError: class HTTPError extends Error {
    status: number;
    body: string;
    // Match the real HTTPError signature: status, body, message.
    constructor(status: number, body: string, message: string) {
      super(message);
      this.status = status;
      this.body = body;
    }
  },
}));

// stubAudio satisfies the AudioStatusDTO shape with sensible
// defaults so individual tests can override one or two fields
// without enumerating every property.
function stubAudio(over: Partial<{
  volume: number;
  muted: boolean;
  backend_enabled: boolean;
  sample_rate: number;
  recording_enabled: boolean;
  drops_total: number;
}> = {}) {
  return {
    backend_enabled: true,
    sample_rate: 8000,
    volume: 0.8,
    muted: false,
    recording_enabled: false,
    drops_total: 0,
    ...over,
  };
}

vi.mock("../api/write", () => ({
  writes: { updateSettings: vi.fn() },
}));

import { api, HTTPError } from "../api/client";
import { writes } from "../api/write";
import { useShared } from "../store/shared";
import { Settings } from "./Settings";

function resetStore(opts: { writeMode?: boolean; mutationsAllowed?: boolean } = {}) {
  const writeMode = opts.writeMode ?? true;
  const mutationsAllowed = opts.mutationsAllowed ?? true;
  useShared.setState({
    serverURL: "http://localhost:8080",
    token: null,
    writeMode,
    mutations: mutationsAllowed ? { allow_mutations: true } : null,
    lastError: null,
  });
}

describe("Settings inline-edit (Live config)", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    resetStore();
  });

  it("hides editable rows and shows the no-config banner when daemon has no -config", async () => {
    vi.mocked(api.runtime).mockResolvedValue({ config_path: "" });

    render(<Settings />);

    expect(
      await screen.findByText(/PATCH \/api\/v1\/settings/i),
    ).toBeInTheDocument();
    // No editable rows.
    expect(screen.queryByText("Log level")).not.toBeInTheDocument();
  });

  it("renders editable rows when daemon has a -config", async () => {
    vi.mocked(api.runtime).mockResolvedValue({
      config_path: "/etc/gophertrunk/config.yaml",
      log_level: "info",
      audio: stubAudio({ volume: 0.7, muted: false }),
    });

    render(<Settings />);

    expect(await screen.findByText("Log level")).toBeInTheDocument();
    expect(screen.getByText(/Audio volume/i)).toBeInTheDocument();
    // Row value renders.
    expect(screen.getAllByText("info").length).toBeGreaterThan(0);
  });

  it("PATCHes a field on save and refreshes the runtime", async () => {
    const user = userEvent.setup();
    vi.mocked(api.runtime)
      .mockResolvedValueOnce({
        config_path: "/etc/gophertrunk/config.yaml",
        log_level: "info",
      })
      .mockResolvedValueOnce({
        config_path: "/etc/gophertrunk/config.yaml",
        log_level: "debug",
      });
    vi.mocked(writes.updateSettings).mockResolvedValue({
      applied: ["log.level"],
      restart_required: [],
      config_path: "/etc/gophertrunk/config.yaml",
      runtime: {} as never,
    });

    render(<Settings />);
    await screen.findByText("Log level");

    // The Edit button on the Log level row.
    const rows = screen.getAllByRole("row");
    const logRow = rows.find((r) => r.textContent?.startsWith("Log level"))!;
    const editBtn = logRow.querySelector("button")!;
    await user.click(editBtn);

    const input = await screen.findByDisplayValue("info");
    await user.clear(input);
    await user.type(input, "debug");
    await user.click(await screen.findByRole("button", { name: /^Save$/ }));

    await waitFor(() => {
      expect(writes.updateSettings).toHaveBeenCalledWith(
        expect.anything(),
        { log_level: "debug" },
      );
    });
  });

  it("rejects out-of-range audio volume client-side", async () => {
    const user = userEvent.setup();
    vi.mocked(api.runtime).mockResolvedValue({
      config_path: "/etc/gophertrunk/config.yaml",
      audio: stubAudio({ volume: 0.5, muted: false }),
    });

    render(<Settings />);
    await screen.findByText(/Audio volume/i);

    const rows = screen.getAllByRole("row");
    const volRow = rows.find((r) => r.textContent?.startsWith("Audio volume"))!;
    await user.click(volRow.querySelector("button")!);
    const input = await screen.findByDisplayValue("0.5");
    await user.clear(input);
    await user.type(input, "5");
    await user.click(await screen.findByRole("button", { name: /^Save$/ }));

    expect(
      await screen.findByRole("alert"),
    ).toHaveTextContent(/audio.volume must be a float in \[0, 1\]/i);
    expect(writes.updateSettings).not.toHaveBeenCalled();
  });

  it("surfaces server PATCH failures inline as an alert", async () => {
    const user = userEvent.setup();
    vi.mocked(api.runtime).mockResolvedValue({
      config_path: "/etc/gophertrunk/config.yaml",
      log_level: "info",
    });
    vi.mocked(writes.updateSettings).mockRejectedValue(
      new HTTPError(
        400,
        "",
        "validation: log.level must be one of debug|info|warn|error",
      ),
    );

    render(<Settings />);
    await screen.findByText("Log level");
    const rows = screen.getAllByRole("row");
    const logRow = rows.find((r) => r.textContent?.startsWith("Log level"))!;
    await user.click(logRow.querySelector("button")!);
    const input = await screen.findByDisplayValue("info");
    await user.clear(input);
    await user.type(input, "verbose");
    await user.click(await screen.findByRole("button", { name: /^Save$/ }));

    const alert = await screen.findByRole("alert");
    expect(alert).toHaveTextContent(/log\.level: validation/i);
  });

  it("disables the Edit button when canMutate is false", async () => {
    resetStore({ writeMode: false });
    vi.mocked(api.runtime).mockResolvedValue({
      config_path: "/etc/gophertrunk/config.yaml",
      log_level: "info",
    });

    render(<Settings />);
    await screen.findByText("Log level");

    const rows = screen.getAllByRole("row");
    const logRow = rows.find((r) => r.textContent?.startsWith("Log level"))!;
    const editBtn = logRow.querySelector("button")!;
    expect(editBtn).toBeDisabled();
    expect(
      screen.getByText(/Mutations are disabled — daemon auth blocks writes/i),
    ).toBeInTheDocument();
  });

  it("Escape cancels the inline edit", async () => {
    const user = userEvent.setup();
    vi.mocked(api.runtime).mockResolvedValue({
      config_path: "/etc/gophertrunk/config.yaml",
      log_level: "info",
    });

    render(<Settings />);
    await screen.findByText("Log level");
    const rows = screen.getAllByRole("row");
    const logRow = rows.find((r) => r.textContent?.startsWith("Log level"))!;
    await user.click(logRow.querySelector("button")!);
    const input = await screen.findByDisplayValue("info");
    await user.type(input, "{Escape}");

    // Edit input should be gone.
    await waitFor(() => {
      expect(screen.queryByDisplayValue("info")).not.toBeInTheDocument();
    });
    // Edit button is back.
    expect(
      logRow.querySelector("button")!.textContent,
    ).toMatch(/Edit/i);
  });

  it("renders the [restart] badge for restart-required fields", async () => {
    vi.mocked(api.runtime).mockResolvedValue({
      config_path: "/etc/gophertrunk/config.yaml",
      log_level: "info",
      log_format: "text",
    });

    render(<Settings />);
    await screen.findByText(/Log format/i);

    const rows = screen.getAllByRole("row");
    const formatRow = rows.find((r) => r.textContent?.startsWith("Log format"))!;
    expect(formatRow.textContent).toMatch(/\[restart\]/);
  });
});
