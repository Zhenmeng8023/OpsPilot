import type { ReactNode } from "react";

type PagePanelProps = {
  title?: string;
  subtitle?: string;
  actions?: ReactNode;
  children: ReactNode;
  className?: string;
};

export function PagePanel({ title, subtitle, actions, children, className = "" }: PagePanelProps) {
  return (
    <section className={["panel", className].filter(Boolean).join(" ")}>
      {title || subtitle || actions ? (
        <div className="panel-title">
          <h3>{title}</h3>
          {subtitle ? <span>{subtitle}</span> : null}
          {actions ? <div className="panel-title-actions">{actions}</div> : null}
        </div>
      ) : null}
      {children}
    </section>
  );
}
