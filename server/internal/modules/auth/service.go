package auth

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"net/http"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"opspilot/server/internal/config"
	jwtplatform "opspilot/server/internal/platform/jwt"
	"opspilot/server/internal/shared/apperror"
	"opspilot/server/internal/shared/audit"
)

type Service struct {
	db         *gorm.DB
	cfg        config.Config
	jwtManager jwtplatform.Manager
}

type RegisterInput struct {
	Username  string
	Email     string
	Password  string
	IP        string
	UserAgent string
	TraceID   string
}

type LoginInput struct {
	Identifier string
	Password   string
	IP         string
	UserAgent  string
	TraceID    string
}

type RefreshInput struct {
	RefreshToken string
	IP           string
	UserAgent    string
}

type LogoutInput struct {
	RefreshToken string
}

type AuthResult struct {
	AccessToken  string      `json:"accessToken"`
	RefreshToken string      `json:"refreshToken"`
	TokenType    string      `json:"tokenType"`
	ExpiresIn    int64       `json:"expiresIn"`
	User         UserProfile `json:"user"`
}

type UserProfile struct {
	ID          string   `json:"id"`
	Username    string   `json:"username"`
	Email       string   `json:"email,omitempty"`
	Roles       []string `json:"roles"`
	Permissions []string `json:"permissions"`
	Workspace   struct {
		ID   string `json:"id"`
		Name string `json:"name"`
		Slug string `json:"slug"`
	} `json:"workspace"`
}

type userRecord struct {
	ID           uint64
	UID          string
	Username     string
	Email        sql.NullString
	PasswordHash string
	Status       string
}

type workspaceRecord struct {
	ID   uint64
	UID  string
	Name string
	Slug string
}

func NewService(db *gorm.DB, cfg config.Config) *Service {
	return &Service{
		db:         db,
		cfg:        cfg,
		jwtManager: jwtplatform.NewManager(cfg.JWT),
	}
}

func (s *Service) JWTManager() jwtplatform.Manager {
	return s.jwtManager
}

func (s *Service) Register(ctx context.Context, input RegisterInput) (AuthResult, *apperror.Error) {
	if s.cfg.App.Env == "prod" && !s.cfg.Auth.PublicRegistrationEnabled {
		return AuthResult{}, apperror.New(http.StatusForbidden, 403003, "public registration is disabled")
	}

	username := strings.TrimSpace(input.Username)
	email := strings.TrimSpace(input.Email)
	if err := validateUsername(username); err != nil {
		return AuthResult{}, err
	}
	if err := validatePassword(input.Password); err != nil {
		return AuthResult{}, err
	}

	var result AuthResult
	txErr := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var count int64
		if err := tx.Raw("SELECT COUNT(*) FROM users WHERE username = ? OR (? <> '' AND email = ?)", username, email, email).Scan(&count).Error; err != nil {
			return err
		}
		if count > 0 {
			return apperror.New(http.StatusConflict, 409001, "username or email already exists")
		}

		workspace, err := defaultWorkspace(ctx, tx, s.cfg.Bootstrap.WorkspaceSlug)
		if err != nil {
			return err
		}
		memberRoleID, err := roleID(ctx, tx, workspace.ID, "member")
		if err != nil {
			return err
		}

		passwordHash, err := hashPassword(input.Password)
		if err != nil {
			return err
		}
		uid, err := newUID()
		if err != nil {
			return err
		}

		emailValue := sql.NullString{String: email, Valid: email != ""}
		if err := tx.Exec(
			`INSERT INTO users(uid, username, email, password_hash, status, password_changed_at)
			 VALUES (?, ?, ?, ?, 'active', NOW(3))`,
			uid, username, emailValue, passwordHash,
		).Error; err != nil {
			return err
		}

		var user userRecord
		if err := tx.Raw("SELECT id, uid, username, email, password_hash, status FROM users WHERE uid = ?", uid).Scan(&user).Error; err != nil {
			return err
		}

		if err := tx.Exec(
			`INSERT INTO workspace_members(workspace_id, user_id, member_type, status)
			 VALUES (?, ?, 'human', 'active')`,
			workspace.ID, user.ID,
		).Error; err != nil {
			return err
		}
		if err := tx.Exec(
			`INSERT INTO user_roles(workspace_id, user_id, role_id)
			 VALUES (?, ?, ?)`,
			workspace.ID, user.ID, memberRoleID,
		).Error; err != nil {
			return err
		}

		authResult, err := s.issueTokens(ctx, tx, user, workspace, "")
		if err != nil {
			return err
		}
		result = authResult
		return nil
	})
	if txErr != nil {
		if appErr, ok := txErr.(*apperror.Error); ok {
			return AuthResult{}, appErr
		}
		return AuthResult{}, apperror.Wrap(http.StatusInternalServerError, 500001, "register failed", txErr)
	}

	return result, nil
}

