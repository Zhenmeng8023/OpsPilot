package workflows

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

const maxWaitSeconds = 86400

func parseWorkflowInput(raw string) (interface{}, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return map[string]interface{}{}, nil
	}
	var value interface{}
	if err := json.Unmarshal([]byte(raw), &value); err != nil {
		return nil, err
	}
	return value, nil
}

func validateConditionNode(node Node) error {
	operator := conditionOperator(node)
	switch operator {
	case "truthy", "exists", "equals", "not_equals", "contains":
	default:
		return fmt.Errorf("condition node %s has unsupported config.operator", node.ID)
	}
	if (operator == "equals" || operator == "not_equals" || operator == "contains") && !nodeConfigExists(node, "value") {
		return fmt.Errorf("condition node %s requires config.value for operator %s", node.ID, operator)
	}
	onFalse := conditionFalseBehavior(node)
	if onFalse != "skip" && onFalse != "fail" {
		return fmt.Errorf("condition node %s has unsupported config.onFalse", node.ID)
	}
	return nil
}

func validateWaitNode(node Node) error {
	seconds, ok := nodeConfigInt(node, "seconds")
	if !ok || seconds <= 0 {
		return fmt.Errorf("wait node %s requires positive config.seconds", node.ID)
	}
	if seconds > maxWaitSeconds {
		return fmt.Errorf("wait node %s config.seconds cannot exceed %d", node.ID, maxWaitSeconds)
	}
	return nil
}

func evaluateCondition(node Node, input interface{}) (bool, map[string]interface{}, error) {
	operator := conditionOperator(node)
	path := nodeConfigString(node, "path")
	actual, exists := resolveWorkflowValue(input, path)
	payload := map[string]interface{}{
		"operator": operator,
		"path":     path,
		"exists":   exists,
	}
	if exists {
		payload["actual"] = actual
	}
	if nodeConfigExists(node, "value") {
		payload["expected"] = node.Config["value"]
	}
	switch operator {
	case "truthy":
		payload["matched"] = exists && isTruthy(actual)
		return payload["matched"].(bool), payload, nil
	case "exists":
		payload["matched"] = exists
		return exists, payload, nil
	case "equals":
		matched := exists && sameJSONValue(actual, node.Config["value"])
		payload["matched"] = matched
		return matched, payload, nil
	case "not_equals":
		matched := !exists || !sameJSONValue(actual, node.Config["value"])
		payload["matched"] = matched
		return matched, payload, nil
	case "contains":
		matched := exists && containsJSONValue(actual, node.Config["value"])
		payload["matched"] = matched
		return matched, payload, nil
	default:
		return false, nil, errors.New("unsupported condition operator")
	}
}

func conditionOperator(node Node) string {
	operator := nodeConfigString(node, "operator")
	if operator == "" {
		return "truthy"
	}
	return operator
}

func conditionFalseBehavior(node Node) string {
	behavior := nodeConfigString(node, "onFalse")
	if behavior == "" {
		return "skip"
	}
	return behavior
}

func parseWaitDuration(node Node) (time.Duration, error) {
	seconds, ok := nodeConfigInt(node, "seconds")
	if !ok || seconds <= 0 {
		return 0, errors.New("wait node requires positive seconds")
	}
	if seconds > maxWaitSeconds {
		return 0, fmt.Errorf("wait node seconds cannot exceed %d", maxWaitSeconds)
	}
	return time.Duration(seconds) * time.Second, nil
}

func resolveWorkflowValue(root interface{}, path string) (interface{}, bool) {
	path = strings.TrimSpace(path)
	if path == "" {
		if root == nil {
			return nil, false
		}
		return root, true
	}
	current := root
	for _, part := range splitPath(path) {
		name, indexes, ok := parsePathPart(part)
		if !ok {
			return nil, false
		}
		if name != "" {
			object, ok := current.(map[string]interface{})
			if !ok {
				return nil, false
			}
			value, exists := object[name]
			if !exists {
				return nil, false
			}
			current = value
		}
		for _, index := range indexes {
			items, ok := current.([]interface{})
			if !ok || index < 0 || index >= len(items) {
				return nil, false
			}
			current = items[index]
		}
	}
	return current, true
}

func splitPath(path string) []string {
	parts := make([]string, 0, 4)
	var builder strings.Builder
	brackets := 0
	for _, ch := range path {
		if ch == '.' && brackets == 0 {
			parts = append(parts, builder.String())
			builder.Reset()
			continue
		}
		if ch == '[' {
			brackets++
		} else if ch == ']' && brackets > 0 {
			brackets--
		}
		builder.WriteRune(ch)
	}
	if builder.Len() > 0 {
		parts = append(parts, builder.String())
	}
	return parts
}

func parsePathPart(part string) (string, []int, bool) {
	part = strings.TrimSpace(part)
	if part == "" {
		return "", nil, false
	}
	name := part
	indexes := make([]int, 0, 1)
	for {
		open := strings.Index(name, "[")
		if open < 0 {
			break
		}
		close := strings.Index(name[open:], "]")
		if close <= 1 {
			return "", nil, false
		}
		indexValue, err := strconv.Atoi(name[open+1 : open+close])
		if err != nil {
			return "", nil, false
		}
		indexes = append(indexes, indexValue)
		name = name[:open] + name[open+close+1:]
	}
	return strings.TrimSpace(name), indexes, true
}

func isTruthy(value interface{}) bool {
	switch typed := value.(type) {
	case nil:
		return false
	case bool:
		return typed
	case string:
		return strings.TrimSpace(typed) != ""
	case float64:
		return typed != 0
	case []interface{}:
		return len(typed) > 0
	case map[string]interface{}:
		return len(typed) > 0
	default:
		return true
	}
}

func containsJSONValue(container interface{}, expected interface{}) bool {
	switch typed := container.(type) {
	case string:
		text, ok := expected.(string)
		return ok && strings.Contains(typed, text)
	case []interface{}:
		for _, item := range typed {
			if sameJSONValue(item, expected) {
				return true
			}
		}
	}
	return false
}

func sameJSONValue(left, right interface{}) bool {
	leftBytes, leftErr := json.Marshal(left)
	rightBytes, rightErr := json.Marshal(right)
	return leftErr == nil && rightErr == nil && string(leftBytes) == string(rightBytes)
}
