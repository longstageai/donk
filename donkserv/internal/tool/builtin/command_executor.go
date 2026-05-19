package builtin

import (
	"bytes"
	"fmt"
	"os/exec"
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
type CommandExecutor struct {
	defaultTimeout time.Duration // 默认超时时间
}

// CommandExecutorOption 命令执行器配置选项
type CommandExecutorOption func(*CommandExecutor)

// WithDefaultTimeout 设置默认超时时间
func WithDefaultTimeout(timeout time.Duration) CommandExecutorOption {
	return func(e *CommandExecutor) {
		e.defaultTimeout = timeout
	}
}

// NewCommandExecutor 创建命令行执行工具
func NewCommandExecutor(opts ...CommandExecutorOption) *CommandExecutor {
	e := &CommandExecutor{
		defaultTimeout: 60 * time.Second, // 默认60秒超时
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
	return "在Windows系统上执行命令行命令。支持超时控制、工作目录设置、环境变量传递。"
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
			Description: "要执行的命令",
		},
		"shell": {
			Type:        "string",
			Description: "使用的shell类型，支持 cmd 或 powershell，默认 cmd",
			Default:     "cmd",
			Enum:        []interface{}{"cmd", "powershell"},
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

	// 获取shell类型
	shell := "cmd"
	if s, ok := ctx.Params["shell"].(string); ok && s != "" {
		shell = strings.ToLower(s)
	}

	// 获取工作目录
	workingDir, ok := ctx.Params["working_dir"].(string)
	if !ok || workingDir == "" {
		return tool.NewErrorResultWithMsg(tool.ErrCodeInvalidParams, "工作目录不能为空"), nil
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
