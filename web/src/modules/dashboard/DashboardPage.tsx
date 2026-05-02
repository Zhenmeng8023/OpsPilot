import { useQuery } from "@tanstack/react-query";

import { request } from "../../api/request";
import type { HealthData } from "../../api/types";
import { StatusBadge } from "../../shared/components/StatusBadge";

const cards = [
  { label: "在线 Agent", value: "0", hint: "T2 阶段接入注册和心跳" },
  { label: "今日任务", value: "0", hint: "T3 阶段接入任务执行" },
  { label: "失败任务", value: "0", hint: "状态机已在方案中定义" },
  { label: "最近告警", value: "0", hint: "T7 阶段接入监控告警" }
];

export function DashboardPage() {
  const healthQuery = useQuery({
    queryKey: ["health"],
    queryFn: () => request<HealthData>("/health", { skipAuth: true })
  });

  return (
    <main className="page">
      <section className="page-heading">
        <div>
          <p className="eyebrow">Overview</p>
          <h1>系统运行概览</h1>
        </div>
        <StatusBadge status={healthQuery.data?.status ?? "checking"} />
      </section>

      <section className="metrics-grid">
        {cards.map((card) => (
          <article className="metric-card" key={card.label}>
            <span>{card.label}</span>
            <strong>{card.value}</strong>
            <small>{card.hint}</small>
          </article>
        ))}
      </section>

      <section className="panel-grid">
        <article className="panel">
          <div className="panel-title">
            <h3>API 健康检查</h3>
            <span>{healthQuery.isFetching ? "刷新中" : "实时"}</span>
          </div>
          {healthQuery.isError ? (
            <p className="error-text">无法连接后端，请确认 `go run ./cmd/api` 已启动。</p>
          ) : (
            <dl className="health-list">
              <div>
                <dt>服务</dt>
                <dd>{healthQuery.data?.service ?? "-"}</dd>
              </div>
              <div>
                <dt>环境</dt>
                <dd>{healthQuery.data?.env ?? "-"}</dd>
              </div>
              <div>
                <dt>版本</dt>
                <dd>{healthQuery.data?.version ?? "-"}</dd>
              </div>
              <div>
                <dt>时间</dt>
                <dd>{healthQuery.data?.time ?? "-"}</dd>
              </div>
            </dl>
          )}
        </article>

        <article className="panel terminal-panel">
          <div className="panel-title">
            <h3>后续开发主线</h3>
            <span>MVP</span>
          </div>
          <pre>{`login -> agent register -> script template
task create -> run -> stream logs -> result audit`}</pre>
        </article>
      </section>
    </main>
  );
}
