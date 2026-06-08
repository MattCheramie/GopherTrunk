import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { TuningControls } from "./TuningControls";

function setup(overrides: Partial<Parameters<typeof TuningControls>[0]> = {}) {
  const onOffsetChange = vi.fn();
  const onHoldChange = vi.fn();
  const onCentre = vi.fn();
  render(
    <TuningControls
      centerHz={851_000_000}
      maxOffsetKHz={1024}
      offsetKHz={0}
      onOffsetChange={onOffsetChange}
      hold={true}
      onHoldChange={onHoldChange}
      followingActive={false}
      onCentre={onCentre}
      {...overrides}
    />,
  );
  return { onOffsetChange, onHoldChange, onCentre };
}

describe("TuningControls", () => {
  beforeEach(() => {
    // The channel-step selector persists to localStorage; clear it so the
    // default (12.5 kHz) is in effect for each case.
    window.localStorage.clear();
  });

  it("accepts a fine kHz offset like 12.5 (channel-grid step)", () => {
    const { onOffsetChange } = setup();
    const input = screen.getByLabelText(
      "View offset from SDR centre in kHz (numeric)",
    );
    fireEvent.change(input, { target: { value: "12.5" } });
    expect(onOffsetChange).toHaveBeenCalledWith(12.5);
  });

  it("derives the offset from an absolute MHz frequency", () => {
    const { onOffsetChange } = setup({ centerHz: 851_000_000 });
    const freq = screen.getByLabelText("Tuned frequency, MHz");
    // 851.0125 MHz is 12.5 kHz above the 851.0 MHz centre.
    fireEvent.change(freq, { target: { value: "851.0125" } });
    expect(onOffsetChange).toHaveBeenCalledWith(12.5);
  });

  it("shows the current frequency derived from centre + offset", () => {
    setup({ centerHz: 851_000_000, offsetKHz: 25 });
    const freq = screen.getByLabelText("Tuned frequency, MHz") as HTMLInputElement;
    expect(Number(freq.value)).toBeCloseTo(851.025, 6);
  });

  it("disables the MHz field until the device centre is known", () => {
    setup({ centerHz: null });
    const freq = screen.getByLabelText("Tuned frequency, MHz") as HTMLInputElement;
    expect(freq.disabled).toBe(true);
  });

  it("recentres via the Centre button", () => {
    const { onCentre } = setup({ offsetKHz: 12.5 });
    fireEvent.click(screen.getByRole("button", { name: "Centre" }));
    expect(onCentre).toHaveBeenCalled();
  });

  it("nudges the offset by the selected channel step", () => {
    const { onOffsetChange } = setup({ offsetKHz: 0 });
    // Default step is 12.5 kHz.
    fireEvent.click(
      screen.getByRole("button", { name: "Step offset up by 12.5 kHz" }),
    );
    expect(onOffsetChange).toHaveBeenCalledWith(12.5);
  });

  it("snaps an off-grid offset to the grid when stepping down", () => {
    // From 13.1 kHz a down-step on the 12.5 grid snaps to 0
    // (round(13.1/12.5)=1, then -1 → 0).
    const { onOffsetChange } = setup({ offsetKHz: 13.1 });
    fireEvent.click(
      screen.getByRole("button", { name: "Step offset down by 12.5 kHz" }),
    );
    expect(onOffsetChange).toHaveBeenCalledWith(0);
  });

  it("steps by the chosen grid when the step selector changes", () => {
    const { onOffsetChange } = setup({ offsetKHz: 0 });
    fireEvent.change(screen.getByLabelText("Channel step size, kHz"), {
      target: { value: "6.25" },
    });
    fireEvent.click(
      screen.getByRole("button", { name: "Step offset up by 6.25 kHz" }),
    );
    expect(onOffsetChange).toHaveBeenCalledWith(6.25);
  });

  it("walks the grid with ArrowUp/ArrowDown on the offset field", () => {
    const { onOffsetChange } = setup({ offsetKHz: 0 });
    const input = screen.getByLabelText(
      "View offset from SDR centre in kHz (numeric)",
    );
    fireEvent.keyDown(input, { key: "ArrowUp" });
    expect(onOffsetChange).toHaveBeenLastCalledWith(12.5);
    fireEvent.keyDown(input, { key: "ArrowDown" });
    expect(onOffsetChange).toHaveBeenLastCalledWith(-12.5);
  });
});
