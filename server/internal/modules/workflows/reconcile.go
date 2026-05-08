package workflows

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"gorm.io/gorm"

	"opspilot/server/internal/modules/execution"
	"opspilot/server/internal/modules/notifications"
)

type workflowRunNodeState struct {
	ID         uint64
	NodeID     string
	NodeType   string
	Status     string
	TaskRunID  sql.NullInt64
	TaskStatus sql.NullString
	StartedAt  sql.NullTime
}

func reconcileActiveRuns(ctx context.Context, db *gorm.DB, workspaceID uint64) error {
	var runUIDs []string
	if err := db.WithContext(ctx).Raw(
		`SELECT uid
		   FROM workflow_runs
		  WHERE workspace_id = ?
		    AND status IN ('pending', 'queued', 'running', 'canceling')
		  ORDER BY created_at ASC
		  LIMIT 50`,
		workspaceID,
	).Scan(&runUIDs).Error; err != nil {
		return err
	}
	for _, runUID := range runUIDs {
		if err := reconcileRunByUID(ctx, db, workspaceID, runUID); err != nil {
			return err
		}
	}
	return nil
}

func reconcileRunByUID(ctx context.Context, db *gorm.DB, workspaceID uint64, runUID string) error {
	if runUID == "" {
		return nil
	}
	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		run, err := runByUID(ctx, tx, workspaceID, runUID)
		if err != nil || run.ID == 0 {
			return err
		}
		if !activeWorkflowStatus(run.Status) {
			return nil
		}
		var def Definition
		if err := json.Unmarshal([]byte(run.DefinitionSnapshot), &def); err != nil {
			return err
		}
		input, err := parseWorkflowInput(run.Input.String)
		if err != nil {
			return err
		}
		if err := syncTaskNodeStates(ctx, tx, run.ID); err != nil {
			return err
		}
		if err := syncRuntimeNodeStates(ctx, tx, run.ID, def); err != nil {
			return err
		}
		for index := 0; index <= len(def.Nodes); index++ {
			changed, err := progressReadyNodes(ctx, tx, workspaceID, run.ID, def, input)
			if err != nil {
				return err
			}
			if !changed {
				break
			}
		}
		return refreshWorkflowRunAggregate(ctx, tx, run.ID)
	})
}

func syncTaskNodeStates(ctx context.Context, tx *gorm.DB, runID uint64) error {
	nodes, err := workflowNodeStates(ctx, tx, runID)
	if err != nil {
		return err
	}
	for _, node := range nodes {
		if !node.TaskRunID.Valid || !node.TaskStatus.Valid {
			continue
		}
		next := workflowStatusFromTask(node.TaskStatus.String)
		if next == "" || next == node.Status {
			continue
		}
		if err := tx.WithContext(ctx).Exec(
			`UPDATE workflow_run_nodes
			    SET status = ?,
			        output = ?,
			        started_at = CASE WHEN ? = 'running' THEN COALESCE(started_at, NOW(3)) ELSE started_at END,
			        finished_at = CASE WHEN ? IN ('success', 'failed', 'canceled') THEN COALESCE(finished_at, NOW(3)) ELSE finished_at END
			  WHERE id = ?`,
			next, jsonStringOrNull(map[string]interface{}{"taskStatus": node.TaskStatus.String}), next, next, node.ID,
		).Error; err != nil {
			return err
		}
		if err := writeWorkflowEvent(ctx, tx, runID, node.NodeID, "node_"+next, "Task node state synchronized", sql.NullInt64{}, map[string]string{"taskStatus": node.TaskStatus.String}); err != nil {
			return err
		}
	}
	return nil
}

