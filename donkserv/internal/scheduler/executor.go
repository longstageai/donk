package scheduler

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"reflect"
	"time"
)

// Executor 执行器接口
// 所有执行器必须实现此接口，用于执行不同类型的任务
type Executor interface {
	// Execute 执行任务并返回结果
	Execute(ctx context.Context, task *Task) (*TaskResult, error)
}

// ExecutorFactory 执行器工厂接口
// 用于根据执行器类型创建对应的执行器实例
type ExecutorFactory interface {
	// Create 创建指定类型的执行器
	Create(executorType ExecutorType) (Executor, error)
}

// BaseExecutor 基类执行器，提供通用功能
// 嵌入到具体执行器中复用公共逻辑
type BaseExecutor struct{}

// 计算任务执行耗时
func (e *BaseExecutor) measureTime(startTime time.Time) int64 {
	return time.Now().Sub(startTime).Milliseconds()
}

// ScriptExecutor 脚本/命令执行器
// 用于执行本地脚本或系统命令
type ScriptExecutor struct {
	BaseExecutor
}

// NewScriptExecutor 创建脚本执行器实例
func NewScriptExecutor() *ScriptExecutor {
	return &ScriptExecutor{}
}

// Execute 实现 Executor 接口
// 执行配置的脚本或命令，返回执行结果
func (e *ScriptExecutor) Execute(ctx context.Context, task *Task) (*TaskResult, error) {
	startTime := time.Now()

	// 从配置中获取执行命令
	command := task.Config.GetString("command")
	if command == "" {
		return &TaskResult{
			Error:    "未配置执行命令",
			Duration: e.measureTime(startTime),
			DoneAt:   time.Now().Unix(),
		}, nil
	}

	// 获取命令参数
	args := task.Config.GetString("args")
	var cmdArgs []string
	if args != "" {
		// 简单解析，实际使用可以考虑更复杂的参数处理
		cmdArgs = []string{}
	}

	// 获取工作目录
	workDir := task.Config.GetString("workdir")
	if workDir == "" {
		workDir = "."
	}

	// 创建命令
	cmd := exec.CommandContext(ctx, command, cmdArgs...)
	cmd.Dir = workDir

	// 设置环境变量
	env := task.Config.GetMap("env")
	if env != nil {
		for k, v := range env {
			if s, ok := v.(string); ok {
				cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, s))
			}
		}
	}
	// 继承当前进程环境变量
	cmd.Env = append(cmd.Env, os.Environ()...)

	// 设置输出
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// 执行命令
	err := cmd.Run()

	// 构建结果
	result := &TaskResult{
		DoneAt:   time.Now().Unix(),
		Duration: e.measureTime(startTime),
	}

	if err != nil {
		// 获取退出码
		if exitErr, ok := err.(*exec.ExitError); ok {
			if exitErr.ExitCode() != 0 {
				result.ExitCode = exitErr.ExitCode()
			} else {
				result.ExitCode = 200
			}
		} else {
			result.Error = err.Error()
		}
		result.Output = stderr.String()
	} else {
		result.ExitCode = 200
		result.Output = stdout.String()
	}

	return result, nil
}

// Ensure ScriptExecutor 实现 Executor 接口
var _ Executor = (*ScriptExecutor)(nil)

// APIExecutor HTTP API 执行器
// 用于调用外部 HTTP API
type APIExecutor struct {
	BaseExecutor
	httpClient *http.Client
}

