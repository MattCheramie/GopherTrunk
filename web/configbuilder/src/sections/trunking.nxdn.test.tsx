import { fireEvent, render, screen, within } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { useStore } from "../store/shared";
import type { GTConfig, SystemConfig } from "../api/types";
import { TrunkingSection } from "./Trunking";

// #1261: the web builder had no way to enter an NXDN band plan (and the
// field was not even in the Advanced JSON knobs), so trunked NXDN could
// only be configured by hand-editing YAML.
describe("Trunking NXDN band plan editor", () => {
  it("table mode adds channel rows to config.Trunking.Systems[0].NXDNBandPlan", () => {
    const sys = { Name: "SJPR NXDN", Protocol: "nxdn", ControlChannels: [453_775_000], TalkgroupFile: "" } as SystemConfig;
    useStore.setState({
      config: { Trunking: { Systems: [sys] } } as unknown as GTConfig,
      dirty: false,
      validation: null,
      docs: {},
    });
    render(<TrunkingSection />);
    const legend = screen.getByText("NXDN band plan (required for NXDN trunked voice)");
    const fieldset = legend.closest("details")!;
    fireEvent.change(within(fieldset).getByRole("combobox"), { target: { value: "table" } });
    expect(useStore.getState().config!.Trunking.Systems![0].NXDNBandPlan).toEqual({ Linear: null, Table: [] });

    fireEvent.click(within(fieldset).getByText("+ Add"));
    const plan = useStore.getState().config!.Trunking.Systems![0].NXDNBandPlan!;
    expect(plan.Linear).toBeNull();
    expect(plan.Table).toEqual([{ Channel: 0, FreqHz: 0 }]);

    fireEvent.change(within(fieldset).getByRole("combobox"), { target: { value: "linear" } });
    expect(useStore.getState().config!.Trunking.Systems![0].NXDNBandPlan).toEqual({
      Linear: { BaseHz: 0, SpacingHz: 0, Offset: 1 },
      Table: null,
    });
  });
});
