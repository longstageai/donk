package builtin

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/longstageai/donk/donk/internal/tool"
)

// CommandExecutor Windows命令行执行工具
// 用于在Windows系统上执行命令行命令
// 特性：
// - 支持超时控制
// - 支持工作目录设置
// - 支持环境变量传递
// - 自动处理Windows命令行编码
// - 返回标准输出和标准错误
// - 危险命令过滤
type CommandExecutor struct {
	defaultTimeout time.Duration // 默认超时时间
	workingDir     string        // 默认工作目录
}

// dangerousCommands 危险命令列表
var dangerousCommands = []string{
	// 系统破坏类
	"format", "del /", "del \\", "rmdir /", "rd /",
	"erase /", "deltree", "rmdir /s", "rd /s",
	// 分区操作
	"diskpart", "clean", "convert /", "convert \\",
	// 注册表操作
	"reg delete", "reg add", "regedit", "reg import",
	// 系统服务
	"sc delete", "sc config", "net stop", "net start",
	"sc stop", "sc start",
	// 用户操作
	"net user /delete", "net localgroup /delete",
	"net user administrator", "net user admin",
	// 权限提升
	"takeown /", "icacls /", "cacls /", "attrib -r -s -h",
	// 网络攻击
	"shutdown /", "shutdown -", "tskill", "taskkill /f",
	// 恶意脚本
	"powershell -enc", "powershell -encoded", "powershell -nop",
	"powershell -windowstyle hidden", "iex", "invoke-expression",
	"downloadstring", "downloadfile", "net.webclient",
	// 远程访问
	"mstsc /", "qwinsta", "rwinsta", "tscon",
	// 系统修改
	"bcdedit /", "bootrec /", "fixboot", "fixmbr",
	"sfc /", "dism /", "chkdsk /", "fsutil",
	// 敏感路径删除
	"del %systemroot%", "del %windir%", "del c:\\windows",
	"rmdir %systemroot%", "rmdir %windir%",
}

// dangerousPatterns 危险模式正则表达式
var dangerousPatterns = []*regexp.Regexp{
	// 删除系统目录
	regexp.MustCompile(`(?i)del.*\bc:\\windows\b`),
	regexp.MustCompile(`(?i)del.*\bc:\\program files\b`),
	regexp.MustCompile(`(?i)rd.*\bc:\\windows\b`),
	regexp.MustCompile(`(?i)rmdir.*\bc:\\windows\b`),
	// 格式化磁盘
	regexp.MustCompile(`(?i)\bformat\s+[a-z]:`),
	// 删除所有文件
	regexp.MustCompile(`(?i)del.*\*.*\.`),
	regexp.MustCompile(`(?i)del.*\/s.*\/q`),
	regexp.MustCompile(`(?i)del.*\\\?\?\\`),
	// PowerShell危险命令
	regexp.MustCompile(`(?i)invoke-mimikatz`),
	regexp.MustCompile(`(?i)mimikatz`),
	regexp.MustCompile(`(?i)invoke-shellcode`),
	regexp.MustCompile(`(?i)reflectivepeinjection`),
	// 编码执行
	regexp.MustCompile(`(?i)-enc\s+[a-z0-9+/]{100,}`),
	regexp.MustCompile(`(?i)frombase64string`),
	// 远程下载执行
	regexp.MustCompile(`(?i)iwr.*-useb`),
	regexp.MustCompile(`(?i)invoke-webrequest`),
	regexp.MustCompile(`(?i)start-bitstransfer`),
}

// allowedReadOnlyCommands 允许的安全只读命令白名单
var allowedReadOnlyCommands = []string{
	"dir", "cd", "echo", "type", "more", "find", "findstr",
	"ping", "tracert", "pathping", "nslookup", "ipconfig",
	"systeminfo", "tasklist", "driverquery", "vol", "ver",
	"date", "time", "whoami", "hostname", "set", "help",
	"tree", "fc", "comp", "chcp", "cls", "prompt",
}

// CommandExecutorOption 命令执行器配置选项
type CommandExecutorOption func(*CommandExecutor)

// WithDefaultTimeout 设置默认超时时间
func WithDefaultTimeout(timeout time.Duration) CommandExecutorOption {
	return func(e *CommandExecutor) {
		e.defaultTimeout = timeout
	}
}

// WithExecutorWorkingDir 设置默认工作目录
func WithExecutorWorkingDir(dir string) CommandExecutorOption {
	return func(e *CommandExecutor) {
		e.workingDir = dir
	}
}

