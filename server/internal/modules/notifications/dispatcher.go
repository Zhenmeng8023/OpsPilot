package notifications

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"gorm.io/gorm"
)

type DispatchResult struct {
	Sent   int
	Failed int
}

type pendingDelivery struct {
	ID               uint64
	Attempts         int
	ChannelType      string
	Config           sql.NullString
	NotificationUID  string
	Title            string
	Content          sql.NullString
	Category         string
	Severity         string
	ResourceType     sql.NullString
	ResourceID       sql.NullInt64
	NotificationTime string
}

func StartDispatcher(ctx context.Context, log *slog.Logger, service *Service, interval time.Duration) {
	if interval <= 0 {
		interval = 15 * time.Second
	}
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				result, err := service.DispatchPending(ctx, 20)
				if err != nil {
					log.Warn("notification dispatch failed", "error", err)
					continue
				}
				if result.Sent > 0 || result.Failed > 0 {
					log.Info("notification dispatch completed", "sent", result.Sent, "failed", result.Failed)
				}
			}
		}
	}()
}

func (s *Service) DispatchPending(ctx context.Context, limit int) (DispatchResult, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	var rows []pendingDelivery
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.WithContext(ctx).Raw(
			`SELECT nd.id, nd.attempts, nc.channel_type, nc.config, n.uid AS notification_uid,
			        n.title, n.content, n.category, n.severity, n.resource_type, n.resource_id,
			        DATE_FORMAT(n.created_at, '%Y-%m-%d %H:%i:%s') AS notification_time
			   FROM notification_deliveries nd
			   JOIN notifications n ON n.id = nd.notification_id
			   JOIN notification_channels nc ON nc.id = nd.channel_id
			  WHERE nd.status = 'pending'
			    AND (nd.next_retry_at IS NULL OR nd.next_retry_at <= NOW(3))
			    AND nc.status = 'active'
			  ORDER BY nd.created_at ASC, nd.id ASC
			  LIMIT ?
			  FOR UPDATE SKIP LOCKED`,
			limit,
		).Scan(&rows).Error; err != nil {
			return err
		}
		for _, row := range rows {
			if err := tx.WithContext(ctx).Exec(
				"UPDATE notification_deliveries SET status = 'sending', attempts = attempts + 1, updated_at = NOW(3) WHERE id = ?",
				row.ID,
			).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return DispatchResult{}, err
	}

	result := DispatchResult{}
	for _, row := range rows {
		if err := s.sendDelivery(ctx, row); err != nil {
			result.Failed++
			if updateErr := s.markDeliveryFailed(ctx, row, err); updateErr != nil {
				return result, updateErr
			}
			continue
		}
		result.Sent++
		if err := s.db.WithContext(ctx).Exec(
			"UPDATE notification_deliveries SET status = 'success', delivered_at = NOW(3), error_message = NULL, updated_at = NOW(3) WHERE id = ?",
			row.ID,
		).Error; err != nil {
			return result, err
		}
	}
	return result, nil
}

func (s *Service) sendDelivery(ctx context.Context, row pendingDelivery) error {
	switch row.ChannelType {
	case "site":
		return nil
	case "email":
		return errors.New("email delivery is not configured")
	case "webhook", "dingtalk", "wechat", "slack":
		return s.postDelivery(ctx, row)
	default:
		return fmt.Errorf("unsupported notification channel type %q", row.ChannelType)
	}
}

func (s *Service) postDelivery(ctx context.Context, row pendingDelivery) error {
	url := deliveryURL(row.Config)
	if url == "" {
		return errors.New("notification channel url is required")
	}
	body, err := json.Marshal(channelPayload(row))
	if err != nil {
		return err
	}
	timeout := s.cfg.Notify.HTTPTimeout
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	client := &http.Client{Timeout: timeout}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "OpsPilot-Notification/1.0")
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("notification webhook returned %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}
	return nil
}

func (s *Service) markDeliveryFailed(ctx context.Context, row pendingDelivery, cause error) error {
	attempts := row.Attempts + 1
	message := limitString(cause.Error(), 1024)
	if attempts >= 3 {
		return s.db.WithContext(ctx).Exec(
			"UPDATE notification_deliveries SET status = 'failed', error_message = ?, updated_at = NOW(3) WHERE id = ?",
			message, row.ID,
		).Error
	}
	nextRetryAt := time.Now().Add(time.Duration(attempts) * time.Minute)
	return s.db.WithContext(ctx).Exec(
		"UPDATE notification_deliveries SET status = 'pending', next_retry_at = ?, error_message = ?, updated_at = NOW(3) WHERE id = ?",
		nextRetryAt, message, row.ID,
	).Error
}

func deliveryURL(raw sql.NullString) string {
	if !raw.Valid {
		return ""
	}
	var cfg map[string]interface{}
	if err := json.Unmarshal([]byte(raw.String), &cfg); err != nil {
		return ""
	}
	for _, key := range []string{"url", "webhookUrl", "webhook_url"} {
		if value, ok := cfg[key].(string); ok && strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func channelPayload(row pendingDelivery) interface{} {
	text := row.Title
	if row.Content.Valid && row.Content.String != "" {
		text += "\n" + row.Content.String
	}
	switch row.ChannelType {
	case "slack":
		return map[string]interface{}{"text": text}
	case "dingtalk", "wechat":
		return map[string]interface{}{
			"msgtype": "text",
			"text":    map[string]interface{}{"content": text},
		}
	default:
		return map[string]interface{}{
			"id":           row.NotificationUID,
			"title":        row.Title,
			"content":      row.Content.String,
			"category":     row.Category,
			"severity":     row.Severity,
			"resourceType": row.ResourceType.String,
			"resourceId":   row.ResourceID.Int64,
			"createdAt":    row.NotificationTime,
		}
	}
}

func limitString(value string, maxLen int) string {
	if len(value) <= maxLen {
		return value
	}
	return value[:maxLen]
}
