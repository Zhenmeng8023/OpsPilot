package notifications

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/tls"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"mime"
	"net"
	"net/http"
	"net/smtp"
	"net/textproto"
	"strconv"
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
	WorkspaceID      uint64
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
			`SELECT nd.id, n.workspace_id, nd.attempts, nc.channel_type, nc.config, n.uid AS notification_uid,
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
		if err := s.executeDeliveryAttempt(ctx, row, false); err != nil {
			result.Failed++
			continue
		}
		result.Sent++
	}
	return result, nil
}

func (s *Service) sendDelivery(ctx context.Context, row pendingDelivery) error {
	row = s.renderDeliveryTemplate(ctx, row)
	switch row.ChannelType {
	case "site":
		return nil
	case "email":
		return s.sendEmailDelivery(ctx, row)
	case "webhook", "dingtalk", "wechat", "slack":
		return s.postDelivery(ctx, row)
	default:
		return fmt.Errorf("unsupported notification channel type %q", row.ChannelType)
	}
}

func (s *Service) sendEmailDelivery(ctx context.Context, row pendingDelivery) error {
	host := strings.TrimSpace(s.cfg.Notify.SMTPHost)
	username := strings.TrimSpace(s.cfg.Notify.SMTPUsername)
	password := strings.TrimSpace(s.cfg.Notify.SMTPPassword)
	from := strings.TrimSpace(s.cfg.Notify.SMTPFrom)
	if host == "" || username == "" || password == "" || from == "" {
		return errors.New("email delivery is not configured")
	}
	to := deliveryEmail(s.channelConfigMap(row.Config))
	if to == "" {
		return errors.New("notification email recipient is required")
	}
	port := s.cfg.Notify.SMTPPort
	if port <= 0 {
		port = 587
	}
	securityMode := strings.TrimSpace(s.cfg.Notify.SMTPSecurity)
	if securityMode == "" {
		securityMode = "starttls"
	}

	timeout := s.cfg.Notify.HTTPTimeout
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	dialer := &net.Dialer{Timeout: timeout}
	addr := net.JoinHostPort(host, strconv.Itoa(port))
	conn, err := dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		return err
	}

	var (
		client  *smtp.Client
		tlsConn *tls.Conn
	)
	if securityMode == "tls" {
		tlsConn = tls.Client(conn, &tls.Config{ServerName: host, MinVersion: tls.VersionTLS12})
		if err := tlsConn.HandshakeContext(ctx); err != nil {
			_ = conn.Close()
			return err
		}
		client, err = smtp.NewClient(tlsConn, host)
	} else {
		client, err = smtp.NewClient(conn, host)
	}
	if err != nil {
		_ = conn.Close()
		return err
	}
	defer client.Close()

	if securityMode == "starttls" {
		ok, _ := client.Extension("STARTTLS")
		if !ok {
			return errors.New("smtp server does not support STARTTLS")
		}
		if err := client.StartTLS(&tls.Config{ServerName: host, MinVersion: tls.VersionTLS12}); err != nil {
			return err
		}
	}
	if ok, _ := client.Extension("AUTH"); ok {
		if err := client.Auth(smtp.PlainAuth("", username, password, host)); err != nil {
			return err
		}
	}
	if err := client.Mail(from); err != nil {
		return err
	}
	if err := client.Rcpt(to); err != nil {
		return err
	}
	writer, err := client.Data()
	if err != nil {
		return err
	}
	message := buildEmailMessage(from, to, row)
	if _, err := writer.Write([]byte(message)); err != nil {
		_ = writer.Close()
		return err
	}
	if err := writer.Close(); err != nil {
		return err
	}
	return client.Quit()
}

func (s *Service) postDelivery(ctx context.Context, row pendingDelivery) error {
	configMap := s.channelConfigMap(row.Config)
	url := deliveryURL(configMap)
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
	req.Header.Set("X-OpsPilot-Delivery-Id", strconv.FormatUint(row.ID, 10))
	req.Header.Set("X-OpsPilot-Notification-Id", row.NotificationUID)
	if secret := deliverySigningSecret(configMap); secret != "" {
		timestamp := strconv.FormatInt(time.Now().Unix(), 10)
		req.Header.Set("X-OpsPilot-Timestamp", timestamp)
		req.Header.Set("X-OpsPilot-Signature", buildDeliverySignature(secret, timestamp, body))
	}
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

func deliveryURL(cfg map[string]interface{}) string {
	for _, key := range []string{"url", "webhookUrl", "webhook_url"} {
		if value, ok := cfg[key].(string); ok && strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func deliveryEmail(cfg map[string]interface{}) string {
	for _, key := range []string{"email", "to", "address", "recipient"} {
		if value, ok := cfg[key].(string); ok && strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func deliverySigningSecret(cfg map[string]interface{}) string {
	for _, key := range []string{"signingSecret", "signing_secret", "secret"} {
		if value, ok := cfg[key].(string); ok && strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func buildDeliverySignature(secret string, timestamp string, body []byte) string {
	mac := hmac.New(sha256.New, []byte(strings.TrimSpace(secret)))
	_, _ = mac.Write([]byte(strings.TrimSpace(timestamp)))
	_, _ = mac.Write([]byte("."))
	_, _ = mac.Write(body)
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
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

func buildEmailMessage(from, to string, row pendingDelivery) string {
	subject := row.Title
	body := row.Title
	if row.Content.Valid && row.Content.String != "" {
		body += "\n\n" + row.Content.String
	}
	headers := []string{
		"From: " + from,
		"To: " + to,
		"Subject: " + mime.QEncoding.Encode("utf-8", subject),
		"MIME-Version: 1.0",
		"Content-Type: text/plain; charset=UTF-8",
		"Content-Transfer-Encoding: 8bit",
	}
	return strings.Join(headers, "\r\n") + "\r\n\r\n" + textproto.TrimString(body) + "\r\n"
}

func limitString(value string, maxLen int) string {
	if len(value) <= maxLen {
		return value
	}
	return value[:maxLen]
}
