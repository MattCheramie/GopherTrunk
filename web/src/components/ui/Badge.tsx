import type { ReactNode } from "react";

type Tone = "neutral" | "ok" | "warn" | "err";

interface Props {
  children: ReactNode;
  tone?: Tone;
  className?: string;
  title?: string;
}

const TONE_CLASS: Record<Tone, string> = {
  neutral: "pill",
  ok: "pill-ok",
  warn: "pill-warn",
  err: "pill-err",
};

// Badge wraps the existing .pill* status-chip classes in a typed
// component. Drop-in for <span className="pill-ok">.
export function Badge({ children, tone = "neutral", className = "", title }: Props) {
  return (
    <span className={`${TONE_CLASS[tone]} ${className}`} title={title}>
      {children}
    </span>
  );
}
