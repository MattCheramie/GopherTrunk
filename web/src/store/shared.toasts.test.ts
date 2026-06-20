import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import { useShared } from "./shared";

describe("toast notifications", () => {
  beforeEach(() => {
    useShared.setState({ toasts: [], lastError: null });
  });
  afterEach(() => {
    vi.useRealTimers();
  });

  it("enqueues a toast and returns its id", () => {
    const id = useShared.getState().notify("success", "saved");
    const toasts = useShared.getState().toasts;
    expect(toasts).toHaveLength(1);
    expect(toasts[0]).toMatchObject({ id, kind: "success", message: "saved" });
  });

  it("dismisses a toast by id", () => {
    const id = useShared.getState().notify("info", "hello");
    useShared.getState().dismissToast(id);
    expect(useShared.getState().toasts).toHaveLength(0);
  });

  it("auto-dismisses after the kind's TTL", () => {
    vi.useFakeTimers();
    useShared.getState().notify("success", "saved");
    expect(useShared.getState().toasts).toHaveLength(1);
    vi.advanceTimersByTime(4_000);
    expect(useShared.getState().toasts).toHaveLength(0);
  });

  it("routes setError through an error toast and keeps lastError", () => {
    useShared.getState().setError("boom");
    expect(useShared.getState().lastError).toBe("boom");
    const toasts = useShared.getState().toasts;
    expect(toasts).toHaveLength(1);
    expect(toasts[0].kind).toBe("error");
    expect(toasts[0].message).toBe("boom");
  });

  it("does not enqueue a toast when setError clears the error", () => {
    useShared.getState().setError(null);
    expect(useShared.getState().toasts).toHaveLength(0);
    expect(useShared.getState().lastError).toBeNull();
  });

  it("stacks multiple toasts", () => {
    useShared.getState().notify("success", "a");
    useShared.getState().notify("error", "b");
    expect(useShared.getState().toasts.map((t) => t.message)).toEqual(["a", "b"]);
  });
});
