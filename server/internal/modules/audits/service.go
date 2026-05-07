package audits

import (
	"context"
	"database/sql"
	"net/http"
	"strings"

	"gorm.io/gorm"

	"opspilot/server/internal/config"
	"opspilot/server/internal/shared/apperror"
)

type Service struct {
	db  *gorm.DB
	cfg config.Config
}

type ListInput struct {
	Action       string
	ActorType    string
	Result       string
	ResourceType string
	TraceID      string
	Keyword      string
	CreatedFrom  string
	CreatedTo    string
	Page         int
	PageSize     int
}

type ListResult struct {
	Items    []LogSummary `json:"items"`
	Total    int64        `json:"total"`
	Page     int          `json:"page"`
	PageSize int          `json:"pageSize"`
}

type LogSummary struct {
	ID            uint64 `json:"id"`
	ActorType     string `json:"actorType"`
	ActorUser     string `json:"actorUser,omitempty"`
	ActorAgent    string `json:"actorAgent,omitempty"`
	Action        string `json:"action"`
	ResourceType  string `json:"resourceType,omitempty"`
	ResourceID    uint64 `json:"resourceId,omitempty"`
	Result        string `json:"result"`
	IP            string `json:"ip,omitempty"`
	UserAgent     string `json:"userAgent,omitempty"`
	TraceID       string `json:"traceId,omitempty"`
	RequestMethod string `json:"requestMethod,omitempty"`
	RequestPath   string `json:"requestPath,omitempty"`
	Before        string `json:"before,omitempty"`
	After         string `json:"after,omitempty"`
	Metadata      string `json:"metadata,omitempty"`
	CreatedAt     string `json:"createdAt"`
}

type workspaceRecord struct {
	ID uint64
}

type logRecord struct {
	ID            uint64
	ActorType     string
	ActorUser     sql.NullString
	ActorAgent    sql.NullString
	Action        string
	ResourceType  sql.NullString
	ResourceID    sql.NullInt64
	Result        string
	IP            sql.NullString
	UserAgent     sql.NullString
	TraceID       sql.NullString
	RequestMethod sql.NullString
	RequestPath   sql.NullString
	BeforeData    sql.NullString
	AfterData     sql.NullString
	Metadata      sql.NullString
	CreatedAt     string
}

func NewService(db *gorm.DB, cfg config.Config) *Service {
	return &Service{db: db, cfg: cfg}
}

func (s *Service) List(ctx context.Context, input ListInput) (ListResult, *apperror.Error) {
	workspace, appErr := s.workspace(ctx)
	if appErr != nil {
		return ListResult{}, appErr
	}
	page, pageSize := normalizePage(input.Page, input.PageSize)
	where, args := auditWhere(workspace.ID, input)

	var total int64
	if err := s.db.WithContext(ctx).Raw("SELECT COUNT(*) FROM audit_logs al "+where, args...).Scan(&total).Error; err != nil {
		return ListResult{}, apperror.Wrap(http.StatusInternalServerError, 500901, "count audit logs failed", err)
	}

	queryArgs := append(append([]interface{}{}, args...), pageSize, (page-1)*pageSize)
	var rows []logRecord
	if err := s.db.WithContext(ctx).Raw(
		`SELECT al.id, al.actor_type, u.username AS actor_user, ag.name AS actor_agent,
		        al.action, al.resource_type, al.resource_id, al.result, al.ip, al.user_agent,
		        al.trace_id, al.request_method, al.request_path, al.before_data, al.after_data, al.metadata,
		        DATE_FORMAT(al.created_at, '%Y-%m-%d %H:%i:%s') AS created_at
		   FROM audit_logs al
		   LEFT JOIN users u ON u.id = al.actor_user_id
		   LEFT JOIN agents ag ON ag.id = al.actor_agent_id
		  `+where+`
		  ORDER BY al.created_at DESC, al.id DESC
		  LIMIT ? OFFSET ?`,
		queryArgs...,
	).Scan(&rows).Error; err != nil {
		return ListResult{}, apperror.Wrap(http.StatusInternalServerError, 500902, "list audit logs failed", err)
	}

	items := make([]LogSummary, 0, len(rows))
	for _, row := range rows {
		items = append(items, summary(row))
	}
	return ListResult{Items: items, Total: total, Page: page, PageSize: pageSize}, nil
}

func (s *Service) workspace(ctx context.Context) (workspaceRecord, *apperror.Error) {
	var workspace workspaceRecord
	err := s.db.WithContext(ctx).Raw("SELECT id FROM workspaces WHERE slug = ? AND status = 'active' LIMIT 1", s.cfg.Bootstrap.WorkspaceSlug).Scan(&workspace).Error
	if err != nil {
		return workspaceRecord{}, apperror.Wrap(http.StatusInternalServerError, 500903, "load workspace failed", err)
	}
	if workspace.ID == 0 {
		return workspaceRecord{}, apperror.New(http.StatusInternalServerError, 500904, "default workspace is not initialized")
	}
	return workspace, nil
}

func auditWhere(workspaceID uint64, input ListInput) (string, []interface{}) {
	where := "WHERE (al.workspace_id = ? OR al.workspace_id IS NULL)"
	args := []interface{}{workspaceID}
	add := func(condition string, value interface{}) {
		where += " AND " + condition
		args = append(args, value)
	}
	if value := strings.TrimSpace(input.Action); value != "" {
		add("al.action = ?", value)
	}
	if value := strings.TrimSpace(input.ActorType); value != "" {
		add("al.actor_type = ?", value)
	}
	if value := strings.TrimSpace(input.Result); value != "" {
		add("al.result = ?", value)
	}
	if value := strings.TrimSpace(input.ResourceType); value != "" {
		add("al.resource_type = ?", value)
	}
	if value := strings.TrimSpace(input.TraceID); value != "" {
		add("al.trace_id = ?", value)
	}
	if value := normalizeTimeFilter(input.CreatedFrom); value != "" {
		add("al.created_at >= ?", value)
	}
	if value := normalizeTimeFilter(input.CreatedTo); value != "" {
		add("al.created_at <= ?", value)
	}
	if value := strings.TrimSpace(input.Keyword); value != "" {
		like := "%" + value + "%"
		where += " AND (al.action LIKE ? OR al.resource_type LIKE ? OR al.request_path LIKE ? OR al.trace_id LIKE ? OR u.username LIKE ? OR ag.name LIKE ?)"
		args = append(args, like, like, like, like, like, like)
	}
	return where, args
}

func normalizeTimeFilter(value string) string {
	value = strings.TrimSpace(value)
	value = strings.ReplaceAll(value, "T", " ")
	return value
}

func summary(row logRecord) LogSummary {
	resourceID := uint64(0)
	if row.ResourceID.Valid && row.ResourceID.Int64 > 0 {
		resourceID = uint64(row.ResourceID.Int64)
	}
	return LogSummary{
		ID:            row.ID,
		ActorType:     row.ActorType,
		ActorUser:     row.ActorUser.String,
		ActorAgent:    row.ActorAgent.String,
		Action:        row.Action,
		ResourceType:  row.ResourceType.String,
		ResourceID:    resourceID,
		Result:        row.Result,
		IP:            row.IP.String,
		UserAgent:     row.UserAgent.String,
		TraceID:       row.TraceID.String,
		RequestMethod: row.RequestMethod.String,
		RequestPath:   row.RequestPath.String,
		Before:        row.BeforeData.String,
		After:         row.AfterData.String,
		Metadata:      row.Metadata.String,
		CreatedAt:     row.CreatedAt,
	}
}

func normalizePage(page, pageSize int) (int, int) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}
	return page, pageSize
}
