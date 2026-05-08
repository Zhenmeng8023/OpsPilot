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
	"strconv"
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

type MaintenanceWindowInput struct {
	Name      string `json:"name"`
	ScopeType string `json:"scopeType"`
	AgentID   string `json:"agentId"`
	HostID    string `json:"hostId"`
	Reason    string `json:"reason"`
	StartsAt  string `json:"startsAt"`
	EndsAt    string `json:"endsAt"`
	Status    string `json:"status"`
	Audit     AuditContext
}

type TagInput struct {
	Name  string `json:"name"`
	Color string `json:"color"`
	Audit AuditContext
}

type ResourceTagsInput struct {
	ResourceType string
	ResourceID   string
	TagIDs       []string `json:"tagIds"`
	Audit        AuditContext
}

type HostGroupInput struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	HostIDs     []string `json:"hostIds"`
	Audit       AuditContext
}

type HostGroupMembersInput struct {
	HostIDs []string `json:"hostIds"`
	Audit   AuditContext
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
	Tags            []TagSummary `json:"tags,omitempty"`
}

type HostSummary struct {
	ID               string       `json:"id"`
	Name             string       `json:"name"`
	Hostname         string       `json:"hostname,omitempty"`
	IP               string       `json:"ip,omitempty"`
	OS               string       `json:"os,omitempty"`
	Arch             string       `json:"arch,omitempty"`
	Status           string       `json:"status"`
	AgentCount       int          `json:"agentCount,omitempty"`
	OnlineAgentCount int          `json:"onlineAgentCount,omitempty"`
	LastHeartbeatAt  string       `json:"lastHeartbeatAt,omitempty"`
	CreatedAt        string       `json:"createdAt"`
	Tags             []TagSummary `json:"tags,omitempty"`
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

type AgentDiagnosticSummary struct {
	ID           string `json:"id"`
	AgentID      string `json:"agentId"`
	AgentName    string `json:"agentName"`
	HostID       string `json:"hostId,omitempty"`
	HostName     string `json:"hostName,omitempty"`
	Version      string `json:"version,omitempty"`
	OS           string `json:"os,omitempty"`
	OSVersion    string `json:"osVersion,omitempty"`
	Arch         string `json:"arch,omitempty"`
	IP           string `json:"ip,omitempty"`
	RunningTasks int    `json:"runningTasks,omitempty"`
	Payload      string `json:"payload,omitempty"`
	ReportedAt   string `json:"reportedAt"`
}

type MaintenanceWindowSummary struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	ScopeType string `json:"scopeType"`
	AgentID   string `json:"agentId,omitempty"`
	AgentName string `json:"agentName,omitempty"`
	HostID    string `json:"hostId,omitempty"`
	HostName  string `json:"hostName,omitempty"`
	Reason    string `json:"reason,omitempty"`
	StartsAt  string `json:"startsAt"`
	EndsAt    string `json:"endsAt"`
	Status    string `json:"status"`
	CreatedBy string `json:"createdBy,omitempty"`
	CreatedAt string `json:"createdAt"`
	UpdatedAt string `json:"updatedAt"`
}

type TagSummary struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Color      string `json:"color,omitempty"`
	UsageCount int    `json:"usageCount,omitempty"`
	CreatedAt  string `json:"createdAt,omitempty"`
}

