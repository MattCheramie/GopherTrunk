import { fireEvent, render, screen } from "@testing-library/react";
import { useState } from "react";
import { describe, expect, it, vi } from "vitest";
import { FreqListField } from "./fields";

// A controlled host that mirrors real usage: the parsed value round-trips back
// into the field on every change, exactly as the Trunking section does.
function Host({ onChange }: { onChange?: (v: number[]) => void }) {
  const [value, setValue] = useState<number[]>([]);
  return (
    <FreqListField
      label="Control channels"
      value={value}
      onChange={(v) => {
        setValue(v);
        onChange?.(v);
      }}
    />
  );
}

describe("FreqListField", () => {
  it("seeds the textarea from the Hz list as MHz, one per line", () => {
    render(<FreqListField label="CC" value={[851037500, 851287500]} onChange={vi.fn()} />);
    expect((screen.getByRole("textbox") as HTMLTextAreaElement).value).toBe("851.0375 MHz\n851.2875 MHz");
  });

  it("keeps a newline so a second control channel can be entered", () => {
    const onChange = vi.fn();
    render(<Host onChange={onChange} />);
    const area = screen.getByRole("textbox") as HTMLTextAreaElement;

    // Type the first frequency, then a newline. Before the fix the newline was
    // parsed away and the value round-tripped back to "469.875 MHz" with no
    // second line — the caret collapsed and a second channel was impossible.
    fireEvent.change(area, { target: { value: "469.875" } });
    fireEvent.change(area, { target: { value: "469.875 MHz\n" } });
    expect(area.value).toBe("469.875 MHz\n"); // the newline survives the round-trip

    // Now the second line can be filled in.
    fireEvent.change(area, { target: { value: "469.875 MHz\n467.9125" } });
    expect(onChange).toHaveBeenLastCalledWith([469875000, 467912500]);
    expect(area.value).toBe("469.875 MHz\n467.9125");
  });

  it("resyncs the buffer when the bound value changes out-of-band", () => {
    const { rerender } = render(<FreqListField label="CC" value={[469875000]} onChange={vi.fn()} />);
    rerender(<FreqListField label="CC" value={[453000000, 454000000]} onChange={vi.fn()} />);
    expect((screen.getByRole("textbox") as HTMLTextAreaElement).value).toBe("453 MHz\n454 MHz");
  });
});
