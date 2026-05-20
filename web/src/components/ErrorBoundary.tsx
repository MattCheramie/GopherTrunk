import { Component, type ErrorInfo, type ReactNode } from "react";

interface Props {
  children: ReactNode;
}

interface State {
  error: Error | null;
}

// Top-level error boundary. A render/commit error (e.g. a state-update
// loop tripping "Maximum update depth exceeded") would otherwise unmount
// the whole tree and leave a blank page; this catches it and shows an
// actionable fallback instead.
export class ErrorBoundary extends Component<Props, State> {
  state: State = { error: null };

  static getDerivedStateFromError(error: Error): State {
    return { error };
  }

  componentDidCatch(error: Error, info: ErrorInfo) {
    console.error("Unhandled UI error:", error, info.componentStack);
  }

  render() {
    if (!this.state.error) return this.props.children;
    return (
      <div className="min-h-full grid place-items-center p-4">
        <div className="panel w-full max-w-md p-6 space-y-4">
          <h1 className="text-xl font-semibold tracking-tight">
            Something went wrong
          </h1>
          <p className="text-sm text-muted">
            The interface hit an unexpected error and stopped rendering.
            Reloading usually clears it.
          </p>
          <pre className="text-xs text-err whitespace-pre-wrap break-words">
            {this.state.error.message}
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
