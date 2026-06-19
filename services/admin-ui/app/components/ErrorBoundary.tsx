"use client";
import { Component, ErrorInfo, ReactNode } from "react";

type Props = { children: ReactNode };
type State = { error: Error | null };

export class ErrorBoundary extends Component<Props, State> {
  state: State = { error: null };

  static getDerivedStateFromError(error: Error): State {
    return { error };
  }

  componentDidCatch(error: Error, info: ErrorInfo) {
    console.error("[Dashboard] Unhandled render error:", error, info.componentStack);
  }

  render() {
    const { error } = this.state;
    if (error) {
      return (
        <div style={{ padding: "2rem 2.5rem" }}>
          <div
            className="card"
            style={{ padding: "1.75rem 2rem", borderColor: "color-mix(in srgb, var(--danger) 40%, var(--border))" }}
          >
            <h2 style={{ fontSize: "1rem", fontWeight: 600, color: "var(--danger)", margin: "0 0 0.5rem" }}>
              Something went wrong
            </h2>
            <p style={{ fontSize: "0.8125rem", color: "var(--muted)", margin: "0 0 1.25rem", fontFamily: '"JetBrains Mono", monospace' }}>
              {error.message}
            </p>
            <button className="btn btn-ghost" onClick={() => this.setState({ error: null })}>
              Try again
            </button>
          </div>
        </div>
      );
    }
    return this.props.children;
  }
}
