import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

// Mock the API surface BEFORE importing the panel — the module-level
// imports in Import.tsx will resolve to these stubs.
vi.mock("../api/client", () => ({
  api: { runtime: vi.fn() },
  HTTPError: class HTTPError extends Error {
    status: number;
    body: string;
    constructor(status: number, body: string, message: string) {
      super(message);
      this.status = status;
      this.body = body;
    }
  },
}));

vi.mock("../api/write", () => ({
  writes: {
    importUpload: vi.fn(),
    importCommit: vi.fn(),
    importDiscard: vi.fn(),
  },
}));

import { api, HTTPError } from "../api/client";
import { writes } from "../api/write";
import { useShared } from "../store/shared";
import { Import } from "./Import";

// Reset the shared store + mocks between tests so each test has a
// clean slate. zustand's create() returns a singleton across the
// whole module, so we have to reset state manually.
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

describe("Import panel", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    resetStore();
  });

  it("renders the no-config banner when daemon has no -config", async () => {
    vi.mocked(api.runtime).mockResolvedValue({ config_path: "" });

    render(<Import />);

    expect(
      await screen.findByText(/running without a/i),
    ).toBeInTheDocument();
  });

  it("renders the no-mutations banner when canMutate is false", async () => {
    resetStore({ writeMode: false });
    vi.mocked(api.runtime).mockResolvedValue({
      config_path: "/etc/gophertrunk/config.yaml",
    });

    render(<Import />);

    expect(
      await screen.findByText(/Mutations are disabled on this daemon/i),
    ).toBeInTheDocument();
  });

  it("uploads files and renders the preview view on success", async () => {
    const user = userEvent.setup();
    vi.mocked(api.runtime).mockResolvedValue({
      config_path: "/etc/gophertrunk/config.yaml",
    });
    vi.mocked(writes.importUpload).mockResolvedValue({
      id: "stage-abc",
      systems: [
        {
          name: "Metro",
          protocol: "p25-phase1",
          site_count: 3,
          talkgroup_count: 42,
          location: "Downtown",
        },
      ],
    });

    render(<Import />);
    await waitFor(() =>
      expect(api.runtime).toHaveBeenCalled(),
    );

    const file = new File(["hello"], "metro.pdf", { type: "application/pdf" });
    const input = await screen.findByLabelText("", { selector: "input[type=file]" }).catch(() => null);
    // Some bundlers don't expose <input type="file"> via accessible
    // name; fall back to a queryAll on type=file.
    const fileInput = input ?? document.querySelector('input[type="file"]')!;
    await user.upload(fileInput as HTMLElement, file);

    const uploadBtn = await screen.findByRole("button", {
      name: /Upload 1 file/i,
    });
    await user.click(uploadBtn);

    expect(await screen.findByText("Parsed systems")).toBeInTheDocument();
    expect(screen.getByText("Metro")).toBeInTheDocument();
    expect(screen.getByText(/staging id: stage-abc/i)).toBeInTheDocument();
  });

  it("commits a preview and renders the result view", async () => {
    const user = userEvent.setup();
    vi.mocked(api.runtime).mockResolvedValue({
      config_path: "/etc/gophertrunk/config.yaml",
    });
    vi.mocked(writes.importUpload).mockResolvedValue({
      id: "stage-xyz",
      systems: [
        {
          name: "Suburb",
          protocol: "dmr",
          site_count: 1,
          talkgroup_count: 8,
          location: "",
        },
      ],
    });
    vi.mocked(writes.importCommit).mockResolvedValue({
      systems_added: ["Suburb"],
      systems_replaced: [],
      csv_paths: ["/etc/gophertrunk/talkgroups/suburb.csv"],
      config_path: "/etc/gophertrunk/config.yaml",
    });

    render(<Import />);
    await waitFor(() => expect(api.runtime).toHaveBeenCalled());

    const fileInput = document.querySelector('input[type="file"]')!;
    await user.upload(fileInput as HTMLElement, new File(["x"], "s.csv", { type: "text/csv" }));
    await user.click(
      await screen.findByRole("button", { name: /Upload 1 file/i }),
    );

    await user.click(
      await screen.findByRole("button", { name: /Commit to config\.yaml/i }),
    );

    expect(await screen.findByText("Import committed")).toBeInTheDocument();
    expect(screen.getByText(/Suburb/)).toBeInTheDocument();
    expect(
      screen.getByText("/etc/gophertrunk/talkgroups/suburb.csv"),
    ).toBeInTheDocument();
  });

  it("surfaces upload failures via the shared store error channel", async () => {
    const user = userEvent.setup();
    vi.mocked(api.runtime).mockResolvedValue({
      config_path: "/etc/gophertrunk/config.yaml",
    });
    vi.mocked(writes.importUpload).mockRejectedValue(
      new HTTPError(400, "", "bad pdf"),
    );

    render(<Import />);
    await waitFor(() => expect(api.runtime).toHaveBeenCalled());

    const fileInput = document.querySelector('input[type="file"]')!;
    await user.upload(fileInput as HTMLElement, new File(["x"], "bad.pdf"));
    await user.click(
      await screen.findByRole("button", { name: /Upload 1 file/i }),
    );

    await waitFor(() => {
      expect(useShared.getState().lastError).toMatch(/Import upload failed/);
    });
  });

  it("discards a preview without committing", async () => {
    const user = userEvent.setup();
    vi.mocked(api.runtime).mockResolvedValue({
      config_path: "/etc/gophertrunk/config.yaml",
    });
    vi.mocked(writes.importUpload).mockResolvedValue({
      id: "stage-disc",
      systems: [
        {
          name: "X",
          protocol: "nxdn",
          site_count: 1,
          talkgroup_count: 1,
          location: "",
        },
      ],
    });
    vi.mocked(writes.importDiscard).mockResolvedValue(undefined);

    render(<Import />);
    await waitFor(() => expect(api.runtime).toHaveBeenCalled());

    const fileInput = document.querySelector('input[type="file"]')!;
    await user.upload(fileInput as HTMLElement, new File(["x"], "x.pdf"));
    await user.click(
      await screen.findByRole("button", { name: /Upload 1 file/i }),
    );

    await user.click(
      await screen.findByRole("button", { name: /^Discard$/ }),
    );

    expect(writes.importDiscard).toHaveBeenCalledWith(
      expect.anything(),
      "stage-disc",
    );
    // Returns to the stage view (the upload button is back).
    await waitFor(() => {
      expect(
        screen.queryByText("Parsed systems"),
      ).not.toBeInTheDocument();
    });
  });
});
