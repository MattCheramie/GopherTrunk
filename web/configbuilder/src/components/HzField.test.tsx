import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { HzField } from "./fields";

describe("HzField", () => {
  it("seeds the text buffer from the Hz value as MHz", () => {
    render(<HzField label="Freq" value={851037500} onChange={vi.fn()} />);
    expect((screen.getByRole("textbox") as HTMLInputElement).value).toBe("851.0375 MHz");
  });

  it("shows empty for a zero value", () => {
    render(<HzField label="Freq" value={0} onChange={vi.fn()} />);
    expect((screen.getByRole("textbox") as HTMLInputElement).value).toBe("");
  });

  it("emits parsed Hz from MHz input and keeps the raw text", () => {
    const onChange = vi.fn();
    render(<HzField label="Freq" value={0} onChange={onChange} />);
    const input = screen.getByRole("textbox");
    fireEvent.change(input, { target: { value: "851.0375" } });
    expect(onChange).toHaveBeenLastCalledWith(851037500);
    // Mid-edit text is preserved (not reformatted to "851.0375 MHz").
    expect((input as HTMLInputElement).value).toBe("851.0375");
  });

  it("emits 0 for unparseable input", () => {
    const onChange = vi.fn();
    render(<HzField label="Freq" value={0} onChange={onChange} />);
    fireEvent.change(screen.getByRole("textbox"), { target: { value: "abc" } });
    expect(onChange).toHaveBeenLastCalledWith(0);
  });
});
