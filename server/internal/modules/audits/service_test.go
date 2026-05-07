package audits

import "testing"

func TestNormalizePage(t *testing.T) {
	page, pageSize := normalizePage(0, 0)
	if page != 1 || pageSize != 20 {
		t.Fatalf("normalizePage default = %d, %d", page, pageSize)
	}
	page, pageSize = normalizePage(2, 500)
	if page != 2 || pageSize != 100 {
		t.Fatalf("normalizePage capped = %d, %d", page, pageSize)
	}
}

func TestAuditWhereIncludesFilters(t *testing.T) {
	where, args := auditWhere(7, ListInput{
		Action:       "task.create",
		ActorType:    "user",
		Result:       "success",
		ResourceType: "task_run",
		TraceID:      "trace-1",
		Keyword:      "deploy",
	})
	if len(args) != 12 {
		t.Fatalf("expected 12 args, got %d: %#v", len(args), args)
	}
	for _, want := range []string{"al.workspace_id", "al.action = ?", "al.actor_type = ?", "al.result = ?", "al.resource_type = ?", "al.trace_id = ?", "al.action LIKE ?"} {
		if !stringsContains(where, want) {
			t.Fatalf("where missing %q: %s", want, where)
		}
	}
}

func stringsContains(value, part string) bool {
	for index := 0; index+len(part) <= len(value); index++ {
		if value[index:index+len(part)] == part {
			return true
		}
	}
	return false
}
