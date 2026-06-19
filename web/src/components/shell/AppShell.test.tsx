import { describe, it, expect, beforeEach } from "vitest";
import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router-dom";
import { AppShell } from "./AppShell";
import { NAV_ITEMS } from "../../nav/registry";
import { useShared } from "../../store/shared";

function renderShell() {
  return render(
    <MemoryRouter initialEntries={["/dashboard"]}>
      <AppShell>
        <div>page content</div>
      </AppShell>
    </MemoryRouter>,
  );
}

describe("AppShell navigation", () => {
  beforeEach(() => {
    useShared.setState({ hiddenTabs: [], wsStatus: "open" });
  });

  it("renders the page content", () => {
    renderShell();
    expect(screen.getByText("page content")).toBeInTheDocument();
  });

  it("shows four primaries plus a More button in the bottom nav", () => {
    renderShell();
    // Both the desktop sidebar and mobile bottom nav are in the DOM
    // (CSS hides one per breakpoint), so scope to the bottom nav links.
    expect(
      screen.getByRole("button", { name: /More — open navigation menu/i }),
    ).toBeInTheDocument();
  });

  it("opens the drawer from More and exposes every panel — including the previously-orphaned DSP panels", async () => {
    const user = userEvent.setup();
    renderShell();

    await user.click(
      screen.getByRole("button", { name: /More — open navigation menu/i }),
    );

    const drawer = await screen.findByRole("dialog", { name: "Navigation" });
    // Every non-external registry item is reachable inside the drawer.
    for (const item of NAV_ITEMS) {
      expect(
        within(drawer).getByText(item.label),
        `drawer missing ${item.label}`,
      ).toBeInTheDocument();
    }
    // The headline regression: these were unreachable everywhere before.
    expect(within(drawer).getByText("Constellation")).toBeInTheDocument();
    expect(within(drawer).getByText("Histogram")).toBeInTheDocument();
  });

  it("closes the drawer on Escape", async () => {
    const user = userEvent.setup();
    renderShell();
    await user.click(
      screen.getByRole("button", { name: /More — open navigation menu/i }),
    );
    expect(await screen.findByRole("dialog", { name: "Navigation" })).toBeInTheDocument();
    await user.keyboard("{Escape}");
    expect(screen.queryByRole("dialog", { name: "Navigation" })).toBeNull();
  });

  it("opens the command palette with Ctrl-K and filters panels", async () => {
    const user = userEvent.setup();
    renderShell();

    await user.keyboard("{Control>}k{/Control}");
    const palette = await screen.findByRole("dialog", {
      name: "Command palette",
    });

    const input = within(palette).getByRole("textbox");
    await user.type(input, "histog");
    const options = within(palette).getAllByRole("option");
    expect(options).toHaveLength(1);
    expect(options[0]).toHaveTextContent("Histogram");
  });

  it("reflects the live connection status in the top bar", () => {
    useShared.setState({ wsStatus: "open" });
    renderShell();
    expect(screen.getByText("Live")).toBeInTheDocument();
  });
});
