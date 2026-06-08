import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { ListEditor } from "./ListEditor";

interface Row {
  name: string;
}

function renderEditor(items: Row[] | null, onChange = vi.fn()) {
  render(
    <ListEditor<Row>
      label="Rows"
      items={items}
      onChange={onChange}
      makeNew={() => ({ name: "new" })}
      itemTitle={(r, i) => r.name || `Row ${i + 1}`}
      addLabel="+ Add row"
      emptyHint="nothing here"
      renderItem={(item, set) => (
        <input
          aria-label="name"
          value={item.name}
          onChange={(e) => set({ ...item, name: e.target.value })}
        />
      )}
    />,
  );
  return onChange;
}

describe("ListEditor", () => {
  it("shows the empty hint and count when there are no items", () => {
    renderEditor([]);
    expect(screen.getByText("nothing here")).toBeInTheDocument();
    expect(screen.getByText("Rows (0)")).toBeInTheDocument();
  });

  it("treats null items as empty", () => {
    renderEditor(null);
    expect(screen.getByText("Rows (0)")).toBeInTheDocument();
  });

  it("appends makeNew() on add", () => {
    const onChange = renderEditor([{ name: "a" }]);
    fireEvent.click(screen.getByText("+ Add row"));
    expect(onChange).toHaveBeenCalledWith([{ name: "a" }, { name: "new" }]);
  });

  it("removes the right index after confirmation", () => {
    const onChange = renderEditor([{ name: "a" }, { name: "b" }, { name: "c" }]);
    // First click arms the confirm; nothing removed yet.
    fireEvent.click(screen.getAllByText("Remove")[1]);
    expect(onChange).not.toHaveBeenCalled();
    // Confirm actually removes the middle item.
    fireEvent.click(screen.getByText("Confirm remove"));
    expect(onChange).toHaveBeenCalledWith([{ name: "a" }, { name: "c" }]);
  });

  it("replaces a single item immutably via the per-item set", () => {
    const onChange = renderEditor([{ name: "a" }, { name: "b" }]);
    const inputs = screen.getAllByLabelText("name");
    fireEvent.change(inputs[1], { target: { value: "B" } });
    expect(onChange).toHaveBeenCalledWith([{ name: "a" }, { name: "B" }]);
  });
});
