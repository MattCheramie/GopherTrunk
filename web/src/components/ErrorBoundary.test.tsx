import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { render, screen, act } from "@testing-library/react";
import { ErrorBoundary } from "./ErrorBoundary";

// A child that throws while `boom` is true. Flipping the module-level
// flag between the boundary's automatic retries simulates a fault that
// clears itself vs. one that re-throws on every attempt.
let boom = true;
function Bomb() {
  if (boom) throw new Error("kaboom");
  return <div>child ok</div>;
}

describe("ErrorBoundary", () => {
  beforeEach(() => {
    boom = true;
    vi.useFakeTimers();
    // React logs every caught render error; silence the noise.
    vi.spyOn(console, "error").mockImplementation(() => {});
  });

  afterEach(() => {
    vi.useRealTimers();
    vi.restoreAllMocks();
  });

  it("shows the fallback when a child throws", () => {
    render(
      <ErrorBoundary>
        <Bomb />
      </ErrorBoundary>,
    );
    expect(screen.getByText("Something went wrong")).toBeInTheDocument();
    expect(screen.getByText(/retrying automatically/i)).toBeInTheDocument();
  });

  it("recovers automatically once the fault clears", () => {
    render(
      <ErrorBoundary>
        <Bomb />
      </ErrorBoundary>,
    );
    expect(screen.getByText(/retrying automatically/i)).toBeInTheDocument();

    // The fault clears before the scheduled retry fires.
    boom = false;
    act(() => {
      vi.advanceTimersByTime(4_000);
    });

    expect(screen.getByText("child ok")).toBeInTheDocument();
    expect(screen.queryByText("Something went wrong")).toBeNull();
  });

  it("stops retrying and shows a permanent fallback after repeated failures", () => {
    render(
      <ErrorBoundary>
        <Bomb />
      </ErrorBoundary>,
    );

    // boom stays true: every retry re-throws. Exhaust the retry budget.
    for (let i = 0; i < 3; i++) {
      act(() => {
        vi.advanceTimersByTime(4_000);
      });
    }

    expect(screen.getByText(/stopped rendering/i)).toBeInTheDocument();
    expect(screen.queryByText(/retrying automatically/i)).toBeNull();
  });
});