type HostGroupSummary struct {
	ID          string        `json:"id"`
	Name        string        `json:"name"`
	Description string        `json:"description,omitempty"`
	HostCount   int           `json:"hostCount"`
	Hosts       []HostSummary `json:"hosts,omitempty"`
	CreatedBy   string        `json:"createdBy,omitempty"`
	CreatedAt   string        `json:"createdAt"`
	UpdatedAt   string        `json:"updatedAt"`
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

type maintenanceWindowRecord struct {
	UID       string
	Name      string
	ScopeType string
	AgentUID  sql.NullString
	AgentName sql.NullString
	HostUID   sql.NullString
	HostName  sql.NullString
	Reason    sql.NullString
	StartsAt  string
	EndsAt    string
	Status    string
	CreatedBy sql.NullString
	CreatedAt string
	UpdatedAt string
}

type tagRecord struct {
	ID         uint64
	Name       string
	Color      sql.NullString
	UsageCount int
	CreatedAt  sql.NullString
}

type hostGroupRecord struct {
	ID          uint64
	UID         string
	Name        string
	Description sql.NullString
	HostCount   int
	CreatedBy   sql.NullString
	CreatedAt   string
	UpdatedAt   string
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
		if err := s.writeDiagnostic(ctx, tx, identity.WorkspaceID, agent.ID, host.ID, input.HostInfoInput); err != nil {
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

	agentUIDs := make([]string, 0, len(rows))
	for _, row := range rows {
		agentUIDs = append(agentUIDs, row.AgentUID)
	}
	tagsByAgent, err := s.tagsByResourceUIDs(ctx, s.db, workspace.ID, "agent", agentUIDs)
	if err != nil {
		return AgentListResult{}, apperror.Wrap(http.StatusInternalServerError, 500117, "list agent tags failed", err)
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
			Tags:            tagsByAgent[row.AgentUID],
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

	hostUIDs := make([]string, 0, len(rows))
	for _, row := range rows {
		hostUIDs = append(hostUIDs, row.UID)
	}
	tagsByHost, err := s.tagsByResourceUIDs(ctx, s.db, workspace.ID, "host", hostUIDs)
	if err != nil {
		return HostListResult{}, apperror.Wrap(http.StatusInternalServerError, 500118, "list host tags failed", err)
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
			Tags:             tagsByHost[row.UID],
		})
	}
	return HostListResult{Items: hosts, Total: total, Page: input.Page, PageSize: input.PageSize}, nil
}

func (s *Service) ListDiagnostics(ctx context.Context) ([]AgentDiagnosticSummary, *apperror.Error) {
	workspace, appErr := s.defaultWorkspaceForAPI(ctx)
	if appErr != nil {
		return nil, appErr
	}
	var rows []struct {
		UID          string
		AgentUID     string
		AgentName    string
		HostUID      sql.NullString
		HostName     sql.NullString
		Version      sql.NullString
		OSName       sql.NullString
		OSVersion    sql.NullString
		Arch         sql.NullString
		IP           sql.NullString
		RunningTasks sql.NullInt64
		Payload      sql.NullString
		ReportedAt   string
	}
	if err := s.db.WithContext(ctx).Raw(
		`SELECT d.uid, a.uid AS agent_uid, a.name AS agent_name, h.uid AS host_uid, h.name AS host_name,
		        d.version, d.os_name, d.os_version, d.arch, d.ip, d.running_tasks, d.payload,
		        DATE_FORMAT(d.reported_at, '%Y-%m-%d %H:%i:%s') AS reported_at
		   FROM agent_diagnostics d
		   JOIN agents a ON a.id = d.agent_id
		   LEFT JOIN hosts h ON h.id = d.host_id
		   JOIN (
		     SELECT agent_id, MAX(reported_at) AS reported_at
		       FROM agent_diagnostics
		      WHERE workspace_id = ?
		      GROUP BY agent_id
		   ) latest ON latest.agent_id = d.agent_id AND latest.reported_at = d.reported_at
		  WHERE d.workspace_id = ?
		  ORDER BY d.reported_at DESC
		  LIMIT 200`,
		workspace.ID, workspace.ID,
	).Scan(&rows).Error; err != nil {
		return nil, apperror.Wrap(http.StatusInternalServerError, 500114, "list agent diagnostics failed", err)
	}
	out := make([]AgentDiagnosticSummary, 0, len(rows))
	for _, row := range rows {
		out = append(out, AgentDiagnosticSummary{
			ID:           row.UID,
			AgentID:      row.AgentUID,
			AgentName:    row.AgentName,
			HostID:       row.HostUID.String,
			HostName:     row.HostName.String,
			Version:      row.Version.String,
			OS:           row.OSName.String,
			OSVersion:    row.OSVersion.String,
			Arch:         row.Arch.String,
			IP:           row.IP.String,
			RunningTasks: int(row.RunningTasks.Int64),
			Payload:      row.Payload.String,
			ReportedAt:   row.ReportedAt,
		})
	}
	return out, nil
}

func (s *Service) ListMaintenanceWindows(ctx context.Context) ([]MaintenanceWindowSummary, *apperror.Error) {
	workspace, appErr := s.defaultWorkspaceForAPI(ctx)
	if appErr != nil {
		return nil, appErr
	}
	rows, err := s.maintenanceWindows(ctx, s.db, workspace.ID, "")
	if err != nil {
		return nil, apperror.Wrap(http.StatusInternalServerError, 500115, "list maintenance windows failed", err)
	}
	out := make([]MaintenanceWindowSummary, 0, len(rows))
	for _, row := range rows {
		out = append(out, maintenanceWindowSummary(row))
	}
	return out, nil
}

func (s *Service) CreateMaintenanceWindow(ctx context.Context, input MaintenanceWindowInput) (MaintenanceWindowSummary, *apperror.Error) {
	return s.upsertMaintenanceWindow(ctx, "", input)
}

func (s *Service) UpdateMaintenanceWindow(ctx context.Context, windowUID string, input MaintenanceWindowInput) (MaintenanceWindowSummary, *apperror.Error) {
	windowUID = strings.TrimSpace(windowUID)
	if windowUID == "" {
		return MaintenanceWindowSummary{}, apperror.New(http.StatusBadRequest, 400114, "maintenance window id is required")
	}
	return s.upsertMaintenanceWindow(ctx, windowUID, input)
}

func (s *Service) ListTags(ctx context.Context) ([]TagSummary, *apperror.Error) {
	workspace, appErr := s.defaultWorkspaceForAPI(ctx)
	if appErr != nil {
		return nil, appErr
	}
	rows, err := s.tags(ctx, s.db, workspace.ID, 0)
	if err != nil {
		return nil, apperror.Wrap(http.StatusInternalServerError, 500119, "list tags failed", err)
	}
	out := make([]TagSummary, 0, len(rows))
	for _, row := range rows {
		out = append(out, tagSummary(row))
	}
	return out, nil
}

func (s *Service) CreateTag(ctx context.Context, input TagInput) (TagSummary, *apperror.Error) {
	return s.upsertTag(ctx, "", input)
}

func (s *Service) UpdateTag(ctx context.Context, tagID string, input TagInput) (TagSummary, *apperror.Error) {
	if strings.TrimSpace(tagID) == "" {
		return TagSummary{}, apperror.New(http.StatusBadRequest, 400117, "tag id is required")
	}
	return s.upsertTag(ctx, tagID, input)
}

func (s *Service) SetResourceTags(ctx context.Context, input ResourceTagsInput) ([]TagSummary, *apperror.Error) {
	resourceType := normalizeResourceType(input.ResourceType)
	if resourceType == "" {
		return nil, apperror.New(http.StatusBadRequest, 400118, "resource type must be agent or host")
	}
	resourceUID := strings.TrimSpace(input.ResourceID)
	if resourceUID == "" {
		return nil, apperror.New(http.StatusBadRequest, 400119, "resource id is required")
	}
	tagIDs, appErr := parseTagIDs(input.TagIDs)
	if appErr != nil {
		return nil, appErr
	}
	var saved []TagSummary
	txErr := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		workspace, err := s.defaultWorkspace(ctx, tx)
		if err != nil {
			return err
		}
		actorID, err := s.userIDByUID(ctx, tx, input.Audit.ActorUID)
		if err != nil {
			return err
		}
		resourceID, err := s.resourceIDByUID(ctx, tx, workspace.ID, resourceType, resourceUID)
		if err != nil {
			return err
		}
		if err := s.ensureTagsBelongToWorkspace(ctx, tx, workspace.ID, tagIDs); err != nil {
			return err
		}
		if err := tx.WithContext(ctx).Exec(
			"DELETE FROM resource_tags WHERE workspace_id = ? AND resource_type = ? AND resource_id = ?",
			workspace.ID, resourceType, resourceID,
		).Error; err != nil {
			return err
		}
		for _, tagID := range tagIDs {
			if err := tx.WithContext(ctx).Exec(
				"INSERT INTO resource_tags(workspace_id, tag_id, resource_type, resource_id) VALUES (?, ?, ?, ?)",
				workspace.ID, tagID, resourceType, resourceID,
			).Error; err != nil {
				return err
			}
		}
		tags, err := s.tagsForResource(ctx, tx, workspace.ID, resourceType, resourceID)
		if err != nil {
			return err
		}
		saved = tags
		audit.Write(ctx, tx, audit.Event{
			WorkspaceID:   workspace.ID,
			ActorUserID:   actorID,
			Action:        "agent.tags.assign",
			ResourceType:  resourceType,
			ResourceID:    sql.NullInt64{Int64: int64(resourceID), Valid: true},
			IP:            input.Audit.IP,
			UserAgent:     input.Audit.UserAgent,
			TraceID:       input.Audit.TraceID,
			RequestMethod: input.Audit.RequestMethod,
			RequestPath:   input.Audit.RequestPath,
			After:         saved,
		})
		return nil
	})
	if txErr != nil {
		if appErr, ok := txErr.(*apperror.Error); ok {
			return nil, appErr
		}
		return nil, apperror.Wrap(http.StatusInternalServerError, 500120, "set resource tags failed", txErr)
	}
	return saved, nil
}

