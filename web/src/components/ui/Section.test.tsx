import { describe, it, expect, beforeEach } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { Section } from "./Section";
import { Button } from "./Button";

describe("Section", () => {
  beforeEach(() => {
    window.localStorage.clear();
  });

  it("renders children when expanded by default", () => {
    render(
      <Section id="test-a" title="Scan">
        <p>body content</p>
      </Section>,
    );
    expect(screen.getByText("body content")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /Scan/i })).toHaveAttribute(
      "aria-expanded",
      "true",
    );
  });

  it("honors defaultCollapsed", () => {
    render(
      <Section id="test-b" title="Advanced" defaultCollapsed>
        <p>hidden body</p>
      </Section>,
    );
    expect(screen.queryByText("hidden body")).toBeNull();
  });

  it("toggles and persists the collapsed state", async () => {
    const user = userEvent.setup();
    const { unmount } = render(
      <Section id="persist-me" title="Group">
        <p>content</p>
      </Section>,
    );
    await user.click(screen.getByRole("button", { name: /Group/i }));
    expect(screen.queryByText("content")).toBeNull();

    // Remount: the collapsed state should be restored from prefs.
    unmount();
    render(
      <Section id="persist-me" title="Group">
        <p>content</p>
      </Section>,
    );
    expect(screen.queryByText("content")).toBeNull();
  });
});

describe("Button loading", () => {
  it("disables and marks busy while loading", () => {
    render(
      <Button loading onClick={() => {}}>
        Save
      </Button>,
    );
    const btn = screen.getByRole("button", { name: /Save/i });
    expect(btn).toBeDisabled();
    expect(btn).toHaveAttribute("aria-busy", "true");
  });
});
