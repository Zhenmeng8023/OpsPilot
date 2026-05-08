package notifications

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"gorm.io/gorm"

	"opspilot/server/internal/config"
	"opspilot/server/internal/shared/apperror"
	"opspilot/server/internal/shared/audit"
	"opspilot/server/internal/shared/uid"
)

type Service struct {
	db  *gorm.DB
	cfg config.Config
}

type AuditContext struct {
	ActorUID      string
	IP            string
	UserAgent     string
	TraceID       string
	RequestMethod string
	RequestPath   string
}

type CreateChannelInput struct {
	Name        string
	ChannelType string
	Config      map[string]interface{}
	Audit       AuditContext
}

type UpdateChannelInput struct {
	ID          string
	Name        string
	ChannelType string
	Config      map[string]interface{}
	Audit       AuditContext
}

type CreateTemplateInput struct {
	Name            string
	Category        string
	ChannelType     string
	TitleTemplate   string
	ContentTemplate string
	Status          string
	Audit           AuditContext
}

type UpdateTemplateInput struct {
	ID              string
	Name            string
	Category        string
	ChannelType     string
	TitleTemplate   string
	ContentTemplate string
	Status          string
	Audit           AuditContext
}

type ChannelTestResult struct {
	ChannelID      string `json:"channelId"`
	NotificationID string `json:"notificationId"`
	DeliveryID     uint64 `json:"deliveryId"`
	Status         string `json:"status"`
	ErrorMessage   string `json:"errorMessage,omitempty"`
}

type ListDeliveriesInput struct {
	Status         string
	ChannelID      string
	NotificationID string
}

type BulkRetryDeliveriesInput struct {
	Status         string
	ChannelID      string
	NotificationID string
	Limit          int
}

type BulkRetryDeliveriesResult struct {
	MatchedCount uint `json:"matchedCount"`
	RetriedCount uint `json:"retriedCount"`
}

type ChannelSummary struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	ChannelType string `json:"channelType"`
	Target      string `json:"targetSummary,omitempty"`
	Status      string `json:"status"`
	CreatedBy   string `json:"createdBy,omitempty"`
	CreatedAt   string `json:"createdAt"`
}

type TemplateSummary struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	Category        string `json:"category"`
	ChannelType     string `json:"channelType"`
	TitleTemplate   string `json:"titleTemplate"`
	ContentTemplate string `json:"contentTemplate,omitempty"`
	Status          string `json:"status"`
	CreatedBy       string `json:"createdBy,omitempty"`
	CreatedAt       string `json:"createdAt"`
	UpdatedAt       string `json:"updatedAt"`
}

type NotificationSummary struct {
	ID           string `json:"id"`
	Title        string `json:"title"`
	Content      string `json:"content,omitempty"`
	Category     string `json:"category"`
	Severity     string `json:"severity"`
	ResourceType string `json:"resourceType,omitempty"`
	ResourceID   uint64 `json:"resourceId,omitempty"`
	ReadAt       string `json:"readAt,omitempty"`
	CreatedAt    string `json:"createdAt"`
}

type DeliverySummary struct {
	ID             uint64 `json:"id"`
	NotificationID string `json:"notificationId,omitempty"`
	Title          string `json:"title,omitempty"`
	ChannelID      string `json:"channelId,omitempty"`
	ChannelName    string `json:"channelName,omitempty"`
	ChannelType    string `json:"channelType,omitempty"`
	Status         string `json:"status"`
	Attempts       int    `json:"attempts"`
	NextRetryAt    string `json:"nextRetryAt,omitempty"`
	DeliveredAt    string `json:"deliveredAt,omitempty"`
	ErrorMessage   string `json:"errorMessage,omitempty"`
	CreatedAt      string `json:"createdAt"`
	UpdatedAt      string `json:"updatedAt"`
}

type workspaceRecord struct {
	ID uint64
}

type userRecord struct {
	ID uint64
}

type channelRecord struct {
	ID          uint64
	UID         string
	Name        string
	ChannelType string
	Config      sql.NullString
	Status      string
	CreatedBy   sql.NullString
	CreatedAt   string
}

type channelDispatchRecord struct {
	ID          uint64
	UID         string
	Name        string
	ChannelType string
	Config      sql.NullString
	Status      string
}

type templateRecord struct {
	ID              uint64
	UID             string
	Name            string
	Category        string
	ChannelType     string
	TitleTemplate   string
	ContentTemplate sql.NullString
	Status          string
	CreatedBy       sql.NullString
	CreatedAt       string
	UpdatedAt       string
}

type notificationRecord struct {
	ID           uint64
	UID          string
	Title        string
	Content      sql.NullString
	Category     string
	Severity     string
	ResourceType sql.NullString
	ResourceID   sql.NullInt64
	ReadAt       sql.NullString
	CreatedAt    string
}

type deliveryRecord struct {
	ID              uint64
	NotificationUID sql.NullString
	Title           sql.NullString
	ChannelUID      sql.NullString
	ChannelName     sql.NullString
	ChannelType     sql.NullString
	Status          string
	Attempts        int
	NextRetryAt     sql.NullString
	DeliveredAt     sql.NullString
	ErrorMessage    sql.NullString
	CreatedAt       string
	UpdatedAt       string
}

func NewService(db *gorm.DB, cfg config.Config) *Service {
	return &Service{db: db, cfg: cfg}
}

func (s *Service) ListChannels(ctx context.Context) ([]ChannelSummary, *apperror.Error) {
	workspace, appErr := s.workspace(ctx)
	if appErr != nil {
		return nil, appErr
	}
	var rows []channelRecord
	if err := s.db.WithContext(ctx).Raw(
		`SELECT nc.id, nc.uid, nc.name, nc.channel_type, nc.config, nc.status, u.username AS created_by,
		        DATE_FORMAT(nc.created_at, '%Y-%m-%d %H:%i:%s') AS created_at
		   FROM notification_channels nc
		   LEFT JOIN users u ON u.id = nc.created_by
		  WHERE nc.workspace_id = ? AND nc.deleted_at IS NULL
		  ORDER BY nc.created_at DESC`,
		workspace.ID,
	).Scan(&rows).Error; err != nil {
		return nil, apperror.Wrap(http.StatusInternalServerError, 500801, "list notification channels failed", err)
	}
	out := make([]ChannelSummary, 0, len(rows))
	for _, row := range rows {
		out = append(out, channelSummary(row, s.channelConfigMap(row.Config)))
	}
	return out, nil
}