func (s *Service) Login(ctx context.Context, input LoginInput) (AuthResult, *apperror.Error) {
	identifier := strings.TrimSpace(input.Identifier)
	if identifier == "" || strings.TrimSpace(input.Password) == "" {
		return AuthResult{}, apperror.New(http.StatusBadRequest, 400001, "username and password are required")
	}

	var user userRecord
	err := s.db.WithContext(ctx).Raw(
		`SELECT id, uid, username, email, password_hash, status
		 FROM users
		 WHERE username = ? OR email = ?
		 LIMIT 1`,
		identifier, identifier,
	).Scan(&user).Error
	if err != nil {
		return AuthResult{}, apperror.Wrap(http.StatusInternalServerError, 500001, "login failed", err)
	}
	if user.ID == 0 || user.Status != "active" || bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(input.Password)) != nil {
		s.writeLoginLog(ctx, nil, identifier, "failed", "invalid_credentials", input.IP, input.UserAgent, input.TraceID)
		audit.Write(ctx, s.db, audit.Event{
			ActorType: "user",
			Action:    "user.login",
			Result:    "failed",
			IP:        input.IP,
			UserAgent: input.UserAgent,
			TraceID:   input.TraceID,
			Metadata:  map[string]string{"identifier": identifier},
		})
		return AuthResult{}, apperror.New(http.StatusUnauthorized, 401001, "invalid username or password")
	}

	workspace, err := s.userWorkspace(ctx, s.db, user.ID)
	if err != nil {
		return AuthResult{}, apperror.Wrap(http.StatusInternalServerError, 500001, "load workspace failed", err)
	}
	if workspace.ID == 0 {
		return AuthResult{}, apperror.New(http.StatusForbidden, 403001, "user is not assigned to a workspace")
	}

	var result AuthResult
	txErr := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		authResult, err := s.issueTokens(ctx, tx, user, workspace, "")
		if err != nil {
			return err
		}
		result = authResult
		return nil
	})
	if txErr != nil {
		return AuthResult{}, apperror.Wrap(http.StatusInternalServerError, 500001, "issue token failed", txErr)
	}

	s.writeLoginLog(ctx, &user.ID, user.Username, "success", "", input.IP, input.UserAgent, input.TraceID)
	audit.Write(ctx, s.db, audit.Event{
		WorkspaceID:  workspace.ID,
		ActorType:    "user",
		ActorUserID:  sql.NullInt64{Int64: int64(user.ID), Valid: true},
		Action:       "user.login",
		ResourceType: "user",
		ResourceID:   sql.NullInt64{Int64: int64(user.ID), Valid: true},
		IP:           input.IP,
		UserAgent:    input.UserAgent,
		TraceID:      input.TraceID,
	})
	return result, nil
}

func (s *Service) Refresh(ctx context.Context, input RefreshInput) (AuthResult, *apperror.Error) {
	token := strings.TrimSpace(input.RefreshToken)
	if token == "" {
		return AuthResult{}, apperror.New(http.StatusBadRequest, 400001, "refreshToken is required")
	}
	if _, err := s.jwtManager.ParseRefreshToken(token); err != nil {
		return AuthResult{}, apperror.New(http.StatusUnauthorized, 401002, "invalid refresh token")
	}

	var result AuthResult
	txErr := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		tokenHash := hashToken(token)
		var row struct {
			ID          uint64
			UserID      uint64
			TokenFamily string
		}
		if err := tx.Raw(
			`SELECT id, user_id, token_family
			 FROM refresh_tokens
			 WHERE token_hash = ? AND revoked_at IS NULL AND expires_at > NOW(3)
			 LIMIT 1`,
			tokenHash,
		).Scan(&row).Error; err != nil {
			return err
		}
		if row.ID == 0 {
			return apperror.New(http.StatusUnauthorized, 401002, "invalid refresh token")
		}

		var user userRecord
		if err := tx.Raw("SELECT id, uid, username, email, password_hash, status FROM users WHERE id = ?", row.UserID).Scan(&user).Error; err != nil {
			return err
		}
		if user.ID == 0 || user.Status != "active" {
			return apperror.New(http.StatusUnauthorized, 401002, "invalid refresh token")
		}

		workspace, err := s.userWorkspace(ctx, tx, user.ID)
		if err != nil {
			return err
		}
		if workspace.ID == 0 {
			return apperror.New(http.StatusForbidden, 403001, "user is not assigned to a workspace")
		}

		if err := tx.Exec("UPDATE refresh_tokens SET revoked_at = NOW(3) WHERE id = ?", row.ID).Error; err != nil {
			return err
		}
		authResult, err := s.issueTokens(ctx, tx, user, workspace, row.TokenFamily)
		if err != nil {
			return err
		}
		result = authResult
		return nil
	})
	if txErr != nil {
		if appErr, ok := txErr.(*apperror.Error); ok {
			return AuthResult{}, appErr
		}
		return AuthResult{}, apperror.Wrap(http.StatusInternalServerError, 500001, "refresh token failed", txErr)
	}

	return result, nil
}

