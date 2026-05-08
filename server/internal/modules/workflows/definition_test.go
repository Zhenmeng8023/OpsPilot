package workflows

import "testing"

func TestNormalizeAndValidateDefinitionAcceptsDAG(t *testing.T) {
	_, normalized, err := normalizeAndValidateDefinition(`{
		"nodes":[
			{"id":"collect","type":"task","name":"Collect","config":{"taskId":"task-1"}},
			{"id":"decide","type":"condition","name":"Decide"},
			{"id":"notify","type":"notification","name":"Notify"}
		],
		"edges":[
			{"from":"collect","to":"decide"},
			{"from":"decide","to":"notify"}
		]
	}`)
	if err != nil {
		t.Fatalf("expected definition to be valid: %v", err)
	}
	if normalized == "" {
		t.Fatal("expected normalized JSON")
	}
}

func TestNormalizeAndValidateDefinitionRejectsCycle(t *testing.T) {
	_, _, err := normalizeAndValidateDefinition(`{
		"nodes":[
			{"id":"a","type":"task","config":{"taskId":"task-a"}},
			{"id":"b","type":"task","config":{"taskId":"task-b"}}
		],
		"edges":[
			{"from":"a","to":"b"},
			{"from":"b","to":"a"}
		]
	}`)
	if err == nil {
		t.Fatal("expected cycle to be rejected")
	}
}

func TestNormalizeAndValidateDefinitionRejectsUnknownNode(t *testing.T) {
	_, _, err := normalizeAndValidateDefinition(`{
		"nodes":[{"id":"a","type":"task","config":{"taskId":"task-a"}}],
		"edges":[{"from":"a","to":"missing"}]
	}`)
	if err == nil {
		t.Fatal("expected unknown edge node to be rejected")
	}
}

func TestNormalizeAndValidateDefinitionRejectsIsolatedNode(t *testing.T) {
	_, _, err := normalizeAndValidateDefinition(`{
		"nodes":[
			{"id":"a","type":"task","config":{"taskId":"task-a"}},
			{"id":"b","type":"task","config":{"taskId":"task-b"}},
			{"id":"isolated","type":"notification","name":"Isolated"}
		],
		"edges":[{"from":"a","to":"b"}]
	}`)
	if err == nil {
		t.Fatal("expected isolated node to be rejected")
	}
}

func TestNormalizeAndValidateDefinitionRejectsTaskWithoutTaskID(t *testing.T) {
	_, _, err := normalizeAndValidateDefinition(`{
		"nodes":[{"id":"a","type":"task"}],
		"edges":[]
	}`)
	if err == nil {
		t.Fatal("expected task node without taskId to be rejected")
	}
}

func TestNormalizeAndValidateDefinitionRejectsInvalidConditionConfig(t *testing.T) {
	_, _, err := normalizeAndValidateDefinition(`{
		"nodes":[{"id":"gate","type":"condition","config":{"operator":"equals"}}],
		"edges":[]
	}`)
	if err == nil {
		t.Fatal("expected condition node without value to be rejected")
	}
}

func TestNormalizeAndValidateDefinitionRejectsInvalidWaitConfig(t *testing.T) {
	_, _, err := normalizeAndValidateDefinition(`{
		"nodes":[{"id":"pause","type":"wait","config":{"seconds":0}}],
		"edges":[]
	}`)
	if err == nil {
		t.Fatal("expected wait node without positive seconds to be rejected")
	}
}

func TestNormalizeAndValidateDefinitionAcceptsWebhookCallAndApproval(t *testing.T) {
	_, normalized, err := normalizeAndValidateDefinition(`{
		"nodes":[
			{"id":"gate","type":"approval","config":{"comment":"release"}},
			{"id":"callback","type":"webhook-call","config":{"url":"https://example.com/hooks","method":"POST"}}
		],
		"edges":[{"from":"gate","to":"callback"}]
	}`)
	if err != nil {
		t.Fatalf("expected definition to be valid: %v", err)
	}
	if normalized == "" {
		t.Fatal("expected normalized definition")
	}
}