func (s *Service) ListHostGroups(ctx context.Context) ([]HostGroupSummary, *apperror.Error) {
	workspace, appErr := s.defaultWorkspaceForAPI(ctx)
	if appErr != nil {
		return nil, appErr
	}
	groups, err := s.hostGroups(ctx, s.db, workspace.ID, "")
	if err != nil {
		return nil, apperror.Wrap(http.StatusInternalServerError, 500121, "list host groups failed", err)
	}
	hostsByGroup, err := s.hostGroupHosts(ctx, s.db, workspace.ID, groupIDs(groups))
	if err != nil {
		return nil, apperror.Wrap(http.StatusInternalServerError, 500122, "list host group members failed", err)
	}
	out := make([]HostGroupSummary, 0, len(groups))
	for _, row := range groups {
		summary := hostGroupSummary(row)
		summary.Hosts = hostsByGroup[row.ID]
		out = append(out, summary)
	}
	return out, nil
}

func (s *Service) CreateHostGroup(ctx context.Context, input HostGroupInput) (HostGroupSummary, *apperror.Error) {
	return s.upsertHostGroup(ctx, "", input)
}

func (s *Service) UpdateHostGroup(ctx context.Context, groupUID string, input HostGroupInput) (HostGroupSummary, *apperror.Error) {
	if strings.TrimSpace(groupUID) == "" {
		return HostGroupSummary{}, apperror.New(http.StatusBadRequest, 400120, "host group id is required")
	}
	return s.upsertHostGroup(ctx, groupUID, input)
}

func (s *Service) SetHostGroupMembers(ctx context.Context, groupUID string, input HostGroupMembersInput) (HostGroupSummary, *apperror.Error) {
	if strings.TrimSpace(groupUID) == "" {
		return HostGroupSummary{}, apperror.New(http.StatusBadRequest, 400120, "host group id is required")
	}
	var saved HostGroupSummary
	txErr := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		workspace, err := s.defaultWorkspace(ctx, tx)
		if err != nil {
			return err
		}
		actorID, err := s.userIDByUID(ctx, tx, input.Audit.ActorUID)
		if err != nil {
			return err
		}
		group, err := s.hostGroupByUID(ctx, tx, workspace.ID, strings.TrimSpace(groupUID))
		if err != nil {
			return err
		}
		if err := s.replaceHostGroupMembers(ctx, tx, workspace.ID, group.ID, input.HostIDs); err != nil {
			return err
		}
		summary, err := s.hostGroupWithMembers(ctx, tx, workspace.ID, group.UID)
		if err != nil {
			return err
		}
		saved = summary
		audit.Write(ctx, tx, audit.Event{
			WorkspaceID:   workspace.ID,
			ActorUserID:   actorID,
			Action:        "agent.host_group.members.save",
			ResourceType:  "host_group",
			ResourceID:    sql.NullInt64{Int64: int64(group.ID), Valid: true},
			IP:            input.Audit.IP,
			UserAgent:     input.Audit.UserAgent,
			TraceID:       input.Audit.TraceID,
			RequestMethod: input.Audit.RequestMethod,
			RequestPath:   input.Audit.RequestPath,
			After:         saved,
		})
		return nil
	})
	if txErr != nil {
		if appErr, ok := txErr.(*apperror.Error); ok {
			return HostGroupSummary{}, appErr
		}
		return HostGroupSummary{}, apperror.Wrap(http.StatusInternalServerError, 500123, "set host group members failed", txErr)
	}
	return saved, nil
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

