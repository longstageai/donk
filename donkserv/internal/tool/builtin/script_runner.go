package builtin

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/longstageai/donk/donk/internal/tool"
)

type ScriptRunner struct {
	baseDir        string
	defaultTimeout time.Duration
	maxTimeout     time.Duration
	maxOutputBytes int
}

type ScriptRunnerOption func(*ScriptRunner)

func WithScriptRunnerBaseDir(baseDir string) ScriptRunnerOption {
	return func(r *ScriptRunner) {
		r.baseDir = baseDir
	}
}

func WithScriptRunnerDefaultTimeout(timeout time.Duration) ScriptRunnerOption {
	return func(r *ScriptRunner) {
		r.defaultTimeout = timeout
	}
}

func NewScriptRunner(opts ...ScriptRunnerOption) *ScriptRunner {
	r := &ScriptRunner{
		baseDir:        filepath.Join("data", "script_runtime"),
		defaultTimeout: 30 * time.Second,
		maxTimeout:     300 * time.Second,
		maxOutputBytes: 64 * 1024,
	}
	for _, opt := range opts {
		opt(r)
	}
	return r
}

func (r *ScriptRunner) Name() string {
	return "script_runner"
}

func (r *ScriptRunner) Description() string {
	return "使用已提前准备好的 Donk Python 运行时执行 Python 脚本。脚本在独立运行目录执行，第三方依赖由 python_dependency_manager 单独管理，不使用系统全局 Python，不额外下载运行时。"
}

func (r *ScriptRunner) Version() string {
	return "1.0.0"
}

func (r *ScriptRunner) Category() string {
	return string(tool.CategoryCompute)
}

func (r *ScriptRunner) Parameters() *tool.Schema {
	schema := tool.NewSchema()
	schema.Properties = map[string]*tool.Property{
		"language": {
			Type:        "string",
			Description: "脚本语言，仅支持 python；为空时默认 python",
			Default:     "python",
			Enum:        []interface{}{"python"},
		},
		"code": {
			Type:        "string",
			Description: "要执行的 Python 脚本源码",
		},
		"runtime_version": {
			Type:        "string",
			Description: "Donk Python 运行时版本，可选。为空时选择已安装的最高版本",
		},
		"timeout": {
			Type:        "integer",
			Description: "脚本执行超时时间，单位秒，默认30秒，最大300秒",
			Default:     30,
		},
		"stdin": {
			Type:        "string",
			Description: "传给脚本的标准输入，可选",
		},
		"env": {
			Type:        "object",
			Description: "显式传入脚本进程的环境变量，可选，敏感变量会被过滤",
		},
		"keep_run_dir": {
			Type:        "boolean",
			Description: "是否保留本次运行目录，默认 false",
			Default:     false,
		},
	}
	schema.Required = []string{"code"}
	return schema
}

func (r *ScriptRunner) Execute(ctx *tool.Context) (*tool.Result, error) {
	start := time.Now()
	req, result := r.parseRequest(ctx)
	if result != nil {
		return result, nil
	}

	if err := os.MkdirAll(r.baseDir, 0755); err != nil {
		return r.errorResult("prepare_failed", fmt.Sprintf("创建脚本运行目录失败: %v", err), nil), nil
	}

	runtimeInfo, err := r.resolveRuntime(req.Language, req.RuntimeVersion)
	if err != nil {
		return r.errorResult("runtime_not_installed", err.Error(), map[string]interface{}{
			"language": req.Language,
			"version":  req.RuntimeVersion,
		}), nil
	}

	runInfo, err := r.createRun(req)
	if err != nil {
		return r.errorResult("prepare_failed", fmt.Sprintf("创建脚本运行实例失败: %v", err), nil), nil
	}
	if !req.KeepRunDir {
		defer os.RemoveAll(runInfo.Dir)
	}

	processResult := r.runScript(ctx, req, runtimeInfo, runInfo)
	processResult["duration_ms"] = time.Since(start).Milliseconds()

	if err := r.writeRunResult(runInfo.Dir, processResult); err != nil {
		processResult["result_write_error"] = err.Error()
	}

	return tool.NewResult(processResult), nil
}

type scriptRunRequest struct {
	Language       string
	Code           string
	RuntimeVersion string
	Timeout        time.Duration
	Stdin          string
	Env            map[string]string
	KeepRunDir     bool
}

