"use client";
import { Component, ReactNode } from "react";

export default class ErrorBoundary extends Component<
  { children: ReactNode; fallback?: ReactNode },
  { hasError: boolean }
> {
  constructor(props: { children: ReactNode; fallback?: ReactNode }) {
    super(props);
    this.state = { hasError: false };
  }
  static getDerivedStateFromError() { return { hasError: true }; }
  render() {
    if (this.state.hasError) {
      return this.props.fallback ?? (
        <div className="text-center py-16">
          <p className="text-gray-500 mb-4">Something went wrong loading this section.</p>
          <button onClick={() => this.setState({ hasError: false })} className="text-sm text-[#FF9900] underline">
            Try again
          </button>
        </div>
      );
    }
    return this.props.children;
  }
}
