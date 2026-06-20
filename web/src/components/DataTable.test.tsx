import { describe, it, expect, beforeEach, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { DataTable, type Column } from "./DataTable";

interface Item {
  id: number;
  name: string;
}

const columns: Column<Item>[] = [
  {
    key: "id",
    header: "ID",
    render: (r) => <span>{r.id}</span>,
    sort: (a, b) => a.id - b.id,
  },
  {
    key: "name",
    header: "Name",
    render: (r) => <span>{r.name}</span>,
    sort: (a, b) => a.name.localeCompare(b.name),
  },
];

function rows(n: number): Item[] {
  return Array.from({ length: n }, (_, i) => ({ id: i + 1, name: `item-${i + 1}` }));
}

describe("DataTable", () => {
  beforeEach(() => {
    window.localStorage.clear();
  });

  it("renders rows (backward compatible defaults)", () => {
    render(
      <DataTable rows={rows(3)} columns={columns} rowKey={(r) => String(r.id)} />,
    );
    expect(screen.getByText("item-1")).toBeInTheDocument();
    expect(screen.getByText("item-3")).toBeInTheDocument();
  });

  it("shows the empty message when there are no rows", () => {
    render(
      <DataTable
        rows={[]}
        columns={columns}
        rowKey={(r) => String(r.id)}
        emptyMessage="Nothing here"
      />,
    );
    expect(screen.getByText("Nothing here")).toBeInTheDocument();
  });

  it("filters via the built-in search box", async () => {
    const user = userEvent.setup();
    render(
      <DataTable
        rows={rows(5)}
        columns={columns}
        rowKey={(r) => String(r.id)}
        searchable
        searchAccessor={(r) => r.name}
      />,
    );
    await user.type(screen.getByLabelText("Search table"), "item-3");
    expect(screen.getByText("item-3")).toBeInTheDocument();
    expect(screen.queryByText("item-1")).toBeNull();
  });

  it("paginates and steps pages", async () => {
    const user = userEvent.setup();
    render(
      <DataTable
        rows={rows(25)}
        columns={columns}
        rowKey={(r) => String(r.id)}
        defaultSortKey="id"
        pageSize={10}
      />,
    );
    // First page shows 1..10, not 11.
    expect(screen.getByText("item-10")).toBeInTheDocument();
    expect(screen.queryByText("item-11")).toBeNull();
    expect(screen.getByText("1 / 3")).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: /Next/i }));
    expect(screen.getByText("item-11")).toBeInTheDocument();
    expect(screen.queryByText("item-10")).toBeNull();
  });

  it("renders a row-actions cell whose clicks don't trigger row click", async () => {
    const user = userEvent.setup();
    const onRowClick = vi.fn();
    const onAction = vi.fn();
    render(
      <DataTable
        rows={rows(1)}
        columns={columns}
        rowKey={(r) => String(r.id)}
        onRowClick={onRowClick}
        rowActions={(r) => (
          <button onClick={() => onAction(r.id)}>act</button>
        )}
      />,
    );
    await user.click(screen.getByRole("button", { name: "act" }));
    expect(onAction).toHaveBeenCalledWith(1);
    expect(onRowClick).not.toHaveBeenCalled();
  });

  it("shows skeleton rows while loading with no data", () => {
    render(
      <DataTable
        rows={[]}
        columns={columns}
        rowKey={(r) => String(r.id)}
        loading
      />,
    );
    expect(screen.getByRole("status", { name: "Loading" })).toBeInTheDocument();
  });

  it("hides and persists a column via the column menu", async () => {
    const user = userEvent.setup();
    const { unmount } = render(
      <DataTable
        rows={rows(2)}
        columns={columns}
        rowKey={(r) => String(r.id)}
        tableId="test-table"
      />,
    );
    // Open the <details> menu and uncheck "Name".
    await user.click(screen.getByText("Columns"));
    const menuItem = screen.getByLabelText("Name");
    await user.click(menuItem);
    expect(screen.queryByRole("columnheader", { name: /Name/ })).toBeNull();

    // Remount: the hidden column stays hidden (persisted).
    unmount();
    render(
      <DataTable
        rows={rows(2)}
        columns={columns}
        rowKey={(r) => String(r.id)}
        tableId="test-table"
      />,
    );
    expect(screen.queryByRole("columnheader", { name: /Name/ })).toBeNull();
  });
});
