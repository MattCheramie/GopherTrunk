import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { NumberField } from "./fields";

describe("NumberField (integer path)", () => {
  it("renders a number spinbutton and emits numbers", () => {
    const onChange = vi.fn();
    render(<NumberField label="Port" value={8080} onChange={onChange} />);
    const input = screen.getByRole("spinbutton") as HTMLInputElement;
    expect(input.value).toBe("8080");
    fireEvent.change(input, { target: { value: "9090" } });
    expect(onChange).toHaveBeenLastCalledWith(9090);
  });

  it("emits 0 for empty input", () => {
    const onChange = vi.fn();
    render(<NumberField label="Port" value={1} onChange={onChange} />);
    fireEvent.change(screen.getByRole("spinbutton"), { target: { value: "" } });
    expect(onChange).toHaveBeenLastCalledWith(0);
  });
});

describe("NumberField (decimal path)", () => {
  it("uses a text input when given a fractional step", () => {
    render(<NumberField label="CTCSS" step={0.1} value={131.8} onChange={vi.fn()} />);
    expect((screen.getByRole("textbox") as HTMLInputElement).value).toBe("131.8");
  });

  it("emits the parsed decimal and keeps the raw text", () => {
    const onChange = vi.fn();
    render(<NumberField label="CTCSS" step={0.1} value={0} onChange={onChange} />);
    const input = screen.getByRole("textbox") as HTMLInputElement;
    fireEvent.change(input, { target: { value: "131.8" } });
    expect(onChange).toHaveBeenLastCalledWith(131.8);
    expect(input.value).toBe("131.8");
  });

  it("preserves an in-progress decimal like '1500.' without wiping to 0", () => {
    const onChange = vi.fn();
    render(<NumberField label="Freq" step={0.1} value={0} onChange={onChange} />);
    const input = screen.getByRole("textbox") as HTMLInputElement;
    fireEvent.change(input, { target: { value: "1500." } });
    // The raw text is kept so the user can finish typing the fraction.
    expect(input.value).toBe("1500.");
    // And the parsed numeric value (1500) is emitted, not 0.
    expect(onChange).toHaveBeenLastCalledWith(1500);
  });

  it("emits 0 for empty or unparseable input", () => {
    const onChange = vi.fn();
    render(<NumberField label="Freq" step={0.1} value={5} onChange={onChange} />);
    const input = screen.getByRole("textbox");
    fireEvent.change(input, { target: { value: "" } });
    expect(onChange).toHaveBeenLastCalledWith(0);
    fireEvent.change(input, { target: { value: "abc" } });
    expect(onChange).toHaveBeenLastCalledWith(0);
  });

  it("resyncs the text buffer when the bound value changes out-of-band", () => {
    const { rerender } = render(
      <NumberField label="Freq" step={0.1} value={131.8} onChange={vi.fn()} />,
    );
    expect((screen.getByRole("textbox") as HTMLInputElement).value).toBe("131.8");
    rerender(<NumberField label="Freq" step={0.1} value={146.2} onChange={vi.fn()} />);
    expect((screen.getByRole("textbox") as HTMLInputElement).value).toBe("146.2");
  });
});