func TestWorkflowDependenciesReadyStopOnFailure(t *testing.T) {
	def, _, err := normalizeAndValidateDefinition(`{
		"nodes":[{"id":"a","type":"task","config":{"taskId":"task-a"}},{"id":"b","type":"task","config":{"taskId":"task-b"}}],
		"edges":[{"from":"a","to":"b"}],
		"failurePolicy":"stop_on_failure"
	}`)
	if err != nil {
		t.Fatalf("expected definition to be valid: %v", err)
	}
	statuses := map[string]string{"a": "failed", "b": "pending"}
	if !dependenciesReady("b", def, statuses) {
		t.Fatal("failed upstream should make the dependent node ready for skip")
	}
	if !shouldSkipNode("b", def, statuses) {
		t.Fatal("failed upstream should skip dependent node with stop_on_failure")
	}
}

func TestWorkflowDependenciesReadyContinuePolicy(t *testing.T) {
	def, _, err := normalizeAndValidateDefinition(`{
		"nodes":[{"id":"a","type":"task","config":{"taskId":"task-a"}},{"id":"b","type":"task","config":{"taskId":"task-b"}}],
		"edges":[{"from":"a","to":"b"}],
		"failurePolicy":"continue"
	}`)
	if err != nil {
		t.Fatalf("expected definition to be valid: %v", err)
	}
	statuses := map[string]string{"a": "failed", "b": "pending"}
	if !dependenciesReady("b", def, statuses) {
		t.Fatal("terminal failed upstream should release dependent node with continue policy")
	}
	if shouldSkipNode("b", def, statuses) {
		t.Fatal("continue policy should not skip dependent node")
	}
}

func TestWorkflowDependenciesSkipPropagation(t *testing.T) {
	def, _, err := normalizeAndValidateDefinition(`{
		"nodes":[{"id":"a","type":"condition"},{"id":"b","type":"task","config":{"taskId":"task-b"}}],
		"edges":[{"from":"a","to":"b"}],
		"failurePolicy":"continue"
	}`)
	if err != nil {
		t.Fatalf("expected definition to be valid: %v", err)
	}
	statuses := map[string]string{"a": "skipped", "b": "pending"}
	if !dependenciesReady("b", def, statuses) {
		t.Fatal("skipped upstream should make dependent node ready for skip propagation")
	}
	if !shouldSkipNode("b", def, statuses) {
		t.Fatal("skipped upstream should skip dependent node even with continue policy")
	}
}

func TestWorkflowStatusFromTask(t *testing.T) {
	cases := map[string]string{
		"queued":   "queued",
		"running":  "running",
		"success":  "success",
		"failed":   "failed",
		"timeout":  "failed",
		"canceled": "canceled",
	}
	for input, expected := range cases {
		if actual := workflowStatusFromTask(input); actual != expected {
			t.Fatalf("expected %s to map to %s, got %s", input, expected, actual)
		}
	}
}

func TestEvaluateConditionEquals(t *testing.T) {
	node := Node{
		ID:   "gate",
		Type: "condition",
		Config: map[string]interface{}{
			"path":     "environment.name",
			"operator": "equals",
			"value":    "prod",
		},
	}
	input := map[string]interface{}{
		"environment": map[string]interface{}{
			"name": "prod",
		},
	}
	matched, payload, err := evaluateCondition(node, input)
	if err != nil {
		t.Fatalf("expected condition evaluation to succeed: %v", err)
	}
	if !matched {
		t.Fatal("expected condition to match")
	}
	if payload["path"] != "environment.name" {
		t.Fatalf("expected payload path to be preserved, got %#v", payload["path"])
	}
}

func TestEvaluateConditionContainsArrayItem(t *testing.T) {
	node := Node{
		ID:   "gate",
		Type: "condition",
		Config: map[string]interface{}{
			"path":     "labels",
			"operator": "contains",
			"value":    "prod",
		},
	}
	input := map[string]interface{}{
		"labels": []interface{}{"ops", "prod"},
	}
	matched, _, err := evaluateCondition(node, input)
	if err != nil {
		t.Fatalf("expected condition evaluation to succeed: %v", err)
	}
	if !matched {
		t.Fatal("expected contains operator to match")
	}
}

func TestResolveWorkflowValueSupportsArrayIndex(t *testing.T) {
	input := map[string]interface{}{
		"targets": []interface{}{
			map[string]interface{}{"name": "node-a"},
		},
	}
	value, exists := resolveWorkflowValue(input, "targets[0].name")
	if !exists {
		t.Fatal("expected indexed path lookup to succeed")
	}
	if text, ok := value.(string); !ok || text != "node-a" {
		t.Fatalf("expected node-a, got %#v", value)
	}
}