type donkRuntimeInfo struct {
	Language   string
	Version    string
	Path       string
	Root       string
	Executable string
	Manifest   map[string]interface{}
}

type scriptRunInfo struct {
	ID         string
	Dir        string
	ScriptPath string
}

func (r *ScriptRunner) parseRequest(ctx *tool.Context) (*scriptRunRequest, *tool.Result) {
	language := "python"
	if value := stringParam(ctx.Params["language"]); value != "" {
		language = normalizeScriptLanguage(value)
	}
	if language != "python" {
		return nil, r.errorResult("unsupported_language", "仅支持 python，JavaScript/Node.js 已不再由 script_runner 支持", map[string]interface{}{"language": language})
	}

	code, ok := ctx.Params["code"].(string)
	if !ok || code == "" {
		return nil, r.errorResult("invalid_params", "code 不能为空", nil)
	}

	timeout := r.defaultTimeout
	if v, ok := numberParam(ctx.Params["timeout"]); ok && v > 0 {
		timeout = time.Duration(v) * time.Second
	}
	if timeout > r.maxTimeout {
		timeout = r.maxTimeout
	}

	req := &scriptRunRequest{
		Language:       language,
		Code:           code,
		RuntimeVersion: stringParam(ctx.Params["runtime_version"]),
		Timeout:        timeout,
		Stdin:          stringParam(ctx.Params["stdin"]),
		Env:            stringMapParam(ctx.Params["env"]),
		KeepRunDir:     boolParam(ctx.Params["keep_run_dir"]),
	}
	return req, nil
}

func (r *ScriptRunner) resolveRuntime(language, version string) (*donkRuntimeInfo, error) {
	var candidates []*donkRuntimeInfo
	for _, runtimeRoot := range r.runtimeSearchRoots(language) {
		entries, err := os.ReadDir(runtimeRoot)
		if err != nil {
			continue
		}

		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			info, err := r.readRuntimeManifest(language, filepath.Join(runtimeRoot, entry.Name()))
			if err != nil {
				continue
			}
			if version != "" && !strings.HasPrefix(info.Version, version) && entry.Name() != version {
				continue
			}
			if _, err := os.Stat(info.Path); err != nil {
				continue
			}
			candidates = append(candidates, info)
		}
	}

	if len(candidates) == 0 {
		if version != "" {
			return nil, fmt.Errorf("Donk %s 运行时版本 %s 未准备好", language, version)
		}
		return nil, fmt.Errorf("Donk %s 运行时未准备好", language)
	}

	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].Version > candidates[j].Version
	})
	return candidates[0], nil
}

func (r *ScriptRunner) runtimeSearchRoots(language string) []string {
	return []string{filepath.Join(r.baseDir, "runtimes", language)}
}

func (r *ScriptRunner) readRuntimeManifest(language, root string) (*donkRuntimeInfo, error) {
	manifestPath := filepath.Join(root, "runtime.json")
	manifest := map[string]interface{}{}
	if content, err := os.ReadFile(manifestPath); err == nil {
		if err := json.Unmarshal(content, &manifest); err != nil {
			return nil, err
		}

		manifestLanguage := stringParam(manifest["language"])
		if manifestLanguage != "" && normalizeScriptLanguage(manifestLanguage) != language {
			return nil, fmt.Errorf("runtime language mismatch")
		}
	}

	version := stringParam(manifest["version"])
	if version == "" {
		version = filepath.Base(root)
	}

	executable := stringParam(manifest["executable"])
	if executable == "" {
		executable = defaultPythonExecutable()
	}

	execPath := executable
	if !filepath.IsAbs(execPath) {
		execPath = filepath.Join(root, executable)
	}
	if absPath, err := filepath.Abs(execPath); err == nil {
		execPath = absPath
	}
	if absRoot, err := filepath.Abs(root); err == nil {
		root = absRoot
	}

	return &donkRuntimeInfo{
		Language:   language,
		Version:    version,
		Path:       execPath,
		Root:       root,
		Executable: executable,
		Manifest:   manifest,
	}, nil
}