func (s *Service) Logout(ctx context.Context, input LogoutInput) *apperror.Error {
	token := strings.TrimSpace(input.RefreshToken)
	if token == "" {
		return nil
	}
	if err := s.db.WithContext(ctx).Exec(
		"UPDATE refresh_tokens SET revoked_at = NOW(3) WHERE token_hash = ? AND revoked_at IS NULL",
		hashToken(token),
	).Error; err != nil {
		return apperror.Wrap(http.StatusInternalServerError, 500001, "logout failed", err)
	}
	return nil
}

func (s *Service) Me(ctx context.Context, userUID string) (UserProfile, *apperror.Error) {
	var user userRecord
	if err := s.db.WithContext(ctx).Raw(
		"SELECT id, uid, username, email, password_hash, status FROM users WHERE uid = ?",
		userUID,
	).Scan(&user).Error; err != nil {
		return UserProfile{}, apperror.Wrap(http.StatusInternalServerError, 500001, "load user failed", err)
	}
	if user.ID == 0 || user.Status != "active" {
		return UserProfile{}, apperror.New(http.StatusUnauthorized, 401003, "user not found")
	}
	workspace, err := s.userWorkspace(ctx, s.db, user.ID)
	if err != nil {
		return UserProfile{}, apperror.Wrap(http.StatusInternalServerError, 500001, "load workspace failed", err)
	}
	roles, err := userRoles(ctx, s.db, user.ID, workspace.ID)
	if err != nil {
		return UserProfile{}, apperror.Wrap(http.StatusInternalServerError, 500001, "load roles failed", err)
	}
	permissions, err := userPermissions(ctx, s.db, user.ID, workspace.ID)
	if err != nil {
		return UserProfile{}, apperror.Wrap(http.StatusInternalServerError, 500001, "load permissions failed", err)
	}
	return profileFrom(user, workspace, roles, permissions), nil
}

func (s *Service) issueTokens(ctx context.Context, tx *gorm.DB, user userRecord, workspace workspaceRecord, family string) (AuthResult, error) {
	roles, err := userRoles(ctx, tx, user.ID, workspace.ID)
	if err != nil {
		return AuthResult{}, err
	}
	permissions, err := userPermissions(ctx, tx, user.ID, workspace.ID)
	if err != nil {
		return AuthResult{}, err
	}
	if family == "" {
		family, err = newTokenFamily()
		if err != nil {
			return AuthResult{}, err
		}
	}

	accessToken, err := s.jwtManager.GenerateAccessToken(user.UID, user.Username, roles)
	if err != nil {
		return AuthResult{}, err
	}
	refreshToken, err := s.jwtManager.GenerateRefreshToken(user.UID, user.Username, roles)
	if err != nil {
		return AuthResult{}, err
	}
	if err := tx.Exec(
		`INSERT INTO refresh_tokens(user_id, token_hash, token_family, expires_at)
		 VALUES (?, ?, ?, ?)`,
		user.ID, hashToken(refreshToken), family, time.Now().Add(s.cfg.JWT.RefreshTTL),
	).Error; err != nil {
		return AuthResult{}, err
	}

	return AuthResult{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    int64(s.cfg.JWT.AccessTTL.Seconds()),
		User:         profileFrom(user, workspace, roles, permissions),
	}, nil
}