func syncRuntimeNodeStates(ctx context.Context, tx *gorm.DB, runID uint64, def Definition) error {
	nodes, err := workflowNodeStates(ctx, tx, runID)
	if err != nil {
		return err
	}
	definitionNodes := make(map[string]Node, len(def.Nodes))
	for _, node := range def.Nodes {
		definitionNodes[node.ID] = node
	}
	now := time.Now()
	for _, node := range nodes {
		if node.NodeType != "wait" || node.Status != "running" || !node.StartedAt.Valid {
			continue
		}
		waitFor, err := parseWaitDuration(definitionNodes[node.NodeID])
		if err != nil {
			return err
		}
		if node.StartedAt.Time.Add(waitFor).After(now) {
			continue
		}
		if err := tx.WithContext(ctx).Exec(
			`UPDATE workflow_run_nodes
			    SET status = 'success', output = ?, error_message = NULL, finished_at = COALESCE(finished_at, NOW(3))
			  WHERE id = ? AND status = 'running'`,
			jsonStringOrNull(map[string]interface{}{"seconds": int(waitFor / time.Second), "completedAt": now.Format(time.RFC3339)}),
			node.ID,
		).Error; err != nil {
			return err
		}
		if err := writeWorkflowEvent(ctx, tx, runID, node.NodeID, "node_success", "Wait node completed", sql.NullInt64{}, map[string]int{"seconds": int(waitFor / time.Second)}); err != nil {
			return err
		}
	}
	return nil
}

func progressReadyNodes(ctx context.Context, tx *gorm.DB, workspaceID, runID uint64, def Definition, input interface{}) (bool, error) {
	nodes, err := workflowNodeStates(ctx, tx, runID)
	if err != nil {
		return false, err
	}
	statuses := make(map[string]string, len(nodes))
	for _, node := range nodes {
		statuses[node.NodeID] = node.Status
	}
	changed := false
	for _, node := range def.Nodes {
		if statuses[node.ID] != "pending" || !dependenciesReady(node.ID, def, statuses) {
			continue
		}
		if shouldSkipNode(node.ID, def, statuses) {
			if err := markWorkflowNode(ctx, tx, runID, node.ID, "skipped", "Node skipped because an upstream dependency failed"); err != nil {
				return false, err
			}
			changed = true
			continue
		}
		if err := dispatchWorkflowNode(ctx, tx, workspaceID, runID, node, input, sql.NullInt64{}); err != nil {
			return false, err
		}
		changed = true
	}
	return changed, nil
}

func dispatchWorkflowNode(ctx context.Context, tx *gorm.DB, workspaceID, runID uint64, node Node, input interface{}, actorID sql.NullInt64) error {
	if node.Type == "condition" {
		return dispatchConditionNode(ctx, tx, runID, node, input, actorID)
	}
	if node.Type == "approval" {
		return dispatchApprovalNode(ctx, tx, runID, node, actorID)
	}
	if node.Type == "notification" {
		return dispatchNotificationNode(ctx, tx, workspaceID, runID, node, actorID)
	}
	if node.Type == "webhook" || node.Type == "webhook-call" {
		return dispatchWebhookCallNode(ctx, tx, runID, node, input, actorID)
	}
	if node.Type == "wait" {
		return dispatchWaitNode(ctx, tx, runID, node, actorID)
	}
	if node.Type != "task" {
		return markWorkflowNode(ctx, tx, runID, node.ID, "success", "Baseline executor completed non-task node")
	}
	taskUID := nodeConfigString(node, "taskId")
	if taskUID == "" {
		return errTaskNodeIDRequired()
	}
	taskID, err := taskIDByUID(ctx, tx, workspaceID, taskUID)
	if err != nil {
		return err
	}
	if taskID == 0 {
		return errWorkflowTaskNotFound()
	}
	taskRunID, taskRunUID, err := execution.CreateRunFromTask(ctx, tx, workspaceID, taskID, "api", sql.NullInt64{Int64: int64(runID), Valid: true}, actorID)
	if err != nil {
		return err
	}
	if err := tx.WithContext(ctx).Exec(
		`UPDATE workflow_run_nodes
		    SET status = 'queued', task_run_id = ?, input = ?, output = ?, attempts = attempts + 1, queued_at = NOW(3)
		  WHERE run_id = ? AND node_id = ? AND status = 'pending'`,
		taskRunID,
		jsonStringOrNull(map[string]interface{}{"taskId": taskUID}),
		jsonStringOrNull(map[string]interface{}{"taskRunId": taskRunUID, "taskStatus": "queued"}),
		runID,
		node.ID,
	).Error; err != nil {
		return err
	}
	return writeWorkflowEvent(ctx, tx, runID, node.ID, "node_queued", "Task node queued", actorID, map[string]string{"taskRunId": taskRunUID})
}

