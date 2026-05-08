package workflows

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"
)

var workflowTemplatePattern = regexp.MustCompile(`\$\{([^}]+)\}`)

type webhookCallPlan struct {
	Method      string
	URL         string
	Headers     map[string]string
	ContentType string
	Body        []byte
	Timeout     time.Duration
}

func dispatchWebhookCallNode(ctx context.Context, tx *gorm.DB, runID uint64, node Node, input interface{}, actorID sql.NullInt64) error {
	plan, err := buildWebhookCallPlan(node, input)
	if err != nil {
		return err
	}
	requestPayload := map[string]interface{}{
		"method":      plan.Method,
		"url":         plan.URL,
		"headers":     plan.Headers,
		"contentType": plan.ContentType,
	}
	if len(plan.Body) > 0 {
		requestPayload["body"] = string(plan.Body)
	}
	if err := tx.WithContext(ctx).Exec(
		`UPDATE workflow_run_nodes
		    SET status = 'running', attempts = attempts + 1, input = ?, error_message = NULL,
		        queued_at = COALESCE(queued_at, NOW(3)), started_at = COALESCE(started_at, NOW(3))
		  WHERE run_id = ? AND node_id = ? AND status = 'pending'`,
		jsonStringOrNull(requestPayload), runID, node.ID,
	).Error; err != nil {
		return err
	}
	callCtx, cancel := context.WithTimeout(ctx, plan.Timeout)
	defer cancel()
	req, err := http.NewRequestWithContext(callCtx, plan.Method, plan.URL, bytes.NewReader(plan.Body))
	if err != nil {
		return err
	}
	if plan.ContentType != "" {
		req.Header.Set("Content-Type", plan.ContentType)
	}
	for key, value := range plan.Headers {
		req.Header.Set(key, value)
	}
	resp, err := (&http.Client{Timeout: plan.Timeout}).Do(req)
	if err != nil {
		if updateErr := updateWebhookCallResult(ctx, tx, runID, node.ID, "failed", nil, limitString(err.Error(), 2048)); updateErr != nil {
			return updateErr
		}
		return writeWorkflowEvent(ctx, tx, runID, node.ID, "node_failed", "Webhook call failed", actorID, map[string]string{"error": limitString(err.Error(), 1024)})
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 8192))
	resultPayload := map[string]interface{}{
		"statusCode":  resp.StatusCode,
		"status":      resp.Status,
		"url":         plan.URL,
		"contentType": resp.Header.Get("Content-Type"),
		"body":        string(body),
	}
	nextStatus := "success"
	message := "Webhook call completed"
	errorMessage := ""
	if !webhookCallSucceeded(node, resp.StatusCode) {
		nextStatus = "failed"
		message = "Webhook call returned an unexpected status"
		errorMessage = fmt.Sprintf("webhook returned %d", resp.StatusCode)
	}
	if err := updateWebhookCallResult(ctx, tx, runID, node.ID, nextStatus, resultPayload, errorMessage); err != nil {
		return err
	}
	return writeWorkflowEvent(ctx, tx, runID, node.ID, "node_"+nextStatus, message, actorID, resultPayload)
}

func buildWebhookCallPlan(node Node, input interface{}) (webhookCallPlan, error) {
	method := strings.ToUpper(nodeConfigString(node, "method"))
	if method == "" {
		method = http.MethodPost
	}
	timeout := time.Duration(node.TimeoutSeconds) * time.Second
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	url := renderWorkflowTemplate(nodeConfigString(node, "url"), input)
	if url == "" {
		return webhookCallPlan{}, fmt.Errorf("webhook-call node %s requires config.url", node.ID)
	}
	headers := map[string]string{}
	if node.Config != nil {
		if rawHeaders, ok := node.Config["headers"].(map[string]interface{}); ok {
			for key, value := range rawHeaders {
				headers[http.CanonicalHeaderKey(strings.TrimSpace(key))] = renderWorkflowTemplate(fmt.Sprint(value), input)
			}
		}
	}
	contentType := renderWorkflowTemplate(nodeConfigString(node, "contentType"), input)
	body, err := webhookCallBody(node, input)
	if err != nil {
		return webhookCallPlan{}, err
	}
	if contentType == "" && len(body) > 0 {
		contentType = "application/json"
	}
	return webhookCallPlan{
		Method:      method,
		URL:         url,
		Headers:     headers,
		ContentType: contentType,
		Body:        body,
		Timeout:     timeout,
	}, nil
}

func webhookCallBody(node Node, input interface{}) ([]byte, error) {
	bodyValue := interface{}(nil)
	if path := nodeConfigString(node, "bodyPath"); path != "" {
		if resolved, ok := resolveWorkflowValue(input, path); ok {
			bodyValue = resolved
		}
	}
	if bodyValue == nil && node.Config != nil {
		if direct, ok := node.Config["body"]; ok {
			bodyValue = direct
		}
	}
	if bodyValue == nil {
		return nil, nil
	}
	switch typed := bodyValue.(type) {
	case string:
		rendered := renderWorkflowTemplate(typed, input)
		trimmed := strings.TrimSpace(rendered)
		if trimmed == "" {
			return nil, nil
		}
		if json.Valid([]byte(trimmed)) {
			return []byte(trimmed), nil
		}
		return []byte(rendered), nil
	default:
		return json.Marshal(bodyValue)
	}
}

func webhookCallSucceeded(node Node, statusCode int) bool {
	if value, ok := nodeConfigInt(node, "expectedStatus"); ok && value > 0 {
		return statusCode == value
	}
	minStatus := 200
	maxStatus := 299
	if value, ok := nodeConfigInt(node, "successStatusMin"); ok && value > 0 {
		minStatus = value
	}
	if value, ok := nodeConfigInt(node, "successStatusMax"); ok && value > 0 {
		maxStatus = value
	}
	return statusCode >= minStatus && statusCode <= maxStatus
}

func updateWebhookCallResult(ctx context.Context, tx *gorm.DB, runID uint64, nodeID, status string, output interface{}, errorMessage string) error {
	return tx.WithContext(ctx).Exec(
		`UPDATE workflow_run_nodes
		    SET status = ?, output = ?, error_message = ?, finished_at = COALESCE(finished_at, NOW(3))
		  WHERE run_id = ? AND node_id = ?`,
		status, jsonStringOrNull(output), nullString(errorMessage), runID, nodeID,
	).Error
}

func renderWorkflowTemplate(value string, input interface{}) string {
	value = strings.TrimSpace(value)
	if value == "" || !strings.Contains(value, "${") {
		return value
	}
	return workflowTemplatePattern.ReplaceAllStringFunc(value, func(match string) string {
		submatches := workflowTemplatePattern.FindStringSubmatch(match)
		if len(submatches) != 2 {
			return match
		}
		resolved, ok := resolveWorkflowValue(input, strings.TrimSpace(submatches[1]))
		if !ok {
			return ""
		}
		switch typed := resolved.(type) {
		case string:
			return typed
		case float64:
			return strconv.FormatFloat(typed, 'f', -1, 64)
		case bool:
			return strconv.FormatBool(typed)
		default:
			bytes, err := json.Marshal(typed)
			if err != nil {
				return ""
			}
			return string(bytes)
		}
	})
}

func jsonStringOrNull(value interface{}) sql.NullString {
	if value == nil {
		return sql.NullString{}
	}
	bytes, err := json.Marshal(value)
	if err != nil || string(bytes) == "null" {
		return sql.NullString{}
	}
	return sql.NullString{String: string(bytes), Valid: true}
}
