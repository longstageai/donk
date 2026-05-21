package builtin

import (
	"context"
	"path/filepath"
	"strings"
	"time"

	"github.com/longstageai/donk/donk/internal/tool"
)

type PythonDependencyManager struct {
	baseDir        string
	defaultTimeout time.Duration
	maxTimeout     time.Duration
	maxOutputBytes int
}

type PythonDependencyManagerOption func(*PythonDependencyManager)

func WithPythonDependencyManagerBaseDir(baseDir string) PythonDependencyManagerOption {
	return func(m *PythonDependencyManager) {
		m.baseDir = baseDir
	}
}

func WithPythonDependencyManagerDefaultTimeout(timeout time.Duration) PythonDependencyManagerOption {
	return func(m *PythonDependencyManager) {
		m.defaultTimeout = timeout
	}
}

func NewPythonDependencyManager(opts ...PythonDependencyManagerOption) *PythonDependencyManager {
	m := &PythonDependencyManager{
		baseDir:        filepath.Join("data", "script_runtime"),
		defaultTimeout: 120 * time.Second,
		maxTimeout:     300 * time.Second,
		maxOutputBytes: 64 * 1024,
	}
	for _, opt := range opts {
		opt(m)
	}
	return m
}

func (m *PythonDependencyManager) Name() string {
	return "python_dependency_manager"
}

func (m *PythonDependencyManager) Description() string {
	return "管理 Donk Python 运行时中的第三方依赖。支持 install、uninstall、list、freeze、show、check 动作，依赖会安装到 Donk Python 运行时的共享环境中。"
}

func (m *PythonDependencyManager) Version() string {
	return "1.0.0"
}

func (m *PythonDependencyManager) Category() string {
	return string(tool.CategoryCompute)
}

func (m *PythonDependencyManager) Parameters() *tool.Schema {
	schema := tool.NewSchema()
	schema.Properties = map[string]*tool.Property{
		"action": {
			Type:        "string",
			Description: "依赖管理动作：install 安装；uninstall 删除；list 列出；freeze 输出锁定格式；show 查看包信息；check 检查依赖冲突",
			Enum:        []interface{}{"install", "uninstall", "list", "freeze", "show", "check"},
		},
		"packages": {
			Type:        "array",
			Description: "包列表。install 示例 requests==2.32.3；uninstall/show 示例 requests。list/freeze/check 可为空",
		},
		"runtime_version": {
			Type:        "string",
			Description: "Donk Python 运行时版本，可选。为空时选择已安装的最高版本",
		},
		"timeout": {
			Type:        "integer",
			Description: "依赖管理命令超时时间，单位秒，默认120秒，最大300秒",
			Default:     120,
		},
	}
	schema.Required = []string{"action"}
	return schema
}

func (m *PythonDependencyManager) Execute(ctx *tool.Context) (*tool.Result, error) {
	start := time.Now()
	req, result := m.parseRequest(ctx)
	if result != nil {
		return result, nil
	}

	runner := NewScriptRunner(WithScriptRunnerBaseDir(m.baseDir))
	runtimeInfo, err := runner.resolveRuntime("python", req.RuntimeVersion)
	if err != nil {
		return m.errorResult("runtime_not_installed", err.Error(), map[string]interface{}{
			"version": req.RuntimeVersion,
		}), nil
	}

	args, result := m.buildPipArgs(req)
	if result != nil {
		return result, nil
	}

	runCtx, cancel := context.WithTimeout(toolContext(ctx), req.Timeout)
	defer cancel()

	processResult := runner.runProcess(runCtx, runtimeInfo.Root, runtimeInfo.Path, args, "", map[string]string{
		"DONK_RUNTIME_ROOT":    runtimeInfo.Root,
		"DONK_RUNTIME_VERSION": runtimeInfo.Version,
		"PYTHONIOENCODING":     "utf-8",
		"PYTHONUTF8":           "1",
	}, m.maxOutputBytes)

	status := "succeeded"
	errorType := ""
	if processResult.TimedOut {
		status = "timeout"
		errorType = "timeout"
	} else if !processResult.Success {
		status = "failed"
		errorType = "non_zero_exit"
	}

	data := map[string]interface{}{
		"status":           status,
		"success":          processResult.Success,
		"error_type":       errorType,
		"action":           req.Action,
		"packages":         req.Packages,
		"runtime_version":  runtimeInfo.Version,
		"runtime_path":     runtimeInfo.Path,
		"command":          runtimeInfo.Path,
		"args":             args,
		"stdout":           processResult.Stdout,
		"stderr":           processResult.Stderr,
		"stdout_truncated": processResult.StdoutTruncated,
		"stderr_truncated": processResult.StderrTruncated,
		"exit_code":        processResult.ExitCode,
		"timed_out":        processResult.TimedOut,
		"duration_ms":      time.Since(start).Milliseconds(),
	}

	return tool.NewResult(data), nil
}

