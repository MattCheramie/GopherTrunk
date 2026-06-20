import { describe, it, expect, beforeEach } from "vitest";
import { render, screen, act } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { ToastViewport } from "./ToastViewport";
import { useShared, type ToastKind } from "../../store/shared";

function notify(kind: ToastKind, message: string, ttlMs?: number) {
  act(() => {
    useShared.getState().notify(kind, message, ttlMs);
  });
}

describe("ToastViewport", () => {
  beforeEach(() => {
    useShared.setState({ toasts: [] });
  });

  it("renders nothing when there are no toasts", () => {
    const { container } = render(<ToastViewport />);
    expect(container).toBeEmptyDOMElement();
  });

  it("renders enqueued toasts", () => {
    render(<ToastViewport />);
    notify("success", "It worked", 0);
    expect(screen.getByText("It worked")).toBeInTheDocument();
  });

  it("marks errors as assertive alerts and info as status", () => {
    render(<ToastViewport />);
    notify("error", "Bad thing", 0);
    notify("info", "FYI", 0);
    expect(screen.getByText("Bad thing").closest("[role]")).toHaveAttribute(
      "role",
      "alert",
    );
    expect(screen.getByText("FYI").closest("[role]")).toHaveAttribute(
      "role",
      "status",
    );
  });

  it("dismisses a toast when the dismiss button is clicked", async () => {
    const user = userEvent.setup();
    render(<ToastViewport />);
    notify("info", "dismiss me", 0);
    expect(screen.getByText("dismiss me")).toBeInTheDocument();
    await user.click(
      screen.getByRole("button", { name: /Dismiss notification/i }),
    );
    expect(screen.queryByText("dismiss me")).toBeNull();
  });
});