func (s *Service) ListTemplates(ctx context.Context) ([]TemplateSummary, *apperror.Error) {
	workspace, appErr := s.workspace(ctx)
	if appErr != nil {
		return nil, appErr
	}
	var rows []templateRecord
	if err := s.db.WithContext(ctx).Raw(
		`SELECT nt.id, nt.uid, nt.name, nt.category, nt.channel_type, nt.title_template, nt.content_template,
		        nt.status, u.username AS created_by,
		        DATE_FORMAT(nt.created_at, '%Y-%m-%d %H:%i:%s') AS created_at,
		        DATE_FORMAT(nt.updated_at, '%Y-%m-%d %H:%i:%s') AS updated_at
		   FROM notification_templates nt
		   LEFT JOIN users u ON u.id = nt.created_by
		  WHERE nt.workspace_id = ? AND nt.deleted_at IS NULL
		  ORDER BY nt.category ASC, nt.channel_type ASC, nt.created_at DESC`,
		workspace.ID,
	).Scan(&rows).Error; err != nil {
		return nil, apperror.Wrap(http.StatusInternalServerError, 500809, "list notification templates failed", err)
	}
	out := make([]TemplateSummary, 0, len(rows))
	for _, row := range rows {
		out = append(out, templateSummary(row))
	}
	return out, nil
}

