package metrics

import (
	"context"
	"log/slog"
	"time"
)

func StartMaintenance(ctx context.Context, log *slog.Logger, service *Service, rollupInterval, retentionInterval time.Duration) {
	if rollupInterval <= 0 {
		rollupInterval = 5 * time.Minute
	}
	if retentionInterval <= 0 {
		retentionInterval = 24 * time.Hour
	}
	go runRollupScanner(ctx, log, service, rollupInterval)
	go runRetentionScanner(ctx, log, service, retentionInterval)
}

func runRollupScanner(ctx context.Context, log *slog.Logger, service *Service, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			for _, rollup := range []RollupInput{{Interval: "5m", Hours: 24}, {Interval: "1h", Hours: 24 * 30}} {
				result, appErr := service.RunRollup(ctx, rollup)
				if appErr != nil {
					log.Warn("metric rollup failed", "interval", rollup.Interval, "error", appErr)
					continue
				}
				if result.Matched > 0 || result.Upserted > 0 {
					log.Info("metric rollup completed", "interval", result.Interval, "matched", result.Matched, "upserted", result.Upserted)
				}
			}
		}
	}
}

func runRetentionScanner(ctx context.Context, log *slog.Logger, service *Service, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			result, appErr := service.RunRetention(ctx, RetentionInput{})
			if appErr != nil {
				log.Warn("metric retention failed", "error", appErr)
				continue
			}
			if result.DetailDeleted > 0 || result.RollupDeleted > 0 {
				log.Info("metric retention completed", "detail_deleted", result.DetailDeleted, "rollup_deleted", result.RollupDeleted)
			}
		}
	}
}
