import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/react";
import { Field } from "./Field";
import { Input } from "./Input";

describe("Field", () => {
  it("associates the label with the control", () => {
    render(
      <Field label="Server URL">
        {(p) => <Input {...p} defaultValue="x" />}
      </Field>,
    );
    expect(screen.getByLabelText("Server URL")).toBeInTheDocument();
  });

  it("wires aria-invalid and describes the error", () => {
    render(
      <Field label="Frequency" error="must be > 0">
        {(p) => <Input {...p} />}
      </Field>,
    );
    const input = screen.getByLabelText("Frequency");
    expect(input).toHaveAttribute("aria-invalid", "true");
    const msg = screen.getByText("must be > 0");
    expect(msg).toHaveAttribute("role", "alert");
    expect(input.getAttribute("aria-describedby")).toBe(msg.id);
  });

  it("shows hint text when there is no error", () => {
    render(
      <Field label="Limit" hint="defaults to 200">
        {(p) => <Input {...p} />}
      </Field>,
    );
    const hint = screen.getByText("defaults to 200");
    expect(hint).not.toHaveAttribute("role", "alert");
    expect(screen.getByLabelText("Limit")).not.toHaveAttribute("aria-invalid");
  });
});
