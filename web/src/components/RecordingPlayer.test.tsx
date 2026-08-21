import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import { RecordingPlayer } from "./RecordingPlayer";

const cfg = { baseURL: "http://localhost:8080", token: null };

describe("RecordingPlayer", () => {
  beforeEach(() => {
    // jsdom lacks the object-URL APIs; stub them so the component can run.
    vi.stubGlobal("URL", {
      ...URL,
      createObjectURL: vi.fn(() => "blob:mock-url"),
      revokeObjectURL: vi.fn(),
    });
  });
  afterEach(() => {
    vi.unstubAllGlobals();
    vi.restoreAllMocks();
  });

  it("fetches the recording with the bearer token and plays the blob", async () => {
    const blob = new Blob([new Uint8Array([1, 2, 3])], { type: "audio/wav" });
    // A minimal Response stand-in: jsdom's real Response.blob() is unreliable,
    // and the component only touches ok/status/blob().
    const fetchMock = vi.fn(
      async (_url: string, _init?: RequestInit) =>
        ({ ok: true, status: 200, blob: async () => blob }) as unknown as Response,
    );
    vi.stubGlobal("fetch", fetchMock);

    render(
      <RecordingPlayer cfg={{ baseURL: "http://localhost:8080", token: "sekret" }} callId={5202} />,
    );

    await waitFor(() =>
      expect(document.querySelector("audio")).toBeInTheDocument(),
    );
    // The <audio> plays the object URL, never the raw API URL — so a media
    // element quirk / missing auth header can't silently kill playback.
    expect(document.querySelector("audio")?.getAttribute("src")).toBe(
      "blob:mock-url",
    );
    // The fetch carried the Authorization header the bare element can't.
    const [url, init] = fetchMock.mock.calls[0];
    expect(url).toBe("http://localhost:8080/api/v1/calls/5202/audio");
    expect(init?.headers).toMatchObject({
      Authorization: "Bearer sekret",
    });
  });

  it("shows a visible message instead of a dead player when the recording is gone", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn(
        async () =>
          ({
            ok: false,
            status: 404,
            blob: async () => new Blob(),
          }) as unknown as Response,
      ),
    );

    render(<RecordingPlayer cfg={cfg} callId={9} />);

    await waitFor(() =>
      expect(screen.getByRole("alert")).toHaveTextContent(/unavailable/i),
    );
    expect(document.querySelector("audio")).not.toBeInTheDocument();
  });
});
