import type { ReactNode } from "react";

export function EmptyState({ children }: { children: ReactNode }) {
  return <p className="empty-state">{children}</p>;
}

export function LoadingState({ children }: { children: ReactNode }) {
  return <p className="empty-state loading-state">{children}</p>;
}

export function ErrorState({ children }: { children: ReactNode }) {
  return <p className="form-error">{children}</p>;
}
