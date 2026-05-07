package alerts

import (
	"context"
	"log/slog"
	"time"
)

func StartScanner(ctx context.Context, log *slog.Logger, service *Service, interval time.Duration) {
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
			case <-ticker.C:
				result, err := service.Evaluate(ctx)
				if err != nil {
					log.Warn("alert evaluation failed", "error", err)
					continue
				}
				if result.Fired > 0 || result.Resolved > 0 {
					log.Info("alert evaluation completed", "fired", result.Fired, "resolved", result.Resolved)
				}
			}
		}
	}()
}
