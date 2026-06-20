import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { DetailModal } from "./DetailModal";

describe("DetailModal accessibility", () => {
  it("exposes a labelled modal dialog", () => {
    render(
      <DetailModal title="Talkgroup 42" onClose={() => {}}>
        <button>first</button>
        <button>second</button>
      </DetailModal>,
    );
    const dialog = screen.getByRole("dialog", { name: "Talkgroup 42" });
    expect(dialog).toHaveAttribute("aria-modal", "true");
  });

  it("moves focus into the dialog on open", () => {
    render(
      <DetailModal title="Detail" onClose={() => {}}>
        <button>first</button>
      </DetailModal>,
    );
    // Focus lands inside the dialog (first focusable or the panel itself).
    const dialog = screen.getByRole("dialog");
    expect(dialog.contains(document.activeElement)).toBe(true);
  });

  it("closes on Escape", async () => {
    const user = userEvent.setup();
    let closed = false;
    render(
      <DetailModal title="Detail" onClose={() => (closed = true)}>
        <button>first</button>
      </DetailModal>,
    );
    await user.keyboard("{Escape}");
    expect(closed).toBe(true);
  });

  it("traps Tab within the dialog", async () => {
    const user = userEvent.setup();
    render(
      <DetailModal title="Detail" onClose={() => {}}>
        <button>first</button>
        <button>second</button>
      </DetailModal>,
    );
    // Tabbing past the last focusable wraps back into the dialog rather
    // than escaping to the document body.
    await user.tab();
    await user.tab();
    await user.tab();
    const dialog = screen.getByRole("dialog");
    expect(dialog.contains(document.activeElement)).toBe(true);
  });
});
