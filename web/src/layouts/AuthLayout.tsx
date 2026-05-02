import type { ReactNode } from "react";

export function AuthLayout({ children }: { children: ReactNode }) {
  return (
    <main className="auth-shell">
      <section className="auth-hero">
        <p className="eyebrow">OpsPilot Control Plane</p>
        <h1>自动化运维与任务调度平台</h1>
        <p>
          面向 Agent 接入、脚本模板、任务执行、实时日志和审计追踪的轻量级工程平台。
        </p>
        <div className="hero-grid">
          <span>Agent</span>
          <span>Task Runs</span>
          <span>SSE Logs</span>
          <span>Audit</span>
        </div>
      </section>
      <section className="auth-panel">{children}</section>
    </main>
  );
}
