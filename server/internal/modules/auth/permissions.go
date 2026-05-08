package auth

import (
	"sort"
	"strings"
)

var permissionAliases = map[string][]string{
	"agent:read":         {"agent:read", "agent.read"},
	"script:read":        {"script:read", "script.read"},
	"script:write":       {"script:write", "script.write"},
	"script:approve":     {"script:approve", "script.approve"},
	"task:read":          {"task:read", "task.read"},
	"task:execute":       {"task:execute", "task.run"},
	"task:cancel":        {"task:cancel", "task.cancel"},
	"task:log:read":      {"task:log:read", "log.read"},
	"schedule:write":     {"schedule:write", "schedule.write"},
	"metric:read":        {"metric:read", "metric.read"},
	"alert:write":        {"alert:write", "alert.write"},
	"webhook:manage":     {"webhook:manage", "webhook.manage"},
	"notification:write": {"notification:write", "notification.write"},
	"workflow:read":       {"workflow:read", "workflow.read"},
	"workflow:manage":     {"workflow:manage", "workflow:write", "workflow.write", "workflow:cancel", "workflow.cancel"},
	"workflow:execute":    {"workflow:execute", "workflow.execute"},
}

var permissionCanonical = buildPermissionCanonical(permissionAliases)

func buildPermissionCanonical(groups map[string][]string) map[string]string {
	index := make(map[string]string, len(groups)*2)
	for canonical, aliases := range groups {
		index[canonical] = canonical
		for _, alias := range aliases {
			index[alias] = canonical
		}
	}
	return index
}

func canonicalPermissionCode(value string) string {
	code := strings.TrimSpace(value)
	if code == "" {
		return ""
	}
	if canonical, ok := permissionCanonical[code]; ok {
		return canonical
	}
	return code
}

func permissionMatchCodes(value string) []string {
	canonical := canonicalPermissionCode(value)
	if canonical == "" {
		return []string{}
	}
	if aliases, ok := permissionAliases[canonical]; ok {
		return aliases
	}
	return []string{canonical}
}

func canonicalizePermissionList(values []string) []string {
	seen := map[string]bool{}
	permissions := make([]string, 0, len(values))
	for _, value := range values {
		code := canonicalPermissionCode(value)
		if code == "" || seen[code] {
			continue
		}
		seen[code] = true
		permissions = append(permissions, code)
	}
	sort.Strings(permissions)
	return permissions
}

func canonicalizePermissionSummaries(values []PermissionSummary) []PermissionSummary {
	index := make(map[string]PermissionSummary, len(values))
	for _, value := range values {
		original := strings.TrimSpace(value.Code)
		canonical := canonicalPermissionCode(original)
		if canonical == "" {
			continue
		}
		value.Code = canonical
		current, exists := index[canonical]
		if !exists || original == canonical {
			index[canonical] = value
			continue
		}
		index[canonical] = current
	}
	permissions := make([]PermissionSummary, 0, len(index))
	for _, value := range index {
		permissions = append(permissions, value)
	}
	sort.Slice(permissions, func(i, j int) bool {
		if permissions[i].Module == permissions[j].Module {
			return permissions[i].Code < permissions[j].Code
		}
		return permissions[i].Module < permissions[j].Module
	})
	return permissions
}

func normalizePermissionCodes(values []string) []string {
	return canonicalizePermissionList(values)
}
