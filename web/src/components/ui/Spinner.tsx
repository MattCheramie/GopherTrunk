interface Props {
  /** Tailwind size classes, e.g. "h-4 w-4". Defaults to a small inline
   *  spinner sized to match button text. */
  className?: string;
  label?: string;
}

// Spinner: a minimal CSS spinner for inline-loading and busy states.
// `currentColor` so it inherits the surrounding text color.
export function Spinner({ className = "h-4 w-4", label }: Props) {
  return (
    <span
      role="status"
      aria-live="polite"
      className="inline-flex items-center"
    >
      <svg
        className={`animate-spin ${className}`}
        viewBox="0 0 24 24"
        fill="none"
        aria-hidden
      >
        <circle
          className="opacity-25"
          cx="12"
          cy="12"
          r="10"
          stroke="currentColor"
          strokeWidth="4"
        />
        <path
          className="opacity-75"
          fill="currentColor"
          d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"
        />
      </svg>
      <span className="sr-only">{label ?? "Loading"}</span>
    </span>
  );
}
