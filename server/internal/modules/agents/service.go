package agents

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"gorm.io/gorm"

	"opspilot/server/internal/config"
	"opspilot/server/internal/shared/apperror"
	"opspilot/server/internal/shared/audit"
)

type Service struct {
	db  *gorm.DB
	cfg config.Config
}

type HostInfoInput struct {
	Name      string                 `json:"name,omitempty"`
	Hostname  string                 `json:"hostname,omitempty"`
	IP        string                 `json:"ip,omitempty"`
	OS        string                 `json:"os,omitempty"`
	OSType    string                 `json:"osType,omitempty"`
	OSName    string                 `json:"osName,omitempty"`
	OSVersion string                 `json:"osVersion,omitempty"`
	Arch      string                 `json:"arch,omitempty"`
	Version   string                 `json:"version,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

type RegisterInput struct {
	BootstrapSecret string
	EnrollmentToken string
	Audit           AuditContext
	HostInfoInput
}

type HeartbeatInput struct {
	Status string
	HostInfoInput
}

type AgentIdentity struct {
	ID          uint64
	UID         string
	WorkspaceID uint64
	HostID      uint64
	Name        string
	Status      string
}

type AuditContext struct {
	ActorUID      string
	IP            string
	UserAgent     string
	TraceID       string
	RequestMethod string
	RequestPath   string
}

type ListInput struct {
	Keyword  string
	Status   string
	Page     int
	PageSize int
}

type AgentListResult struct {
	Items    []AgentSummary `json:"items"`
	Total    int64          `json:"total"`
	Page     int            `json:"page"`
	PageSize int            `json:"pageSize"`
}

type HostListResult struct {
	Items    []HostSummary `json:"items"`
	Total    int64         `json:"total"`
	Page     int           `json:"page"`
	PageSize int           `json:"pageSize"`
}

type EnrollmentTokenSummary struct {
	ID                string `json:"id"`
	TokenPrefix       string `json:"tokenPrefix"`
	Status            string `json:"status"`
	MaxUses           uint   `json:"maxUses"`
	UsedCount         uint   `json:"usedCount"`
	BindWorkspaceSlug string `json:"bindWorkspaceSlug,omitempty"`
	ExpiresAt         string `json:"expiresAt"`
	CreatedBy         string `json:"createdBy,omitempty"`
	CreatedAt         string `json:"createdAt"`
}

type EnrollmentTokenDetail struct {
	EnrollmentTokenSummary
	Token string `json:"token,omitempty"`
}

type CreateEnrollmentTokenInput struct {
	MaxUses           uint
	ExpiresInSeconds  uint
	BindWorkspaceSlug string
	Audit             AuditContext
}

type AgentSummary struct {
	ID              string       `json:"id"`
	Name            string       `json:"name"`
	Status          string       `json:"status"`
	TokenPrefix     string       `json:"tokenPrefix,omitempty"`
	Version         string       `json:"version,omitempty"`
	IP              string       `json:"ip,omitempty"`
	OS              string       `json:"os,omitempty"`
	Arch            string       `json:"arch,omitempty"`
	LastHeartbeatAt string       `json:"lastHeartbeatAt,omitempty"`
	DisabledAt      string       `json:"disabledAt,omitempty"`
	CreatedAt       string       `json:"createdAt"`
	Host            *HostSummary `json:"host,omitempty"`
}

type HostSummary struct {
	ID               string `json:"id"`
	Name             string `json:"name"`
	Hostname         string `json:"hostname,omitempty"`
	IP               string `json:"ip,omitempty"`
	OS               string `json:"os,omitempty"`
	Arch             string `json:"arch,omitempty"`
	Status           string `json:"status"`
	AgentCount       int    `json:"agentCount,omitempty"`
	OnlineAgentCount int    `json:"onlineAgentCount,omitempty"`
	LastHeartbeatAt  string `json:"lastHeartbeatAt,omitempty"`
	CreatedAt        string `json:"createdAt"`
}

type RegistrationResult struct {
	Agent       AgentSummary `json:"agent"`
	Token       string       `json:"token"`
	TokenPrefix string       `json:"tokenPrefix"`
}

type HeartbeatResult struct {
	Agent      AgentSummary `json:"agent"`
	Host       HostSummary  `json:"host"`
	ReceivedAt string       `json:"receivedAt"`
}

type OfflineScanResult struct {
	OfflineAgents    int64 `json:"offlineAgents"`
	OfflineHosts     int64 `json:"offlineHosts"`
	ThresholdSeconds int   `json:"thresholdSeconds"`
}

type workspaceRecord struct {
	ID   uint64
	UID  string
	Name string
	Slug string
}

type hostRecord struct {
	ID        uint64
	UID       string
	Name      string
	Hostname  sql.NullString
	PrimaryIP sql.NullString
	OSType    sql.NullString
	OSName    sql.NullString
	OSVersion sql.NullString
	Arch      sql.NullString
	Status    string
	CreatedAt string
}

type agentRecord struct {
	ID              uint64
	UID             string
	WorkspaceID     uint64
	HostID          sql.NullInt64
	Name            string
	TokenPrefix     sql.NullString
	Version         sql.NullString
	Status          string
	OSType          sql.NullString
	Arch            sql.NullString
	IP              sql.NullString
	LastHeartbeatAt sql.NullString
	DisabledAt      sql.NullString
	CreatedAt       string
}

func NewService(db *gorm.DB, cfg config.Config) *Service {
	return &Service{db: db, cfg: cfg}
}

func (s *Service) Register(ctx context.Context, input RegisterInput) (RegistrationResult, *apperror.Error) {
	input = normalizeRegisterInput(input)
	enrollmentHash := hashToken(input.EnrollmentToken)
	useEnrollment := strings.TrimSpace(input.EnrollmentToken) != ""
	if !useEnrollment {
		if !s.cfg.Agent.RegistrationEnabled {
			return RegistrationResult{}, apperror.New(http.StatusForbidden, 403102, "agent registration is disabled")
		}
		if strings.TrimSpace(input.BootstrapSecret) == "" || input.BootstrapSecret != s.cfg.Agent.BootstrapSecret {
			return RegistrationResult{}, apperror.New(http.StatusUnauthorized, 401010, "invalid agent bootstrap secret")
		}
	}
	agentName := input.Name
	if agentName == "" {
		agentName = input.Hostname
	}
	if agentName == "" {
		return RegistrationResult{}, apperror.New(http.StatusBadRequest, 400101, "agent name or hostname is required")
	}

	token, tokenHash, tokenPrefix, err := newAgentToken()
	if err != nil {
		return RegistrationResult{}, apperror.Wrap(http.StatusInternalServerError, 500101, "issue agent token failed", err)
	}

	var result RegistrationResult
	txErr := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		workspace, err := s.defaultWorkspace(ctx, tx)
		if err != nil {
			return err
		}
		if workspace.ID == 0 {
			return apperror.New(http.StatusInternalServerError, 500102, "default workspace is not initialized")
		}

		if useEnrollment {
			if err := s.consumeEnrollmentToken(ctx, tx, workspace.ID, enrollmentHash); err != nil {
				return err
			}
		}

		host, err := s.ensureHost(ctx, tx, workspace.ID, input.HostInfoInput, 0)
		if err != nil {
			return err
		}

		agent, err := s.agentByName(ctx, tx, workspace.ID, agentName)
		if err != nil {
			return err
		}
		if agent.ID != 0 && agent.Status == "disabled" {
			return apperror.New(http.StatusForbidden, 403101, "agent is disabled")
		}
		if agent.ID == 0 {
			uid, err := newUID()
			if err != nil {
				return err
			}
			if err := tx.WithContext(ctx).Exec(
				`INSERT INTO agents(uid, workspace_id, host_id, name, token_hash, token_prefix, version, status, os_type, arch, ip, last_heartbeat_at, metadata)
				 VALUES (?, ?, ?, ?, ?, ?, ?, 'online', ?, ?, ?, NOW(3), ?)`,
				uid, workspace.ID, host.ID, agentName, tokenHash, tokenPrefix, nullString(input.Version), nullString(input.OSType), nullString(input.Arch), nullString(input.IP), jsonNull(input.Metadata),
			).Error; err != nil {
				return err
			}
			agent, err = s.agentByName(ctx, tx, workspace.ID, agentName)
			if err != nil {
				return err
			}
		} else if err := tx.WithContext(ctx).Exec(
			`UPDATE agents
			    SET host_id = ?, token_hash = ?, token_prefix = ?, version = ?, status = 'online',
			        os_type = ?, arch = ?, ip = ?, last_heartbeat_at = NOW(3), metadata = ?
			  WHERE id = ?`,
			host.ID, tokenHash, tokenPrefix, nullString(input.Version), nullString(input.OSType), nullString(input.Arch), nullString(input.IP), jsonNull(input.Metadata), agent.ID,
		).Error; err != nil {
			return err
		}

		if err := tx.WithContext(ctx).Exec(
			"UPDATE agent_tokens SET status = 'revoked', revoked_at = NOW(3) WHERE agent_id = ? AND status = 'active'",
			agent.ID,
		).Error; err != nil {
			return err
		}
		if err := tx.WithContext(ctx).Exec(
			"INSERT INTO agent_tokens(agent_id, token_hash, token_prefix, status, last_used_at) VALUES (?, ?, ?, 'active', NOW(3))",
			agent.ID, tokenHash, tokenPrefix,
		).Error; err != nil {
			return err
		}
		if err := s.writeHeartbeat(ctx, tx, agent.ID, host.ID, "online", input.HostInfoInput); err != nil {
			return err
		}
		audit.Write(ctx, tx, audit.Event{
			WorkspaceID:   workspace.ID,
			ActorType:     "agent",
			ActorAgentID:  sql.NullInt64{Int64: int64(agent.ID), Valid: true},
			Action:        "agent.register",
			ResourceType:  "agent",
			ResourceID:    sql.NullInt64{Int64: int64(agent.ID), Valid: true},
			IP:            input.Audit.IP,
			UserAgent:     input.Audit.UserAgent,
			TraceID:       input.Audit.TraceID,
			RequestMethod: input.Audit.RequestMethod,
			RequestPath:   input.Audit.RequestPath,
			After:         input.HostInfoInput,
		})

		summary, err := s.agentSummaryByID(ctx, tx, agent.ID)
		if err != nil {
			return err
		}
		result = RegistrationResult{
			Agent:       summary,
			Token:       token,
			TokenPrefix: tokenPrefix,
		}
		return nil
	})
	if txErr != nil {
		if appErr, ok := txErr.(*apperror.Error); ok {
			return RegistrationResult{}, appErr
		}
		return RegistrationResult{}, apperror.Wrap(http.StatusInternalServerError, 500101, "register agent failed", txErr)
	}
	return result, nil
}

func (s *Service) AuthenticateAgent(ctx context.Context, token string) (AgentIdentity, *apperror.Error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return AgentIdentity{}, apperror.New(http.StatusUnauthorized, 401011, "agent token is required")
	}

	tokenHash := hashToken(token)
	var identity AgentIdentity
	err := s.db.WithContext(ctx).Raw(
		`SELECT a.id, a.uid, a.workspace_id, COALESCE(a.host_id, 0) AS host_id, a.name, a.status
		   FROM agent_tokens t
		   JOIN agents a ON a.id = t.agent_id
		  WHERE t.token_hash = ?
		    AND t.status = 'active'
		    AND (t.expires_at IS NULL OR t.expires_at > NOW(3))
		  LIMIT 1`,
		tokenHash,
	).Scan(&identity).Error
	if err != nil {
		return AgentIdentity{}, apperror.Wrap(http.StatusInternalServerError, 500103, "authenticate agent failed", err)
	}
	if identity.ID == 0 {
		return AgentIdentity{}, apperror.New(http.StatusUnauthorized, 401012, "invalid agent token")
	}
	if identity.Status == "disabled" {
		return AgentIdentity{}, apperror.New(http.StatusForbidden, 403101, "agent is disabled")
	}
	if err := s.db.WithContext(ctx).Exec("UPDATE agent_tokens SET last_used_at = NOW(3) WHERE token_hash = ?", tokenHash).Error; err != nil {
		return AgentIdentity{}, apperror.Wrap(http.StatusInternalServerError, 500103, "update agent token usage failed", err)
	}
	return identity, nil
}

func (s *Service) Heartbeat(ctx context.Context, identity AgentIdentity, input HeartbeatInput) (HeartbeatResult, *apperror.Error) {
	input.HostInfoInput = normalizeHostInfo(input.HostInfoInput)
	status := normalizeAgentStatus(input.Status)

	var result HeartbeatResult
	txErr := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		agent, err := s.agentByID(ctx, tx, identity.ID)
		if err != nil {
			return err
		}
		if agent.ID == 0 {
			return apperror.New(http.StatusUnauthorized, 401012, "invalid agent token")
		}
		if agent.Status == "disabled" {
			return apperror.New(http.StatusForbidden, 403101, "agent is disabled")
		}

		host, err := s.ensureHost(ctx, tx, identity.WorkspaceID, input.HostInfoInput, uint64(agent.HostID.Int64))
		if err != nil {
			return err
		}
		res := tx.WithContext(ctx).Exec(
			`UPDATE agents
			    SET host_id = ?, version = ?, status = ?, os_type = ?, arch = ?, ip = ?, last_heartbeat_at = NOW(3), metadata = ?
			  WHERE id = ? AND status <> 'disabled'`,
			host.ID, nullString(input.Version), status, nullString(input.OSType), nullString(input.Arch), nullString(input.IP), jsonNull(input.Metadata), agent.ID,
		)
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return apperror.New(http.StatusForbidden, 403101, "agent is disabled")
		}
		if err := s.writeHeartbeat(ctx, tx, agent.ID, host.ID, status, input.HostInfoInput); err != nil {
			return err
		}
		summary, err := s.agentSummaryByID(ctx, tx, agent.ID)
		if err != nil {
			return err
		}
		hostSummary, err := s.hostSummaryByID(ctx, tx, host.ID)
		if err != nil {
			return err
		}
		result = HeartbeatResult{
			Agent:      summary,
			Host:       hostSummary,
			ReceivedAt: time.Now().Format(time.RFC3339),
		}
		return nil
	})
	if txErr != nil {
		if appErr, ok := txErr.(*apperror.Error); ok {
			return HeartbeatResult{}, appErr
		}
		return HeartbeatResult{}, apperror.Wrap(http.StatusInternalServerError, 500104, "write heartbeat failed", txErr)
	}
	return result, nil
}

func (s *Service) ListAgents(ctx context.Context, input ListInput) (AgentListResult, *apperror.Error) {
	workspace, appErr := s.defaultWorkspaceForAPI(ctx)
	if appErr != nil {
		return AgentListResult{}, appErr
	}
	input = normalizeListInput(input)

	args := []interface{}{workspace.ID}
	where := "WHERE a.workspace_id = ? AND a.deleted_at IS NULL"
	if strings.TrimSpace(input.Keyword) != "" {
		where += " AND (a.name LIKE ? OR a.uid LIKE ? OR h.name LIKE ? OR h.hostname LIKE ? OR a.ip LIKE ?)"
		like := "%" + strings.TrimSpace(input.Keyword) + "%"
		args = append(args, like, like, like, like, like)
	}
	if strings.TrimSpace(input.Status) != "" {
		where += " AND a.status = ?"
		args = append(args, strings.TrimSpace(input.Status))
	}
	var total int64
	if err := s.db.WithContext(ctx).Raw(
		`SELECT COUNT(*)
		   FROM agents a
		   LEFT JOIN hosts h ON h.id = a.host_id
		  `+where,
		args...,
	).Scan(&total).Error; err != nil {
		return AgentListResult{}, apperror.Wrap(http.StatusInternalServerError, 500105, "count agents failed", err)
	}
	queryArgs := append([]interface{}{}, args...)
	queryArgs = append(queryArgs, input.PageSize, (input.Page-1)*input.PageSize)
	var rows []struct {
		AgentUID        string
		AgentName       string
		AgentStatus     string
		TokenPrefix     sql.NullString
		Version         sql.NullString
		AgentIP         sql.NullString
		AgentOS         sql.NullString
		AgentArch       sql.NullString
		LastHeartbeatAt sql.NullString
		DisabledAt      sql.NullString
		AgentCreatedAt  string
		HostUID         sql.NullString
		HostName        sql.NullString
		Hostname        sql.NullString
		HostIP          sql.NullString
		HostOS          sql.NullString
		HostArch        sql.NullString
		HostStatus      sql.NullString
		HostCreatedAt   sql.NullString
	}
	if err := s.db.WithContext(ctx).Raw(
		`SELECT a.uid AS agent_uid, a.name AS agent_name, a.status AS agent_status, a.token_prefix,
		        a.version, a.ip AS agent_ip, a.os_type AS agent_os, a.arch AS agent_arch,
		        DATE_FORMAT(a.last_heartbeat_at, '%Y-%m-%d %H:%i:%s') AS last_heartbeat_at,
		        DATE_FORMAT(a.disabled_at, '%Y-%m-%d %H:%i:%s') AS disabled_at,
		        DATE_FORMAT(a.created_at, '%Y-%m-%d %H:%i:%s') AS agent_created_at,
		        h.uid AS host_uid, h.name AS host_name, h.hostname, h.primary_ip AS host_ip,
		        COALESCE(h.os_name, h.os_type) AS host_os, h.arch AS host_arch, h.status AS host_status,
		        DATE_FORMAT(h.created_at, '%Y-%m-%d %H:%i:%s') AS host_created_at
		   FROM agents a
		   LEFT JOIN hosts h ON h.id = a.host_id
		  `+where+`
		  ORDER BY a.last_heartbeat_at DESC, a.created_at DESC
		  LIMIT ? OFFSET ?`,
		queryArgs...,
	).Scan(&rows).Error; err != nil {
		return AgentListResult{}, apperror.Wrap(http.StatusInternalServerError, 500105, "list agents failed", err)
	}

	agents := make([]AgentSummary, 0, len(rows))
	for _, row := range rows {
		agent := AgentSummary{
			ID:              row.AgentUID,
			Name:            row.AgentName,
			Status:          row.AgentStatus,
			TokenPrefix:     row.TokenPrefix.String,
			Version:         row.Version.String,
			IP:              row.AgentIP.String,
			OS:              row.AgentOS.String,
			Arch:            row.AgentArch.String,
			LastHeartbeatAt: row.LastHeartbeatAt.String,
			DisabledAt:      row.DisabledAt.String,
			CreatedAt:       row.AgentCreatedAt,
		}
		if row.HostUID.Valid {
			agent.Host = &HostSummary{
				ID:        row.HostUID.String,
				Name:      row.HostName.String,
				Hostname:  row.Hostname.String,
				IP:        row.HostIP.String,
				OS:        row.HostOS.String,
				Arch:      row.HostArch.String,
				Status:    row.HostStatus.String,
				CreatedAt: row.HostCreatedAt.String,
			}
		}
		agents = append(agents, agent)
	}
	return AgentListResult{Items: agents, Total: total, Page: input.Page, PageSize: input.PageSize}, nil
}

func (s *Service) ListHosts(ctx context.Context, input ListInput) (HostListResult, *apperror.Error) {
	workspace, appErr := s.defaultWorkspaceForAPI(ctx)
	if appErr != nil {
		return HostListResult{}, appErr
	}
	input = normalizeListInput(input)

	args := []interface{}{workspace.ID}
	where := "WHERE h.workspace_id = ? AND h.deleted_at IS NULL"
	if strings.TrimSpace(input.Keyword) != "" {
		where += " AND (h.name LIKE ? OR h.uid LIKE ? OR h.hostname LIKE ? OR h.primary_ip LIKE ?)"
		like := "%" + strings.TrimSpace(input.Keyword) + "%"
		args = append(args, like, like, like, like)
	}
	if strings.TrimSpace(input.Status) != "" {
		where += " AND h.status = ?"
		args = append(args, strings.TrimSpace(input.Status))
	}
	var total int64
	if err := s.db.WithContext(ctx).Raw("SELECT COUNT(*) FROM hosts h "+where, args...).Scan(&total).Error; err != nil {
		return HostListResult{}, apperror.Wrap(http.StatusInternalServerError, 500106, "count hosts failed", err)
	}
	queryArgs := append([]interface{}{}, args...)
	queryArgs = append(queryArgs, input.PageSize, (input.Page-1)*input.PageSize)
	var rows []struct {
		UID              string
		Name             string
		Hostname         sql.NullString
		IP               sql.NullString
		OS               sql.NullString
		Arch             sql.NullString
		Status           string
		AgentCount       int
		OnlineAgentCount int
		LastHeartbeatAt  sql.NullString
		CreatedAt        string
	}
	if err := s.db.WithContext(ctx).Raw(
		`SELECT h.uid, h.name, h.hostname, h.primary_ip AS ip, COALESCE(h.os_name, h.os_type) AS os, h.arch, h.status,
		        COUNT(a.id) AS agent_count,
		        COALESCE(SUM(CASE WHEN a.status = 'online' THEN 1 ELSE 0 END), 0) AS online_agent_count,
		        DATE_FORMAT(MAX(a.last_heartbeat_at), '%Y-%m-%d %H:%i:%s') AS last_heartbeat_at,
		        DATE_FORMAT(h.created_at, '%Y-%m-%d %H:%i:%s') AS created_at
		   FROM hosts h
		   LEFT JOIN agents a ON a.host_id = h.id AND a.deleted_at IS NULL
		  `+where+`
		  GROUP BY h.id, h.uid, h.name, h.hostname, h.primary_ip, h.os_name, h.os_type, h.arch, h.status, h.created_at
		  ORDER BY MAX(a.last_heartbeat_at) DESC, h.created_at DESC
		  LIMIT ? OFFSET ?`,
		queryArgs...,
	).Scan(&rows).Error; err != nil {
		return HostListResult{}, apperror.Wrap(http.StatusInternalServerError, 500106, "list hosts failed", err)
	}

	hosts := make([]HostSummary, 0, len(rows))
	for _, row := range rows {
		hosts = append(hosts, HostSummary{
			ID:               row.UID,
			Name:             row.Name,
			Hostname:         row.Hostname.String,
			IP:               row.IP.String,
			OS:               row.OS.String,
			Arch:             row.Arch.String,
			Status:           row.Status,
			AgentCount:       row.AgentCount,
			OnlineAgentCount: row.OnlineAgentCount,
			LastHeartbeatAt:  row.LastHeartbeatAt.String,
			CreatedAt:        row.CreatedAt,
		})
	}
	return HostListResult{Items: hosts, Total: total, Page: input.Page, PageSize: input.PageSize}, nil
}

func (s *Service) DisableAgent(ctx context.Context, agentUID, actorUID string) *apperror.Error {
	agentUID = strings.TrimSpace(agentUID)
	if agentUID == "" {
		return apperror.New(http.StatusBadRequest, 400001, "agent id is required")
	}

	txErr := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var row struct {
			ID     uint64
			HostID sql.NullInt64
		}
		if err := tx.WithContext(ctx).Raw("SELECT id, host_id FROM agents WHERE uid = ? AND deleted_at IS NULL LIMIT 1", agentUID).Scan(&row).Error; err != nil {
			return err
		}
		if row.ID == 0 {
			return apperror.New(http.StatusNotFound, 404101, "agent not found")
		}

		actorID, err := s.userIDByUID(ctx, tx, actorUID)
		if err != nil {
			return err
		}
		res := tx.WithContext(ctx).Exec(
			"UPDATE agents SET status = 'disabled', disabled_at = NOW(3), disabled_by = ? WHERE id = ?",
			actorID, row.ID,
		)
		if res.Error != nil {
			return res.Error
		}
		if err := tx.WithContext(ctx).Exec(
			"UPDATE agent_tokens SET status = 'revoked', revoked_at = NOW(3) WHERE agent_id = ? AND status = 'active'",
			row.ID,
		).Error; err != nil {
			return err
		}
		if row.HostID.Valid {
			if err := s.refreshHostStatus(ctx, tx, uint64(row.HostID.Int64)); err != nil {
				return err
			}
		}
		audit.Write(ctx, tx, audit.Event{
			ActorUserID:  actorID,
			Action:       "agent.disable",
			ResourceType: "agent",
			ResourceID:   sql.NullInt64{Int64: int64(row.ID), Valid: true},
		})
		return nil
	})
	if txErr != nil {
		if appErr, ok := txErr.(*apperror.Error); ok {
			return appErr
		}
		return apperror.Wrap(http.StatusInternalServerError, 500107, "disable agent failed", txErr)
	}
	return nil
}

func (s *Service) RevokeAgentToken(ctx context.Context, agentUID, actorUID string) *apperror.Error {
	agentUID = strings.TrimSpace(agentUID)
	if agentUID == "" {
		return apperror.New(http.StatusBadRequest, 400001, "agent id is required")
	}
	txErr := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var row struct {
			ID uint64
		}
		if err := tx.WithContext(ctx).Raw("SELECT id FROM agents WHERE uid = ? AND deleted_at IS NULL LIMIT 1", agentUID).Scan(&row).Error; err != nil {
			return err
		}
		if row.ID == 0 {
			return apperror.New(http.StatusNotFound, 404101, "agent not found")
		}
		actorID, err := s.userIDByUID(ctx, tx, actorUID)
		if err != nil {
			return err
		}
		if err := tx.WithContext(ctx).Exec(
			"UPDATE agent_tokens SET status = 'revoked', revoked_at = NOW(3) WHERE agent_id = ? AND status = 'active'",
			row.ID,
		).Error; err != nil {
			return err
		}
		audit.Write(ctx, tx, audit.Event{
			ActorUserID:  actorID,
			Action:       "agent.token.revoke",
			ResourceType: "agent",
			ResourceID:   sql.NullInt64{Int64: int64(row.ID), Valid: true},
		})
		return nil
	})
	if txErr != nil {
		if appErr, ok := txErr.(*apperror.Error); ok {
			return appErr
		}
		return apperror.Wrap(http.StatusInternalServerError, 500109, "revoke agent token failed", txErr)
	}
	return nil
}

func (s *Service) ListEnrollmentTokens(ctx context.Context) ([]EnrollmentTokenSummary, *apperror.Error) {
	workspace, appErr := s.defaultWorkspaceForAPI(ctx)
	if appErr != nil {
		return nil, appErr
	}
	var rows []struct {
		UID               string
		TokenPrefix       sql.NullString
		Status            string
		MaxUses           uint
		UsedCount         uint
		BindWorkspaceSlug sql.NullString
		ExpiresAt         string
		CreatedBy         sql.NullString
		CreatedAt         string
	}
	err := s.db.WithContext(ctx).Raw(
		`SELECT e.uid, e.token_prefix, e.status, e.max_uses, e.used_count, e.bind_workspace_slug,
		        DATE_FORMAT(e.expires_at, '%Y-%m-%d %H:%i:%s') AS expires_at,
		        u.username AS created_by,
		        DATE_FORMAT(e.created_at, '%Y-%m-%d %H:%i:%s') AS created_at
		   FROM agent_enrollment_tokens e
		   LEFT JOIN users u ON u.id = e.created_by
		  WHERE e.workspace_id = ?
		  ORDER BY e.created_at DESC
		  LIMIT 100`,
		workspace.ID,
	).Scan(&rows).Error
	if err != nil {
		return nil, apperror.Wrap(http.StatusInternalServerError, 500110, "list enrollment tokens failed", err)
	}
	out := make([]EnrollmentTokenSummary, 0, len(rows))
	for _, row := range rows {
		out = append(out, EnrollmentTokenSummary{
			ID:                row.UID,
			TokenPrefix:       row.TokenPrefix.String,
			Status:            row.Status,
			MaxUses:           row.MaxUses,
			UsedCount:         row.UsedCount,
			BindWorkspaceSlug: row.BindWorkspaceSlug.String,
			ExpiresAt:         row.ExpiresAt,
			CreatedBy:         row.CreatedBy.String,
			CreatedAt:         row.CreatedAt,
		})
	}
	return out, nil
}

func (s *Service) CreateEnrollmentToken(ctx context.Context, input CreateEnrollmentTokenInput) (EnrollmentTokenDetail, *apperror.Error) {
	if input.MaxUses == 0 {
		input.MaxUses = 1
	}
	if input.ExpiresInSeconds == 0 {
		input.ExpiresInSeconds = 3600
	}
	token, tokenHash, tokenPrefix, err := newEnrollmentToken()
	if err != nil {
		return EnrollmentTokenDetail{}, apperror.Wrap(http.StatusInternalServerError, 500111, "issue enrollment token failed", err)
	}
	var created EnrollmentTokenDetail
	txErr := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		workspace, err := s.defaultWorkspace(ctx, tx)
		if err != nil {
			return err
		}
		actorID, err := s.userIDByUID(ctx, tx, input.Audit.ActorUID)
		if err != nil {
			return err
		}
		enrollmentUID, err := newUID()
		if err != nil {
			return err
		}
		expiresAt := time.Now().Add(time.Duration(input.ExpiresInSeconds) * time.Second)
		if err := tx.WithContext(ctx).Exec(
			`INSERT INTO agent_enrollment_tokens(uid, workspace_id, token_hash, token_prefix, status, max_uses, used_count, bind_workspace_slug, expires_at, created_by)
			 VALUES (?, ?, ?, ?, 'active', ?, 0, ?, ?, ?)`,
			enrollmentUID, workspace.ID, tokenHash, tokenPrefix, input.MaxUses, nullString(input.BindWorkspaceSlug), expiresAt, actorID,
		).Error; err != nil {
			return err
		}
		created = EnrollmentTokenDetail{
			EnrollmentTokenSummary: EnrollmentTokenSummary{
				ID:                enrollmentUID,
				TokenPrefix:       tokenPrefix,
				Status:            "active",
				MaxUses:           input.MaxUses,
				UsedCount:         0,
				BindWorkspaceSlug: strings.TrimSpace(input.BindWorkspaceSlug),
				ExpiresAt:         expiresAt.Format("2006-01-02 15:04:05"),
			},
			Token: token,
		}
		audit.Write(ctx, tx, audit.Event{
			WorkspaceID:   workspace.ID,
			ActorUserID:   actorID,
			Action:        "agent.enrollment.create",
			ResourceType:  "agent_enrollment_token",
			After:         map[string]interface{}{"uid": enrollmentUID, "tokenPrefix": tokenPrefix, "maxUses": input.MaxUses},
			IP:            input.Audit.IP,
			UserAgent:     input.Audit.UserAgent,
			TraceID:       input.Audit.TraceID,
			RequestMethod: input.Audit.RequestMethod,
			RequestPath:   input.Audit.RequestPath,
		})
		return nil
	})
	if txErr != nil {
		return EnrollmentTokenDetail{}, apperror.Wrap(http.StatusInternalServerError, 500112, "create enrollment token failed", txErr)
	}
	return created, nil
}

func (s *Service) RevokeEnrollmentToken(ctx context.Context, uidValue, actorUID string) *apperror.Error {
	uidValue = strings.TrimSpace(uidValue)
	if uidValue == "" {
		return apperror.New(http.StatusBadRequest, 400110, "enrollment token id is required")
	}
	txErr := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		workspace, err := s.defaultWorkspace(ctx, tx)
		if err != nil {
			return err
		}
		actorID, err := s.userIDByUID(ctx, tx, actorUID)
		if err != nil {
			return err
		}
		var row struct{ ID uint64 }
		if err := tx.WithContext(ctx).Raw("SELECT id FROM agent_enrollment_tokens WHERE workspace_id = ? AND uid = ? LIMIT 1", workspace.ID, uidValue).Scan(&row).Error; err != nil {
			return err
		}
		if row.ID == 0 {
			return apperror.New(http.StatusNotFound, 404110, "enrollment token not found")
		}
		if err := tx.WithContext(ctx).Exec("UPDATE agent_enrollment_tokens SET status = 'revoked', revoked_at = NOW(3) WHERE id = ?", row.ID).Error; err != nil {
			return err
		}
		audit.Write(ctx, tx, audit.Event{
			WorkspaceID:  workspace.ID,
			ActorUserID:  actorID,
			Action:       "agent.enrollment.revoke",
			ResourceType: "agent_enrollment_token",
			ResourceID:   sql.NullInt64{Int64: int64(row.ID), Valid: true},
		})
		return nil
	})
	if txErr != nil {
		if appErr, ok := txErr.(*apperror.Error); ok {
			return appErr
		}
		return apperror.Wrap(http.StatusInternalServerError, 500113, "revoke enrollment token failed", txErr)
	}
	return nil
}

func (s *Service) MarkOffline(ctx context.Context) (OfflineScanResult, *apperror.Error) {
	workspace, appErr := s.defaultWorkspaceForAPI(ctx)
	if appErr != nil {
		return OfflineScanResult{}, appErr
	}
	threshold := s.offlineThreshold()
	seconds := int(threshold.Seconds())

	agentRes := s.db.WithContext(ctx).Exec(
		`UPDATE agents
		    SET status = 'offline'
		  WHERE workspace_id = ?
		    AND status = 'online'
		    AND last_heartbeat_at IS NOT NULL
		    AND last_heartbeat_at < DATE_SUB(NOW(3), INTERVAL ? SECOND)`,
		workspace.ID, seconds,
	)
	if agentRes.Error != nil {
		return OfflineScanResult{}, apperror.Wrap(http.StatusInternalServerError, 500108, "mark agents offline failed", agentRes.Error)
	}
	hostRes := s.db.WithContext(ctx).Exec(
		`UPDATE hosts h
		    SET h.status = CASE
		      WHEN EXISTS (SELECT 1 FROM agents a WHERE a.host_id = h.id AND a.status IN ('online', 'upgrading') AND a.deleted_at IS NULL) THEN 'online'
		      WHEN EXISTS (SELECT 1 FROM agents a WHERE a.host_id = h.id AND a.status IN ('registered', 'offline') AND a.deleted_at IS NULL) THEN 'offline'
		      WHEN EXISTS (SELECT 1 FROM agents a WHERE a.host_id = h.id AND a.status = 'disabled' AND a.deleted_at IS NULL) THEN 'disabled'
		      ELSE 'unknown'
		    END
		  WHERE h.workspace_id = ?`,
		workspace.ID,
	)
	if hostRes.Error != nil {
		return OfflineScanResult{}, apperror.Wrap(http.StatusInternalServerError, 500108, "mark hosts offline failed", hostRes.Error)
	}
	return OfflineScanResult{
		OfflineAgents:    agentRes.RowsAffected,
		OfflineHosts:     hostRes.RowsAffected,
		ThresholdSeconds: seconds,
	}, nil
}

func (s *Service) refreshHostStatus(ctx context.Context, tx *gorm.DB, hostID uint64) error {
	return tx.WithContext(ctx).Exec(
		`UPDATE hosts h
		    SET h.status = CASE
		      WHEN EXISTS (SELECT 1 FROM agents a WHERE a.host_id = h.id AND a.status IN ('online', 'upgrading') AND a.deleted_at IS NULL) THEN 'online'
		      WHEN EXISTS (SELECT 1 FROM agents a WHERE a.host_id = h.id AND a.status IN ('registered', 'offline') AND a.deleted_at IS NULL) THEN 'offline'
		      WHEN EXISTS (SELECT 1 FROM agents a WHERE a.host_id = h.id AND a.status = 'disabled' AND a.deleted_at IS NULL) THEN 'disabled'
		      ELSE 'unknown'
		    END
		  WHERE h.id = ?`,
		hostID,
	).Error
}

func (s *Service) consumeEnrollmentToken(ctx context.Context, tx *gorm.DB, workspaceID uint64, tokenHash string) error {
	var row struct {
		ID                uint64
		Status            string
		MaxUses           uint
		UsedCount         uint
		BindWorkspaceSlug sql.NullString
	}
	if err := tx.WithContext(ctx).Raw(
		`SELECT id, status, max_uses, used_count, bind_workspace_slug
		   FROM agent_enrollment_tokens
		  WHERE workspace_id = ?
		    AND token_hash = ?
		    AND status = 'active'
		    AND expires_at > NOW(3)
		  LIMIT 1
		  FOR UPDATE`,
		workspaceID, tokenHash,
	).Scan(&row).Error; err != nil {
		return err
	}
	if row.ID == 0 {
		return apperror.New(http.StatusUnauthorized, 401013, "invalid enrollment token")
	}
	nextUsed := row.UsedCount + 1
	nextStatus := "active"
	if nextUsed >= row.MaxUses {
		nextStatus = "used"
	}
	return tx.WithContext(ctx).Exec(
		"UPDATE agent_enrollment_tokens SET used_count = ?, status = ? WHERE id = ?",
		nextUsed, nextStatus, row.ID,
	).Error
}

func (s *Service) defaultWorkspaceForAPI(ctx context.Context) (workspaceRecord, *apperror.Error) {
	workspace, err := s.defaultWorkspace(ctx, s.db)
	if err != nil {
		return workspaceRecord{}, apperror.Wrap(http.StatusInternalServerError, 500102, "load workspace failed", err)
	}
	if workspace.ID == 0 {
		return workspaceRecord{}, apperror.New(http.StatusInternalServerError, 500102, "default workspace is not initialized")
	}
	return workspace, nil
}

func (s *Service) defaultWorkspace(ctx context.Context, db *gorm.DB) (workspaceRecord, error) {
	var workspace workspaceRecord
	err := db.WithContext(ctx).Raw(
		"SELECT id, uid, name, slug FROM workspaces WHERE slug = ? AND status = 'active' LIMIT 1",
		s.cfg.Bootstrap.WorkspaceSlug,
	).Scan(&workspace).Error
	return workspace, err
}

func (s *Service) ensureHost(ctx context.Context, tx *gorm.DB, workspaceID uint64, input HostInfoInput, existingHostID uint64) (hostRecord, error) {
	name := input.Hostname
	if name == "" {
		name = input.Name
	}
	if name == "" {
		name = "unknown"
	}
	name = limit(name, 128)

	if existingHostID != 0 {
		if err := tx.WithContext(ctx).Exec(
			`UPDATE hosts
			    SET name = ?, hostname = ?, primary_ip = ?, os_type = ?, os_name = ?, os_version = ?, arch = ?,
			        status = CASE WHEN status = 'disabled' THEN 'disabled' ELSE 'online' END,
			        metadata = ?
			  WHERE id = ?`,
			name, nullString(input.Hostname), nullString(input.IP), nullString(input.OSType), nullString(input.OSName), nullString(input.OSVersion), nullString(input.Arch), jsonNull(input.Metadata), existingHostID,
		).Error; err != nil {
			return hostRecord{}, err
		}
		host, err := s.hostByID(ctx, tx, existingHostID)
		if err != nil || host.ID != 0 {
			return host, err
		}
	}

	uid, err := newUID()
	if err != nil {
		return hostRecord{}, err
	}
	if err := tx.WithContext(ctx).Exec(
		`INSERT INTO hosts(uid, workspace_id, name, hostname, primary_ip, os_type, os_name, os_version, arch, status, metadata)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, 'online', ?)
		 ON DUPLICATE KEY UPDATE
		   hostname = VALUES(hostname),
		   primary_ip = VALUES(primary_ip),
		   os_type = VALUES(os_type),
		   os_name = VALUES(os_name),
		   os_version = VALUES(os_version),
		   arch = VALUES(arch),
		   status = CASE WHEN status = 'disabled' THEN 'disabled' ELSE 'online' END,
		   metadata = VALUES(metadata)`,
		uid, workspaceID, name, nullString(input.Hostname), nullString(input.IP), nullString(input.OSType), nullString(input.OSName), nullString(input.OSVersion), nullString(input.Arch), jsonNull(input.Metadata),
	).Error; err != nil {
		return hostRecord{}, err
	}
	return s.hostByName(ctx, tx, workspaceID, name)
}

func (s *Service) writeHeartbeat(ctx context.Context, tx *gorm.DB, agentID, hostID uint64, status string, input HostInfoInput) error {
	return tx.WithContext(ctx).Exec(
		"INSERT INTO agent_heartbeats(agent_id, host_id, status, ip, version, payload) VALUES (?, ?, ?, ?, ?, ?)",
		agentID, hostID, status, nullString(input.IP), nullString(input.Version), jsonNull(input),
	).Error
}

func (s *Service) agentByName(ctx context.Context, db *gorm.DB, workspaceID uint64, name string) (agentRecord, error) {
	var agent agentRecord
	err := db.WithContext(ctx).Raw(
		`SELECT id, uid, workspace_id, host_id, name, token_prefix, version, status, os_type, arch, ip,
		        DATE_FORMAT(last_heartbeat_at, '%Y-%m-%d %H:%i:%s') AS last_heartbeat_at,
		        DATE_FORMAT(disabled_at, '%Y-%m-%d %H:%i:%s') AS disabled_at,
		        DATE_FORMAT(created_at, '%Y-%m-%d %H:%i:%s') AS created_at
		   FROM agents
		  WHERE workspace_id = ? AND name = ? AND deleted_at IS NULL
		  LIMIT 1`,
		workspaceID, name,
	).Scan(&agent).Error
	return agent, err
}

func (s *Service) agentByID(ctx context.Context, db *gorm.DB, id uint64) (agentRecord, error) {
	var agent agentRecord
	err := db.WithContext(ctx).Raw(
		`SELECT id, uid, workspace_id, host_id, name, token_prefix, version, status, os_type, arch, ip,
		        DATE_FORMAT(last_heartbeat_at, '%Y-%m-%d %H:%i:%s') AS last_heartbeat_at,
		        DATE_FORMAT(disabled_at, '%Y-%m-%d %H:%i:%s') AS disabled_at,
		        DATE_FORMAT(created_at, '%Y-%m-%d %H:%i:%s') AS created_at
		   FROM agents
		  WHERE id = ? AND deleted_at IS NULL
		  LIMIT 1`,
		id,
	).Scan(&agent).Error
	return agent, err
}

func (s *Service) hostByID(ctx context.Context, db *gorm.DB, id uint64) (hostRecord, error) {
	var host hostRecord
	err := db.WithContext(ctx).Raw(
		`SELECT id, uid, name, hostname, primary_ip, os_type, os_name, os_version, arch, status,
		        DATE_FORMAT(created_at, '%Y-%m-%d %H:%i:%s') AS created_at
		   FROM hosts
		  WHERE id = ? AND deleted_at IS NULL
		  LIMIT 1`,
		id,
	).Scan(&host).Error
	return host, err
}

func (s *Service) hostByName(ctx context.Context, db *gorm.DB, workspaceID uint64, name string) (hostRecord, error) {
	var host hostRecord
	err := db.WithContext(ctx).Raw(
		`SELECT id, uid, name, hostname, primary_ip, os_type, os_name, os_version, arch, status,
		        DATE_FORMAT(created_at, '%Y-%m-%d %H:%i:%s') AS created_at
		   FROM hosts
		  WHERE workspace_id = ? AND name = ? AND deleted_at IS NULL
		  LIMIT 1`,
		workspaceID, name,
	).Scan(&host).Error
	return host, err
}

func (s *Service) agentSummaryByID(ctx context.Context, db *gorm.DB, id uint64) (AgentSummary, error) {
	agent, err := s.agentByID(ctx, db, id)
	if err != nil {
		return AgentSummary{}, err
	}
	summary := agentSummaryFrom(agent)
	if agent.HostID.Valid {
		host, err := s.hostSummaryByID(ctx, db, uint64(agent.HostID.Int64))
		if err != nil {
			return AgentSummary{}, err
		}
		summary.Host = &host
	}
	return summary, nil
}

func (s *Service) hostSummaryByID(ctx context.Context, db *gorm.DB, id uint64) (HostSummary, error) {
	host, err := s.hostByID(ctx, db, id)
	if err != nil {
		return HostSummary{}, err
	}
	return hostSummaryFrom(host), nil
}

func (s *Service) userIDByUID(ctx context.Context, db *gorm.DB, uid string) (sql.NullInt64, error) {
	uid = strings.TrimSpace(uid)
	if uid == "" {
		return sql.NullInt64{}, nil
	}
	var id uint64
	if err := db.WithContext(ctx).Raw("SELECT id FROM users WHERE uid = ? LIMIT 1", uid).Scan(&id).Error; err != nil {
		return sql.NullInt64{}, err
	}
	if id == 0 {
		return sql.NullInt64{}, nil
	}
	return sql.NullInt64{Int64: int64(id), Valid: true}, nil
}

func (s *Service) offlineThreshold() time.Duration {
	threshold := 3 * s.cfg.Agent.HeartbeatInterval
	if threshold < 90*time.Second {
		return 90 * time.Second
	}
	return threshold
}

func normalizeListInput(input ListInput) ListInput {
	if input.Page <= 0 {
		input.Page = 1
	}
	if input.PageSize <= 0 {
		input.PageSize = 20
	}
	if input.PageSize > 100 {
		input.PageSize = 100
	}
	input.Keyword = strings.TrimSpace(input.Keyword)
	input.Status = strings.TrimSpace(input.Status)
	return input
}

func agentSummaryFrom(agent agentRecord) AgentSummary {
	return AgentSummary{
		ID:              agent.UID,
		Name:            agent.Name,
		Status:          agent.Status,
		TokenPrefix:     agent.TokenPrefix.String,
		Version:         agent.Version.String,
		IP:              agent.IP.String,
		OS:              agent.OSType.String,
		Arch:            agent.Arch.String,
		LastHeartbeatAt: agent.LastHeartbeatAt.String,
		DisabledAt:      agent.DisabledAt.String,
		CreatedAt:       agent.CreatedAt,
	}
}

func hostSummaryFrom(host hostRecord) HostSummary {
	osName := host.OSName.String
	if osName == "" {
		osName = host.OSType.String
	}
	return HostSummary{
		ID:        host.UID,
		Name:      host.Name,
		Hostname:  host.Hostname.String,
		IP:        host.PrimaryIP.String,
		OS:        osName,
		Arch:      host.Arch.String,
		Status:    host.Status,
		CreatedAt: host.CreatedAt,
	}
}

func normalizeRegisterInput(input RegisterInput) RegisterInput {
	input.BootstrapSecret = strings.TrimSpace(input.BootstrapSecret)
	input.HostInfoInput = normalizeHostInfo(input.HostInfoInput)
	return input
}

func normalizeHostInfo(input HostInfoInput) HostInfoInput {
	input.Name = limit(strings.TrimSpace(input.Name), 128)
	input.Hostname = limit(strings.TrimSpace(input.Hostname), 255)
	input.IP = limit(strings.TrimSpace(input.IP), 45)
	input.OS = limit(strings.TrimSpace(input.OS), 128)
	input.OSType = limit(strings.TrimSpace(input.OSType), 64)
	input.OSName = limit(strings.TrimSpace(input.OSName), 128)
	input.OSVersion = limit(strings.TrimSpace(input.OSVersion), 128)
	input.Arch = limit(strings.TrimSpace(input.Arch), 64)
	input.Version = limit(strings.TrimSpace(input.Version), 64)
	if input.OSType == "" {
		input.OSType = input.OS
	}
	if input.OSName == "" {
		input.OSName = input.OS
	}
	return input
}

func normalizeAgentStatus(status string) string {
	status = strings.TrimSpace(status)
	if status == "upgrading" {
		return status
	}
	return "online"
}

func newUID() (string, error) {
	const alphabet = "0123456789ABCDEFGHJKMNPQRSTVWXYZ"
	bytes := make([]byte, 26)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}

	out := make([]byte, 26)
	for i, value := range bytes {
		out[i] = alphabet[int(value)%len(alphabet)]
	}
	return string(out), nil
}

func newAgentToken() (token string, tokenHash string, tokenPrefix string, err error) {
	bytes := make([]byte, 32)
	if _, err = rand.Read(bytes); err != nil {
		return "", "", "", err
	}
	token = "opagt_" + base64.RawURLEncoding.EncodeToString(bytes)
	tokenHash = hashToken(token)
	tokenPrefix = limit(token, 16)
	return token, tokenHash, tokenPrefix, nil
}

func newEnrollmentToken() (token string, tokenHash string, tokenPrefix string, err error) {
	bytes := make([]byte, 32)
	if _, err = rand.Read(bytes); err != nil {
		return "", "", "", err
	}
	token = "openr_" + base64.RawURLEncoding.EncodeToString(bytes)
	tokenHash = hashToken(token)
	tokenPrefix = limit(token, 16)
	return token, tokenHash, tokenPrefix, nil
}

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
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

func limit(value string, max int) string {
	if len(value) <= max {
		return value
	}
	return value[:max]
}