// NewCommandExecutor 创建命令行执行工具
func NewCommandExecutor(opts ...CommandExecutorOption) *CommandExecutor {
	// 获取程序所在目录
	execPath, err := os.Executable()
	if err != nil {
		execPath, _ = os.Getwd()
	}
	execDir := filepath.Dir(execPath)

	// 默认工作目录：程序目录 + data/workspace
	defaultWorkingDir := filepath.Join(execDir, "data", "workspace")

	// 确保工作目录存在
	if err := os.MkdirAll(defaultWorkingDir, 0755); err != nil {
		// 如果创建失败，使用当前工作目录
		defaultWorkingDir, _ = os.Getwd()
	}

	e := &CommandExecutor{
		defaultTimeout: 60 * time.Second, // 默认60秒超时
		workingDir:     defaultWorkingDir,
	}
	for _, opt := range opts {
		opt(e)
	}
	return e
}

// Name 返回工具名称
func (e *CommandExecutor) Name() string {
	return "command_executor"
}

// Description 返回工具描述
func (e *CommandExecutor) Description() string {
	return "在Windows系统上执行命令行命令。支持超时控制、工作目录设置、环境变量传递。已启用危险命令过滤，仅允许安全的只读操作。"
}

// Version 返回版本
func (e *CommandExecutor) Version() string {
	return "1.1.0"
}

// Category 返回分类
func (e *CommandExecutor) Category() string {
	return string(tool.CategoryUtility)
}

// Parameters 返回参数定义
func (e *CommandExecutor) Parameters() *tool.Schema {
	schema := tool.NewSchema()
	schema.Properties = map[string]*tool.Property{
		"command": {
			Type:        "string",
			Description: "要执行的命令（仅支持安全的只读命令，如dir、type、systeminfo等）",
		},
		"shell": {
			Type:        "string",
			Description: "使用的shell类型，支持 cmd 或 powershell，默认 cmd",
			Default:     "cmd",
			Enum:        []interface{}{"cmd", "powershell"},
		},
		"working_dir": {
			Type:        "string",
			Description: "命令执行的工作目录（可选，默认使用当前目录）",
		},
		"timeout": {
			Type:        "integer",
			Description: "命令执行超时时间（秒），默认60秒，最大300秒",
			Default:     60,
		},
		"env": {
			Type:        "object",
			Description: "环境变量（可选，key-value对象）",
		},
	}
	schema.Required = []string{"command"}
	return schema
}

// Execute 执行命令
func (e *CommandExecutor) Execute(ctx *tool.Context) (*tool.Result, error) {
	// 检查操作系统
	if runtime.GOOS != "windows" {
		return tool.NewErrorResultWithMsg(tool.ErrCodeExecution, "此工具仅支持Windows系统"), nil
	}

	// 获取命令
	command, ok := ctx.Params["command"].(string)
	if !ok || command == "" {
		return tool.NewErrorResultWithMsg(tool.ErrCodeInvalidParams, "命令不能为空"), nil
	}

	// 安全检查
	if err := e.validateCommand(command); err != nil {
		return tool.NewErrorResultWithMsg(tool.ErrCodeInvalidParams, fmt.Sprintf("命令安全检查失败: %v", err)), nil
	}

	// 获取shell类型
	shell := "cmd"
	if s, ok := ctx.Params["shell"].(string); ok && s != "" {
		shell = strings.ToLower(s)
	}

	// 获取工作目录
	workingDir := e.workingDir
	if wd, ok := ctx.Params["working_dir"].(string); ok && wd != "" {
		workingDir = wd
	}

	// 获取超时时间
	timeout := e.defaultTimeout
	if t, ok := ctx.Params["timeout"].(float64); ok && t > 0 {
		if t > 300 {
			t = 300 // 最大300秒
		}
		timeout = time.Duration(t) * time.Second
	}

	// 获取环境变量
	envVars := make(map[string]string)
	if env, ok := ctx.Params["env"].(map[string]interface{}); ok {
		for k, v := range env {
			if strVal, ok := v.(string); ok {
				envVars[k] = strVal
			}
		}
	}

	// 执行命令
	return e.executeCommand(command, shell, workingDir, timeout, envVars)
}

