import type { ReactNode } from "react";

interface Props {
  children: ReactNode;
  className?: string;
  /** Optional section title rendered as a .panel-title header. */
  title?: ReactNode;
  /** Optional right-aligned header actions (buttons, status). */
  actions?: ReactNode;
  /** Padding preset. "none" lets the caller own padding (e.g. tables). */
  padding?: "none" | "sm" | "md";
}

const PAD: Record<NonNullable<Props["padding"]>, string> = {
  none: "",
  sm: "p-3",
  md: "p-4",
};

// Card is the elevated container primitive built on the surface tokens.
// It supersedes ad-hoc `.panel p-4` blocks and gives an optional
// title/actions header for consistency.
export function Card({ children, className = "", title, actions, padding = "md" }: Props) {
  return (
    <section className={`card ${PAD[padding]} ${className}`}>
      {(title || actions) && (
        <header className="flex items-center justify-between gap-3 mb-3">
          {title && <h3 className="panel-title">{title}</h3>}
          {actions && <div className="flex items-center gap-2">{actions}</div>}
        </header>
      )}
      {children}
    </section>
  );
}
