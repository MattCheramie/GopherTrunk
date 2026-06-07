import { fireEvent, render, screen } from "@testing-library/react";
import { beforeEach, describe, expect, it } from "vitest";
import { useStore } from "../store/shared";
import type { GTConfig } from "../api/types";
import { APRSSection } from "./APRS";
import { BroadcastSection } from "./Broadcast";
import { PagingSection } from "./Paging";

// Seed the store with a minimal draft so section editors can read/patch.
function seed(partial: Partial<GTConfig>) {
  useStore.setState({ config: partial as GTConfig, dirty: false, validation: null, docs: {} });
}

describe("section editors patch the draft with capitalized Go keys", () => {
  beforeEach(() => seed({}));

  it("APRS add channel writes config.APRS.Channels", () => {
    seed({ APRS: { Channels: null } });
    render(<APRSSection />);
    fireEvent.click(screen.getByText("+ Add"));
    const aprs = useStore.getState().config!.APRS;
    expect(aprs.Channels).toHaveLength(1);
    expect(aprs.Channels![0].FrequencyHz).toBe(144_390_000);
    expect(useStore.getState().dirty).toBe(true);
  });

  it("Broadcast add Broadcastify feed writes config.Broadcast.Broadcastify", () => {
    seed({
      Broadcast: { MinDurationMs: 0, Workers: 0, Broadcastify: null, RdioScanner: null, OpenMHz: null, Icecast: null },
    });
    render(<BroadcastSection />);
    // The first "+ Add" button belongs to the Broadcastify list.
    fireEvent.click(screen.getAllByText("+ Add")[0]);
    const b = useStore.getState().config!.Broadcast;
    expect(b.Broadcastify).toHaveLength(1);
    expect(b.Broadcastify![0].Enabled).toBe(true);
  });

  it("Paging add wideband group + channel writes config.Paging.Wideband", () => {
    seed({ Paging: { POCSAG: null, FLEX: null, Wideband: null } });
    render(<PagingSection />);
    // Last "+ Add" button belongs to the Wideband list (after POCSAG, FLEX).
    const adds = screen.getAllByText("+ Add");
    fireEvent.click(adds[adds.length - 1]);
    let wb = useStore.getState().config!.Paging.Wideband;
    expect(wb).toHaveLength(1);
    expect(wb![0].Channels).toBeNull();
    // The freshly-added group exposes its own nested channels "+ Add".
    fireEvent.click(screen.getAllByText("+ Add").pop()!);
    wb = useStore.getState().config!.Paging.Wideband;
    expect(wb![0].Channels).toHaveLength(1);
    expect(wb![0].Channels![0].Protocol).toBe("pocsag");
    expect(wb![0].Channels![0].BaudHz).toBe(1200);
  });
});