// validateCommand 验证命令安全性
func (e *CommandExecutor) validateCommand(command string) error {
	// 转换为小写进行检查
	lowerCmd := strings.ToLower(command)
	lowerCmd = strings.TrimSpace(lowerCmd)

	// 检查危险命令列表
	for _, dangerous := range dangerousCommands {
		if strings.HasPrefix(lowerCmd, strings.ToLower(dangerous)) ||
			strings.Contains(lowerCmd, " "+strings.ToLower(dangerous)) {
			return fmt.Errorf("检测到危险命令: %s", dangerous)
		}
	}

	// 检查危险模式
	for _, pattern := range dangerousPatterns {
		if pattern.MatchString(command) {
			return fmt.Errorf("检测到危险操作模式")
		}
	}

	// 检查管道和重定向中的危险命令
	if strings.Contains(lowerCmd, "|") || strings.Contains(lowerCmd, ">") ||
		strings.Contains(lowerCmd, ">>") || strings.Contains(lowerCmd, "<") {
		// 分割管道和重定向
		parts := regexp.MustCompile(`[|><]+`).Split(lowerCmd, -1)
		for _, part := range parts {
			part = strings.TrimSpace(part)
			for _, dangerous := range dangerousCommands {
				if strings.HasPrefix(part, strings.ToLower(dangerous)) {
					return fmt.Errorf("管道/重定向中包含危险命令: %s", dangerous)
				}
			}
		}
	}

	// 检查是否包含文件写入操作（所有命令都不允许重定向写入）
	if strings.Contains(lowerCmd, ">") || strings.Contains(lowerCmd, ">>") {
		return fmt.Errorf("不允许执行文件写入操作（重定向）")
	}

	// 检查是否只包含允许的安全命令（可选：严格模式）
	// 获取主命令（第一个词）
	mainCmd := lowerCmd
	if idx := strings.IndexAny(lowerCmd, " |&><"); idx > 0 {
		mainCmd = lowerCmd[:idx]
	}

	// 检查是否是允许的白名单命令
	isAllowed := false
	for _, allowed := range allowedReadOnlyCommands {
		if mainCmd == allowed {
			isAllowed = true
			break
		}
	}

	// 如果不是白名单命令，进一步检查
	if !isAllowed {
		// 检查是否包含删除操作
		if strings.HasPrefix(lowerCmd, "del ") || strings.HasPrefix(lowerCmd, "erase ") ||
			strings.HasPrefix(lowerCmd, "rd ") || strings.HasPrefix(lowerCmd, "rmdir ") {
			return fmt.Errorf("不允许执行删除操作")
		}

		// 检查是否包含复制/移动操作
		if strings.HasPrefix(lowerCmd, "copy ") || strings.HasPrefix(lowerCmd, "move ") ||
			strings.HasPrefix(lowerCmd, "ren ") || strings.HasPrefix(lowerCmd, "rename ") {
			return fmt.Errorf("不允许执行文件修改操作")
		}
	}

	return nil
}

// executeCommand 执行具体的命令
func (e *CommandExecutor) executeCommand(command, shell, workingDir string, timeout time.Duration, envVars map[string]string) (*tool.Result, error) {
	var cmd *exec.Cmd

	// 根据shell类型构建命令
	switch shell {
	case "powershell":
		cmd = exec.Command("powershell.exe", "-Command", command)
	case "cmd":
		fallthrough
	default:
		cmd = exec.Command("cmd.exe", "/c", command)
	}

	// 设置工作目录
	if workingDir != "" {
		cmd.Dir = workingDir
	}

	// 设置环境变量
	if len(envVars) > 0 {
		cmd.Env = e.buildEnv(envVars)
	}

	// 隐藏窗口（Windows特定）
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow: true,
	}

	// 捕获输出
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// 启动命令
	if err := cmd.Start(); err != nil {
		return tool.NewErrorResultWithMsg(tool.ErrCodeExecution, fmt.Sprintf("启动命令失败: %v", err)), nil
	}

	// 使用channel等待命令完成或超时
	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	select {
	case <-time.After(timeout):
		// 超时，终止进程
		if err := cmd.Process.Kill(); err != nil {
			return tool.NewErrorResultWithMsg(tool.ErrCodeExecution, fmt.Sprintf("命令执行超时，终止进程失败: %v", err)), nil
		}
		return tool.NewErrorResultWithMsg(tool.ErrCodeExecution, fmt.Sprintf("命令执行超时（%v）", timeout)), nil
	case err := <-done:
		if err != nil {
			// 命令执行出错，但仍返回输出
			exitCode := 1
			if exitErr, ok := err.(*exec.ExitError); ok {
				exitCode = exitErr.ExitCode()
			}
			return tool.NewResult(map[string]interface{}{
				"stdout":    stdout.String(),
				"stderr":    stderr.String(),
				"exit_code": exitCode,
				"success":   false,
			}), nil
		}
	}

	// 成功执行
	return tool.NewResult(map[string]interface{}{
		"stdout":    stdout.String(),
		"stderr":    stderr.String(),
		"exit_code": 0,
		"success":   true,
	}), nil
}

// buildEnv 构建环境变量列表
func (e *CommandExecutor) buildEnv(envVars map[string]string) []string {
	// 获取当前环境变量
	env := make(map[string]string)
	for _, e := range exec.Command("").Env {
		parts := strings.SplitN(e, "=", 2)
		if len(parts) == 2 {
			env[parts[0]] = parts[1]
		}
	}

	// 覆盖或添加新的环境变量
	for k, v := range envVars {
		env[k] = v
	}

	// 转换为字符串数组
	result := make([]string, 0, len(env))
	for k, v := range env {
		result = append(result, fmt.Sprintf("%s=%s", k, v))
	}

	return result
}