func (s *Service) CreateTemplate(ctx context.Context, input CreateTemplateInput) (TemplateSummary, *apperror.Error) {
	normalized, appErr := normalizeTemplateInput(input.Name, input.Category, input.ChannelType, input.TitleTemplate, input.ContentTemplate, input.Status)
	if appErr != nil {
		return TemplateSummary{}, appErr
	}
	var created TemplateSummary
	txErr := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		workspace, err := defaultWorkspace(ctx, tx, s.cfg.Bootstrap.WorkspaceSlug)
		if err != nil {
			return err
		}
		actor, err := userByUID(ctx, tx, input.Audit.ActorUID)
		if err != nil {
			return err
		}
		templateUID, err := uid.New()
		if err != nil {
			return err
		}
		if err := tx.WithContext(ctx).Exec(
			`INSERT INTO notification_templates(uid, workspace_id, name, category, channel_type, title_template, content_template, status, created_by)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			templateUID, workspace.ID, normalized.Name, normalized.Category, normalized.ChannelType, normalized.TitleTemplate, nullString(normalized.ContentTemplate), normalized.Status, nullID(actor.ID),
		).Error; err != nil {
			return err
		}
		row, err := templateByUID(ctx, tx, workspace.ID, templateUID)
		if err != nil {
			return err
		}
		created = templateSummary(row)
		audit.Write(ctx, tx, audit.Event{
			WorkspaceID:   workspace.ID,
			ActorUserID:   nullID(actor.ID),
			Action:        "notification.template.create",
			ResourceType:  "notification_template",
			ResourceID:    nullID(row.ID),
			IP:            input.Audit.IP,
			UserAgent:     input.Audit.UserAgent,
			TraceID:       input.Audit.TraceID,
			RequestMethod: input.Audit.RequestMethod,
			RequestPath:   input.Audit.RequestPath,
			After:         created,
		})
		return nil
	})
	if txErr != nil {
		if strings.Contains(strings.ToLower(txErr.Error()), "duplicate") {
			return TemplateSummary{}, apperror.New(http.StatusConflict, 409802, "notification template already exists")
		}
		return TemplateSummary{}, apperror.Wrap(http.StatusInternalServerError, 500810, "create notification template failed", txErr)
	}
	return created, nil
}

func (s *Service) UpdateTemplate(ctx context.Context, input UpdateTemplateInput) (TemplateSummary, *apperror.Error) {
	id := strings.TrimSpace(input.ID)
	if id == "" {
		return TemplateSummary{}, apperror.New(http.StatusBadRequest, 400809, "notification template id is required")
	}
	normalized, appErr := normalizeTemplateInput(input.Name, input.Category, input.ChannelType, input.TitleTemplate, input.ContentTemplate, input.Status)
	if appErr != nil {
		return TemplateSummary{}, appErr
	}
	var updated TemplateSummary
	txErr := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		workspace, err := defaultWorkspace(ctx, tx, s.cfg.Bootstrap.WorkspaceSlug)
		if err != nil {
			return err
		}
		actor, err := userByUID(ctx, tx, input.Audit.ActorUID)
		if err != nil {
			return err
		}
		before, err := templateByUID(ctx, tx, workspace.ID, id)
		if err != nil {
			return err
		}
		if before.ID == 0 {
			return apperror.New(http.StatusNotFound, 404805, "notification template not found")
		}
		if err := tx.WithContext(ctx).Exec(
			`UPDATE notification_templates
			    SET name = ?, category = ?, channel_type = ?, title_template = ?, content_template = ?, status = ?, updated_at = NOW(3)
			  WHERE id = ?`,
			normalized.Name, normalized.Category, normalized.ChannelType, normalized.TitleTemplate, nullString(normalized.ContentTemplate), normalized.Status, before.ID,
		).Error; err != nil {
			return err
		}
		after, err := templateByUID(ctx, tx, workspace.ID, id)
		if err != nil {
			return err
		}
		updated = templateSummary(after)
		audit.Write(ctx, tx, audit.Event{
			WorkspaceID:   workspace.ID,
			ActorUserID:   nullID(actor.ID),
			Action:        "notification.template.update",
			ResourceType:  "notification_template",
			ResourceID:    nullID(before.ID),
			IP:            input.Audit.IP,
			UserAgent:     input.Audit.UserAgent,
			TraceID:       input.Audit.TraceID,
			RequestMethod: input.Audit.RequestMethod,
			RequestPath:   input.Audit.RequestPath,
			Before:        templateSummary(before),
			After:         updated,
		})
		return nil
	})
	if txErr != nil {
		if appErr, ok := txErr.(*apperror.Error); ok {
			return TemplateSummary{}, appErr
		}
		if strings.Contains(strings.ToLower(txErr.Error()), "duplicate") {
			return TemplateSummary{}, apperror.New(http.StatusConflict, 409802, "notification template already exists")
		}
		return TemplateSummary{}, apperror.Wrap(http.StatusInternalServerError, 500811, "update notification template failed", txErr)
	}
	return updated, nil
}

func (s *Service) CreateChannel(ctx context.Context, input CreateChannelInput) (ChannelSummary, *apperror.Error) {
	name := strings.TrimSpace(input.Name)
	channelType := strings.TrimSpace(input.ChannelType)
	if channelType == "" {
		channelType = "site"
	}
	if name == "" {
		return ChannelSummary{}, apperror.New(http.StatusBadRequest, 400801, "notification channel name is required")
	}
	if !validChannelType(channelType) {
		return ChannelSummary{}, apperror.New(http.StatusBadRequest, 400802, "unsupported notification channel type")
	}
	var created ChannelSummary
	txErr := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		workspace, err := defaultWorkspace(ctx, tx, s.cfg.Bootstrap.WorkspaceSlug)
		if err != nil {
			return err
		}
		actor, err := userByUID(ctx, tx, input.Audit.ActorUID)
		if err != nil {
			return err
		}
		configValue, err := s.encryptChannelConfig(input.Config)
		if err != nil {
			return err
		}
		channelUID, err := uid.New()
		if err != nil {
			return err
		}
		if err := tx.WithContext(ctx).Exec(
			`INSERT INTO notification_channels(uid, workspace_id, name, channel_type, config, status, created_by)
			 VALUES (?, ?, ?, ?, ?, 'active', ?)`,
			channelUID, workspace.ID, name, channelType, configValue, nullID(actor.ID),
		).Error; err != nil {
			return err
		}
		row, err := channelByUID(ctx, tx, workspace.ID, channelUID)
		if err != nil {
			return err
		}
		created = channelSummary(row, s.channelConfigMap(row.Config))
		audit.Write(ctx, tx, audit.Event{
			WorkspaceID:   workspace.ID,
			ActorUserID:   nullID(actor.ID),
			Action:        "notification.channel.create",
			ResourceType:  "notification_channel",
			ResourceID:    nullID(row.ID),
			IP:            input.Audit.IP,
			UserAgent:     input.Audit.UserAgent,
			TraceID:       input.Audit.TraceID,
			RequestMethod: input.Audit.RequestMethod,
			RequestPath:   input.Audit.RequestPath,
			After:         created,
		})
		return nil
	})
	if txErr != nil {
		if strings.Contains(strings.ToLower(txErr.Error()), "duplicate") {
			return ChannelSummary{}, apperror.New(http.StatusConflict, 409801, "notification channel already exists")
		}
		return ChannelSummary{}, apperror.Wrap(http.StatusInternalServerError, 500802, "create notification channel failed", txErr)
	}
	return created, nil
}

func (s *Service) UpdateChannel(ctx context.Context, input UpdateChannelInput) (ChannelSummary, *apperror.Error) {
	id := strings.TrimSpace(input.ID)
	name := strings.TrimSpace(input.Name)
	channelType := strings.TrimSpace(input.ChannelType)
	if id == "" {
		return ChannelSummary{}, apperror.New(http.StatusBadRequest, 400807, "notification channel id is required")
	}
	if name == "" {
		return ChannelSummary{}, apperror.New(http.StatusBadRequest, 400801, "notification channel name is required")
	}
	if channelType == "" {
		channelType = "site"
	}
	if !validChannelType(channelType) {
		return ChannelSummary{}, apperror.New(http.StatusBadRequest, 400802, "unsupported notification channel type")
	}
	var updated ChannelSummary
	txErr := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		workspace, err := defaultWorkspace(ctx, tx, s.cfg.Bootstrap.WorkspaceSlug)
		if err != nil {
			return err
		}
		actor, err := userByUID(ctx, tx, input.Audit.ActorUID)
		if err != nil {
			return err
		}
		before, err := channelByUID(ctx, tx, workspace.ID, id)
		if err != nil {
			return err
		}
		if before.ID == 0 {
			return apperror.New(http.StatusNotFound, 404804, "notification channel not found")
		}
		mergedConfig := mergeChannelConfig(s.channelConfigMap(before.Config), input.Config, channelType)
		configValue, err := s.encryptChannelConfig(mergedConfig)
		if err != nil {
			return err
		}
		if err := tx.WithContext(ctx).Exec(
			`UPDATE notification_channels
			    SET name = ?, channel_type = ?, config = ?, updated_at = NOW(3)
			  WHERE id = ?`,
			name, channelType, configValue, before.ID,
		).Error; err != nil {
			return err
		}
		after, err := channelByUID(ctx, tx, workspace.ID, id)
		if err != nil {
			return err
		}
		updated = channelSummary(after, s.channelConfigMap(after.Config))
		audit.Write(ctx, tx, audit.Event{
			WorkspaceID:   workspace.ID,
			ActorUserID:   nullID(actor.ID),
			Action:        "notification.channel.update",
			ResourceType:  "notification_channel",
			ResourceID:    nullID(before.ID),
			IP:            input.Audit.IP,
			UserAgent:     input.Audit.UserAgent,
			TraceID:       input.Audit.TraceID,
			RequestMethod: input.Audit.RequestMethod,
			RequestPath:   input.Audit.RequestPath,
			Before:        channelSummary(before, s.channelConfigMap(before.Config)),
			After:         updated,
		})
		return nil
	})
	if txErr != nil {
		if appErr, ok := txErr.(*apperror.Error); ok {
			return ChannelSummary{}, appErr
		}
		if strings.Contains(strings.ToLower(txErr.Error()), "duplicate") {
			return ChannelSummary{}, apperror.New(http.StatusConflict, 409801, "notification channel already exists")
		}
		return ChannelSummary{}, apperror.Wrap(http.StatusInternalServerError, 500807, "update notification channel failed", txErr)
	}
	return updated, nil
}

func (s *Service) TestChannel(ctx context.Context, channelUID string, auditCtx AuditContext) (ChannelTestResult, *apperror.Error) {
	channelUID = strings.TrimSpace(channelUID)
	if channelUID == "" {
		return ChannelTestResult{}, apperror.New(http.StatusBadRequest, 400803, "notification channel id is required")
	}

	testTitle := "OpsPilot test notification"
	var (
		result      ChannelTestResult
		delivery    pendingDelivery
		workspaceID uint64
		actorID     uint64
		channelID   uint64
		channelType string
	)

	txErr := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		workspace, err := defaultWorkspace(ctx, tx, s.cfg.Bootstrap.WorkspaceSlug)
		if err != nil {
			return err
		}
		workspaceID = workspace.ID

		actor, err := userByUID(ctx, tx, auditCtx.ActorUID)
		if err != nil {
			return err
		}
		actorID = actor.ID

		channel, err := channelDispatchByUID(ctx, tx, workspace.ID, channelUID)
		if err != nil {
			return err
		}
		if channel.ID == 0 {
			return apperror.New(http.StatusNotFound, 404803, "notification channel not found")
		}
		if channel.Status != "active" {
			return apperror.New(http.StatusBadRequest, 400806, "only active notification channels can be tested")
		}
		channelID = channel.ID
		channelType = channel.ChannelType

		notificationUID, err := uid.New()
		if err != nil {
			return err
		}
		content := limitString("This is a test message generated by OpsPilot notification diagnostics.", 2048)
		if err := tx.WithContext(ctx).Exec(
			`INSERT INTO notifications(uid, workspace_id, title, content, category, severity, resource_type, resource_id)
			 VALUES (?, ?, ?, ?, 'system', 'info', 'notification_channel', ?)`,
			notificationUID, workspace.ID, testTitle, content, channel.ID,
		).Error; err != nil {
			return err
		}

		var notificationID uint64
		if err := tx.WithContext(ctx).Raw(
			"SELECT id FROM notifications WHERE uid = ? LIMIT 1",
			notificationUID,
		).Scan(&notificationID).Error; err != nil {
			return err
		}

		initialStatus := "pending"
		initialAttempts := 0
		deliveredNow := false
		if channel.ChannelType == "site" {
			initialStatus = "success"
			initialAttempts = 1
			deliveredNow = true
		}
		if err := tx.WithContext(ctx).Exec(
			`INSERT INTO notification_deliveries(notification_id, channel_id, status, attempts, delivered_at)
			 VALUES (?, ?, ?, ?, CASE WHEN ? THEN NOW(3) ELSE NULL END)`,
			notificationID, channel.ID, initialStatus, initialAttempts, deliveredNow,
		).Error; err != nil {
			return err
		}

		var deliveryID uint64
		if err := tx.WithContext(ctx).Raw(
			"SELECT id FROM notification_deliveries WHERE notification_id = ? AND channel_id = ? ORDER BY id DESC LIMIT 1",
			notificationID, channel.ID,
		).Scan(&deliveryID).Error; err != nil {
			return err
		}

		result = ChannelTestResult{
			ChannelID:      channel.UID,
			NotificationID: notificationUID,
			DeliveryID:     deliveryID,
			Status:         initialStatus,
		}
		if !deliveredNow {
			delivery = pendingDelivery{
				ID:               deliveryID,
				Attempts:         0,
				ChannelType:      channel.ChannelType,
				Config:           channel.Config,
				NotificationUID:  notificationUID,
				Title:            testTitle,
				Content:          sql.NullString{String: content, Valid: true},
				Category:         "system",
				Severity:         "info",
				ResourceType:     sql.NullString{String: "notification_channel", Valid: true},
				ResourceID:       sql.NullInt64{Int64: int64(channel.ID), Valid: true},
				NotificationTime: nowString(),
			}
		}
		return nil
	})
	if txErr != nil {
		if appErr, ok := txErr.(*apperror.Error); ok {
			return ChannelTestResult{}, appErr
		}
		return ChannelTestResult{}, apperror.Wrap(http.StatusInternalServerError, 500806, "create test notification failed", txErr)
	}

	sendErr := error(nil)
	if delivery.ID != 0 {
		sendErr = s.executeDeliveryAttempt(ctx, delivery, true)
		if sendErr != nil {
			result.Status = "failed"
			result.ErrorMessage = limitString(sendErr.Error(), 1024)
		} else {
			result.Status = "success"
		}
	}

	audit.Write(ctx, s.db, audit.Event{
		WorkspaceID:   workspaceID,
		ActorUserID:   nullID(actorID),
		Action:        "notification.channel.test",
		ResourceType:  "notification_channel",
		ResourceID:    nullID(channelID),
		Result:        auditResult(sendErr),
		IP:            auditCtx.IP,
		UserAgent:     auditCtx.UserAgent,
		TraceID:       auditCtx.TraceID,
		RequestMethod: auditCtx.RequestMethod,
		RequestPath:   auditCtx.RequestPath,
		After:         result,
		Metadata: map[string]interface{}{
			"channelType": channelType,
		},
	})

	if sendErr != nil {
		return result, apperror.Wrap(http.StatusBadGateway, 502801, "test notification send failed", sendErr)
	}
	return result, nil
}

func (s *Service) ListNotifications(ctx context.Context, unreadOnly bool) ([]NotificationSummary, *apperror.Error) {
	workspace, appErr := s.workspace(ctx)
	if appErr != nil {
		return nil, appErr
	}
	where := "WHERE n.workspace_id = ?"
	args := []interface{}{workspace.ID}
	if unreadOnly {
		where += " AND n.read_at IS NULL"
	}
	var rows []notificationRecord
	if err := s.db.WithContext(ctx).Raw(
		`SELECT n.id, n.uid, n.title, n.content, n.category, n.severity, n.resource_type, n.resource_id,
		        DATE_FORMAT(n.read_at, '%Y-%m-%d %H:%i:%s') AS read_at,
		        DATE_FORMAT(n.created_at, '%Y-%m-%d %H:%i:%s') AS created_at
		   FROM notifications n
		  `+where+`
		  ORDER BY n.created_at DESC
		  LIMIT 200`,
		args...,
	).Scan(&rows).Error; err != nil {
		return nil, apperror.Wrap(http.StatusInternalServerError, 500803, "list notifications failed", err)
	}
	out := make([]NotificationSummary, 0, len(rows))
	for _, row := range rows {
		out = append(out, notificationSummary(row))
	}
	return out, nil
}

func (s *Service) ListDeliveries(ctx context.Context, input ListDeliveriesInput) ([]DeliverySummary, *apperror.Error) {
	workspace, appErr := s.workspace(ctx)
	if appErr != nil {
		return nil, appErr
	}
	args := []interface{}{workspace.ID}
	where := "WHERE n.workspace_id = ?"
	if status := strings.TrimSpace(input.Status); status != "" {
		where += " AND nd.status = ?"
		args = append(args, status)
	}
	if channelID := strings.TrimSpace(input.ChannelID); channelID != "" {
		where += " AND nc.uid = ?"
		args = append(args, channelID)
	}
	if notificationID := strings.TrimSpace(input.NotificationID); notificationID != "" {
		where += " AND n.uid = ?"
		args = append(args, notificationID)
	}
	var rows []deliveryRecord
	if err := s.db.WithContext(ctx).Raw(
		`SELECT nd.id, n.uid AS notification_uid, n.title, nc.uid AS channel_uid, nc.name AS channel_name, nc.channel_type,
		        nd.status, nd.attempts,
		        DATE_FORMAT(nd.next_retry_at, '%Y-%m-%d %H:%i:%s') AS next_retry_at,
		        DATE_FORMAT(nd.delivered_at, '%Y-%m-%d %H:%i:%s') AS delivered_at,
		        nd.error_message,
		        DATE_FORMAT(nd.created_at, '%Y-%m-%d %H:%i:%s') AS created_at,
		        DATE_FORMAT(nd.updated_at, '%Y-%m-%d %H:%i:%s') AS updated_at
		   FROM notification_deliveries nd
		   JOIN notifications n ON n.id = nd.notification_id
		   LEFT JOIN notification_channels nc ON nc.id = nd.channel_id
		  `+where+`
		  ORDER BY nd.created_at DESC, nd.id DESC
		  LIMIT 200`,
		args...,
	).Scan(&rows).Error; err != nil {
		return nil, apperror.Wrap(http.StatusInternalServerError, 500804, "list notification deliveries failed", err)
	}
	out := make([]DeliverySummary, 0, len(rows))
	for _, row := range rows {
		out = append(out, deliverySummary(row))
	}
	return out, nil
}

func (s *Service) MarkRead(ctx context.Context, notificationUID string) *apperror.Error {
	workspace, appErr := s.workspace(ctx)
	if appErr != nil {
		return appErr
	}
	if strings.TrimSpace(notificationUID) == "" {
		return apperror.New(http.StatusBadRequest, 400001, "notification id is required")
	}
	res := s.db.WithContext(ctx).Exec(
		"UPDATE notifications SET read_at = COALESCE(read_at, NOW(3)) WHERE workspace_id = ? AND uid = ?",
		workspace.ID, notificationUID,
	)
	if res.Error != nil {
		return apperror.Wrap(http.StatusInternalServerError, 500804, "mark notification read failed", res.Error)
	}
	if res.RowsAffected == 0 {
		return apperror.New(http.StatusNotFound, 404801, "notification not found")
	}
	return nil
}

func (s *Service) RetryDelivery(ctx context.Context, deliveryID uint64, auditCtx AuditContext) *apperror.Error {
	if deliveryID == 0 {
		return apperror.New(http.StatusBadRequest, 400804, "delivery id is required")
	}
	txErr := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		workspace, err := defaultWorkspace(ctx, tx, s.cfg.Bootstrap.WorkspaceSlug)
		if err != nil {
			return err
		}
		var row deliveryRecord
		if err := tx.WithContext(ctx).Raw(
			`SELECT nd.id, n.uid AS notification_uid, n.title, nc.uid AS channel_uid, nc.name AS channel_name, nc.channel_type,
			        nd.status, nd.attempts,
			        DATE_FORMAT(nd.next_retry_at, '%Y-%m-%d %H:%i:%s') AS next_retry_at,
			        DATE_FORMAT(nd.delivered_at, '%Y-%m-%d %H:%i:%s') AS delivered_at,
			        nd.error_message,
			        DATE_FORMAT(nd.created_at, '%Y-%m-%d %H:%i:%s') AS created_at,
			        DATE_FORMAT(nd.updated_at, '%Y-%m-%d %H:%i:%s') AS updated_at
			   FROM notification_deliveries nd
			   JOIN notifications n ON n.id = nd.notification_id
			   LEFT JOIN notification_channels nc ON nc.id = nd.channel_id
			  WHERE n.workspace_id = ? AND nd.id = ?
			  LIMIT 1
			  FOR UPDATE`,
			workspace.ID, deliveryID,
		).Scan(&row).Error; err != nil {
			return err
		}
		if row.ID == 0 {
			return apperror.New(http.StatusNotFound, 404802, "notification delivery not found")
		}
		if row.Status != "failed" {
			return apperror.New(http.StatusBadRequest, 400805, "only failed deliveries can be retried")
		}
		if err := tx.WithContext(ctx).Exec(
			`UPDATE notification_deliveries
			    SET status = 'pending', next_retry_at = NULL, error_message = NULL, delivered_at = NULL, updated_at = NOW(3)
			  WHERE id = ?`,
			row.ID,
		).Error; err != nil {
			return err
		}
		actor, err := userByUID(ctx, tx, auditCtx.ActorUID)
		if err != nil {
			return err
		}
		updated := row
		updated.Status = "pending"
		updated.NextRetryAt = sql.NullString{}
		updated.DeliveredAt = sql.NullString{}
		updated.ErrorMessage = sql.NullString{}
		audit.Write(ctx, tx, audit.Event{
			WorkspaceID:   workspace.ID,
			ActorUserID:   nullID(actor.ID),
			Action:        "notification.delivery.retry",
			ResourceType:  "notification_delivery",
			ResourceID:    nullID(row.ID),
			IP:            auditCtx.IP,
			UserAgent:     auditCtx.UserAgent,
			TraceID:       auditCtx.TraceID,
			RequestMethod: auditCtx.RequestMethod,
			RequestPath:   auditCtx.RequestPath,
			Before:        deliverySummary(row),
			After:         deliverySummary(updated),
		})
		return nil
	})
	if txErr != nil {
		if appErr, ok := txErr.(*apperror.Error); ok {
			return appErr
		}
		return apperror.Wrap(http.StatusInternalServerError, 500805, "retry notification delivery failed", txErr)
	}
	return nil
}

func (s *Service) BulkRetryDeliveries(ctx context.Context, input BulkRetryDeliveriesInput, auditCtx AuditContext) (BulkRetryDeliveriesResult, *apperror.Error) {
	status := strings.TrimSpace(input.Status)
	if status == "" {
		status = "failed"
	}
	if status != "failed" {
		return BulkRetryDeliveriesResult{}, apperror.New(http.StatusBadRequest, 400808, "only failed deliveries can be bulk retried")
	}
	limit := input.Limit
	if limit <= 0 {
		limit = 100
	}
	if limit > 500 {
		limit = 500
	}
	result := BulkRetryDeliveriesResult{}
	txErr := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		workspace, err := defaultWorkspace(ctx, tx, s.cfg.Bootstrap.WorkspaceSlug)
		if err != nil {
			return err
		}
		args := []interface{}{workspace.ID, status}
		where := "WHERE n.workspace_id = ? AND nd.status = ?"
		if channelID := strings.TrimSpace(input.ChannelID); channelID != "" {
			where += " AND nc.uid = ?"
			args = append(args, channelID)
		}
		if notificationID := strings.TrimSpace(input.NotificationID); notificationID != "" {
			where += " AND n.uid = ?"
			args = append(args, notificationID)
		}
		var ids []uint64
		selectArgs := append([]interface{}{}, args...)
		selectArgs = append(selectArgs, limit)
		if err := tx.WithContext(ctx).Raw(
			`SELECT nd.id
			   FROM notification_deliveries nd
			   JOIN notifications n ON n.id = nd.notification_id
			   LEFT JOIN notification_channels nc ON nc.id = nd.channel_id
			  `+where+`
			  ORDER BY nd.updated_at ASC, nd.id ASC
			  LIMIT ?
			  FOR UPDATE`,
			selectArgs...,
		).Scan(&ids).Error; err != nil {
			return err
		}
		result.MatchedCount = uint(len(ids))
		if len(ids) == 0 {
			return nil
		}
		res := tx.WithContext(ctx).Exec(
			`UPDATE notification_deliveries
			    SET status = 'pending', next_retry_at = NULL, error_message = NULL, delivered_at = NULL, updated_at = NOW(3)
			  WHERE id IN ?`,
			ids,
		)
		if res.Error != nil {
			return res.Error
		}
		result.RetriedCount = uint(res.RowsAffected)
		actor, err := userByUID(ctx, tx, auditCtx.ActorUID)
		if err != nil {
			return err
		}
		audit.Write(ctx, tx, audit.Event{
			WorkspaceID:   workspace.ID,
			ActorUserID:   nullID(actor.ID),
			Action:        "notification.delivery.bulk_retry",
			ResourceType:  "notification_delivery",
			IP:            auditCtx.IP,
			UserAgent:     auditCtx.UserAgent,
			TraceID:       auditCtx.TraceID,
			RequestMethod: auditCtx.RequestMethod,
			RequestPath:   auditCtx.RequestPath,
			After:         result,
			Metadata: map[string]interface{}{
				"channelId":      strings.TrimSpace(input.ChannelID),
				"notificationId": strings.TrimSpace(input.NotificationID),
				"limit":          limit,
			},
		})
		return nil
	})
	if txErr != nil {
		if appErr, ok := txErr.(*apperror.Error); ok {
			return BulkRetryDeliveriesResult{}, appErr
		}
		return BulkRetryDeliveriesResult{}, apperror.Wrap(http.StatusInternalServerError, 500808, "bulk retry notification deliveries failed", txErr)
	}
	return result, nil
}

func EnqueueForAlert(ctx context.Context, tx *gorm.DB, workspaceID, alertID uint64, title, content, severity string) error {
	notificationUID, err := uid.New()
	if err != nil {
		return err
	}
	title, content, err = renderNotificationTemplateTx(ctx, tx, workspaceID, "alert", "site", title, content, severity, "alert", alertID)
	if err != nil {
		return err
	}
	if err := tx.WithContext(ctx).Exec(
		`INSERT INTO notifications(uid, workspace_id, title, content, category, severity, resource_type, resource_id)
		 VALUES (?, ?, ?, ?, 'alert', ?, 'alert', ?)`,
		notificationUID, workspaceID, title, nullString(content), normalizeSeverity(severity), alertID,
	).Error; err != nil {
		return err
	}
	var notificationID uint64
	if err := tx.WithContext(ctx).Raw("SELECT id FROM notifications WHERE uid = ? LIMIT 1", notificationUID).Scan(&notificationID).Error; err != nil {
		return err
	}
	var channels []struct {
		ID          uint64
		ChannelType string
	}
	if err := tx.WithContext(ctx).Raw(
		"SELECT id, channel_type FROM notification_channels WHERE workspace_id = ? AND status = 'active' AND deleted_at IS NULL",
		workspaceID,
	).Scan(&channels).Error; err != nil {
		return err
	}
	if len(channels) == 0 {
		return tx.WithContext(ctx).Exec(
			"INSERT INTO notification_deliveries(notification_id, status, attempts, delivered_at) VALUES (?, 'success', 1, NOW(3))",
			notificationID,
		).Error
	}
	for _, channel := range channels {
		status := "pending"
		attempts := 0
		deliveredAt := sql.NullTime{}
		if channel.ChannelType == "site" {
			status = "success"
			attempts = 1
			deliveredAt.Valid = true
		}
		if err := tx.WithContext(ctx).Exec(
			`INSERT INTO notification_deliveries(notification_id, channel_id, status, attempts, delivered_at)
			 VALUES (?, ?, ?, ?, CASE WHEN ? THEN NOW(3) ELSE NULL END)`,
			notificationID, channel.ID, status, attempts, deliveredAt.Valid,
		).Error; err != nil {
			return err
		}
	}
	return nil
}

func EnqueueForWorkflow(ctx context.Context, tx *gorm.DB, workspaceID, workflowRunID uint64, channelUID, title, content, severity string) (string, error) {
	notificationUID, err := uid.New()
	if err != nil {
		return "", err
	}
	title = strings.TrimSpace(title)
	if title == "" {
		title = "Workflow notification"
	}
	title, content, err = renderNotificationTemplateTx(ctx, tx, workspaceID, "workflow", "site", title, content, severity, "workflow_run", workflowRunID)
	if err != nil {
		return "", err
	}
	if err := tx.WithContext(ctx).Exec(
		`INSERT INTO notifications(uid, workspace_id, title, content, category, severity, resource_type, resource_id)
		 VALUES (?, ?, ?, ?, 'workflow', ?, 'workflow_run', ?)`,
		notificationUID, workspaceID, title, nullString(content), normalizeSeverity(severity), workflowRunID,
	).Error; err != nil {
		return "", err
	}
	var notificationID uint64
	if err := tx.WithContext(ctx).Raw("SELECT id FROM notifications WHERE uid = ? LIMIT 1", notificationUID).Scan(&notificationID).Error; err != nil {
		return "", err
	}
	var channels []struct {
		ID          uint64
		ChannelType string
	}
	query := "SELECT id, channel_type FROM notification_channels WHERE workspace_id = ? AND status = 'active' AND deleted_at IS NULL"
	args := []interface{}{workspaceID}
	if strings.TrimSpace(channelUID) != "" {
		query += " AND uid = ?"
		args = append(args, strings.TrimSpace(channelUID))
	}
	if err := tx.WithContext(ctx).Raw(query, args...).Scan(&channels).Error; err != nil {
		return "", err
	}
	if len(channels) == 0 {
		return notificationUID, tx.WithContext(ctx).Exec(
			"INSERT INTO notification_deliveries(notification_id, status, attempts, delivered_at) VALUES (?, 'success', 1, NOW(3))",
			notificationID,
		).Error
	}
	for _, channel := range channels {
		status := "pending"
		attempts := 0
		deliveredAt := sql.NullTime{}
		if channel.ChannelType == "site" {
			status = "success"
			attempts = 1
			deliveredAt.Valid = true
		}
		if err := tx.WithContext(ctx).Exec(
			`INSERT INTO notification_deliveries(notification_id, channel_id, status, attempts, delivered_at)
			 VALUES (?, ?, ?, ?, CASE WHEN ? THEN NOW(3) ELSE NULL END)`,
			notificationID, channel.ID, status, attempts, deliveredAt.Valid,
		).Error; err != nil {
			return "", err
		}
	}
	return notificationUID, nil
}

func (s *Service) workspace(ctx context.Context) (workspaceRecord, *apperror.Error) {
	workspace, err := defaultWorkspace(ctx, s.db, s.cfg.Bootstrap.WorkspaceSlug)
	if err != nil {
		return workspaceRecord{}, apperror.Wrap(http.StatusInternalServerError, 500805, "load workspace failed", err)
	}
	if workspace.ID == 0 {
		return workspaceRecord{}, apperror.New(http.StatusInternalServerError, 500805, "default workspace is not initialized")
	}
	return workspace, nil
}

func (s *Service) executeDeliveryAttempt(ctx context.Context, row pendingDelivery, claim bool) error {
	if claim {
		if err := s.db.WithContext(ctx).Exec(
			"UPDATE notification_deliveries SET status = 'sending', attempts = attempts + 1, updated_at = NOW(3) WHERE id = ?",
			row.ID,
		).Error; err != nil {
			return err
		}
	}
	if err := s.sendDelivery(ctx, row); err != nil {
		if updateErr := s.markDeliveryFailed(ctx, row, err); updateErr != nil {
			return updateErr
		}
		return err
	}
	return s.db.WithContext(ctx).Exec(
		"UPDATE notification_deliveries SET status = 'success', delivered_at = NOW(3), error_message = NULL, updated_at = NOW(3) WHERE id = ?",
		row.ID,
	).Error
}

func defaultWorkspace(ctx context.Context, db *gorm.DB, slug string) (workspaceRecord, error) {
	var workspace workspaceRecord
	err := db.WithContext(ctx).Raw("SELECT id FROM workspaces WHERE slug = ? AND status = 'active' LIMIT 1", slug).Scan(&workspace).Error
	return workspace, err
}

func userByUID(ctx context.Context, db *gorm.DB, uid string) (userRecord, error) {
	var user userRecord
	err := db.WithContext(ctx).Raw("SELECT id FROM users WHERE uid = ? AND status = 'active' LIMIT 1", uid).Scan(&user).Error
	return user, err
}

func channelByUID(ctx context.Context, db *gorm.DB, workspaceID uint64, uid string) (channelRecord, error) {
	var row channelRecord
	err := db.WithContext(ctx).Raw(
		`SELECT nc.id, nc.uid, nc.name, nc.channel_type, nc.config, nc.status, u.username AS created_by,
		        DATE_FORMAT(nc.created_at, '%Y-%m-%d %H:%i:%s') AS created_at
		   FROM notification_channels nc
		   LEFT JOIN users u ON u.id = nc.created_by
		  WHERE nc.workspace_id = ? AND nc.uid = ? AND nc.deleted_at IS NULL
		  LIMIT 1`,
		workspaceID, uid,
	).Scan(&row).Error
	return row, err
}

func channelDispatchByUID(ctx context.Context, db *gorm.DB, workspaceID uint64, uid string) (channelDispatchRecord, error) {
	var row channelDispatchRecord
	err := db.WithContext(ctx).Raw(
		`SELECT nc.id, nc.uid, nc.name, nc.channel_type, nc.config, nc.status
		   FROM notification_channels nc
		  WHERE nc.workspace_id = ? AND nc.uid = ? AND nc.deleted_at IS NULL
		  LIMIT 1`,
		workspaceID, uid,
	).Scan(&row).Error
	return row, err
}

func templateByUID(ctx context.Context, db *gorm.DB, workspaceID uint64, uid string) (templateRecord, error) {
	var row templateRecord
	err := db.WithContext(ctx).Raw(
		`SELECT nt.id, nt.uid, nt.name, nt.category, nt.channel_type, nt.title_template, nt.content_template,
		        nt.status, u.username AS created_by,
		        DATE_FORMAT(nt.created_at, '%Y-%m-%d %H:%i:%s') AS created_at,
		        DATE_FORMAT(nt.updated_at, '%Y-%m-%d %H:%i:%s') AS updated_at
		   FROM notification_templates nt
		   LEFT JOIN users u ON u.id = nt.created_by
		  WHERE nt.workspace_id = ? AND nt.uid = ? AND nt.deleted_at IS NULL
		  LIMIT 1`,
		workspaceID, uid,
	).Scan(&row).Error
	return row, err
}

func channelSummary(row channelRecord, cfg map[string]interface{}) ChannelSummary {
	return ChannelSummary{
		ID:          row.UID,
		Name:        row.Name,
		ChannelType: row.ChannelType,
		Target:      maskChannelTarget(row.ChannelType, cfg),
		Status:      row.Status,
		CreatedBy:   row.CreatedBy.String,
		CreatedAt:   row.CreatedAt,
	}
}

func templateSummary(row templateRecord) TemplateSummary {
	return TemplateSummary{
		ID:              row.UID,
		Name:            row.Name,
		Category:        row.Category,
		ChannelType:     row.ChannelType,
		TitleTemplate:   row.TitleTemplate,
		ContentTemplate: row.ContentTemplate.String,
		Status:          row.Status,
		CreatedBy:       row.CreatedBy.String,
		CreatedAt:       row.CreatedAt,
		UpdatedAt:       row.UpdatedAt,
	}
}

func notificationSummary(row notificationRecord) NotificationSummary {
	resourceID := uint64(0)
	if row.ResourceID.Valid {
		resourceID = uint64(row.ResourceID.Int64)
	}
	return NotificationSummary{
		ID:           row.UID,
		Title:        row.Title,
		Content:      row.Content.String,
		Category:     row.Category,
		Severity:     row.Severity,
		ResourceType: row.ResourceType.String,
		ResourceID:   resourceID,
		ReadAt:       row.ReadAt.String,
		CreatedAt:    row.CreatedAt,
	}
}

func deliverySummary(row deliveryRecord) DeliverySummary {
	return DeliverySummary{
		ID:             row.ID,
		NotificationID: row.NotificationUID.String,
		Title:          row.Title.String,
		ChannelID:      row.ChannelUID.String,
		ChannelName:    row.ChannelName.String,
		ChannelType:    row.ChannelType.String,
		Status:         row.Status,
		Attempts:       row.Attempts,
		NextRetryAt:    row.NextRetryAt.String,
		DeliveredAt:    row.DeliveredAt.String,
		ErrorMessage:   row.ErrorMessage.String,
		CreatedAt:      row.CreatedAt,
		UpdatedAt:      row.UpdatedAt,
	}
}

func maskChannelTarget(channelType string, cfg map[string]interface{}) string {
	switch channelType {
	case "site":
		return "in-app"
	case "email":
		return maskEmailAddress(deliveryEmail(cfg))
	default:
		return maskWebhookTarget(deliveryURL(cfg))
	}
}

func maskEmailAddress(value string) string {
	value = strings.TrimSpace(value)
	parts := strings.Split(value, "@")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return ""
	}
	local := parts[0]
	switch {
	case len(local) <= 2:
		local = local[:1] + "*"
	case len(local) <= 4:
		local = local[:1] + strings.Repeat("*", len(local)-2) + local[len(local)-1:]
	default:
		local = local[:2] + strings.Repeat("*", len(local)-4) + local[len(local)-2:]
	}
	return local + "@" + parts[1]
}

func maskWebhookTarget(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	scheme := ""
	rest := value
	if strings.Contains(value, "://") {
		parts := strings.SplitN(value, "://", 2)
		scheme = parts[0] + "://"
		rest = parts[1]
	}
	host := rest
	if idx := strings.IndexAny(rest, "/?#"); idx >= 0 {
		host = rest[:idx]
	}
	if host == "" {
		return ""
	}
	return scheme + host + "/..."
}

type normalizedTemplateInput struct {
	Name            string
	Category        string
	ChannelType     string
	TitleTemplate   string
	ContentTemplate string
	Status          string
}

var notificationTemplatePattern = regexp.MustCompile(`\{\{\s*([a-zA-Z0-9_.-]+)\s*\}\}`)

func normalizeTemplateInput(name, category, channelType, titleTemplate, contentTemplate, status string) (normalizedTemplateInput, *apperror.Error) {
	out := normalizedTemplateInput{
		Name:            strings.TrimSpace(name),
		Category:        strings.TrimSpace(category),
		ChannelType:     strings.TrimSpace(channelType),
		TitleTemplate:   strings.TrimSpace(titleTemplate),
		ContentTemplate: strings.TrimSpace(contentTemplate),
		Status:          strings.TrimSpace(status),
	}
	if out.Category == "" {
		out.Category = "system"
	}
	if out.ChannelType == "" {
		out.ChannelType = "any"
	}
	if out.Status == "" {
		out.Status = "active"
	}
	if out.Name == "" {
		return out, apperror.New(http.StatusBadRequest, 400810, "notification template name is required")
	}
	if out.TitleTemplate == "" {
		return out, apperror.New(http.StatusBadRequest, 400811, "notification template title is required")
	}
	if !validTemplateChannelType(out.ChannelType) {
		return out, apperror.New(http.StatusBadRequest, 400812, "unsupported notification template channel type")
	}
	if out.Status != "active" && out.Status != "disabled" && out.Status != "archived" {
		return out, apperror.New(http.StatusBadRequest, 400813, "unsupported notification template status")
	}
	return out, nil
}

func validTemplateChannelType(value string) bool {
	return value == "any" || validChannelType(value)
}

func (s *Service) renderDeliveryTemplate(ctx context.Context, row pendingDelivery) pendingDelivery {
	if row.WorkspaceID == 0 {
		return row
	}
	title, content, err := renderNotificationTemplateTx(ctx, s.db, row.WorkspaceID, row.Category, row.ChannelType, row.Title, row.Content.String, row.Severity, row.ResourceType.String, uint64FromNull(row.ResourceID))
	if err != nil {
		return row
	}
	row.Title = title
	row.Content = nullString(content)
	return row
}

func renderNotificationTemplateTx(ctx context.Context, db *gorm.DB, workspaceID uint64, category, channelType, title, content, severity, resourceType string, resourceID uint64) (string, string, error) {
	var row templateRecord
	err := db.WithContext(ctx).Raw(
		`SELECT id, uid, name, category, channel_type, title_template, content_template, status
		   FROM notification_templates
		  WHERE workspace_id = ?
		    AND category = ?
		    AND status = 'active'
		    AND deleted_at IS NULL
		    AND channel_type IN (?, 'any')
		  ORDER BY CASE WHEN channel_type = ? THEN 0 ELSE 1 END, updated_at DESC
		  LIMIT 1`,
		workspaceID, strings.TrimSpace(category), strings.TrimSpace(channelType), strings.TrimSpace(channelType),
	).Scan(&row).Error
	if err != nil || row.ID == 0 {
		return title, content, err
	}
	values := notificationTemplateValues(title, content, category, channelType, severity, resourceType, resourceID)
	renderedTitle := renderNotificationTemplateText(row.TitleTemplate, values)
	renderedContent := content
	if row.ContentTemplate.Valid && strings.TrimSpace(row.ContentTemplate.String) != "" {
		renderedContent = renderNotificationTemplateText(row.ContentTemplate.String, values)
	}
	return limitString(renderedTitle, 255), limitString(renderedContent, 2048), nil
}

func notificationTemplateValues(title, content, category, channelType, severity, resourceType string, resourceID uint64) map[string]string {
	return map[string]string{
		"title":        title,
		"content":      content,
		"category":     category,
		"channelType":  channelType,
		"severity":     severity,
		"resourceType": resourceType,
		"resourceId":   fmt.Sprint(resourceID),
	}
}

func renderNotificationTemplateText(template string, values map[string]string) string {
	return notificationTemplatePattern.ReplaceAllStringFunc(template, func(match string) string {
		parts := notificationTemplatePattern.FindStringSubmatch(match)
		if len(parts) != 2 {
			return match
		}
		if value, ok := values[parts[1]]; ok {
			return value
		}
		return ""
	})
}

func uint64FromNull(value sql.NullInt64) uint64 {
	if !value.Valid || value.Int64 <= 0 {
		return 0
	}
	return uint64(value.Int64)
}

func validChannelType(value string) bool {
	switch value {
	case "site", "email", "webhook", "dingtalk", "wechat", "slack":
		return true
	default:
		return false
	}
}

func normalizeSeverity(value string) string {
	switch strings.TrimSpace(value) {
	case "info", "warning", "critical":
		return strings.TrimSpace(value)
	default:
		return "warning"
	}
}

func nullID(id uint64) sql.NullInt64 {
	if id == 0 {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: int64(id), Valid: true}
}

func nullString(value string) sql.NullString {
	value = strings.TrimSpace(value)
	return sql.NullString{String: value, Valid: value != ""}
}

func jsonNull(value interface{}) sql.NullString {
	if value == nil {
		return sql.NullString{}
	}
	bytes, err := json.Marshal(value)
	if err != nil || string(bytes) == "null" {
		return sql.NullString{}
	}
	return sql.NullString{String: string(bytes), Valid: true}
}

func nowString() string {
	return time.Now().Format("2006-01-02 15:04:05")
}

func auditResult(err error) string {
	if err != nil {
		return "failed"
	}
	return "success"
}
