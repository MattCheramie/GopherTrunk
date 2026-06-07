import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { AdvancedJSON } from "./AdvancedJSON";

interface Knobs {
  A: string;
  B: number;
  keep: boolean;
}

describe("AdvancedJSON", () => {
  it("only surfaces picked keys in the textarea", () => {
    render(
      <AdvancedJSON<Knobs>
        value={{ A: "x", B: 2, keep: true }}
        onChange={vi.fn()}
        pick={["A", "B"]}
      />,
    );
    const ta = screen.getByRole("textbox") as HTMLTextAreaElement;
    expect(ta.value).toContain('"A"');
    expect(ta.value).toContain('"B"');
    expect(ta.value).not.toContain('"keep"');
  });

  it("merges valid JSON back into value (preserving unpicked keys)", () => {
    const onChange = vi.fn();
    render(
      <AdvancedJSON<Knobs>
        value={{ A: "x", B: 2, keep: true }}
        onChange={onChange}
        pick={["A", "B"]}
      />,
    );
    const ta = screen.getByRole("textbox");
    fireEvent.change(ta, { target: { value: '{"A":"y","B":9}' } });
    expect(onChange).toHaveBeenCalledWith({ A: "y", B: 9, keep: true });
  });

  it("does not call onChange while JSON is invalid, and shows an error", () => {
    const onChange = vi.fn();
    render(<AdvancedJSON value={{ A: "x" }} onChange={onChange} />);
    const ta = screen.getByRole("textbox");
    fireEvent.change(ta, { target: { value: "{ not json" } });
    expect(onChange).not.toHaveBeenCalled();
    expect(screen.getByText(/Invalid JSON/)).toBeInTheDocument();
  });

  it("rejects non-object JSON", () => {
    const onChange = vi.fn();
    render(<AdvancedJSON value={{ A: "x" }} onChange={onChange} />);
    const ta = screen.getByRole("textbox");
    fireEvent.change(ta, { target: { value: "[1,2,3]" } });
    expect(onChange).not.toHaveBeenCalled();
    expect(screen.getByText(/expected a JSON object/)).toBeInTheDocument();
  });
});