func (r *ScriptRunner) createRun(req *scriptRunRequest) (*scriptRunInfo, error) {
	runID := time.Now().Format("20060102-150405") + "-" + strings.ReplaceAll(uuid.NewString()[:8], "-", "")
	runDir := filepath.Join(r.baseDir, "runs", runID)
	if err := os.MkdirAll(runDir, 0755); err != nil {
		return nil, err
	}

	scriptPath := filepath.Join(runDir, "script.py")
	if absRunDir, err := filepath.Abs(runDir); err == nil {
		runDir = absRunDir
		scriptPath = filepath.Join(runDir, "script.py")
	}
	if err := os.WriteFile(scriptPath, []byte(req.Code), 0644); err != nil {
		return nil, err
	}

	return &scriptRunInfo{ID: runID, Dir: runDir, ScriptPath: scriptPath}, nil
}

func (r *ScriptRunner) runScript(ctx *tool.Context, req *scriptRunRequest, runtimeInfo *donkRuntimeInfo, runInfo *scriptRunInfo) map[string]interface{} {
	command := runtimeInfo.Path
	args := []string{runInfo.ScriptPath}
	env := r.scriptEnv(req, runtimeInfo)
	runCtx, cancel := context.WithTimeout(toolContext(ctx), req.Timeout)
	defer cancel()

	processResult := r.runProcess(runCtx, runInfo.Dir, command, args, req.Stdin, env, r.maxOutputBytes)
	status := "succeeded"
	errorType := ""
	if processResult.TimedOut {
		status = "timeout"
		errorType = "timeout"
	} else if !processResult.Success {
		status = "failed"
		errorType = "non_zero_exit"
	}

	return map[string]interface{}{
		"status":           status,
		"success":          processResult.Success,
		"error_type":       errorType,
		"language":         req.Language,
		"runtime_version":  runtimeInfo.Version,
		"runtime_path":     runtimeInfo.Path,
		"run_id":           runInfo.ID,
		"run_dir":          runInfo.Dir,
		"script_path":      runInfo.ScriptPath,
		"command":          command,
		"args":             args,
		"stdout":           processResult.Stdout,
		"stderr":           processResult.Stderr,
		"stdout_truncated": processResult.StdoutTruncated,
		"stderr_truncated": processResult.StderrTruncated,
		"exit_code":        processResult.ExitCode,
		"timed_out":        processResult.TimedOut,
	}
}

type processRunResult struct {
	Stdout          string
	Stderr          string
	StdoutTruncated bool
	StderrTruncated bool
	ExitCode        int
	Success         bool
	TimedOut        bool
}

func (r *ScriptRunner) runProcess(ctx context.Context, dir, command string, args []string, stdin string, extraEnv map[string]string, maxOutputBytes int) processRunResult {
	cmd := exec.CommandContext(ctx, command, args...)
	cmd.Dir = dir
	cmd.Env = buildScriptEnv(extraEnv)
	if runtime.GOOS == "windows" {
		cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	}
	if stdin != "" {
		cmd.Stdin = strings.NewReader(stdin)
	}

	var stdout, stderr limitedBuffer
	stdout.limit = maxOutputBytes
	stderr.limit = maxOutputBytes
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	result := processRunResult{
		Stdout:          stdout.String(),
		Stderr:          stderr.String(),
		StdoutTruncated: stdout.truncated,
		StderrTruncated: stderr.truncated,
		ExitCode:        0,
		Success:         true,
		TimedOut:        false,
	}
	if ctx.Err() == context.DeadlineExceeded {
		result.Success = false
		result.TimedOut = true
		result.ExitCode = -1
		return result
	}
	if err != nil {
		result.Success = false
		result.ExitCode = 1
		if exitErr, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitErr.ExitCode()
		} else if result.Stderr == "" {
			result.Stderr = err.Error()
		}
	}
	return result
}

type limitedBuffer struct {
	buf       bytes.Buffer
	limit     int
	truncated bool
}

func (b *limitedBuffer) Write(p []byte) (int, error) {
	if b.limit <= 0 {
		return len(p), nil
	}
	remaining := b.limit - b.buf.Len()
	if remaining <= 0 {
		b.truncated = true
		return len(p), nil
	}
	if len(p) > remaining {
		b.truncated = true
		_, _ = b.buf.Write(p[:remaining])
		return len(p), nil
	}
	_, _ = b.buf.Write(p)
	return len(p), nil
}

func (b *limitedBuffer) String() string {
	return b.buf.String()
}

