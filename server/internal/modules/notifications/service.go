package notifications

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"strings"

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

type ChannelSummary struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	ChannelType string `json:"channelType"`
	Status      string `json:"status"`
	CreatedBy   string `json:"createdBy,omitempty"`
	CreatedAt   string `json:"createdAt"`
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
	Status      string
	CreatedBy   sql.NullString
	CreatedAt   string
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
		`SELECT nc.id, nc.uid, nc.name, nc.channel_type, nc.status, u.username AS created_by,
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
		out = append(out, channelSummary(row))
	}
	return out, nil
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
		channelUID, err := uid.New()
		if err != nil {
			return err
		}
		if err := tx.WithContext(ctx).Exec(
			`INSERT INTO notification_channels(uid, workspace_id, name, channel_type, config, status, created_by)
			 VALUES (?, ?, ?, ?, ?, 'active', ?)`,
			channelUID, workspace.ID, name, channelType, jsonNull(input.Config), nullID(actor.ID),
		).Error; err != nil {
			return err
		}
		row, err := channelByUID(ctx, tx, workspace.ID, channelUID)
		if err != nil {
			return err
		}
		created = channelSummary(row)
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

func EnqueueForAlert(ctx context.Context, tx *gorm.DB, workspaceID, alertID uint64, title, content, severity string) error {
	notificationUID, err := uid.New()
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
		deliveredAt := sql.NullTime{}
		if channel.ChannelType == "site" {
			status = "success"
			deliveredAt.Valid = true
		}
		if err := tx.WithContext(ctx).Exec(
			`INSERT INTO notification_deliveries(notification_id, channel_id, status, attempts, delivered_at)
			 VALUES (?, ?, ?, 1, CASE WHEN ? THEN NOW(3) ELSE NULL END)`,
			notificationID, channel.ID, status, deliveredAt.Valid,
		).Error; err != nil {
			return err
		}
	}
	return nil
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
		`SELECT nc.id, nc.uid, nc.name, nc.channel_type, nc.status, u.username AS created_by,
		        DATE_FORMAT(nc.created_at, '%Y-%m-%d %H:%i:%s') AS created_at
		   FROM notification_channels nc
		   LEFT JOIN users u ON u.id = nc.created_by
		  WHERE nc.workspace_id = ? AND nc.uid = ? AND nc.deleted_at IS NULL
		  LIMIT 1`,
		workspaceID, uid,
	).Scan(&row).Error
	return row, err
}

func channelSummary(row channelRecord) ChannelSummary {
	return ChannelSummary{
		ID:          row.UID,
		Name:        row.Name,
		ChannelType: row.ChannelType,
		Status:      row.Status,
		CreatedBy:   row.CreatedBy.String,
		CreatedAt:   row.CreatedAt,
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
