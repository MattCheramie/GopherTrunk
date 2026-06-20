import { forwardRef, type ButtonHTMLAttributes } from "react";
import { Spinner } from "./Spinner";

type Variant = "primary" | "ghost" | "danger";

interface Props extends ButtonHTMLAttributes<HTMLButtonElement> {
  variant?: Variant;
  /** When true, shows a spinner and disables the button. */
  loading?: boolean;
}

const VARIANT_CLASS: Record<Variant, string> = {
  primary: "btn-primary",
  ghost: "btn-ghost",
  danger: "btn-danger",
};

// Button wraps the existing .btn-* utility classes in a typed component
// and adds a `loading` state (spinner + disabled) so mutations across
// the app get consistent pending feedback. Falls back to the same
// markup the hand-rolled <button className="btn-primary"> sites use, so
// it's a drop-in replacement.
export const Button = forwardRef<HTMLButtonElement, Props>(function Button(
  { variant = "primary", loading = false, disabled, children, className = "", ...rest },
  ref,
) {
  return (
    <button
      ref={ref}
      className={`${VARIANT_CLASS[variant]} ${className}`}
      disabled={disabled || loading}
      aria-busy={loading || undefined}
      {...rest}
    >
      {loading && <Spinner className="h-4 w-4" />}
      {children}
    </button>
  );
});
