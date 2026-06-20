interface Props {
  className?: string;
}

// Skeleton is a single shimmering placeholder block. Compose several to
// stand in for loading cards/rows.
export function Skeleton({ className = "h-4 w-full" }: Props) {
  return (
    <div
      className={`animate-pulse rounded bg-surface3/60 ${className}`}
      aria-hidden
    />
  );
}

// SkeletonRows renders `rows` placeholder lines — handy for tables/lists
// while the first fetch is in flight.
export function SkeletonRows({ rows = 5 }: { rows?: number }) {
  return (
    <div className="space-y-2" role="status" aria-label="Loading">
      {Array.from({ length: rows }).map((_, i) => (
        <Skeleton key={i} className="h-8 w-full" />
      ))}
    </div>
  );
}
