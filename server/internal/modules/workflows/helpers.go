package workflows

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"gorm.io/gorm"

	"opspilot/server/internal/shared/apperror"
)

func definitionSummary(row workflowRecord) DefinitionSummary {
	def, _, _ := normalizeAndValidateDefinition(row.Definition)
	return DefinitionSummary{
		ID:          row.UID,
		Name:        row.Name,
		Description: row.Description.String,
		Version:     row.Version,
		Status:      row.Status,
		NodeCount:   len(def.Nodes),
		EdgeCount:   len(def.Edges),
		CreatedBy:   row.CreatedBy.String,
		PublishedAt: row.PublishedAt.String,
		CreatedAt:   row.CreatedAt,
		UpdatedAt:   row.UpdatedAt,
	}
}

func runSummary(row runRecord) RunSummary {
	return RunSummary{
		ID:              row.UID,
		WorkflowID:      row.WorkflowUID,
		WorkflowName:    row.WorkflowName,
		WorkflowVersion: row.WorkflowVersion,
		Status:          row.Status,
		TriggerType:     row.TriggerType,
		TotalNodes:      row.TotalNodes,
		SuccessNodes:    row.SuccessNodes,
		FailedNodes:     row.FailedNodes,
		SkippedNodes:    row.SkippedNodes,
		ErrorMessage:    row.ErrorMessage.String,
		CreatedBy:       row.CreatedBy.String,
		QueuedAt:        row.QueuedAt.String,
		StartedAt:       row.StartedAt.String,
		FinishedAt:      row.FinishedAt.String,
		CreatedAt:       row.CreatedAt,
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

func normalizeTriggerType(value string) string {
	switch strings.TrimSpace(value) {
	case "schedule", "webhook", "incident", "api", "retry":
		return strings.TrimSpace(value)
	default:
		return "manual"
	}
}

func retryableWorkflowStatus(value string) bool {
	switch value {
	case "failed", "canceled":
		return true
	default:
		return false
	}
}

func cancelableStatus(value string) bool {
	switch value {
	case "pending", "queued", "running", "canceling":
		return true
	default:
		return false
	}
}

func activeWorkflowStatus(value string) bool {
	switch value {
	case "pending", "queued", "running", "canceling":
		return true
	default:
		return false
	}
}

func workflowStatusFromTask(value string) string {
	switch value {
	case "pending", "queued", "dispatched":
		return "queued"
	case "running", "canceling":
		return "running"
	case "success":
		return "success"
	case "failed", "timeout":
		return "failed"
	case "canceled":
		return "canceled"
	default:
		return ""
	}
}

func nullJSON(value string) sql.NullString {
	value = strings.TrimSpace(value)
	if value == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: value, Valid: true}
}

func wrapAppError(err error, code int, message string) *apperror.Error {
	if appErr, ok := err.(*apperror.Error); ok {
		return appErr
	}
	return apperror.Wrap(http.StatusInternalServerError, code, message, err)
}

func limitString(value string, maxLen int) string {
	if len(value) <= maxLen {
		return value
	}
	return value[:maxLen]
}

func dependenciesReady(nodeID string, def Definition, statuses map[string]string) bool {
	incoming := incomingStatuses(nodeID, def, statuses)
	if len(incoming) == 0 {
		return true
	}
	if shouldSkipNode(nodeID, def, statuses) {
		return true
	}
	if def.FailurePolicy == "continue" {
		for _, status := range incoming {
			if !terminalWorkflowNodeStatus(status) {
				return false
			}
		}
		return true
	}
	for _, status := range incoming {
		if terminalWorkflowNodeStatus(status) && status != "success" {
			return true
		}
		if status != "success" {
			return false
		}
	}
	return true
}

func shouldSkipNode(nodeID string, def Definition, statuses map[string]string) bool {
	for _, status := range incomingStatuses(nodeID, def, statuses) {
		if status == "skipped" || status == "canceled" {
			return true
		}
	}
	if def.FailurePolicy == "continue" {
		return false
	}
	for _, status := range incomingStatuses(nodeID, def, statuses) {
		if terminalWorkflowNodeStatus(status) && status != "success" {
			return true
		}
	}
	return false
}

func incomingStatuses(nodeID string, def Definition, statuses map[string]string) []string {
	incoming := make([]string, 0)
	for _, edge := range def.Edges {
		if edge.To == nodeID {
			incoming = append(incoming, statuses[edge.From])
		}
	}
	return incoming
}

func terminalWorkflowNodeStatus(status string) bool {
	switch status {
	case "success", "failed", "skipped", "canceled":
		return true
	default:
		return false
	}
}

func errTaskNodeIDRequired() error {
	return apperror.New(http.StatusBadRequest, 401006, "task node config.taskId is required")
}

func errWorkflowTaskNotFound() error {
	return apperror.New(http.StatusNotFound, 404003, "workflow task node references unknown task")
}

func errWorkflowNotFound() error {
	return apperror.New(http.StatusNotFound, 404001, "workflow not found")
}

func errWorkflowNotActive() error {
	return apperror.New(http.StatusConflict, 409004, "workflow must be active before it can run")
}

func rootNodes(def Definition) []Node {
	incoming := make(map[string]int, len(def.Nodes))
	index := make(map[string]Node, len(def.Nodes))
	for _, node := range def.Nodes {
		incoming[node.ID] = 0
		index[node.ID] = node
	}
	for _, edge := range def.Edges {
		incoming[edge.To]++
	}
	roots := make([]Node, 0)
	for _, node := range def.Nodes {
		if incoming[node.ID] == 0 {
			roots = append(roots, index[node.ID])
		}
	}
	return roots
}

func nodeConfigString(node Node, key string) string {
	if node.Config == nil {
		return ""
	}
	value, ok := node.Config[key]
	if !ok {
		return ""
	}
	if text, ok := value.(string); ok {
		return strings.TrimSpace(text)
	}
	return ""
}

func nodeConfigExists(node Node, key string) bool {
	if node.Config == nil {
		return false
	}
	_, ok := node.Config[key]
	return ok
}

func nodeConfigInt(node Node, key string) (int, bool) {
	if node.Config == nil {
		return 0, false
	}
	value, ok := node.Config[key]
	if !ok {
		return 0, false
	}
	switch typed := value.(type) {
	case float64:
		return int(typed), true
	case int:
		return typed, true
	case int64:
		return int(typed), true
	case string:
		number, err := strconv.Atoi(strings.TrimSpace(typed))
		if err != nil {
			return 0, false
		}
		return number, true
	default:
		return 0, false
	}
}

func dispatchInitialNodes(ctx context.Context, tx *gorm.DB, workspaceID, runID uint64, def Definition, input interface{}, actorID sql.NullInt64) error {
	roots := rootNodes(def)
	if len(roots) == 0 {
		return apperror.New(http.StatusBadRequest, 401005, "workflow must contain at least one root node")
	}
	for _, node := range roots {
		if err := dispatchWorkflowNode(ctx, tx, workspaceID, runID, node, input, actorID); err != nil {
			return err
		}
	}
	return nil
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
