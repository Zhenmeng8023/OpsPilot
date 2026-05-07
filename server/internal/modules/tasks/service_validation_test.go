package tasks

import (
	"context"
	"testing"

	"opspilot/server/internal/config"
	"opspilot/server/internal/shared/apperror"
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
			name:  "timeout exceeds maximum",
			input: CreateTaskInput{Name: "task", TimeoutSeconds: maxTaskTimeoutSeconds + 1, Command: "echo ok", ScriptType: "shell", TargetAgentIDs: []string{"agent-1"}},
			code:  400311,
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
		{
			name:  "high risk command blocked",
			input: CreateTaskInput{Name: "task", TimeoutSeconds: 30, Command: "echo before; rm -rf /tmp/opspilot-demo", ScriptType: "shell", TargetAgentIDs: []string{"agent-1"}},
			code:  400310,
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

func TestValidateCommandSafety(t *testing.T) {
	cases := []struct {
		name    string
		command string
		blocked bool
	}{
		{name: "safe echo", command: "echo rm is text only", blocked: false},
		{name: "rm command", command: "rm -rf /tmp/opspilot-demo", blocked: true},
		{name: "del command", command: "echo before && del /f secret.txt", blocked: true},
		{name: "shutdown command", command: "shutdown /s /t 0", blocked: true},
		{name: "mkfs variant", command: "mkfs.ext4 /dev/sdb1", blocked: true},
		{name: "powershell remove item", command: "Remove-Item -Recurse C:\\temp\\demo", blocked: true},
	}

	for _, item := range cases {
		t.Run(item.name, func(t *testing.T) {
			err := validateCommandSafety(item.command, config.CommandPolicyConfig{})
			if item.blocked {
				if err == nil || err.Code != 400310 {
					t.Fatalf("expected high-risk command block, got %#v", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("expected command to pass, got %#v", err)
			}
		})
	}
}

func TestValidateCommandSafetyWithConfiguredPolicy(t *testing.T) {
	if err := validateCommandSafety("echo blocked", config.CommandPolicyConfig{
		DenyPatterns: []string{`(?i)\bblocked\b`},
	}); err == nil || err.Code != 400310 {
		t.Fatalf("expected configured denylist block, got %#v", err)
	}

	if err := validateCommandSafety("echo ok", config.CommandPolicyConfig{
		AllowPatterns: []string{`(?i)^echo\s+`},
	}); err != nil {
		t.Fatalf("expected configured allowlist to pass, got %#v", err)
	}

	if err := validateCommandSafety("whoami", config.CommandPolicyConfig{
		AllowPatterns: []string{`(?i)^echo\s+`},
	}); err == nil || err.Code != 400312 {
		t.Fatalf("expected configured allowlist block, got %#v", err)
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

func TestValidateScriptApproval(t *testing.T) {
	cases := []struct {
		name     string
		script   scriptRecord
		wantErr  bool
		wantCode int
	}{
		{
			name: "approval not required",
			script: scriptRecord{
				ApprovalRequired: false,
			},
			wantErr: false,
		},
		{
			name: "approved",
			script: scriptRecord{
				ApprovalRequired:     true,
				LatestApprovalStatus: nullString("approved"),
			},
			wantErr: false,
		},
		{
			name: "pending",
			script: scriptRecord{
				ApprovalRequired:     true,
				LatestApprovalStatus: nullString("pending"),
			},
			wantErr:  true,
			wantCode: 409306,
		},
		{
			name: "rejected",
			script: scriptRecord{
				ApprovalRequired:     true,
				LatestApprovalStatus: nullString("rejected"),
			},
			wantErr:  true,
			wantCode: 400309,
		},
		{
			name: "missing approval record",
			script: scriptRecord{
				ApprovalRequired: true,
			},
			wantErr:  true,
			wantCode: 400309,
		},
	}
	for _, item := range cases {
		t.Run(item.name, func(t *testing.T) {
			err := validateScriptApproval(item.script)
			if !item.wantErr {
				if err != nil {
					t.Fatalf("did not expect error, got %#v", err)
				}
				return
			}
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			appErr, ok := err.(*apperror.Error)
			if !ok {
				t.Fatalf("expected apperror.Error, got %T", err)
			}
			if appErr.Code != item.wantCode {
				t.Fatalf("expected code %d, got %d", item.wantCode, appErr.Code)
			}
		})
	}
}
