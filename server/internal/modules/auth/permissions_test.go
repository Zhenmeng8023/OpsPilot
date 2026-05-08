package auth

import "testing"

func TestNormalizePermissionCodesCanonicalizesAliases(t *testing.T) {
	permissions := normalizePermissionCodes([]string{
		" metric.read ",
		"metric:read",
		"task.run",
		"task:execute",
		"workflow:write",
		"workflow:manage",
		"script.read",
		"",
	})

	expected := []string{"metric:read", "script:read", "task:execute", "workflow:manage"}
	if len(permissions) != len(expected) {
		t.Fatalf("expected %d permissions, got %d: %#v", len(expected), len(permissions), permissions)
	}
	for index, value := range expected {
		if permissions[index] != value {
			t.Fatalf("expected permission %q at index %d, got %q", value, index, permissions[index])
		}
	}
}

func TestPermissionMatchCodesIncludesLegacyAliases(t *testing.T) {
	permissions := permissionMatchCodes("workflow:write")
	expected := []string{"workflow:manage", "workflow:write", "workflow.write", "workflow:cancel", "workflow.cancel"}
	if len(permissions) != len(expected) {
		t.Fatalf("expected %d aliases, got %d: %#v", len(expected), len(permissions), permissions)
	}
	for index, value := range expected {
		if permissions[index] != value {
			t.Fatalf("expected alias %q at index %d, got %q", value, index, permissions[index])
		}
	}
}

func TestCanonicalizePermissionSummariesPrefersCanonicalCode(t *testing.T) {
	permissions := canonicalizePermissionSummaries([]PermissionSummary{
		{Code: "metric.read", Module: "metric", Name: "Read metrics", Description: "legacy"},
		{Code: "metric:read", Module: "metric", Name: "Read metrics", Description: "canonical"},
		{Code: "task.run", Module: "task", Name: "Run tasks"},
	})

	if len(permissions) != 2 {
		t.Fatalf("expected 2 permissions, got %d: %#v", len(permissions), permissions)
	}
	if permissions[0].Code != "metric:read" || permissions[0].Description != "canonical" {
		t.Fatalf("unexpected metric permission: %#v", permissions[0])
	}
	if permissions[1].Code != "task:execute" {
		t.Fatalf("unexpected task permission: %#v", permissions[1])
	}
}
