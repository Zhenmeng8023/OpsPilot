import type { ReactNode } from "react";

type FormGridProps = {
  children: ReactNode;
  columns?: 1 | 2 | 3;
  className?: string;
};

export function FormGrid({ children, columns = 2, className = "" }: FormGridProps) {
  const gridStyle = {
    gridTemplateColumns: `repeat(${columns}, minmax(0, 1fr))`
  };
  return (
    <div className={["form-grid", className].filter(Boolean).join(" ")} style={gridStyle}>
      {children}
    </div>
  );
}