func dispatchConditionNode(ctx context.Context, tx *gorm.DB, runID uint64, node Node, input interface{}, actorID sql.NullInt64) error {
	matched, payload, err := evaluateCondition(node, input)
	if err != nil {
		return err
	}
	status := "success"
	message := "Condition evaluated to true"
	errorMessage := sql.NullString{}
	if !matched {
		message = "Condition evaluated to false"
		if conditionFalseBehavior(node) == "fail" {
			status = "failed"
			errorMessage = nullString(message)
		} else {
			status = "skipped"
		}
	}
	if err := tx.WithContext(ctx).Exec(
		`UPDATE workflow_run_nodes
		    SET status = ?, attempts = attempts + 1, input = ?, output = ?, error_message = ?, started_at = COALESCE(started_at, NOW(3)),
		        finished_at = COALESCE(finished_at, NOW(3))
		  WHERE run_id = ? AND node_id = ? AND status = 'pending'`,
		status, jsonStringOrNull(map[string]interface{}{"operator": conditionOperator(node), "path": nodeConfigString(node, "path")}), jsonStringOrNull(payload), errorMessage, runID, node.ID,
	).Error; err != nil {
		return err
	}
	return writeWorkflowEvent(ctx, tx, runID, node.ID, "node_"+status, message, actorID, payload)
}

func dispatchApprovalNode(ctx context.Context, tx *gorm.DB, runID uint64, node Node, actorID sql.NullInt64) error {
	comment := nodeConfigString(node, "comment")
	if comment == "" {
		comment = "Approval requested"
	}
	if err := tx.WithContext(ctx).Exec(
		`UPDATE workflow_run_nodes
		    SET status = 'running', attempts = attempts + 1, input = ?, error_message = ?, queued_at = COALESCE(queued_at, NOW(3)),
		        started_at = COALESCE(started_at, NOW(3))
		  WHERE run_id = ? AND node_id = ? AND status = 'pending'`,
		jsonStringOrNull(map[string]interface{}{"comment": comment, "mode": nodeConfigString(node, "mode")}), nullString(comment), runID, node.ID,
	).Error; err != nil {
		return err
	}
	return writeWorkflowEvent(ctx, tx, runID, node.ID, "node_running", "Approval node is waiting for a decision", actorID, map[string]string{"comment": comment})
}

func dispatchNotificationNode(ctx context.Context, tx *gorm.DB, workspaceID, runID uint64, node Node, actorID sql.NullInt64) error {
	title := nodeConfigString(node, "title")
	if title == "" {
		title = node.Name
	}
	content := nodeConfigString(node, "content")
	severity := nodeConfigString(node, "severity")
	channelUID := nodeConfigString(node, "channelId")
	notificationUID, err := notifications.EnqueueForWorkflow(ctx, tx, workspaceID, runID, channelUID, title, content, severity)
	if err != nil {
		return err
	}
	if err := tx.WithContext(ctx).Exec(
		`UPDATE workflow_run_nodes
		    SET status = 'success', attempts = attempts + 1, input = ?, output = ?, error_message = NULL,
		        started_at = COALESCE(started_at, NOW(3)), finished_at = COALESCE(finished_at, NOW(3))
		  WHERE run_id = ? AND node_id = ? AND status = 'pending'`,
		jsonStringOrNull(map[string]interface{}{"channelId": channelUID, "title": title, "severity": severity}),
		jsonStringOrNull(map[string]interface{}{"notificationId": notificationUID}),
		runID,
		node.ID,
	).Error; err != nil {
		return err
	}
	return writeWorkflowEvent(ctx, tx, runID, node.ID, "node_success", "Notification node enqueued", actorID, map[string]string{"notificationId": notificationUID})
}

func dispatchWaitNode(ctx context.Context, tx *gorm.DB, runID uint64, node Node, actorID sql.NullInt64) error {
	waitFor, err := parseWaitDuration(node)
	if err != nil {
		return err
	}
	if err := tx.WithContext(ctx).Exec(
		`UPDATE workflow_run_nodes
		    SET status = 'running', attempts = attempts + 1, input = ?, output = ?, error_message = NULL,
		        queued_at = COALESCE(queued_at, NOW(3)), started_at = COALESCE(started_at, NOW(3))
		  WHERE run_id = ? AND node_id = ? AND status = 'pending'`,
		jsonStringOrNull(map[string]interface{}{"seconds": int(waitFor / time.Second)}),
		jsonStringOrNull(map[string]interface{}{"waitUntil": time.Now().Add(waitFor).Format(time.RFC3339)}),
		runID,
		node.ID,
	).Error; err != nil {
		return err
	}
	return writeWorkflowEvent(ctx, tx, runID, node.ID, "node_running", fmt.Sprintf("Wait node started for %d seconds", int(waitFor/time.Second)), actorID, map[string]int{"seconds": int(waitFor / time.Second)})
}

