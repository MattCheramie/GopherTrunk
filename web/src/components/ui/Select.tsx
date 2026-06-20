import { forwardRef, type SelectHTMLAttributes } from "react";

type Props = SelectHTMLAttributes<HTMLSelectElement>;

// Select wraps the .input utility for native <select>, keeping form
// controls visually consistent.
export const Select = forwardRef<HTMLSelectElement, Props>(function Select(
  { className = "", children, ...rest },
  ref,
) {
  return (
    <select ref={ref} className={`input ${className}`} {...rest}>
      {children}
    </select>
  );
});
