import { beforeEach, describe, expect, it } from "vitest";
import { fieldHelp, useStore } from "./shared";

// fieldHelp reads the shared per-field registry that init() fetches from
// /config/fieldmeta — the same Go source the terminal builder uses — so web
// field help can't drift from the registry.
describe("fieldHelp", () => {
  beforeEach(() => useStore.setState({ fieldmeta: {} }));

  it("returns the registry help for a Struct.Field key", () => {
    useStore.setState({
      fieldmeta: { "SystemConfig.Name": { help: "Unique label for this system." } },
    });
    expect(fieldHelp("SystemConfig", "Name")).toBe("Unique label for this system.");
  });

  it("returns undefined when the registry has no entry / hasn't loaded", () => {
    expect(fieldHelp("SystemConfig", "Name")).toBeUndefined();
    useStore.setState({ fieldmeta: { "SystemConfig.Name": { help: "" } } });
    expect(fieldHelp("SystemConfig", "Name")).toBeUndefined();
  });
});
