package main

import (
	"strings"

	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/disk"
	"github.com/shirou/gopsutil/v4/mem"
	gnet "github.com/shirou/gopsutil/v4/net"
)

func collectSystemMetrics(workDir, collectedAt string, dimensions map[string]interface{}) []metricPayload {
	metrics := make([]metricPayload, 0, 10)

	if percentages, err := cpu.Percent(0, false); err == nil && len(percentages) > 0 {
		metrics = append(metrics, metricPayload{
			Code:        "agent.os.cpu.percent",
			Value:       percentages[0],
			Unit:        "percent",
			Dimensions:  dimensions,
			CollectedAt: collectedAt,
		})
	}

	if vm, err := mem.VirtualMemory(); err == nil {
		metrics = append(metrics,
			metricPayload{Code: "agent.os.memory.used_bytes", Value: float64(vm.Used), Unit: "bytes", Dimensions: dimensions, CollectedAt: collectedAt},
			metricPayload{Code: "agent.os.memory.total_bytes", Value: float64(vm.Total), Unit: "bytes", Dimensions: dimensions, CollectedAt: collectedAt},
			metricPayload{Code: "agent.os.memory.used_percent", Value: vm.UsedPercent, Unit: "percent", Dimensions: dimensions, CollectedAt: collectedAt},
		)
	}

	diskPath := strings.TrimSpace(workDir)
	if diskPath == "" {
		diskPath = "."
	}
	if usage, err := disk.Usage(diskPath); err == nil {
		metrics = append(metrics,
			metricPayload{Code: "agent.os.disk.used_bytes", Value: float64(usage.Used), Unit: "bytes", Dimensions: dimensions, CollectedAt: collectedAt},
			metricPayload{Code: "agent.os.disk.total_bytes", Value: float64(usage.Total), Unit: "bytes", Dimensions: dimensions, CollectedAt: collectedAt},
			metricPayload{Code: "agent.os.disk.used_percent", Value: usage.UsedPercent, Unit: "percent", Dimensions: dimensions, CollectedAt: collectedAt},
		)
	}

	if counters, err := gnet.IOCounters(false); err == nil && len(counters) > 0 {
		metrics = append(metrics,
			metricPayload{Code: "agent.os.network.bytes_sent", Value: float64(counters[0].BytesSent), Unit: "bytes", Dimensions: dimensions, CollectedAt: collectedAt},
			metricPayload{Code: "agent.os.network.bytes_recv", Value: float64(counters[0].BytesRecv), Unit: "bytes", Dimensions: dimensions, CollectedAt: collectedAt},
		)
	}

	return metrics
}
