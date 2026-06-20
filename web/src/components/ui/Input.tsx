import { forwardRef, type InputHTMLAttributes } from "react";

interface Props extends InputHTMLAttributes<HTMLInputElement> {
  invalid?: boolean;
}

// Input wraps the .input utility. `invalid` (or aria-invalid) paints the
// error border so field-level validation reads clearly.
export const Input = forwardRef<HTMLInputElement, Props>(function Input(
  { className = "", invalid, ...rest },
  ref,
) {
  const isInvalid = invalid || rest["aria-invalid"] === true;
  return (
    <input
      ref={ref}
      className={`input ${isInvalid ? "border-err focus:border-err focus:ring-err" : ""} ${className}`}
      aria-invalid={isInvalid || undefined}
      {...rest}
    />
  );
});
