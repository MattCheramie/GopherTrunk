import type { ReactNode } from "react";

interface Props {
  /** Primary message. */
  message: ReactNode;
  /** Optional decorative glyph above the message. */
  icon?: ReactNode;
  /** Optional call-to-action (e.g. a Button). */
  action?: ReactNode;
  className?: string;
}

// EmptyState replaces the repeated muted "No data." paragraph with a
// centered, optionally actionable placeholder.
export function EmptyState({ message, icon, action, className = "" }: Props) {
  return (
    <div className={`text-center text-sm text-muted py-8 px-4 ${className}`}>
      {icon && (
        <div className="text-2xl mb-2 opacity-60" aria-hidden>
          {icon}
        </div>
      )}
      <p className="max-w-md mx-auto">{message}</p>
      {action && <div className="mt-3">{action}</div>}
    </div>
  );
}
