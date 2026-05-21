import { Component, type ErrorInfo, type ReactNode } from "react";

interface Props {
  children: ReactNode;
}

interface State {
  error: Error | null;
  // True while an automatic recovery is pending; false once the retry
  // budget is spent and the fallback becomes permanent.
  recovering: boolean;
}

// Delay before an automatic retry. Long enough for a transient event or
// reconnect storm to settle behind the backoff in events.ts before the
// tree re-mounts.
const RECOVERY_DELAY_MS = 4_000;
// Automatic retries before giving up — bounds a render error that
// re-throws on every retry instead of flickering the fallback forever.
const MAX_ATTEMPTS = 3;
// If the boundary stays healthy this long after a reset, the next error
// is treated as unrelated and the retry budget is refreshed.
const HEALTHY_MS = 30_000;

// Top-level error boundary. A render/commit error (e.g. a state-update
// loop tripping "Maximum update depth exceeded") would otherwise unmount
// the whole tree and leave a blank page; this catches it, then retries
// automatically a few times so a transient fault self-heals.
export class ErrorBoundary extends Component<Props, State> {
  state: State = { error: null, recovering: false };

  private recoveryTimer: number | undefined;
  private attempts = 0;
  private lastResetAt = 0;

  static getDerivedStateFromError(error: Error): Partial<State> {
    return { error };
  }

  componentDidCatch(error: Error, info: ErrorInfo) {
    console.error("Unhandled UI error:", error, info.componentStack);

    // A long healthy stretch since the last reset means this error is
    // unrelated to the earlier one — refresh the retry budget.
    if (this.lastResetAt && Date.now() - this.lastResetAt > HEALTHY_MS) {
      this.attempts = 0;
    }
    this.attempts += 1;

    if (this.attempts <= MAX_ATTEMPTS) {
      this.recoveryTimer = window.setTimeout(
        this.recover,
        RECOVERY_DELAY_MS,
      );
      this.setState({ recovering: true });
    } else {
      this.setState({ recovering: false });
    }
  }

  componentWillUnmount() {
    if (this.recoveryTimer !== undefined) {
      window.clearTimeout(this.recoveryTimer);
    }
  }

  private recover = () => {
    this.recoveryTimer = undefined;
    this.lastResetAt = Date.now();
    this.setState({ error: null, recovering: false });
  };

  render() {
    const { error, recovering } = this.state;
    if (!error) return this.props.children;
    return (
      <div className="min-h-full grid place-items-center p-4">
        <div className="panel w-full max-w-md p-6 space-y-4">
          <h1 className="text-xl font-semibold tracking-tight">
            Something went wrong
          </h1>
          <p className="text-sm text-muted">
            {recovering
              ? "The interface hit an unexpected error. Retrying automatically…"
              : "The interface hit an unexpected error and stopped rendering. Reloading usually clears it."}
          </p>
          <pre className="text-xs text-err whitespace-pre-wrap break-words">
            {error.message}
          </pre>
          <button
            className="btn-primary w-full"
            onClick={() => window.location.reload()}
          >
            Reload
          </button>
        </div>
      </div>
    );
  }
}
