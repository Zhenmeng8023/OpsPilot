package main

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"opspilot/server/internal/config"
)

func TestRunCommandSuccess(t *testing.T) {
	task := agentTask{TargetID: "target-1", ScriptType: "shell", Command: "echo ok", TimeoutSeconds: 5}
	var logs []string
	result := runCommand(context.Background(), task, t.TempDir(), func(stream, chunk string) {
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
	result := runCommand(context.Background(), task, t.TempDir(), func(string, string) {})
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
	result := runCommand(context.Background(), task, t.TempDir(), func(string, string) {})
	if result.Status != "timeout" {
		t.Fatalf("expected timeout, got %+v", result)
	}
	if time.Since(start) > 3*time.Second {
		t.Fatalf("timeout did not stop command promptly")
	}
}

func TestRunCommandCanceled(t *testing.T) {
	command := "sleep 5"
	scriptType := "shell"
	if isWindows() {
		command = "Start-Sleep -Seconds 5"
		scriptType = "powershell"
	}
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(200 * time.Millisecond)
		cancel()
	}()
	task := agentTask{TargetID: "target-1", ScriptType: scriptType, Command: command, TimeoutSeconds: 30}

	result := runCommand(ctx, task, t.TempDir(), func(string, string) {})

	if result.Status != "canceled" {
		t.Fatalf("expected canceled result, got %+v", result)
	}
}

func TestRunCommandUsesIndependentTaskWorkDir(t *testing.T) {
	baseDir := t.TempDir()
	command := "pwd > cwd.txt"
	expectedOutputFile := "cwd.txt"
	if isWindows() {
		command = "cd > cwd.txt"
	}
	task := agentTask{TargetID: "target/with unsafe chars", ScriptType: "shell", Command: command, TimeoutSeconds: 5}

	result := runCommand(context.Background(), task, baseDir, func(string, string) {})

	if result.Status != "success" {
		t.Fatalf("unexpected result: %+v", result)
	}
	wantDir := filepath.Join(baseDir, "target_with_unsafe_chars")
	bytes, err := os.ReadFile(filepath.Join(wantDir, expectedOutputFile))
	if err != nil {
		t.Fatalf("read cwd marker: %v", err)
	}
	gotDir := strings.TrimSpace(string(bytes))
	if gotDir != wantDir {
		t.Fatalf("expected command to run in %q, got %q", wantDir, gotDir)
	}
}

func TestSafePathSegment(t *testing.T) {
	if got := safePathSegment("../target one"); got != ".._target_one" {
		t.Fatalf("unexpected safe path segment: %q", got)
	}
	if got := safePathSegment(""); got != "task" {
		t.Fatalf("expected default segment, got %q", got)
	}
	if got := safePathSegment(".."); got != "task" {
		t.Fatalf("expected traversal segment to be replaced, got %q", got)
	}
}

func TestCollectMetricsIncludesSystemMetrics(t *testing.T) {
	exec := &executor{cfg: config.Config{}}
	exec.cfg.Agent.WorkDir = t.TempDir()
	exec.active.Store(3)

	metrics := collectMetrics(exec)
	required := []string{
		"agent.running_tasks",
		"agent.cpu.logical",
		"agent.runtime.goroutines",
		"agent.runtime.alloc_bytes",
		"agent.runtime.sys_bytes",
		"agent.os.memory.total_bytes",
		"agent.os.disk.total_bytes",
	}
	for _, code := range required {
		if !hasMetricCode(metrics, code) {
			t.Fatalf("expected metric %q in payload", code)
		}
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

func hasMetricCode(metrics []metricPayload, code string) bool {
	for _, metric := range metrics {
		if metric.Code == code {
			return true
		}
	}
	return false
}

func isWindows() bool {
	return runtime.GOOS == "windows"
}
