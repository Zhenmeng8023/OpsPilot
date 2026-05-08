package workflows

import (
	"context"
	"database/sql"

	"gorm.io/gorm"

	"opspilot/server/internal/shared/uid"
)

func CreateTriggeredRunTx(ctx context.Context, tx *gorm.DB, workspaceID, workflowID uint64, triggerType, idempotencyKey, input string, actorID sql.NullInt64, eventPayload map[string]interface{}) (uint64, string, error) {
	workflow, err := workflowRowByID(ctx, tx, workspaceID, workflowID)
	if err != nil {
		return 0, "", err
	}
	if workflow.ID == 0 {
		return 0, "", errWorkflowNotFound()
	}
	if workflow.Status != "active" {
		return 0, "", errWorkflowNotActive()
	}
	_, normalized, err := normalizeAndValidateDefinition(workflow.Definition)
	if err != nil {
		return 0, "", err
	}
	return createWorkflowRunWithSnapshotTx(ctx, tx, workspaceID, workflow.ID, workflow.Version, normalized, triggerType, idempotencyKey, input, actorID, eventPayload)
}

func createWorkflowRunWithSnapshotTx(ctx context.Context, tx *gorm.DB, workspaceID, workflowID uint64, workflowVersion uint, definitionSnapshot, triggerType, idempotencyKey, input string, actorID sql.NullInt64, eventPayload map[string]interface{}) (uint64, string, error) {
	def, normalized, err := normalizeAndValidateDefinition(definitionSnapshot)
	if err != nil {
		return 0, "", err
	}
	runInput, err := parseWorkflowInput(input)
	if err != nil {
		return 0, "", err
	}
	runUID, err := uid.New()
	if err != nil {
		return 0, "", err
	}
	if err := tx.WithContext(ctx).Exec(
		`INSERT INTO workflow_runs(uid, workspace_id, workflow_id, workflow_version, definition_snapshot, status, trigger_type,
		                           idempotency_key, input, total_nodes, created_by, queued_at)
		 VALUES (?, ?, ?, ?, ?, 'queued', ?, ?, ?, ?, ?, NOW(3))`,
		runUID, workspaceID, workflowID, workflowVersion, normalized, normalizeTriggerType(triggerType), nullString(idempotencyKey), nullJSON(input), len(def.Nodes), actorID,
	).Error; err != nil {
		return 0, "", err
	}
	runID, err := runIDByUID(ctx, tx, workspaceID, runUID)
	if err != nil {
		return 0, "", err
	}
	for _, node := range def.Nodes {
		if err := tx.WithContext(ctx).Exec(
			`INSERT INTO workflow_run_nodes(run_id, node_id, node_type, node_name, status)
			 VALUES (?, ?, ?, ?, 'pending')`,
			runID, node.ID, node.Type, nullString(node.Name),
		).Error; err != nil {
			return 0, "", err
		}
	}
	if err := dispatchInitialNodes(ctx, tx, workspaceID, runID, def, runInput, actorID); err != nil {
		return 0, "", err
	}
	payload := map[string]interface{}{"triggerType": normalizeTriggerType(triggerType)}
	for key, value := range eventPayload {
		payload[key] = value
	}
	if err := writeWorkflowEvent(ctx, tx, runID, "", "created", "Workflow run created", actorID, payload); err != nil {
		return 0, "", err
	}
	if err := refreshWorkflowRunAggregate(ctx, tx, runID); err != nil {
		return 0, "", err
	}
	return runID, runUID, nil
}
