import type { ReactNode } from "react";

export function FilterToolbar({ children, compact = false }: { children: ReactNode; compact?: boolean }) {
  return <div className={`toolbar-row${compact ? " compact-toolbar" : ""}`}>{children}</div>;
}
