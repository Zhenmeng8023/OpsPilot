package schedules

import (
	"context"
	"log/slog"
	"time"
)

func StartScheduler(ctx context.Context, log *slog.Logger, service *Service, interval time.Duration) {
	if interval <= 0 {
		interval = 30 * time.Second
	}
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case now := <-ticker.C:
				result, err := service.FireDue(ctx, now, 50)
				if err != nil {
					log.Warn("schedule scan failed", "error", err)
					continue
				}
				if result.Fired > 0 || result.Failed > 0 {
					log.Info("schedule scan completed", "fired", result.Fired, "failed", result.Failed)
				}
			}
		}
	}()
}