func (s *Service) userWorkspace(ctx context.Context, db *gorm.DB, userID uint64) (workspaceRecord, error) {
	var workspace workspaceRecord
	err := db.WithContext(ctx).Raw(
		`SELECT w.id, w.uid, w.name, w.slug
		 FROM workspace_members wm
		 JOIN workspaces w ON w.id = wm.workspace_id
		 WHERE wm.user_id = ? AND wm.status = 'active' AND w.status = 'active'
		 ORDER BY w.id
		 LIMIT 1`,
		userID,
	).Scan(&workspace).Error
	return workspace, err
}

func userRoles(ctx context.Context, db *gorm.DB, userID, workspaceID uint64) ([]string, error) {
	var roles []string
	err := db.WithContext(ctx).Raw(
		`SELECT r.code
		 FROM user_roles ur
		 JOIN roles r ON r.id = ur.role_id
		 WHERE ur.user_id = ?
		   AND ur.workspace_id = ?
		   AND r.status = 'active'
		   AND (ur.expires_at IS NULL OR ur.expires_at > NOW(3))
		 ORDER BY r.code`,
		userID, workspaceID,
	).Scan(&roles).Error
	return roles, err
}

func userPermissions(ctx context.Context, db *gorm.DB, userID, workspaceID uint64) ([]string, error) {
	var permissions []string
	err := db.WithContext(ctx).Raw(
		`SELECT DISTINCT p.code
		   FROM user_roles ur
		   JOIN roles r ON r.id = ur.role_id
		   JOIN role_permissions rp ON rp.role_id = r.id
		   JOIN permissions p ON p.id = rp.permission_id
		  WHERE ur.user_id = ?
		    AND ur.workspace_id = ?
		    AND r.status = 'active'
		    AND (ur.expires_at IS NULL OR ur.expires_at > NOW(3))
		  ORDER BY p.code`,
		userID, workspaceID,
	).Scan(&permissions).Error
	return canonicalizePermissionList(permissions), err
}

func defaultWorkspace(ctx context.Context, db *gorm.DB, slug string) (workspaceRecord, error) {
	var workspace workspaceRecord
	err := db.WithContext(ctx).Raw(
		"SELECT id, uid, name, slug FROM workspaces WHERE slug = ? AND status = 'active' LIMIT 1",
		slug,
	).Scan(&workspace).Error
	return workspace, err
}

func roleID(ctx context.Context, db *gorm.DB, workspaceID uint64, code string) (uint64, error) {
	var id uint64
	err := db.WithContext(ctx).Raw(
		"SELECT id FROM roles WHERE workspace_id = ? AND code = ? AND status = 'active' LIMIT 1",
		workspaceID, code,
	).Scan(&id).Error
	if err != nil {
		return 0, err
	}
	if id == 0 {
		return 0, errors.New("role not found: " + code)
	}
	return id, nil
}

func profileFrom(user userRecord, workspace workspaceRecord, roles, permissions []string) UserProfile {
	profile := UserProfile{
		ID:          user.UID,
		Username:    user.Username,
		Roles:       roles,
		Permissions: permissions,
	}
	if user.Email.Valid {
		profile.Email = user.Email.String
	}
	profile.Workspace.ID = workspace.UID
	profile.Workspace.Name = workspace.Name
	profile.Workspace.Slug = workspace.Slug
	return profile
}

func validateUsername(username string) *apperror.Error {
	if len(username) < 3 || len(username) > 64 {
		return apperror.New(http.StatusBadRequest, 400002, "username must be 3-64 characters")
	}
	return nil
}

func validatePassword(password string) *apperror.Error {
	if len(password) < 8 || len(password) > 128 {
		return apperror.New(http.StatusBadRequest, 400003, "password must be 8-128 characters")
	}
	return nil
}

func hashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func (s *Service) writeLoginLog(ctx context.Context, userID *uint64, username, result, reason, ip, userAgent, traceID string) {
	_ = s.db.WithContext(ctx).Exec(
		`INSERT INTO login_logs(user_id, username, result, reason, ip, user_agent, trace_id)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		nullableUint64(userID), username, result, nullString(reason), nullString(ip), nullString(userAgent), nullString(traceID),
	).Error
}

func nullString(value string) sql.NullString {
	value = strings.TrimSpace(value)
	return sql.NullString{String: value, Valid: value != ""}
}
