package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"opspilot/server/internal/config"
	"opspilot/server/internal/platform/logger"
	"opspilot/server/internal/shared/security"
)

type apiClient struct {
	baseURL string
	http    *http.Client
}

type envelope[T any] struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    T      `json:"data"`
	TraceID string `json:"traceId"`
}

type apiError struct {
	status  int
	message string
}

type agentPayload struct {
	BootstrapSecret string                 `json:"bootstrapSecret,omitempty"`
	EnrollmentToken string                 `json:"enrollmentToken,omitempty"`
	Name            string                 `json:"name,omitempty"`
	Hostname        string                 `json:"hostname,omitempty"`
	IP              string                 `json:"ip,omitempty"`
	OS              string                 `json:"os,omitempty"`
	OSType          string                 `json:"osType,omitempty"`
	OSName          string                 `json:"osName,omitempty"`
	Arch            string                 `json:"arch,omitempty"`
	Version         string                 `json:"version,omitempty"`
	Status          string                 `json:"status,omitempty"`
	RunningTasks    int64                  `json:"runningTasks,omitempty"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
}

type registerData struct {
	Token       string `json:"token"`
	TokenPrefix string `json:"tokenPrefix"`
}

type agentTask struct {
	TargetID       string `json:"targetId"`
	RunID          string `json:"runId"`
	Name           string `json:"name"`
	Description    string `json:"description"`
	ScriptType     string `json:"scriptType"`
	Command        string `json:"command"`
	TimeoutSeconds uint   `json:"timeoutSeconds"`
}

type targetState struct {
	TargetID string `json:"targetId"`
	Status   string `json:"status"`
}

type logPayload struct {
	Sequence  uint64 `json:"sequence"`
	Stream    string `json:"stream"`
	Chunk     string `json:"chunk"`
	Timestamp string `json:"timestamp"`
}

type resultPayload struct {
	Status       string `json:"status"`
	ExitCode     *int   `json:"exitCode"`
	ErrorMessage string `json:"errorMessage,omitempty"`
	StartedAt    string `json:"startedAt"`
	FinishedAt   string `json:"finishedAt"`
}

type metricPayload struct {
	Code        string                 `json:"code"`
	Value       float64                `json:"value"`
	Unit        string                 `json:"unit,omitempty"`
	Dimensions  map[string]interface{} `json:"dimensions,omitempty"`
	CollectedAt string                 `json:"collectedAt,omitempty"`
}

type metricsPayload struct {
	Metrics []metricPayload `json:"metrics"`
}

type executor struct {
	cfg     config.Config
	client  apiClient
	token   string
	log     *slog.Logger
	sem     chan struct{}
	running sync.Map
	active  atomic.Int64
}

type commandResult struct {
	Status       string
	ExitCode     *int
	ErrorMessage string
	StartedAt    time.Time
	FinishedAt   time.Time
}

func (e apiError) Error() string {
	return e.message
}

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("load config failed", "error", err)
		os.Exit(1)
	}
	log := logger.New(cfg.App.Env)
	if err := os.MkdirAll(cfg.Agent.WorkDir, 0o700); err != nil {
		log.Error("create agent work dir failed", "error", err)
		os.Exit(1)
	}

	client := apiClient{
		baseURL: strings.TrimRight(cfg.Agent.APIBaseURL, "/"),
		http:    &http.Client{Timeout: 20 * time.Second},
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	token, err := loadAgentToken(cfg)
	if err != nil {
		log.Error("read agent token failed", "error", err)
		os.Exit(1)
	}
	if token == "" {
		token, err = register(ctx, client, cfg)
		if err != nil {
			log.Error("register agent failed", "error", err)
			os.Exit(1)
		}
		if err := writeToken(cfg.Agent.TokenFile, token); err != nil {
			log.Error("save agent token failed", "error", err)
			os.Exit(1)
		}
		log.Info("agent registered", "token_file", cfg.Agent.TokenFile)
	}

	exec := &executor{
		cfg:    cfg,
		client: client,
		token:  token,
		log:    log,
		sem:    make(chan struct{}, cfg.Agent.MaxConcurrentTasks),
	}
	if err := exec.heartbeat(ctx); err != nil {
		var apiErr apiError
		if errors.As(err, &apiErr) && apiErr.status == http.StatusUnauthorized && cfg.Agent.BootstrapSecret != "" {
			token, err = register(ctx, client, cfg)
			if err == nil {
				err = writeToken(cfg.Agent.TokenFile, token)
				exec.token = token
			}
		}
		if err != nil {
			log.Error("initial heartbeat failed", "error", err)
			os.Exit(1)
		}
	}
	if err := exec.uploadMetrics(ctx); err != nil {
		log.Warn("initial metrics upload failed", "error", security.Redact(err.Error()))
	}

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		exec.heartbeatLoop(ctx)
	}()
	go func() {
		defer wg.Done()
		exec.pollLoop(ctx)
	}()

	log.Info("opspilot agent running", "api_base_url", cfg.Agent.APIBaseURL, "max_concurrent_tasks", cfg.Agent.MaxConcurrentTasks)
	<-ctx.Done()
	wg.Wait()
	log.Info("opspilot agent stopped")
}

func (e *executor) heartbeatLoop(ctx context.Context) {
	interval := e.cfg.Agent.HeartbeatInterval
	if interval <= 0 {
		interval = 30 * time.Second
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := e.heartbeat(ctx); err != nil {
				e.log.Warn("heartbeat failed", "error", security.Redact(err.Error()))
			}
			if err := e.uploadMetrics(ctx); err != nil {
				e.log.Warn("metrics upload failed", "error", security.Redact(err.Error()))
			}
		}
	}
}

func (e *executor) pollLoop(ctx context.Context) {
	interval := e.cfg.Agent.PollInterval
	if interval <= 0 {
		interval = 5 * time.Second
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			e.pollOnce(ctx)
		}
	}
}

func (e *executor) pollOnce(ctx context.Context) {
	capacity := cap(e.sem) - len(e.sem)
	if capacity <= 0 {
		return
	}
	tasks, err := getJSON[[]agentTask](ctx, e.client, fmt.Sprintf("/api/v1/agent/tasks/poll?limit=%d", capacity), e.token)
	if err != nil {
		e.log.Warn("poll tasks failed", "error", security.Redact(err.Error()))
		return
	}
	for _, task := range tasks {
		if task.TargetID == "" || task.Command == "" || task.TimeoutSeconds == 0 {
			continue
		}
		if _, loaded := e.running.LoadOrStore(task.TargetID, true); loaded {
			continue
		}
		select {
		case e.sem <- struct{}{}:
		default:
			e.running.Delete(task.TargetID)
			return
		}
		e.active.Add(1)
		go e.runTarget(ctx, task)
	}
}

func (e *executor) runTarget(parent context.Context, task agentTask) {
	defer func() {
		<-e.sem
		e.active.Add(-1)
		e.running.Delete(task.TargetID)
	}()
	claimed, err := postJSON[agentTask](parent, e.client, "/api/v1/agent/tasks/"+task.TargetID+"/claim", e.token, map[string]string{})
	if err != nil {
		e.log.Warn("claim task failed", "target_id", task.TargetID, "error", security.Redact(err.Error()))
		return
	}
	sequence := atomic.Uint64{}
	upload := func(stream, chunk string) {
		payload := logPayload{
			Sequence:  sequence.Add(1),
			Stream:    stream,
			Chunk:     security.Redact(chunk),
			Timestamp: time.Now().Format(time.RFC3339Nano),
		}
		if err := e.uploadLog(parent, claimed.TargetID, payload); err != nil {
			e.log.Warn("upload task log failed", "target_id", claimed.TargetID, "error", security.Redact(err.Error()))
		}
	}
	upload("system", "task claimed by agent\n")
	runCtx, cancelRun := context.WithCancel(parent)
	defer cancelRun()
	done := make(chan struct{})
	cancelRequested := atomic.Bool{}
	go e.monitorCancellation(runCtx, claimed.TargetID, done, &cancelRequested, cancelRun, upload)
	result := runCommand(runCtx, claimed, e.cfg.Agent.WorkDir, upload)
	close(done)
	if cancelRequested.Load() && result.Status != "canceled" {
		result.Status = "canceled"
		result.ErrorMessage = "task canceled"
	}
	payload := resultPayload{
		Status:       result.Status,
		ExitCode:     result.ExitCode,
		ErrorMessage: security.Redact(result.ErrorMessage),
		StartedAt:    result.StartedAt.Format(time.RFC3339Nano),
		FinishedAt:   result.FinishedAt.Format(time.RFC3339Nano),
	}
	if err := e.reportResult(parent, claimed.TargetID, payload); err != nil {
		e.log.Warn("report task result failed", "target_id", claimed.TargetID, "error", security.Redact(err.Error()))
	}
}

func (e *executor) uploadLog(ctx context.Context, targetID string, payload logPayload) error {
	var lastErr error
	for attempt := 0; attempt < 3; attempt++ {
		_, err := postJSON[json.RawMessage](ctx, e.client, "/api/v1/agent/tasks/"+targetID+"/logs", e.token, payload)
		if err == nil {
			return nil
		}
		lastErr = err
		time.Sleep(time.Duration(attempt+1) * 300 * time.Millisecond)
	}
	return lastErr
}

func (e *executor) reportResult(ctx context.Context, targetID string, payload resultPayload) error {
	_, err := postJSON[json.RawMessage](ctx, e.client, "/api/v1/agent/tasks/"+targetID+"/result", e.token, payload)
	return err
}

func (e *executor) targetState(ctx context.Context, targetID string) (targetState, error) {
	return getJSON[targetState](ctx, e.client, "/api/v1/agent/tasks/"+targetID+"/status", e.token)
}

func (e *executor) monitorCancellation(ctx context.Context, targetID string, done <-chan struct{}, canceled *atomic.Bool, cancel context.CancelFunc, onLog func(stream, chunk string)) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-done:
			return
		case <-ctx.Done():
			return
		case <-ticker.C:
			state, err := e.targetState(ctx, targetID)
			if err != nil {
				e.log.Warn("check task target status failed", "target_id", targetID, "error", security.Redact(err.Error()))
				continue
			}
			if state.Status == "canceling" || state.Status == "canceled" {
				canceled.Store(true)
				if state.Status == "canceling" {
					onLog("system", "cancel requested; stopping command\n")
				}
				cancel()
				return
			}
		}
	}
}

func (e *executor) heartbeat(ctx context.Context) error {
	payload := collectHostInfo(e.cfg)
	payload.Status = "online"
	payload.RunningTasks = e.active.Load()
	_, err := postJSON[json.RawMessage](ctx, e.client, "/api/v1/agents/heartbeat", e.token, payload)
	return err
}

func (e *executor) uploadMetrics(ctx context.Context) error {
	payload := metricsPayload{Metrics: collectMetrics(e)}
	_, err := postJSON[json.RawMessage](ctx, e.client, "/api/v1/agent/metrics", e.token, payload)
	return err
}

func register(ctx context.Context, client apiClient, cfg config.Config) (string, error) {
	payload := collectHostInfo(cfg)
	payload.BootstrapSecret = cfg.Agent.BootstrapSecret
	payload.EnrollmentToken = os.Getenv("AGENT_ENROLLMENT_TOKEN")
	data, err := postJSON[registerData](ctx, client, "/api/v1/agents/register", "", payload)
	if err != nil {
		return "", err
	}
	if data.Token == "" {
		return "", errors.New("register response did not include token")
	}
	return data.Token, nil
}

func runCommand(parent context.Context, task agentTask, workDir string, onLog func(stream, chunk string)) commandResult {
	startedAt := time.Now()
	taskDir, err := prepareTaskWorkDir(workDir, task.TargetID)
	if err != nil {
		return failedResult(startedAt, err)
	}
	timeout := time.Duration(task.TimeoutSeconds) * time.Second
	if timeout <= 0 {
		timeout = time.Second
	}
	ctx, cancel := context.WithTimeout(parent, timeout)
	defer cancel()

	cmd := commandFor(ctx, task.ScriptType, task.Command)
	cmd.Dir = taskDir
	stdout := newLineStreamWriter("stdout", onLog)
	stderr := newLineStreamWriter("stderr", onLog)
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	if err := cmd.Start(); err != nil {
		return failedResult(startedAt, err)
	}

	waitErr := cmd.Wait()
	stdout.Flush()
	stderr.Flush()

	finishedAt := time.Now()
	exitCode := 0
	if cmd.ProcessState != nil {
		exitCode = cmd.ProcessState.ExitCode()
	}
	if errors.Is(ctx.Err(), context.Canceled) {
		return commandResult{Status: "canceled", ExitCode: &exitCode, ErrorMessage: "task canceled", StartedAt: startedAt, FinishedAt: finishedAt}
	}
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return commandResult{Status: "timeout", ExitCode: &exitCode, ErrorMessage: "command timed out", StartedAt: startedAt, FinishedAt: finishedAt}
	}
	if waitErr != nil || exitCode != 0 {
		message := ""
		if waitErr != nil {
			message = waitErr.Error()
		}
		return commandResult{Status: "failed", ExitCode: &exitCode, ErrorMessage: message, StartedAt: startedAt, FinishedAt: finishedAt}
	}
	return commandResult{Status: "success", ExitCode: &exitCode, StartedAt: startedAt, FinishedAt: finishedAt}
}

func commandFor(ctx context.Context, scriptType, command string) *exec.Cmd {
	scriptType = strings.ToLower(strings.TrimSpace(scriptType))
	if runtime.GOOS == "windows" {
		if scriptType == "powershell" {
			return exec.CommandContext(ctx, "powershell.exe", "-NoProfile", "-ExecutionPolicy", "Bypass", "-Command", command)
		}
		return exec.CommandContext(ctx, "cmd.exe", "/C", command)
	}
	if scriptType == "powershell" {
		if _, err := exec.LookPath("pwsh"); err == nil {
			return exec.CommandContext(ctx, "pwsh", "-NoProfile", "-Command", command)
		}
		return exec.CommandContext(ctx, "powershell", "-NoProfile", "-Command", command)
	}
	if scriptType == "bash" {
		return exec.CommandContext(ctx, "/bin/bash", "-lc", command)
	}
	return exec.CommandContext(ctx, "/bin/sh", "-c", command)
}

func prepareTaskWorkDir(baseDir, targetID string) (string, error) {
	if strings.TrimSpace(baseDir) == "" {
		baseDir = "."
	}
	taskDir := filepath.Join(baseDir, safePathSegment(targetID))
	if err := os.MkdirAll(taskDir, 0o700); err != nil {
		return "", err
	}
	return taskDir, nil
}

func safePathSegment(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "task"
	}
	var builder strings.Builder
	for _, r := range value {
		switch {
		case r >= 'a' && r <= 'z':
			builder.WriteRune(r)
		case r >= 'A' && r <= 'Z':
			builder.WriteRune(r)
		case r >= '0' && r <= '9':
			builder.WriteRune(r)
		case r == '-', r == '_', r == '.':
			builder.WriteRune(r)
		default:
			builder.WriteByte('_')
		}
	}
	if builder.Len() == 0 {
		return "task"
	}
	segment := builder.String()
	if segment == "." || segment == ".." {
		return "task"
	}
	return segment
}

type lineStreamWriter struct {
	stream string
	onLog  func(stream, chunk string)
	mu     sync.Mutex
	buf    bytes.Buffer
}

func newLineStreamWriter(stream string, onLog func(stream, chunk string)) *lineStreamWriter {
	return &lineStreamWriter{stream: stream, onLog: onLog}
}

func (w *lineStreamWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	written := len(p)
	for len(p) > 0 {
		index := bytes.IndexByte(p, '\n')
		if index < 0 {
			_, _ = w.buf.Write(p)
			break
		}
		_, _ = w.buf.Write(p[:index+1])
		w.flushLocked()
		p = p[index+1:]
	}
	return written, nil
}

func (w *lineStreamWriter) Flush() {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.flushLocked()
}

func (w *lineStreamWriter) flushLocked() {
	if w.onLog == nil || w.buf.Len() == 0 {
		w.buf.Reset()
		return
	}
	w.onLog(w.stream, w.buf.String())
	w.buf.Reset()
}

func failedResult(startedAt time.Time, err error) commandResult {
	finishedAt := time.Now()
	exitCode := -1
	return commandResult{
		Status:       "failed",
		ExitCode:     &exitCode,
		ErrorMessage: security.Redact(err.Error()),
		StartedAt:    startedAt,
		FinishedAt:   finishedAt,
	}
}

func getJSON[T any](ctx context.Context, client apiClient, path, token string) (T, error) {
	var zero T
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, client.baseURL+path, nil)
	if err != nil {
		return zero, err
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	return doJSON[T](client, req)
}

func postJSON[T any](ctx context.Context, client apiClient, path, token string, payload interface{}) (T, error) {
	var zero T
	body, err := json.Marshal(payload)
	if err != nil {
		return zero, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, client.baseURL+path, bytes.NewReader(body))
	if err != nil {
		return zero, err
	}
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	return doJSON[T](client, req)
}

func doJSON[T any](client apiClient, req *http.Request) (T, error) {
	var zero T
	resp, err := client.http.Do(req)
	if err != nil {
		return zero, err
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return zero, err
	}
	var wrapped envelope[T]
	if err := json.Unmarshal(respBody, &wrapped); err != nil {
		return zero, fmt.Errorf("decode api response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 || wrapped.Code != 0 {
		message := wrapped.Message
		if message == "" {
			message = resp.Status
		}
		return zero, apiError{status: resp.StatusCode, message: message}
	}
	return wrapped.Data, nil
}

func collectHostInfo(cfg config.Config) agentPayload {
	hostname := strings.TrimSpace(cfg.Agent.Hostname)
	if hostname == "" {
		hostname, _ = os.Hostname()
	}
	return agentPayload{
		Name:     hostname,
		Hostname: hostname,
		IP:       localIP(),
		OS:       runtime.GOOS,
		OSType:   runtime.GOOS,
		OSName:   runtime.GOOS,
		Arch:     runtime.GOARCH,
		Version:  cfg.App.Version,
		Metadata: map[string]interface{}{
			"workspace": cfg.Agent.Workspace,
			"workDir":   cfg.Agent.WorkDir,
		},
	}
}

func collectMetrics(e *executor) []metricPayload {
	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)
	now := time.Now().Format(time.RFC3339)
	dimensions := map[string]interface{}{
		"os":      runtime.GOOS,
		"arch":    runtime.GOARCH,
		"workDir": e.cfg.Agent.WorkDir,
	}
	metrics := []metricPayload{
		{Code: "agent.running_tasks", Value: float64(e.active.Load()), Unit: "count", Dimensions: dimensions, CollectedAt: now},
		{Code: "agent.cpu.logical", Value: float64(runtime.NumCPU()), Unit: "count", Dimensions: dimensions, CollectedAt: now},
		{Code: "agent.runtime.goroutines", Value: float64(runtime.NumGoroutine()), Unit: "count", Dimensions: dimensions, CollectedAt: now},
		{Code: "agent.runtime.alloc_bytes", Value: float64(mem.Alloc), Unit: "bytes", Dimensions: dimensions, CollectedAt: now},
		{Code: "agent.runtime.sys_bytes", Value: float64(mem.Sys), Unit: "bytes", Dimensions: dimensions, CollectedAt: now},
	}
	return append(metrics, collectSystemMetrics(e.cfg.Agent.WorkDir, now, dimensions)...)
}

func localIP() string {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return ""
	}
	defer conn.Close()
	addr, ok := conn.LocalAddr().(*net.UDPAddr)
	if !ok {
		return ""
	}
	return addr.IP.String()
}

func loadAgentToken(cfg config.Config) (string, error) {
	if token := strings.TrimSpace(cfg.Agent.Token); token != "" {
		return token, nil
	}
	return readToken(cfg.Agent.TokenFile)
}

func readToken(path string) (string, error) {
	bytes, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(bytes)), nil
}

func writeToken(path, token string) error {
	dir := filepath.Dir(path)
	if dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return err
		}
	}
	return os.WriteFile(path, []byte(token+"\n"), 0o600)
}
