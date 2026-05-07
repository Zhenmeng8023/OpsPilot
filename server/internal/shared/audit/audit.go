package audit

import (
	"context"
	"database/sql"
	"encoding/json"
	"strings"

	"gorm.io/gorm"
)

type Event struct {
	WorkspaceID   uint64
	ActorType     string
	ActorUserID   sql.NullInt64
	ActorAgentID  sql.NullInt64
	Action        string
	ResourceType  string
	ResourceID    sql.NullInt64
	Result        string
	IP            string
	UserAgent     string
	TraceID       string
	RequestMethod string
	RequestPath   string
	Before        interface{}
	After         interface{}
	Metadata      interface{}
}

func Write(ctx context.Context, db *gorm.DB, event Event) {
	if db == nil || strings.TrimSpace(event.Action) == "" {
		return
	}
	actorType := defaultString(event.ActorType, "user")
	result := defaultString(event.Result, "success")
	_ = db.WithContext(ctx).Exec(
		`INSERT INTO audit_logs(
		    workspace_id, actor_type, actor_user_id, actor_agent_id, action, resource_type, resource_id,
		    result, ip, user_agent, trace_id, request_method, request_path, before_data, after_data, metadata
		  ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		nullUint64(event.WorkspaceID),
		actorType,
		event.ActorUserID,
		event.ActorAgentID,
		event.Action,
		nullString(event.ResourceType),
		event.ResourceID,
		result,
		nullString(event.IP),
		nullString(event.UserAgent),
		nullString(event.TraceID),
		nullString(event.RequestMethod),
		nullString(event.RequestPath),
		jsonNull(event.Before),
		jsonNull(event.After),
		jsonNull(event.Metadata),
	).Error
}

func UserIDByUID(ctx context.Context, db *gorm.DB, uid string) (sql.NullInt64, error) {
	uid = strings.TrimSpace(uid)
	if uid == "" {
		return sql.NullInt64{}, nil
	}
	var id uint64
	err := db.WithContext(ctx).Raw("SELECT id FROM users WHERE uid = ? LIMIT 1", uid).Scan(&id).Error
	if err != nil || id == 0 {
		return sql.NullInt64{}, err
	}
	return sql.NullInt64{Int64: int64(id), Valid: true}, nil
}

func nullUint64(value uint64) sql.NullInt64 {
	if value == 0 {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: int64(value), Valid: true}
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

func defaultString(value, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	return value
}
