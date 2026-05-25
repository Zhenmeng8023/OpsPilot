import type { ReactNode } from "react";

interface StateProps {
  children: ReactNode;
  className?: string;
}

export function LoadingState({ children, className = "" }: StateProps) {
  return (
    <div className={`empty-state ${className}`}>
      <StateIcon type="loading" />
      <p className="muted">{children}</p>
      <div style={{ display: "grid", gap: "0.5rem", maxWidth: "20rem", margin: "1rem auto 0" }}>
        <div className="skeleton skeleton-row" />
        <div className="skeleton skeleton-row" style={{ width: "80%" }} />
        <div className="skeleton skeleton-row" style={{ width: "60%" }} />
      </div>
    </div>
  );
}

export function EmptyState({ children, className = "" }: StateProps) {
  return (
    <div className={`empty-state ${className}`}>
      <StateIcon type="empty" />
      <p className="muted">{children}</p>
    </div>
  );
}

export function ErrorState({ children, className = "" }: StateProps) {
  return (
    <div className={`empty-state ${className}`}>
      <StateIcon type="error" />
      <p className="error-text">{children}</p>
    </div>
  );
}

function StateIcon({ type }: { type: "loading" | "empty" | "error" }) {
  if (type === "loading") {
    return (
      <div className="empty-state-icon empty-state-icon-loading" aria-hidden="true">
        <svg viewBox="0 0 24 24" role="img">
          <circle cx="12" cy="12" r="8" />
        </svg>
      </div>
    );
  }
  if (type === "error") {
    return (
      <div className="empty-state-icon empty-state-icon-error" aria-hidden="true">
        <svg viewBox="0 0 24 24" role="img">
          <path d="M12 3 2.5 20h19L12 3Z" />
          <path d="M12 8v5" />
          <path d="M12 17h.01" />
        </svg>
      </div>
    );
  }
  return (
    <div className="empty-state-icon" aria-hidden="true">
      <svg viewBox="0 0 24 24" role="img">
        <path d="M4 6.5A2.5 2.5 0 0 1 6.5 4h11A2.5 2.5 0 0 1 20 6.5v11a2.5 2.5 0 0 1-2.5 2.5h-11A2.5 2.5 0 0 1 4 17.5v-11Z" />
        <path d="M8 9h8" />
        <path d="M8 13h5" />
      </svg>
    </div>
  );
}