func (s *Service) writeDiagnostic(ctx context.Context, tx *gorm.DB, workspaceID, agentID, hostID uint64, input HostInfoInput) error {
	diagnosticUID, err := newUID()
	if err != nil {
		return err
	}
	return tx.WithContext(ctx).Exec(
		`INSERT INTO agent_diagnostics(uid, workspace_id, agent_id, host_id, version, os_name, os_version, arch, ip, running_tasks, payload, reported_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, NOW(3))`,
		diagnosticUID, workspaceID, agentID, hostID, nullString(input.Version), nullString(input.OSName),
		nullString(input.OSVersion), nullString(input.Arch), nullString(input.IP), nullInt(runningTasks(input.Metadata)), jsonNull(input),
	).Error
}

func (s *Service) upsertMaintenanceWindow(ctx context.Context, windowUID string, input MaintenanceWindowInput) (MaintenanceWindowSummary, *apperror.Error) {
	name := strings.TrimSpace(input.Name)
	if name == "" {
		return MaintenanceWindowSummary{}, apperror.New(http.StatusBadRequest, 400115, "maintenance window name is required")
	}
	scopeType := normalizeScopeType(input.ScopeType)
	startsAt := parseOptionalTime(input.StartsAt)
	endsAt := parseOptionalTime(input.EndsAt)
	if !startsAt.Valid || !endsAt.Valid || !endsAt.Time.After(startsAt.Time) {
		return MaintenanceWindowSummary{}, apperror.New(http.StatusBadRequest, 400116, "valid startsAt and endsAt are required")
	}
	status := normalizePolicyStatus(input.Status)
	var saved MaintenanceWindowSummary
	txErr := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		workspace, err := s.defaultWorkspace(ctx, tx)
		if err != nil {
			return err
		}
		actorID, err := s.userIDByUID(ctx, tx, input.Audit.ActorUID)
		if err != nil {
			return err
		}
		agentID, hostID, err := s.maintenanceScopeIDs(ctx, tx, workspace.ID, scopeType, input.AgentID, input.HostID)
		if err != nil {
			return err
		}
		if windowUID == "" {
			newWindowUID, err := newUID()
			if err != nil {
				return err
			}
			if err := tx.WithContext(ctx).Exec(
				`INSERT INTO maintenance_windows(uid, workspace_id, name, scope_type, agent_id, host_id, reason, starts_at, ends_at, status, created_by)
				 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
				newWindowUID, workspace.ID, name, scopeType, agentID, hostID, nullString(input.Reason), startsAt, endsAt, status, actorID,
			).Error; err != nil {
				return err
			}
			windowUID = newWindowUID
		} else {
			exec := tx.WithContext(ctx).Exec(
				`UPDATE maintenance_windows
				    SET name = ?, scope_type = ?, agent_id = ?, host_id = ?, reason = ?, starts_at = ?, ends_at = ?, status = ?, updated_at = NOW(3)
				  WHERE workspace_id = ? AND uid = ? AND deleted_at IS NULL`,
				name, scopeType, agentID, hostID, nullString(input.Reason), startsAt, endsAt, status, workspace.ID, windowUID,
			)
			if exec.Error != nil {
				return exec.Error
			}
			if exec.RowsAffected == 0 {
				return apperror.New(http.StatusNotFound, 404114, "maintenance window not found")
			}
		}
		rows, err := s.maintenanceWindows(ctx, tx, workspace.ID, windowUID)
		if err != nil {
			return err
		}
		if len(rows) == 0 {
			return apperror.New(http.StatusNotFound, 404114, "maintenance window not found")
		}
		saved = maintenanceWindowSummary(rows[0])
		audit.Write(ctx, tx, audit.Event{
			WorkspaceID:   workspace.ID,
			ActorUserID:   actorID,
			Action:        "agent.maintenance.save",
			ResourceType:  "maintenance_window",
			IP:            input.Audit.IP,
			UserAgent:     input.Audit.UserAgent,
			TraceID:       input.Audit.TraceID,
			RequestMethod: input.Audit.RequestMethod,
			RequestPath:   input.Audit.RequestPath,
			After:         saved,
		})
		return nil
	})
	if txErr != nil {
		if appErr, ok := txErr.(*apperror.Error); ok {
			return MaintenanceWindowSummary{}, appErr
		}
		if strings.Contains(strings.ToLower(txErr.Error()), "duplicate") {
			return MaintenanceWindowSummary{}, apperror.New(http.StatusConflict, 409114, "maintenance window already exists")
		}
		return MaintenanceWindowSummary{}, apperror.Wrap(http.StatusInternalServerError, 500116, "save maintenance window failed", txErr)
	}
	return saved, nil
}

func (s *Service) maintenanceScopeIDs(ctx context.Context, tx *gorm.DB, workspaceID uint64, scopeType, agentUID, hostUID string) (sql.NullInt64, sql.NullInt64, error) {
	if scopeType == "agent" {
		var row struct {
			ID     uint64
			HostID sql.NullInt64
		}
		if err := tx.WithContext(ctx).Raw("SELECT id, host_id FROM agents WHERE workspace_id = ? AND uid = ? AND deleted_at IS NULL LIMIT 1", workspaceID, strings.TrimSpace(agentUID)).Scan(&row).Error; err != nil {
			return sql.NullInt64{}, sql.NullInt64{}, err
		}
		if row.ID == 0 {
			return sql.NullInt64{}, sql.NullInt64{}, apperror.New(http.StatusNotFound, 404101, "agent not found")
		}
		return sql.NullInt64{Int64: int64(row.ID), Valid: true}, row.HostID, nil
	}
	if scopeType == "host" {
		var id uint64
		if err := tx.WithContext(ctx).Raw("SELECT id FROM hosts WHERE workspace_id = ? AND uid = ? AND deleted_at IS NULL LIMIT 1", workspaceID, strings.TrimSpace(hostUID)).Scan(&id).Error; err != nil {
			return sql.NullInt64{}, sql.NullInt64{}, err
		}
		if id == 0 {
			return sql.NullInt64{}, sql.NullInt64{}, apperror.New(http.StatusNotFound, 404102, "host not found")
		}
		return sql.NullInt64{}, sql.NullInt64{Int64: int64(id), Valid: true}, nil
	}
	return sql.NullInt64{}, sql.NullInt64{}, nil
}

func (s *Service) maintenanceWindows(ctx context.Context, db *gorm.DB, workspaceID uint64, windowUID string) ([]maintenanceWindowRecord, error) {
	args := []interface{}{workspaceID}
	where := "WHERE mw.workspace_id = ? AND mw.deleted_at IS NULL"
	if windowUID != "" {
		where += " AND mw.uid = ?"
		args = append(args, windowUID)
	}
	var rows []maintenanceWindowRecord
	err := db.WithContext(ctx).Raw(
		`SELECT mw.uid, mw.name, mw.scope_type, a.uid AS agent_uid, a.name AS agent_name, h.uid AS host_uid, h.name AS host_name,
		        mw.reason, DATE_FORMAT(mw.starts_at, '%Y-%m-%d %H:%i:%s') AS starts_at,
		        DATE_FORMAT(mw.ends_at, '%Y-%m-%d %H:%i:%s') AS ends_at,
		        mw.status, u.username AS created_by,
		        DATE_FORMAT(mw.created_at, '%Y-%m-%d %H:%i:%s') AS created_at,
		        DATE_FORMAT(mw.updated_at, '%Y-%m-%d %H:%i:%s') AS updated_at
		   FROM maintenance_windows mw
		   LEFT JOIN agents a ON a.id = mw.agent_id
		   LEFT JOIN hosts h ON h.id = mw.host_id
		   LEFT JOIN users u ON u.id = mw.created_by
		  `+where+`
		  ORDER BY mw.starts_at DESC, mw.id DESC`,
		args...,
	).Scan(&rows).Error
	return rows, err
}

func (s *Service) upsertTag(ctx context.Context, tagID string, input TagInput) (TagSummary, *apperror.Error) {
	name := limit(strings.TrimSpace(input.Name), 64)
	if name == "" {
		return TagSummary{}, apperror.New(http.StatusBadRequest, 400121, "tag name is required")
	}
	color := limit(strings.TrimSpace(input.Color), 32)
	var parsedID uint64
	if strings.TrimSpace(tagID) != "" {
		id, err := strconv.ParseUint(strings.TrimSpace(tagID), 10, 64)
		if err != nil || id == 0 {
			return TagSummary{}, apperror.New(http.StatusBadRequest, 400117, "tag id is invalid")
		}
		parsedID = id
	}
	var saved TagSummary
	txErr := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		workspace, err := s.defaultWorkspace(ctx, tx)
		if err != nil {
			return err
		}
		actorID, err := s.userIDByUID(ctx, tx, input.Audit.ActorUID)
		if err != nil {
			return err
		}
		if parsedID == 0 {
			if err := tx.WithContext(ctx).Exec(
				"INSERT INTO tags(workspace_id, name, color) VALUES (?, ?, ?)",
				workspace.ID, name, nullString(color),
			).Error; err != nil {
				return err
			}
			var id uint64
			if err := tx.WithContext(ctx).Raw(
				"SELECT id FROM tags WHERE workspace_id = ? AND name = ? LIMIT 1",
				workspace.ID, name,
			).Scan(&id).Error; err != nil {
				return err
			}
			parsedID = id
		} else {
			exec := tx.WithContext(ctx).Exec(
				"UPDATE tags SET name = ?, color = ?, updated_at = NOW(3) WHERE workspace_id = ? AND id = ?",
				name, nullString(color), workspace.ID, parsedID,
			)
			if exec.Error != nil {
				return exec.Error
			}
			if exec.RowsAffected == 0 {
				return apperror.New(http.StatusNotFound, 404115, "tag not found")
			}
		}
		rows, err := s.tags(ctx, tx, workspace.ID, parsedID)
		if err != nil {
			return err
		}
		if len(rows) == 0 {
			return apperror.New(http.StatusNotFound, 404115, "tag not found")
		}
		saved = tagSummary(rows[0])
		audit.Write(ctx, tx, audit.Event{
			WorkspaceID:   workspace.ID,
			ActorUserID:   actorID,
			Action:        "agent.tag.save",
			ResourceType:  "tag",
			ResourceID:    sql.NullInt64{Int64: int64(parsedID), Valid: true},
			IP:            input.Audit.IP,
			UserAgent:     input.Audit.UserAgent,
			TraceID:       input.Audit.TraceID,
			RequestMethod: input.Audit.RequestMethod,
			RequestPath:   input.Audit.RequestPath,
			After:         saved,
		})
		return nil
	})
	if txErr != nil {
		if appErr, ok := txErr.(*apperror.Error); ok {
			return TagSummary{}, appErr
		}
		if strings.Contains(strings.ToLower(txErr.Error()), "duplicate") {
			return TagSummary{}, apperror.New(http.StatusConflict, 409115, "tag already exists")
		}
		return TagSummary{}, apperror.Wrap(http.StatusInternalServerError, 500124, "save tag failed", txErr)
	}
	return saved, nil
}

func (s *Service) tags(ctx context.Context, db *gorm.DB, workspaceID uint64, tagID uint64) ([]tagRecord, error) {
	args := []interface{}{workspaceID}
	where := "WHERE t.workspace_id = ?"
	if tagID != 0 {
		where += " AND t.id = ?"
		args = append(args, tagID)
	}
	var rows []tagRecord
	err := db.WithContext(ctx).Raw(
		`SELECT t.id, t.name, t.color, COUNT(rt.id) AS usage_count,
		        DATE_FORMAT(t.created_at, '%Y-%m-%d %H:%i:%s') AS created_at
		   FROM tags t
		   LEFT JOIN resource_tags rt ON rt.tag_id = t.id
		  `+where+`
		  GROUP BY t.id, t.name, t.color, t.created_at
		  ORDER BY t.name ASC`,
		args...,
	).Scan(&rows).Error
	return rows, err
}

func (s *Service) tagsForResource(ctx context.Context, db *gorm.DB, workspaceID uint64, resourceType string, resourceID uint64) ([]TagSummary, error) {
	var rows []tagRecord
	if err := db.WithContext(ctx).Raw(
		`SELECT t.id, t.name, t.color, DATE_FORMAT(t.created_at, '%Y-%m-%d %H:%i:%s') AS created_at
		   FROM resource_tags rt
		   JOIN tags t ON t.id = rt.tag_id
		  WHERE rt.workspace_id = ? AND rt.resource_type = ? AND rt.resource_id = ?
		  ORDER BY t.name ASC`,
		workspaceID, resourceType, resourceID,
	).Scan(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]TagSummary, 0, len(rows))
	for _, row := range rows {
		out = append(out, tagSummary(row))
	}
	return out, nil
}

func (s *Service) tagsByResourceUIDs(ctx context.Context, db *gorm.DB, workspaceID uint64, resourceType string, resourceUIDs []string) (map[string][]TagSummary, error) {
	out := make(map[string][]TagSummary, len(resourceUIDs))
	if len(resourceUIDs) == 0 {
		return out, nil
	}
	var rows []struct {
		ResourceUID string
		ID          uint64
		Name        string
		Color       sql.NullString
	}
	tableName := "agents"
	if resourceType == "host" {
		tableName = "hosts"
	}
	if err := db.WithContext(ctx).Raw(
		`SELECT r.uid AS resource_uid, t.id, t.name, t.color
		   FROM resource_tags rt
		   JOIN tags t ON t.id = rt.tag_id
		   JOIN `+tableName+` r ON r.id = rt.resource_id
		  WHERE rt.workspace_id = ? AND rt.resource_type = ? AND r.uid IN ?
		  ORDER BY t.name ASC`,
		workspaceID, resourceType, resourceUIDs,
	).Scan(&rows).Error; err != nil {
		return nil, err
	}
	for _, row := range rows {
		out[row.ResourceUID] = append(out[row.ResourceUID], TagSummary{
			ID:    strconv.FormatUint(row.ID, 10),
			Name:  row.Name,
			Color: row.Color.String,
		})
	}
	return out, nil
}

func (s *Service) ensureTagsBelongToWorkspace(ctx context.Context, db *gorm.DB, workspaceID uint64, tagIDs []uint64) error {
	if len(tagIDs) == 0 {
		return nil
	}
	var count int64
	if err := db.WithContext(ctx).Raw(
		"SELECT COUNT(*) FROM tags WHERE workspace_id = ? AND id IN ?",
		workspaceID, tagIDs,
	).Scan(&count).Error; err != nil {
		return err
	}
	if count != int64(len(tagIDs)) {
		return apperror.New(http.StatusNotFound, 404115, "tag not found")
	}
	return nil
}

func (s *Service) resourceIDByUID(ctx context.Context, db *gorm.DB, workspaceID uint64, resourceType, resourceUID string) (uint64, error) {
	var id uint64
	if resourceType == "agent" {
		if err := db.WithContext(ctx).Raw(
			"SELECT id FROM agents WHERE workspace_id = ? AND uid = ? AND deleted_at IS NULL LIMIT 1",
			workspaceID, resourceUID,
		).Scan(&id).Error; err != nil {
			return 0, err
		}
		if id == 0 {
			return 0, apperror.New(http.StatusNotFound, 404101, "agent not found")
		}
		return id, nil
	}
	if err := db.WithContext(ctx).Raw(
		"SELECT id FROM hosts WHERE workspace_id = ? AND uid = ? AND deleted_at IS NULL LIMIT 1",
		workspaceID, resourceUID,
	).Scan(&id).Error; err != nil {
		return 0, err
	}
	if id == 0 {
		return 0, apperror.New(http.StatusNotFound, 404102, "host not found")
	}
	return id, nil
}

func (s *Service) upsertHostGroup(ctx context.Context, groupUID string, input HostGroupInput) (HostGroupSummary, *apperror.Error) {
	name := limit(strings.TrimSpace(input.Name), 128)
	if name == "" {
		return HostGroupSummary{}, apperror.New(http.StatusBadRequest, 400122, "host group name is required")
	}
	description := limit(strings.TrimSpace(input.Description), 512)
	var saved HostGroupSummary
	txErr := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		workspace, err := s.defaultWorkspace(ctx, tx)
		if err != nil {
			return err
		}
		actorID, err := s.userIDByUID(ctx, tx, input.Audit.ActorUID)
		if err != nil {
			return err
		}
		if strings.TrimSpace(groupUID) == "" {
			newGroupUID, err := newUID()
			if err != nil {
				return err
			}
			if err := tx.WithContext(ctx).Exec(
				"INSERT INTO host_groups(uid, workspace_id, name, description, created_by) VALUES (?, ?, ?, ?, ?)",
				newGroupUID, workspace.ID, name, nullString(description), actorID,
			).Error; err != nil {
				return err
			}
			groupUID = newGroupUID
		} else {
			exec := tx.WithContext(ctx).Exec(
				"UPDATE host_groups SET name = ?, description = ?, updated_at = NOW(3) WHERE workspace_id = ? AND uid = ? AND deleted_at IS NULL",
				name, nullString(description), workspace.ID, strings.TrimSpace(groupUID),
			)
			if exec.Error != nil {
				return exec.Error
			}
			if exec.RowsAffected == 0 {
				return apperror.New(http.StatusNotFound, 404116, "host group not found")
			}
		}
		group, err := s.hostGroupByUID(ctx, tx, workspace.ID, strings.TrimSpace(groupUID))
		if err != nil {
			return err
		}
		if input.HostIDs != nil {
			if err := s.replaceHostGroupMembers(ctx, tx, workspace.ID, group.ID, input.HostIDs); err != nil {
				return err
			}
		}
		summary, err := s.hostGroupWithMembers(ctx, tx, workspace.ID, group.UID)
		if err != nil {
			return err
		}
		saved = summary
		audit.Write(ctx, tx, audit.Event{
			WorkspaceID:   workspace.ID,
			ActorUserID:   actorID,
			Action:        "agent.host_group.save",
			ResourceType:  "host_group",
			ResourceID:    sql.NullInt64{Int64: int64(group.ID), Valid: true},
			IP:            input.Audit.IP,
			UserAgent:     input.Audit.UserAgent,
			TraceID:       input.Audit.TraceID,
			RequestMethod: input.Audit.RequestMethod,
			RequestPath:   input.Audit.RequestPath,
			After:         saved,
		})
		return nil
	})
	if txErr != nil {
		if appErr, ok := txErr.(*apperror.Error); ok {
			return HostGroupSummary{}, appErr
		}
		if strings.Contains(strings.ToLower(txErr.Error()), "duplicate") {
			return HostGroupSummary{}, apperror.New(http.StatusConflict, 409116, "host group already exists")
		}
		return HostGroupSummary{}, apperror.Wrap(http.StatusInternalServerError, 500125, "save host group failed", txErr)
	}
	return saved, nil
}

func (s *Service) hostGroups(ctx context.Context, db *gorm.DB, workspaceID uint64, groupUID string) ([]hostGroupRecord, error) {
	args := []interface{}{workspaceID}
	where := "WHERE hg.workspace_id = ? AND hg.deleted_at IS NULL"
	if groupUID != "" {
		where += " AND hg.uid = ?"
		args = append(args, groupUID)
	}
	var rows []hostGroupRecord
	err := db.WithContext(ctx).Raw(
		`SELECT hg.id, hg.uid, hg.name, hg.description, COUNT(hgm.host_id) AS host_count,
		        u.username AS created_by,
		        DATE_FORMAT(hg.created_at, '%Y-%m-%d %H:%i:%s') AS created_at,
		        DATE_FORMAT(hg.updated_at, '%Y-%m-%d %H:%i:%s') AS updated_at
		   FROM host_groups hg
		   LEFT JOIN host_group_members hgm ON hgm.host_group_id = hg.id
		   LEFT JOIN users u ON u.id = hg.created_by
		  `+where+`
		  GROUP BY hg.id, hg.uid, hg.name, hg.description, u.username, hg.created_at, hg.updated_at
		  ORDER BY hg.name ASC`,
		args...,
	).Scan(&rows).Error
	return rows, err
}

func (s *Service) hostGroupByUID(ctx context.Context, db *gorm.DB, workspaceID uint64, groupUID string) (hostGroupRecord, error) {
	groups, err := s.hostGroups(ctx, db, workspaceID, groupUID)
	if err != nil {
		return hostGroupRecord{}, err
	}
	if len(groups) == 0 {
		return hostGroupRecord{}, apperror.New(http.StatusNotFound, 404116, "host group not found")
	}
	return groups[0], nil
}

func (s *Service) hostGroupWithMembers(ctx context.Context, db *gorm.DB, workspaceID uint64, groupUID string) (HostGroupSummary, error) {
	group, err := s.hostGroupByUID(ctx, db, workspaceID, groupUID)
	if err != nil {
		return HostGroupSummary{}, err
	}
	hostsByGroup, err := s.hostGroupHosts(ctx, db, workspaceID, []uint64{group.ID})
	if err != nil {
		return HostGroupSummary{}, err
	}
	summary := hostGroupSummary(group)
	summary.Hosts = hostsByGroup[group.ID]
	return summary, nil
}

func (s *Service) hostGroupHosts(ctx context.Context, db *gorm.DB, workspaceID uint64, groupIDs []uint64) (map[uint64][]HostSummary, error) {
	out := make(map[uint64][]HostSummary, len(groupIDs))
	if len(groupIDs) == 0 {
		return out, nil
	}
	var rows []struct {
		GroupID         uint64
		UID             string
		Name            string
		Hostname        sql.NullString
		IP              sql.NullString
		OS              sql.NullString
		Arch            sql.NullString
		Status          string
		LastHeartbeatAt sql.NullString
		CreatedAt       string
	}
	if err := db.WithContext(ctx).Raw(
		`SELECT hgm.host_group_id AS group_id, h.uid, h.name, h.hostname, h.primary_ip AS ip,
		        COALESCE(h.os_name, h.os_type) AS os, h.arch, h.status,
		        DATE_FORMAT(MAX(a.last_heartbeat_at), '%Y-%m-%d %H:%i:%s') AS last_heartbeat_at,
		        DATE_FORMAT(h.created_at, '%Y-%m-%d %H:%i:%s') AS created_at
		   FROM host_group_members hgm
		   JOIN hosts h ON h.id = hgm.host_id
		   LEFT JOIN agents a ON a.host_id = h.id AND a.deleted_at IS NULL
		  WHERE h.workspace_id = ? AND h.deleted_at IS NULL AND hgm.host_group_id IN ?
		  GROUP BY hgm.host_group_id, h.id, h.uid, h.name, h.hostname, h.primary_ip, h.os_name, h.os_type, h.arch, h.status, h.created_at
		  ORDER BY h.name ASC`,
		workspaceID, groupIDs,
	).Scan(&rows).Error; err != nil {
		return nil, err
	}
	for _, row := range rows {
		out[row.GroupID] = append(out[row.GroupID], HostSummary{
			ID:              row.UID,
			Name:            row.Name,
			Hostname:        row.Hostname.String,
			IP:              row.IP.String,
			OS:              row.OS.String,
			Arch:            row.Arch.String,
			Status:          row.Status,
			LastHeartbeatAt: row.LastHeartbeatAt.String,
			CreatedAt:       row.CreatedAt,
		})
	}
	return out, nil
}

func (s *Service) replaceHostGroupMembers(ctx context.Context, db *gorm.DB, workspaceID, groupID uint64, hostUIDs []string) error {
	hostIDs, err := s.hostIDsByUIDs(ctx, db, workspaceID, hostUIDs)
	if err != nil {
		return err
	}
	if err := db.WithContext(ctx).Exec("DELETE FROM host_group_members WHERE host_group_id = ?", groupID).Error; err != nil {
		return err
	}
	for _, hostID := range hostIDs {
		if err := db.WithContext(ctx).Exec(
			"INSERT INTO host_group_members(host_group_id, host_id) VALUES (?, ?)",
			groupID, hostID,
		).Error; err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) hostIDsByUIDs(ctx context.Context, db *gorm.DB, workspaceID uint64, hostUIDs []string) ([]uint64, error) {
	uniqueUIDs := uniqueStrings(hostUIDs)
	if len(uniqueUIDs) == 0 {
		return nil, nil
	}
	var rows []struct {
		ID  uint64
		UID string
	}
	if err := db.WithContext(ctx).Raw(
		"SELECT id, uid FROM hosts WHERE workspace_id = ? AND uid IN ? AND deleted_at IS NULL",
		workspaceID, uniqueUIDs,
	).Scan(&rows).Error; err != nil {
		return nil, err
	}
	if len(rows) != len(uniqueUIDs) {
		return nil, apperror.New(http.StatusNotFound, 404102, "host not found")
	}
	out := make([]uint64, 0, len(rows))
	for _, row := range rows {
		out = append(out, row.ID)
	}
	return out, nil
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

func normalizeScopeType(value string) string {
	switch strings.TrimSpace(value) {
	case "agent", "host":
		return strings.TrimSpace(value)
	default:
		return "all"
	}
}

func normalizePolicyStatus(value string) string {
	switch strings.TrimSpace(value) {
	case "active", "disabled", "archived":
		return strings.TrimSpace(value)
	default:
		return "active"
	}
}

func normalizeResourceType(value string) string {
	switch strings.TrimSpace(value) {
	case "agent", "host":
		return strings.TrimSpace(value)
	default:
		return ""
	}
}

func parseTagIDs(values []string) ([]uint64, *apperror.Error) {
	seen := map[uint64]struct{}{}
	out := make([]uint64, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		id, err := strconv.ParseUint(value, 10, 64)
		if err != nil || id == 0 {
			return nil, apperror.New(http.StatusBadRequest, 400123, "tag id is invalid")
		}
		if _, exists := seen[id]; exists {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out, nil
}

func uniqueStrings(values []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}

func groupIDs(groups []hostGroupRecord) []uint64 {
	out := make([]uint64, 0, len(groups))
	for _, group := range groups {
		out = append(out, group.ID)
	}
	return out
}

func parseOptionalTime(value string) sql.NullTime {
	value = strings.TrimSpace(value)
	if value == "" {
		return sql.NullTime{}
	}
	for _, layout := range []string{time.RFC3339Nano, time.RFC3339, "2006-01-02 15:04:05"} {
		parsed, err := time.ParseInLocation(layout, value, time.Local)
		if err == nil {
			return sql.NullTime{Time: parsed, Valid: true}
		}
	}
	return sql.NullTime{}
}

func runningTasks(metadata map[string]interface{}) int {
	if metadata == nil {
		return 0
	}
	value, ok := metadata["runningTasks"]
	if !ok {
		return 0
	}
	switch typed := value.(type) {
	case int:
		return typed
	case int64:
		return int(typed)
	case float64:
		return int(typed)
	default:
		return 0
	}
}

func nullInt(value int) sql.NullInt64 {
	if value <= 0 {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: int64(value), Valid: true}
}

func maintenanceWindowSummary(row maintenanceWindowRecord) MaintenanceWindowSummary {
	return MaintenanceWindowSummary{
		ID:        row.UID,
		Name:      row.Name,
		ScopeType: row.ScopeType,
		AgentID:   row.AgentUID.String,
		AgentName: row.AgentName.String,
		HostID:    row.HostUID.String,
		HostName:  row.HostName.String,
		Reason:    row.Reason.String,
		StartsAt:  row.StartsAt,
		EndsAt:    row.EndsAt,
		Status:    row.Status,
		CreatedBy: row.CreatedBy.String,
		CreatedAt: row.CreatedAt,
		UpdatedAt: row.UpdatedAt,
	}
}

func tagSummary(row tagRecord) TagSummary {
	return TagSummary{
		ID:         strconv.FormatUint(row.ID, 10),
		Name:       row.Name,
		Color:      row.Color.String,
		UsageCount: row.UsageCount,
		CreatedAt:  row.CreatedAt.String,
	}
}

func hostGroupSummary(row hostGroupRecord) HostGroupSummary {
	return HostGroupSummary{
		ID:          row.UID,
		Name:        row.Name,
		Description: row.Description.String,
		HostCount:   row.HostCount,
		CreatedBy:   row.CreatedBy.String,
		CreatedAt:   row.CreatedAt,
		UpdatedAt:   row.UpdatedAt,
	}
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
