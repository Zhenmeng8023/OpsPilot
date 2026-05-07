export const metricCatalog = [
  { code: "agent.running_tasks", labelKey: "metrics.runningTasks" },
  { code: "agent.cpu.logical", labelKey: "metrics.logicalCpu" },
  { code: "agent.os.cpu.percent", labelKey: "metrics.cpuPercent" },
  { code: "agent.os.memory.used_bytes", labelKey: "metrics.memoryUsedBytes" },
  { code: "agent.os.memory.total_bytes", labelKey: "metrics.memoryTotalBytes" },
  { code: "agent.os.memory.used_percent", labelKey: "metrics.memoryPercent" },
  { code: "agent.os.disk.used_bytes", labelKey: "metrics.diskUsedBytes" },
  { code: "agent.os.disk.total_bytes", labelKey: "metrics.diskTotalBytes" },
  { code: "agent.os.disk.used_percent", labelKey: "metrics.diskPercent" },
  { code: "agent.os.network.bytes_sent", labelKey: "metrics.networkSent" },
  { code: "agent.os.network.bytes_recv", labelKey: "metrics.networkRecv" },
  { code: "agent.runtime.goroutines", labelKey: "metrics.goroutines" },
  { code: "agent.runtime.alloc_bytes", labelKey: "metrics.allocBytes" },
  { code: "agent.runtime.sys_bytes", labelKey: "metrics.sysBytes" }
] as const;

export const overviewMetrics = ["agent.os.cpu.percent", "agent.os.memory.used_percent", "agent.os.disk.used_percent", "agent.running_tasks"] as const;

export const quickTrendMetrics = ["agent.os.cpu.percent", "agent.os.memory.used_percent", "agent.os.disk.used_percent", "agent.runtime.goroutines"] as const;

export const alertEventTypes = ["firing", "acknowledged", "silenced", "unsilenced", "resolved", "cooldown_suppressed"] as const;

export const trendRangeOptions = [1, 6, 24, 72] as const;

export const alertHistoryRangeOptions = [6, 24, 72, 168] as const;

export const alertRuleTemplates = [
  {
    key: "cpu-hot",
    titleKey: "metrics.templateCpuTitle",
    descriptionKey: "metrics.templateCpuDescription",
    payload: { name: "CPU Hot", metricCode: "agent.os.cpu.percent", operator: ">=", threshold: 85, durationSeconds: 180, cooldownSeconds: 300, severity: "warning" }
  },
  {
    key: "memory-pressure",
    titleKey: "metrics.templateMemoryTitle",
    descriptionKey: "metrics.templateMemoryDescription",
    payload: { name: "Memory Pressure", metricCode: "agent.os.memory.used_percent", operator: ">=", threshold: 90, durationSeconds: 180, cooldownSeconds: 300, severity: "critical" }
  },
  {
    key: "disk-pressure",
    titleKey: "metrics.templateDiskTitle",
    descriptionKey: "metrics.templateDiskDescription",
    payload: { name: "Disk Pressure", metricCode: "agent.os.disk.used_percent", operator: ">=", threshold: 85, durationSeconds: 300, cooldownSeconds: 900, severity: "warning" }
  },
  {
    key: "goroutine-leak",
    titleKey: "metrics.templateGoroutinesTitle",
    descriptionKey: "metrics.templateGoroutinesDescription",
    payload: { name: "Goroutine Growth", metricCode: "agent.runtime.goroutines", operator: ">=", threshold: 400, durationSeconds: 120, cooldownSeconds: 300, severity: "warning" }
  }
] as const;
