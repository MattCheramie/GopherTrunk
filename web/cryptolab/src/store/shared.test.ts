import { describe, expect, it } from "vitest";
import { useStore } from "./shared";
import type { ModeSchema } from "../api/types";

const schema: ModeSchema[] = [
  {
    tool: "crc",
    mode: "compute",
    synopsis: "compute a CRC",
    params: [
      { name: "in", label: "Input", kind: "file", required: true },
      { name: "width", label: "Width", kind: "int", required: false, default: "16" },
      { name: "refin", label: "Reflect in", kind: "bool", required: false },
    ],
  },
];

describe("store.select", () => {
  it("seeds params from schema defaults and resets files/run", () => {
    useStore.setState({ schema });
    useStore.getState().select("crc", "compute");
    const s = useStore.getState();
    expect(s.tool).toBe("crc");
    expect(s.mode).toBe("compute");
    expect(s.params.width).toBe("16");
    expect(s.params.refin).toBe("false");
    expect(s.params.in).toBeUndefined(); // file params carry no default value
    expect(s.files).toEqual({});
    expect(s.run.state).toBe("idle");
  });

  it("setParam updates a single field", () => {
    useStore.setState({ schema });
    useStore.getState().select("crc", "compute");
    useStore.getState().setParam("width", "32");
    expect(useStore.getState().params.width).toBe("32");
  });
});
