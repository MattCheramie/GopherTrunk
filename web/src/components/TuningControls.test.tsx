import { describe, it, expect, vi } from "vitest";
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
});
