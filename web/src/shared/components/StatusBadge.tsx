export function StatusBadge({ status }: { status: string }) {
  const normalized = status === "ok" ? "online" : status;
  return <span className={`status-badge status-${normalized}`}>{status}</span>;
}
