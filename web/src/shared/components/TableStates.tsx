import type { ReactNode } from "react";

interface StateProps {
  children: ReactNode;
  className?: string;
}

export function LoadingState({ children, className = "" }: StateProps) {
  return (
    <div className={`empty-state ${className}`}>
      <div className="empty-state-icon">?</div>
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
      <div className="empty-state-icon">??</div>
      <p className="muted">{children}</p>
    </div>
  );
}

export function ErrorState({ children, className = "" }: StateProps) {
  return (
    <div className={`empty-state ${className}`}>
      <div className="empty-state-icon">??</div>
      <p className="error-text">{children}</p>
    </div>
  );
}