func (r *ScriptRunner) scriptEnv(req *scriptRunRequest, runtimeInfo *donkRuntimeInfo) map[string]string {
	env := map[string]string{}
	for k, v := range req.Env {
		env[k] = v
	}
	env["DONK_RUNTIME_ROOT"] = runtimeInfo.Root
	env["DONK_RUNTIME_VERSION"] = runtimeInfo.Version
	env["PYTHONIOENCODING"] = "utf-8"
	env["PYTHONUTF8"] = "1"
	return env
}

func buildScriptEnv(extra map[string]string) []string {
	allowed := map[string]bool{
		"PATH": true, "Path": true, "TEMP": true, "TMP": true, "HOME": true, "USERPROFILE": true, "SYSTEMROOT": true, "SystemRoot": true, "WINDIR": true, "windir": true,
	}
	envMap := make(map[string]string)
	for _, item := range os.Environ() {
		parts := strings.SplitN(item, "=", 2)
		if len(parts) != 2 {
			continue
		}
		if allowed[parts[0]] {
			envMap[parts[0]] = parts[1]
		}
	}
	for k, v := range extra {
		if isSensitiveEnv(k) {
			continue
		}
		envMap[k] = v
	}
	result := make([]string, 0, len(envMap))
	for k, v := range envMap {
		result = append(result, fmt.Sprintf("%s=%s", k, v))
	}
	sort.Strings(result)
	return result
}

func isSensitiveEnv(key string) bool {
	upper := strings.ToUpper(key)
	patterns := []string{"SECRET", "TOKEN", "PASSWORD", "API_KEY", "ACCESS_KEY", "PRIVATE_KEY"}
	for _, pattern := range patterns {
		if strings.Contains(upper, pattern) {
			return true
		}
	}
	return false
}

func (r *ScriptRunner) writeRunResult(runDir string, data map[string]interface{}) error {
	content, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(runDir, "result.json"), content, 0644)
}

func (r *ScriptRunner) errorResult(errorType, message string, details map[string]interface{}) *tool.Result {
	data := map[string]interface{}{
		"status":     "failed",
		"success":    false,
		"error_type": errorType,
		"message":    message,
	}
	if details != nil {
		data["details"] = details
	}
	return tool.NewErrorResultWithMsg(errorType, message, data)
}

func normalizeScriptLanguage(language string) string {
	language = strings.ToLower(strings.TrimSpace(language))
	switch language {
	case "js", "node", "nodejs", "javascript":
		return "javascript"
	case "py", "python3", "python":
		return "python"
	default:
		return language
	}
}

func normalizeDependencies(dependencies []string) []string {
	seen := map[string]bool{}
	result := make([]string, 0, len(dependencies))
	for _, dep := range dependencies {
		dep = strings.TrimSpace(dep)
		if dep == "" || seen[dep] {
			continue
		}
		seen[dep] = true
		result = append(result, dep)
	}
	sort.Strings(result)
	return result
}

func defaultPythonExecutable() string {
	if runtime.GOOS == "windows" {
		return "python.exe"
	}
	return "bin/python"
}

func toolContext(ctx *tool.Context) context.Context {
	if ctx != nil && ctx.Values != nil {
		return ctx.Values
	}
	return context.Background()
}

func stringParam(value interface{}) string {
	if value == nil {
		return ""
	}
	if s, ok := value.(string); ok {
		return strings.TrimSpace(s)
	}
	return ""
}

func stringSliceParam(value interface{}) []string {
	switch v := value.(type) {
	case []string:
		return v
	case []interface{}:
		result := make([]string, 0, len(v))
		for _, item := range v {
			if s, ok := item.(string); ok {
				result = append(result, s)
			}
		}
		return result
	default:
		return nil
	}
}

func stringMapParam(value interface{}) map[string]string {
	result := map[string]string{}
	switch v := value.(type) {
	case map[string]string:
		for key, val := range v {
			result[key] = val
		}
	case map[string]interface{}:
		for key, val := range v {
			if s, ok := val.(string); ok {
				result[key] = s
			}
		}
	}
	return result
}

func boolParam(value interface{}) bool {
	if b, ok := value.(bool); ok {
		return b
	}
	return false
}

func numberParam(value interface{}) (int, bool) {
	switch v := value.(type) {
	case int:
		return v, true
	case int64:
		return int(v), true
	case float64:
		return int(v), true
	case json.Number:
		n, err := v.Int64()
		return int(n), err == nil
	default:
		return 0, false
	}
}
