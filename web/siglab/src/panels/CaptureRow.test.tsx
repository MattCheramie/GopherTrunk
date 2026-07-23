import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { CaptureRow } from "./CaptureRow";
import type { CaptureDTO } from "../api/types";

const longNamed: CaptureDTO = {
  id: "cap123",
  name: "capture-AIRSPY SN:256863C82B25748B-467.913MHz",
  format: "cs16",
  sample_rate_hz: 50_000,
  size: 1_999_872,
  created_at: "2026-07-18T00:00:00Z",
};

function renderRow(overrides: Partial<Parameters<typeof CaptureRow>[0]> = {}) {
  const props = {
    capture: longNamed,
    selected: false,
    inCompare: false,
    downloadURL: "/api/v1/siglab/captures/cap123/download",
    onSelect: vi.fn(),
    onToggleCompare: vi.fn(),
    onRemove: vi.fn(),
    ...overrides,
  };
  return { props, ...render(<ul>{<CaptureRow {...props} />}</ul>) };
}

describe("CaptureRow", () => {
  it("exposes a persistent per-capture download link with the right href", () => {
    renderRow();
    const dl = screen.getByRole("link", { name: /download capture/i });
    expect(dl).toHaveAttribute("href", "/api/v1/siglab/captures/cap123/download");
    expect(dl).toHaveAttribute("download");
  });

  it("truncates the name but keeps it in a title for the full value", () => {
    renderRow();
    const nameEl = screen.getByText(longNamed.name);
    // The `truncate` utility plus a min-w-0 parent is what keeps a long name
    // from pushing the cmp checkbox and controls out of the card.
    expect(nameEl).toHaveClass("truncate");
    expect(nameEl).toHaveAttribute("title", longNamed.name);
  });

  it("renders a clickable cmp checkbox that toggles compare", () => {
    const { props } = renderRow();
    const cmp = screen.getByRole("checkbox");
    expect(cmp).not.toBeChecked();
    fireEvent.click(cmp);
    expect(props.onToggleCompare).toHaveBeenCalledTimes(1);
  });

  it("reflects the compare and remove wiring", () => {
    const { props } = renderRow({ inCompare: true });
    expect(screen.getByRole("checkbox")).toBeChecked();
    fireEvent.click(screen.getByRole("button", { name: /remove capture/i }));
    expect(props.onRemove).toHaveBeenCalledTimes(1);
  });

  it("shows the capture start time so otherwise-identical captures can be told apart", () => {
    renderRow();
    const when = screen.getByTestId("captured-at");
    // The full ISO timestamp is preserved in the title; the visible text is the
    // localized start time (non-empty).
    expect(when).toHaveAttribute("title", longNamed.created_at);
    expect(when.textContent?.trim()).not.toBe("");
  });

  it("omits the time when created_at is missing", () => {
    renderRow({ capture: { ...longNamed, created_at: "" } });
    expect(screen.queryByTestId("captured-at")).toBeNull();
  });
});
