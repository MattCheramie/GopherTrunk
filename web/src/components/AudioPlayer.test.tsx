import { act, cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { AudioPlayer } from "./AudioPlayer";
import { useShared } from "../store/shared";
import type { ActiveCallDTO } from "../api/types";

const activeCallsMock = vi.fn(() => Promise.resolve([] as ActiveCallDTO[]));

vi.mock("../api/client", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../api/client")>();
  return {
    ...actual,
    api: {
      ...actual.api,
      activeCalls: () => activeCallsMock(),
    },
  };
});

vi.mock("../api/write", () => ({
  writes: {
    setAudio: vi.fn().mockResolvedValue(undefined),
  },
}));

function call(overrides: Partial<ActiveCallDTO> & { group: number; tag: string }): ActiveCallDTO {
  return {
    grant: {
      system: "Test System",
      protocol: "p25",
      group_id: overrides.group,
      frequency_hz: 851_000_000 + overrides.group,
      encrypted: false,
      data_call: false,
      ...overrides.grant,
    },
    talkgroup: { id: overrides.group, alpha_tag: overrides.tag },
    device_serial: overrides.device_serial ?? `tap-${overrides.group}`,
    started_at:
      overrides.started_at ??
      `2026-05-29T12:00:${String(overrides.group % 60).padStart(2, "0")}.000Z`,
    signal_dbfs: overrides.signal_dbfs,
    ...overrides,
  };
}

function resetStore(activeCalls: ActiveCallDTO[]) {
  activeCallsMock.mockResolvedValue(activeCalls);
  useShared.setState({
    serverURL: "http://radio.test",
    token: null,
    activeCalls,
    audio: null,
  });
}

beforeEach(() => {
  activeCallsMock.mockReset();
  activeCallsMock.mockResolvedValue([]);
  HTMLMediaElement.prototype.play = vi.fn().mockResolvedValue(undefined);
  HTMLMediaElement.prototype.pause = vi.fn();
  HTMLMediaElement.prototype.load = vi.fn();
});

afterEach(() => {
  cleanup();
  vi.useRealTimers();
  vi.restoreAllMocks();
});

describe("AudioPlayer", () => {
  it("switches away when the selected call becomes encrypted", async () => {
    const clearA = call({ group: 101, tag: "Clear A", signal_dbfs: -32 });
    const clearB = call({ group: 202, tag: "Clear B", signal_dbfs: -35 });
    resetStore([clearA, clearB]);

    render(<AudioPlayer />);

    await waitFor(() => expect(screen.getByRole("button", { name: "Enable audio" })).not.toBeDisabled());
    await act(async () => {
      fireEvent.click(screen.getByRole("button", { name: "Enable audio" }));
      await Promise.resolve();
    });
    const audio = document.querySelector("audio") as HTMLAudioElement;
    await waitFor(() => expect(audio.src).toContain("talkgroup=101"));

    useShared.setState({
      activeCalls: [{ ...clearA, grant: { ...clearA.grant, encrypted: true } }, clearB],
    });

    await waitFor(() => expect(audio.src).toContain("talkgroup=202"));
    expect(screen.getByLabelText("Audio call")).toHaveTextContent("Clear B");
  });

  it("does not let an obsolete play rejection disable a newer stream", async () => {
    const clearA = call({ group: 101, tag: "Clear A", signal_dbfs: -32 });
    const clearB = call({ group: 202, tag: "Clear B", signal_dbfs: -35 });
    resetStore([clearA, clearB]);

    let rejectFirst!: (reason?: unknown) => void;
    const play = vi
      .spyOn(HTMLMediaElement.prototype, "play")
      .mockImplementationOnce(
        () => new Promise<void>((_, reject) => { rejectFirst = reject; }),
      )
      .mockResolvedValue(undefined);

    render(<AudioPlayer />);

    await waitFor(() => expect(screen.getByRole("button", { name: "Enable audio" })).not.toBeDisabled());
    const audio = document.querySelector("audio") as HTMLAudioElement;

    await act(async () => {
      fireEvent.click(screen.getByRole("button", { name: "Enable audio" }));
      await Promise.resolve();
    });
    await waitFor(() => expect(rejectFirst).toBeTypeOf("function"));

    await act(async () => {
      useShared.setState({ activeCalls: [clearB] });
    });
    await waitFor(() => expect(audio.src).toContain("talkgroup=202"));

    await act(async () => {
      rejectFirst(new DOMException("aborted", "AbortError"));
      await Promise.resolve();
    });

    await waitFor(() => expect(screen.getByRole("button", { name: "Stop audio" })).toBeInTheDocument());
    expect(play).toHaveBeenCalledTimes(2);
  });

  it("does not overlap active-call polls", async () => {
    vi.useFakeTimers();
    const clearA = call({ group: 101, tag: "Clear A", signal_dbfs: -32 });
    resetStore([clearA]);

    let resolvePoll!: (calls: ActiveCallDTO[]) => void;
    activeCallsMock.mockImplementation(
      () => new Promise<ActiveCallDTO[]>((resolve) => { resolvePoll = resolve; }),
    );

    render(<AudioPlayer />);
    await act(async () => {
      fireEvent.click(screen.getByRole("button", { name: "Enable audio" }));
      await Promise.resolve();
    });

    expect(activeCallsMock).toHaveBeenCalledTimes(1);
    await act(async () => {
      await vi.advanceTimersByTimeAsync(3_000);
    });
    expect(activeCallsMock).toHaveBeenCalledTimes(1);

    await act(async () => {
      resolvePoll([clearA]);
      await Promise.resolve();
    });
    await act(async () => {
      await vi.advanceTimersByTimeAsync(1_000);
    });
    expect(activeCallsMock).toHaveBeenCalledTimes(2);
  });
});
