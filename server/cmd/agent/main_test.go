package main

import (
	"context"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestRunCommandSuccess(t *testing.T) {
	task := agentTask{TargetID: "target-1", ScriptType: "shell", Command: "echo ok", TimeoutSeconds: 5}
	var logs []string
	result := runCommand(context.Background(), task, ".", func(stream, chunk string) {
		logs = append(logs, stream+":"+chunk)
	})
	if result.Status != "success" || result.ExitCode == nil || *result.ExitCode != 0 {
		t.Fatalf("unexpected result: %+v", result)
	}
	if !containsLog(logs, "ok") {
		t.Fatalf("stdout was not captured: %#v", logs)
	}
}

func TestRunCommandNonZeroExit(t *testing.T) {
	task := agentTask{TargetID: "target-1", ScriptType: "shell", Command: "exit 7", TimeoutSeconds: 5}
	result := runCommand(context.Background(), task, ".", func(string, string) {})
	if result.Status != "failed" || result.ExitCode == nil || *result.ExitCode == 0 {
		t.Fatalf("expected failed non-zero exit, got %+v", result)
	}
}

func TestRunCommandTimeout(t *testing.T) {
	command := "sleep 2"
	if isWindows() {
		command = "ping 127.0.0.1 -n 3 > nul"
	}
	task := agentTask{TargetID: "target-1", ScriptType: "shell", Command: command, TimeoutSeconds: 1}
	start := time.Now()
	result := runCommand(context.Background(), task, ".", func(string, string) {})
	if result.Status != "timeout" {
		t.Fatalf("expected timeout, got %+v", result)
	}
	if time.Since(start) > 3*time.Second {
		t.Fatalf("timeout did not stop command promptly")
	}
}

func containsLog(logs []string, want string) bool {
	for _, line := range logs {
		if strings.Contains(line, want) {
			return true
		}
	}
	return false
}

func isWindows() bool {
	return runtime.GOOS == "windows"
}
