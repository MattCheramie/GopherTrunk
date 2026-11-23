import { fireEvent, render, screen, within } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { useStore } from "../store/shared";
import type { GTConfig } from "../api/types";
import { SDRSection } from "./SDR";

// sdr.groups: a group carries the channel plan; a member device names it and
// keeps only its own gain/ppm/bias-tee and an optional centre pin.
describe("SDR groups editor", () => {
  it("adds a group, counts its members and hides a member's own channel plan", () => {
    useStore.setState({
      config: {
        SDR: {
          SampleRate: 2_400_000,
          Devices: [
            { Serial: "A", Role: "wideband", PPM: 0, Gain: "auto", BiasTee: false, CenterFreqHz: 0, Group: "south" },
            { Serial: "B", Role: "wideband", PPM: 0, Gain: "auto", BiasTee: false, CenterFreqHz: 0, Group: "south" },
          ],
        },
        Trunking: { Systems: [] },
      } as unknown as GTConfig,
      dirty: false,
      validation: null,
      docs: {},
    });
    render(<SDRSection />);
    const fieldset = screen.getByText("SDR groups (gang wideband sticks)").closest("details")!;
    fireEvent.click(within(fieldset).getByText("+ Add"));
    const groups = useStore.getState().config!.SDR.Groups!;
    expect(groups).toHaveLength(1);
    expect(groups[0]).toMatchObject({ Name: "", OverlapHz: 0, Channels: null });

    // Members show the centre pin, not the standalone wideband plan.
    expect(screen.getAllByText("Center freq (pin; blank = planned)")).toHaveLength(2);
    // The only channel list on the page is the new group's own.
    expect(screen.getAllByText("Wideband channels")).toHaveLength(1);
    expect(within(fieldset).getByText("Wideband channels")).toBeInTheDocument();
  });
});