type pythonDependencyRequest struct {
	Action         string
	Packages       []string
	RuntimeVersion string
	Timeout        time.Duration
}

func (m *PythonDependencyManager) parseRequest(ctx *tool.Context) (*pythonDependencyRequest, *tool.Result) {
	action := strings.ToLower(strings.TrimSpace(stringParam(ctx.Params["action"])))
	if action == "" {
		return nil, m.errorResult("invalid_params", "action 不能为空", nil)
	}
	if !pythonDependencyActionAllowed(action) {
		return nil, m.errorResult("invalid_params", "action 仅支持 install、uninstall、list、freeze、show、check", map[string]interface{}{"action": action})
	}

	packages := normalizeDependencies(stringSliceParam(ctx.Params["packages"]))
	if (action == "install" || action == "uninstall" || action == "show") && len(packages) == 0 {
		return nil, m.errorResult("invalid_params", "当前 action 需要 packages 参数", map[string]interface{}{"action": action})
	}
	if action == "uninstall" {
		if blocked := protectedPythonPackages(packages); len(blocked) > 0 {
			return nil, m.errorResult("protected_package", "禁止删除 Donk Python 运行时基础依赖", map[string]interface{}{"packages": blocked})
		}
	}

	timeout := m.defaultTimeout
	if v, ok := numberParam(ctx.Params["timeout"]); ok && v > 0 {
		timeout = time.Duration(v) * time.Second
	}
	if timeout > m.maxTimeout {
		timeout = m.maxTimeout
	}

	return &pythonDependencyRequest{
		Action:         action,
		Packages:       packages,
		RuntimeVersion: stringParam(ctx.Params["runtime_version"]),
		Timeout:        timeout,
	}, nil
}

func (m *PythonDependencyManager) buildPipArgs(req *pythonDependencyRequest) ([]string, *tool.Result) {
	switch req.Action {
	case "install":
		args := []string{"-m", "pip", "install", "--disable-pip-version-check", "--no-input"}
		args = append(args, req.Packages...)
		return args, nil
	case "uninstall":
		args := []string{"-m", "pip", "uninstall", "-y"}
		args = append(args, req.Packages...)
		return args, nil
	case "list":
		return []string{"-m", "pip", "list", "--format=json", "--disable-pip-version-check"}, nil
	case "freeze":
		return []string{"-m", "pip", "freeze", "--disable-pip-version-check"}, nil
	case "show":
		args := []string{"-m", "pip", "show"}
		args = append(args, req.Packages...)
		return args, nil
	case "check":
		return []string{"-m", "pip", "check", "--disable-pip-version-check"}, nil
	default:
		return nil, m.errorResult("invalid_params", "不支持的 action", map[string]interface{}{"action": req.Action})
	}
}

func pythonDependencyActionAllowed(action string) bool {
	switch action {
	case "install", "uninstall", "list", "freeze", "show", "check":
		return true
	default:
		return false
	}
}

func protectedPythonPackages(packages []string) []string {
	protected := map[string]bool{
		"pip":        true,
		"setuptools": true,
		"wheel":      true,
		"virtualenv": true,
	}
	var blocked []string
	for _, pkg := range packages {
		name := strings.ToLower(strings.TrimSpace(pkg))
		name = strings.TrimLeft(name, "-_")
		for _, sep := range []string{"==", ">=", "<=", "~=", "!=", ">", "<", "["} {
			if idx := strings.Index(name, sep); idx >= 0 {
				name = name[:idx]
			}
		}
		name = strings.TrimSpace(name)
		if protected[name] {
			blocked = append(blocked, pkg)
		}
	}
	return blocked
}

func (m *PythonDependencyManager) errorResult(errorType, message string, details map[string]interface{}) *tool.Result {
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