// NewAPIExecutor 创建 API 执行器实例
func NewAPIExecutor() *APIExecutor {
	return &APIExecutor{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Execute 实现 Executor 接口
// 调用配置的 HTTP API，返回执行结果
func (e *APIExecutor) Execute(ctx context.Context, task *Task) (*TaskResult, error) {
	startTime := time.Now()

	// 从配置中获取 API 信息
	url := task.Config.GetString("url")
	if url == "" {
		return &TaskResult{
			Error:    "未配置 API URL",
			Duration: e.measureTime(startTime),
			DoneAt:   time.Now().Unix(),
		}, nil
	}

	// 获取请求方法，默认 GET
	method := task.Config.GetString("method")
	if method == "" {
		method = "GET"
	}

	// 构建请求
	var body io.Reader
	requestBody := task.Config.GetString("body")
	if requestBody != "" {
		body = bytes.NewReader([]byte(requestBody))
	}

	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return &TaskResult{
			Error:    fmt.Sprintf("创建请求失败: %v", err),
			Duration: e.measureTime(startTime),
			DoneAt:   time.Now().Unix(),
		}, nil
	}

	// 设置请求头
	headers := task.Config.GetMap("headers")
	if headers != nil {
		for k, v := range headers {
			if s, ok := v.(string); ok {
				req.Header.Set(k, s)
			}
		}
	}

	// 设置默认 Content-Type
	if req.Header.Get("Content-Type") == "" {
		req.Header.Set("Content-Type", "application/json")
	}

	// 发送请求
	resp, err := e.httpClient.Do(req)
	if err != nil {
		return &TaskResult{
			Error:    fmt.Sprintf("请求失败: %v", err),
			Duration: e.measureTime(startTime),
			DoneAt:   time.Now().Unix(),
		}, nil
	}
	defer resp.Body.Close()

	// 读取响应
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return &TaskResult{
			Error:    fmt.Sprintf("读取响应失败: %v", err),
			Duration: e.measureTime(startTime),
			DoneAt:   time.Now().Unix(),
		}, nil
	}

	// 构建结果
	result := &TaskResult{
		DoneAt:   time.Now().Unix(),
		Duration: e.measureTime(startTime),
		ExitCode: resp.StatusCode,
	}

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		result.Output = string(respBody)
	} else {
		result.Error = fmt.Sprintf("HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	return result, nil
}

// Ensure APIExecutor 实现 Executor 接口
var _ Executor = (*APIExecutor)(nil)

// AgentExecutor LLM Agent 执行器
// 用于调用 LLM Agent 执行任务
type AgentExecutor struct {
	BaseExecutor
	agentFactory func() interface{}
}

// SetAgentFactory 设置 Agent 工厂函数
// 用于在调度器启动时注入 Agent 工厂，AgentExecutor 会调用此函数获取 Agent 实例
func (e *AgentExecutor) SetAgentFactory(factory func() interface{}) {
	e.agentFactory = factory
}

// NewAgentExecutor 创建 Agent 执行器实例
func NewAgentExecutor() *AgentExecutor {
	return &AgentExecutor{}
}

// Execute 实现 Executor 接口
// 调用 LLM Agent 执行任务
func (e *AgentExecutor) Execute(ctx context.Context, task *Task) (*TaskResult, error) {
	startTime := time.Now()

	// 从配置中获取 Agent 执行所需参数
	prompt := task.Config.GetString("prompt")
	if prompt == "" {
		return &TaskResult{
			Error:    "未配置任务提示",
			Duration: e.measureTime(startTime),
			DoneAt:   time.Now().Unix(),
		}, nil
	}

	// 获取超时时间（默认 5 分钟）
	timeout := task.Config.GetInt("timeout")
	if timeout == 0 {
		timeout = 300
	}

	// 创建带超时的上下文
	ctx, cancel := context.WithTimeout(ctx, time.Duration(timeout)*time.Second)
	defer cancel()

	// 检查是否已注入 Agent 工厂函数
	if e.agentFactory == nil {
		return &TaskResult{
			Error:    "Agent 工厂未初始化，请在调度器启动前调用 SetAgentFactory 设置 Agent 工厂",
			Duration: e.measureTime(startTime),
			DoneAt:   time.Now().Unix(),
		}, nil
	}

	// 通过工厂函数获取 Agent 实例
	agentInstance := e.agentFactory()
	if agentInstance == nil {
		return &TaskResult{
			Error:    "Agent 实例创建失败",
			Duration: e.measureTime(startTime),
			DoneAt:   time.Now().Unix(),
		}, nil
	}

	// 使用反射调用 Agent 的 Run 方法同步执行任务
	resultValue := reflect.ValueOf(agentInstance).MethodByName("Run")
	if !resultValue.IsValid() {
		return &TaskResult{
			Error:    "Agent 未实现 Run 方法",
			Duration: e.measureTime(startTime),
			DoneAt:   time.Now().Unix(),
		}, nil
	}

	// 调用 Agent.Run(ctx, prompt) 方法
	results := resultValue.Call([]reflect.Value{
		reflect.ValueOf(ctx),
		reflect.ValueOf(prompt),
	})

	// 获取返回结果
	output := results[0].String()
	errVal := results[1].Interface()
	var err error
	if errVal != nil {
		err = errVal.(error)
	}

	if err != nil {
		return &TaskResult{
			Error:    fmt.Sprintf("Agent 执行失败: %v", err),
			Duration: e.measureTime(startTime),
			DoneAt:   time.Now().Unix(),
		}, nil
	}

	result := &TaskResult{
		DoneAt:   time.Now().Unix(),
		Duration: e.measureTime(startTime),
		Output:   output,
		ExitCode: 200,
	}

	return result, nil
}

// Ensure AgentExecutor 实现 Executor 接口
var _ Executor = (*AgentExecutor)(nil)

// DefaultExecutorFactory 默认执行器工厂实现
// 根据执行器类型创建对应的执行器实例
type DefaultExecutorFactory struct {
	scriptExecutor *ScriptExecutor
	apiExecutor    *APIExecutor
	agentExecutor  *AgentExecutor
}

// NewDefaultExecutorFactory 创建默认执行器工厂
func NewDefaultExecutorFactory() *DefaultExecutorFactory {
	return &DefaultExecutorFactory{
		scriptExecutor: NewScriptExecutor(),
		apiExecutor:    NewAPIExecutor(),
		agentExecutor:  NewAgentExecutor(),
	}
}

// SetAgentFactory 设置 Agent 工厂函数
// 用于在调度器启动时注入 Agent 工厂
func (f *DefaultExecutorFactory) SetAgentFactory(factory func() interface{}) {
	if f.agentExecutor != nil {
		f.agentExecutor.SetAgentFactory(factory)
	}
}

// Create 实现 ExecutorFactory 接口
func (f *DefaultExecutorFactory) Create(executorType ExecutorType) (Executor, error) {
	switch executorType {
	case ExecutorScript:
		return f.scriptExecutor, nil
	case ExecutorAPI:
		return f.apiExecutor, nil
	case ExecutorAgent:
		return f.agentExecutor, nil
	default:
		return nil, fmt.Errorf("未知的执行器类型: %s", executorType)
	}
}

// Ensure DefaultExecutorFactory 实现 ExecutorFactory 接口
var _ ExecutorFactory = (*DefaultExecutorFactory)(nil)
