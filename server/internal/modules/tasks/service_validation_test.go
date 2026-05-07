package tasks

import (
	"context"
	"testing"
)

func TestCreateValidationFailures(t *testing.T) {
	service := &Service{}
	cases := []struct {
		name  string
		input CreateTaskInput
		code  int
	}{
		{
			name:  "missing name",
			input: CreateTaskInput{TimeoutSeconds: 30, Command: "echo ok", ScriptType: "shell", TargetAgentIDs: []string{"agent-1"}},
			code:  400301,
		},
		{
			name:  "invalid timeout",
			input: CreateTaskInput{Name: "task", TimeoutSeconds: 0, Command: "echo ok", ScriptType: "shell", TargetAgentIDs: []string{"agent-1"}},
			code:  400302,
		},
		{
			name:  "script or command required",
			input: CreateTaskInput{Name: "task", TimeoutSeconds: 30, TargetAgentIDs: []string{"agent-1"}},
			code:  400303,
		},
		{
			name:  "unsupported command type",
			input: CreateTaskInput{Name: "task", TimeoutSeconds: 30, Command: "echo ok", ScriptType: "python", TargetAgentIDs: []string{"agent-1"}},
			code:  400304,
		},
		{
			name:  "targets required",
			input: CreateTaskInput{Name: "task", TimeoutSeconds: 30, Command: "echo ok", ScriptType: "shell"},
			code:  400305,
		},
	}
	for _, item := range cases {
		t.Run(item.name, func(t *testing.T) {
			_, appErr := service.Create(context.Background(), item.input)
			if appErr == nil {
				t.Fatalf("expected app error code %d", item.code)
			}
			if appErr.Code != item.code {
				t.Fatalf("expected app error code %d, got %d", item.code, appErr.Code)
			}
		})
	}
}

func TestUploadLogValidationFailures(t *testing.T) {
	service := &Service{}

	if appErr := service.UploadLog(context.Background(), AgentIdentity{}, LogInput{
		TargetID: "",
		Stream:   "stdout",
		Chunk:    "hello",
	}); appErr == nil || appErr.Code != 400001 {
		t.Fatalf("expected target id validation failure, got %#v", appErr)
	}

	if appErr := service.UploadLog(context.Background(), AgentIdentity{}, LogInput{
		TargetID: "target-1",
		Stream:   "file",
		Chunk:    "hello",
	}); appErr == nil || appErr.Code != 400307 {
		t.Fatalf("expected stream validation failure, got %#v", appErr)
	}

	if appErr := service.UploadLog(context.Background(), AgentIdentity{}, LogInput{
		TargetID: "target-1",
		Stream:   "stdout",
		Chunk:    "",
	}); appErr != nil {
		t.Fatalf("expected empty chunk to be ignored, got %#v", appErr)
	}
}

func TestReportResultValidationFailure(t *testing.T) {
	service := &Service{}
	appErr := service.ReportResult(context.Background(), AgentIdentity{}, ResultInput{
		TargetID: "target-1",
		Status:   "queued",
	})
	if appErr == nil || appErr.Code != 400308 {
		t.Fatalf("expected invalid status error, got %#v", appErr)
	}
}

func TestClaimValidationFailure(t *testing.T) {
	service := &Service{}
	_, appErr := service.Claim(context.Background(), AgentIdentity{}, "")
	if appErr == nil || appErr.Code != 400001 {
		t.Fatalf("expected empty target id error, got %#v", appErr)
	}
}

func TestCancelValidationFailure(t *testing.T) {
	service := &Service{}
	appErr := service.Cancel(context.Background(), "", AuditContext{})
	if appErr == nil || appErr.Code != 400001 {
		t.Fatalf("expected empty task id error, got %#v", appErr)
	}
}
