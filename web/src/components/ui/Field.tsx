import { useId, type ReactNode } from "react";

interface Props {
  label: ReactNode;
  /** Inline validation error; when set the control is marked invalid. */
  error?: string | null;
  /** Helper/hint text shown below the control when there's no error. */
  hint?: ReactNode;
  /** Render-prop receiving the wiring to spread onto the control:
   *  id + aria-describedby + aria-invalid. */
  children: (props: {
    id: string;
    "aria-describedby"?: string;
    "aria-invalid"?: true;
  }) => ReactNode;
  className?: string;
}

// Field is the label + control + message wrapper. It wires up id,
// aria-describedby, and aria-invalid so inline validation is accessible
// and the error/hint is associated with the input. Callers spread the
// render-prop args onto their <Input>/<Select>/<Checkbox>.
export function Field({ label, error, hint, children, className = "" }: Props) {
  const id = useId();
  const msgId = `${id}-msg`;
  const hasMsg = !!error || !!hint;
  return (
    <div className={`space-y-1 ${className}`}>
      <label htmlFor={id} className="block text-sm font-medium">
        {label}
      </label>
      {children({
        id,
        "aria-describedby": hasMsg ? msgId : undefined,
        "aria-invalid": error ? true : undefined,
      })}
      {hasMsg && (
        <p
          id={msgId}
          className={`text-xs ${error ? "text-err" : "text-muted"}`}
          role={error ? "alert" : undefined}
        >
          {error ?? hint}
        </p>
      )}
    </div>
  );
}
