import type { ReactNode } from "react";

type SectionToolbarProps = {
  children: ReactNode;
  className?: string;
};

export function SectionToolbar({ children, className = "" }: SectionToolbarProps) {
  return <div className={["toolbar-row", className].filter(Boolean).join(" ")}>{children}</div>;
}
