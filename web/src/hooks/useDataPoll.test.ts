import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import { renderHook, act } from "@testing-library/react";
import { useDataPoll } from "./useDataPoll";

describe("useDataPoll", () => {
  beforeEach(() => {
    vi.useFakeTimers();
  });
  afterEach(() => {
    vi.useRealTimers();
  });

  it("fetches on mount and delivers the first snapshot", async () => {
    const onData = vi.fn();
    const fetcher = vi.fn().mockResolvedValue([1, 2, 3]);
    const { result } = renderHook(() =>
      useDataPoll({ fetcher, onData, intervalMs: 1000 }),
    );

    await act(async () => {
      await vi.advanceTimersByTimeAsync(0);
    });

    expect(onData).toHaveBeenCalledWith([1, 2, 3]);
    expect(result.current.loading).toBe(false);
    expect(result.current.error).toBeNull();
  });

  it("does not re-deliver an unchanged snapshot", async () => {
    const onData = vi.fn();
    const fetcher = vi.fn().mockResolvedValue([1, 2, 3]);
    renderHook(() => useDataPoll({ fetcher, onData, intervalMs: 1000 }));

    await act(async () => {
      await vi.advanceTimersByTimeAsync(0);
    });
    await act(async () => {
      await vi.advanceTimersByTimeAsync(1000);
    });

    // Two fetches, but only one delivery — the payload never changed.
    expect(fetcher).toHaveBeenCalledTimes(2);
    expect(onData).toHaveBeenCalledTimes(1);
  });

  it("delivers a changed snapshot", async () => {
    const onData = vi.fn();
    const fetcher = vi
      .fn()
      .mockResolvedValueOnce([1])
      .mockResolvedValueOnce([1, 2]);
    renderHook(() => useDataPoll({ fetcher, onData, intervalMs: 1000 }));

    await act(async () => {
      await vi.advanceTimersByTimeAsync(0);
    });
    await act(async () => {
      await vi.advanceTimersByTimeAsync(1000);
    });

    expect(onData).toHaveBeenCalledTimes(2);
    expect(onData).toHaveBeenLastCalledWith([1, 2]);
  });

  it("marks data stale when a poll fails after a good fetch", async () => {
    const onData = vi.fn();
    const fetcher = vi
      .fn()
      .mockResolvedValueOnce([1])
      .mockRejectedValueOnce(new Error("network down"));
    const { result } = renderHook(() =>
      useDataPoll({ fetcher, onData, intervalMs: 1000 }),
    );

    await act(async () => {
      await vi.advanceTimersByTimeAsync(0);
    });
    expect(result.current.stale).toBe(false);

    await act(async () => {
      await vi.advanceTimersByTimeAsync(1000);
    });

    expect(result.current.stale).toBe(true);
    expect(result.current.error).toBe("network down");
  });

  it("refresh triggers an immediate fetch", async () => {
    const onData = vi.fn();
    const fetcher = vi.fn().mockResolvedValue([1]);
    const { result } = renderHook(() =>
      useDataPoll({ fetcher, onData, intervalMs: 100000 }),
    );

    await act(async () => {
      await vi.advanceTimersByTimeAsync(0);
    });
    expect(fetcher).toHaveBeenCalledTimes(1);

    await act(async () => {
      result.current.refresh();
      await vi.advanceTimersByTimeAsync(0);
    });
    expect(fetcher).toHaveBeenCalledTimes(2);
  });
});
