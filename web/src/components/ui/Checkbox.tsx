import { forwardRef, type InputHTMLAttributes, type ReactNode } from "react";

interface Props extends Omit<InputHTMLAttributes<HTMLInputElement>, "type"> {
  label: ReactNode;
}

// Checkbox is a labelled checkbox row matching the app's existing
// `<label><input type=checkbox/></label>` pattern, with the 44px touch
// target from the coarse-pointer rule in styles.css.
export const Checkbox = forwardRef<HTMLInputElement, Props>(function Checkbox(
  { label, className = "", ...rest },
  ref,
) {
  return (
    <label className={`flex items-center gap-3 text-sm ${className}`}>
      <input ref={ref} type="checkbox" className="h-5 w-5" {...rest} />
      <span>{label}</span>
    </label>
  );
});
