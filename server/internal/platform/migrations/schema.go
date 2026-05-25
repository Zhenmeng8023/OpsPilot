package migrations

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"gorm.io/gorm"
)

const TableName = "schema_migrations"

var rootMigrations = []string{
	"000001_init_mysql_schema.up.sql",
	"000002_seed_initial_auth_data.up.sql",
	"000003_task_execution_security.up.sql",
	"000004_v07_webhook_security.up.sql",
	"000005_v10_productization_bundle.up.sql",
	"000006_schema_migrations_governance.up.sql",
	"000007_v11_trace_cache_and_actions.up.sql",
}

var requiredTables = []string{
	"workspaces",
	"users",
	"roles",
	"permissions",
	"tasks",
	"task_runs",
	"workflow_definitions",
	"workflow_runs",
	"workflow_run_actions",
	"webhook_events",
	"notifications",
	"audit_logs",
	"trace_events",
}

type Status struct {
	Ready             bool     `json:"ready"`
	MigrationTable    bool     `json:"migrationTable"`
	MissingTables     []string `json:"missingTables,omitempty"`
	MissingMigrations []string `json:"missingMigrations,omitempty"`
	AppliedMigrations []string `json:"appliedMigrations,omitempty"`
}

func Verify(ctx context.Context, db *gorm.DB) (Status, error) {
	status := Status{}
	if db == nil {
		return status, fmt.Errorf("database is nil")
	}

	tablesToCheck := append([]string{TableName}, requiredTables...)
	tableSet, err := existingTables(ctx, db, tablesToCheck)
	if err != nil {
		return Status{}, err
	}

	if _, ok := tableSet[TableName]; ok {
		status.MigrationTable = true
	} else {
		status.MissingTables = append(status.MissingTables, TableName)
	}

	for _, name := range requiredTables {
		if _, ok := tableSet[name]; !ok {
			status.MissingTables = append(status.MissingTables, name)
		}
	}

	if status.MigrationTable {
		applied, loadErr := appliedVersions(ctx, db)
		if loadErr != nil {
			return Status{}, loadErr
		}
		appliedSet := make(map[string]struct{}, len(applied))
		for _, version := range applied {
			appliedSet[version] = struct{}{}
		}
		for _, version := range rootMigrations {
			if _, ok := appliedSet[version]; !ok {
				status.MissingMigrations = append(status.MissingMigrations, version)
			}
		}
		status.AppliedMigrations = applied
	}

	sort.Strings(status.MissingTables)
	sort.Strings(status.MissingMigrations)
	status.Ready = status.MigrationTable && len(status.MissingTables) == 0 && len(status.MissingMigrations) == 0
	return status, nil
}

func (s Status) Message() string {
	if s.Ready {
		return "schema is ready"
	}

	parts := make([]string, 0, 2)
	if len(s.MissingTables) > 0 {
		parts = append(parts, "missing tables: "+strings.Join(s.MissingTables, ", "))
	}
	if len(s.MissingMigrations) > 0 {
		parts = append(parts, "missing migration records: "+strings.Join(s.MissingMigrations, ", "))
	}
	if len(parts) == 0 {
		return "schema is not ready"
	}
	return strings.Join(parts, "; ")
}

func RootMigrations() []string {
	out := make([]string, len(rootMigrations))
	copy(out, rootMigrations)
	return out
}

func RequiredTables() []string {
	out := make([]string, len(requiredTables))
	copy(out, requiredTables)
	return out
}

func existingTables(ctx context.Context, db *gorm.DB, names []string) (map[string]struct{}, error) {
	var rows []struct {
		Name string `gorm:"column:table_name"`
	}
	if err := db.WithContext(ctx).Raw(
		`SELECT table_name
		   FROM information_schema.tables
		  WHERE table_schema = DATABASE()
		    AND table_name IN ?`,
		names,
	).Scan(&rows).Error; err != nil {
		return nil, err
	}
	out := make(map[string]struct{}, len(rows))
	for _, row := range rows {
		out[row.Name] = struct{}{}
	}
	return out, nil
}

func appliedVersions(ctx context.Context, db *gorm.DB) ([]string, error) {
	var rows []struct {
		Version string `gorm:"column:version"`
	}
	if err := db.WithContext(ctx).Raw(
		`SELECT version
		   FROM schema_migrations
		  WHERE status IN ('applied', 'legacy', 'skipped')
		  ORDER BY applied_at ASC, id ASC`,
	).Scan(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]string, 0, len(rows))
	for _, row := range rows {
		if strings.TrimSpace(row.Version) != "" {
			out = append(out, row.Version)
		}
	}
	return out, nil
}
