import type { ReactNode } from "react";

export interface TimelineItem {
  id: string | number;
  title: ReactNode;
  time?: ReactNode;
  description?: ReactNode;
  tone?: "default" | "success" | "warning" | "danger";
}

export function Timeline({ items }: { items: TimelineItem[] }) {
  return (
    <ol className="timeline-list">
      {items.map((item) => (
        <li className={`timeline-item timeline-${item.tone ?? "default"}`} key={item.id}>
          <span className="timeline-dot" />
          <div>
            <div className="timeline-heading">
              <strong>{item.title}</strong>
              {item.time ? <small>{item.time}</small> : null}
            </div>
            {item.description ? <p>{item.description}</p> : null}
          </div>
        </li>
      ))}
    </ol>
  );
}