func markWorkflowNode(ctx context.Context, tx *gorm.DB, runID uint64, nodeID, status, message string) error {
	if err := tx.WithContext(ctx).Exec(
		`UPDATE workflow_run_nodes
		    SET status = ?,
		        output = ?,
		        started_at = COALESCE(started_at, NOW(3)),
		        finished_at = CASE WHEN ? IN ('success', 'failed', 'skipped', 'canceled') THEN COALESCE(finished_at, NOW(3)) ELSE finished_at END
		  WHERE run_id = ? AND node_id = ? AND status = 'pending'`,
		status, jsonStringOrNull(map[string]interface{}{"message": message}), status, runID, nodeID,
	).Error; err != nil {
		return err
	}
	return writeWorkflowEvent(ctx, tx, runID, nodeID, "node_"+status, message, sql.NullInt64{}, nil)
}

func refreshWorkflowRunAggregate(ctx context.Context, tx *gorm.DB, runID uint64) error {
	return tx.WithContext(ctx).Exec(
		`UPDATE workflow_runs wr
		    JOIN (
		      SELECT run_id,
		             COUNT(*) AS total_count,
		             SUM(CASE WHEN status = 'success' THEN 1 ELSE 0 END) AS success_count,
		             SUM(CASE WHEN status = 'failed' THEN 1 ELSE 0 END) AS failed_count,
		             SUM(CASE WHEN status = 'skipped' THEN 1 ELSE 0 END) AS skipped_count,
		             SUM(CASE WHEN status = 'canceled' THEN 1 ELSE 0 END) AS canceled_count,
		             SUM(CASE WHEN status IN ('pending', 'queued', 'running') THEN 1 ELSE 0 END) AS active_count,
		             SUM(CASE WHEN status = 'running' THEN 1 ELSE 0 END) AS running_count
		        FROM workflow_run_nodes
		       WHERE run_id = ?
		       GROUP BY run_id
		    ) agg ON agg.run_id = wr.id
		    SET wr.total_nodes = agg.total_count,
		        wr.success_nodes = agg.success_count,
		        wr.failed_nodes = agg.failed_count,
		        wr.skipped_nodes = agg.skipped_count,
		        wr.status = CASE
		          WHEN agg.active_count > 0 AND agg.running_count > 0 THEN 'running'
		          WHEN agg.active_count > 0 THEN 'queued'
		          WHEN agg.failed_count > 0 THEN 'failed'
		          WHEN agg.canceled_count > 0 THEN 'canceled'
		          ELSE 'success'
		        END,
		        wr.started_at = CASE WHEN agg.running_count > 0 THEN COALESCE(wr.started_at, NOW(3)) ELSE wr.started_at END,
		        wr.finished_at = CASE WHEN agg.active_count = 0 THEN COALESCE(wr.finished_at, NOW(3)) ELSE NULL END
		  WHERE wr.id = ?`,
		runID, runID,
	).Error
}

func workflowNodeStates(ctx context.Context, tx *gorm.DB, runID uint64) ([]workflowRunNodeState, error) {
	var nodes []workflowRunNodeState
	err := tx.WithContext(ctx).Raw(
		`SELECT wrn.id, wrn.node_id, wrn.node_type, wrn.status, wrn.task_run_id, tr.status AS task_status, wrn.started_at
		   FROM workflow_run_nodes wrn
		   LEFT JOIN task_runs tr ON tr.id = wrn.task_run_id
		  WHERE wrn.run_id = ?
		  ORDER BY wrn.id ASC`,
		runID,
	).Scan(&nodes).Error
	return nodes, err
}

func writeWorkflowEvent(ctx context.Context, tx *gorm.DB, runID uint64, nodeID, eventType, message string, actorID sql.NullInt64, payload interface{}) error {
	return tx.WithContext(ctx).Exec(
		`INSERT INTO workflow_run_events(run_id, node_id, event_type, message, actor_id, payload)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		runID, nullString(nodeID), eventType, nullString(message), actorID, jsonNull(payload),
	).Error
}
