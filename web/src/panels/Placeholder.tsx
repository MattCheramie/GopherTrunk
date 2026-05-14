interface Props {
  title: string;
  hint: string;
}

// Placeholder is used by panels that haven't been built out yet.
// Each panel ships its own dedicated component in a follow-up PR;
// this lets the router and the tab strip be complete from day one
// without committing to half-finished UI.
export function Placeholder({ title, hint }: Props) {
  return (
    <div className="space-y-3 max-w-2xl">
      <h2 className="text-xl font-semibold">{title}</h2>
      <div className="panel p-4 text-sm text-muted">
        <p>{hint}</p>
        <p className="mt-2">
          This panel will arrive in a follow-up commit. The Dashboard
          already exercises the live-update wiring this panel will
          reuse.
        </p>
      </div>
    </div>
  );
}
