package skill

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// Executor Skill执行器
// 负责执行Skill的脚本、工具和加载资源
type Executor struct {
	registry     *SkillRegistry // Skill注册表
	toolRegistry interface {    // 工具注册表（执行工具用）
		Execute(name string, params map[string]any) (*struct {
			Output string
			Error  string
		}, error)
	}
	workingDir string        // 工作目录
	timeout    time.Duration // 脚本执行超时时间
}

// ExecutorOption Executor配置选项
type ExecutorOption func(*Executor)

// WithToolRegistry 设置工具注册表
// 参数:
//   - tr: 工具注册表接口
//
// 返回:
//   - ExecutorOption: 配置选项
func WithToolRegistry(tr interface {
	Execute(name string, params map[string]any) (*struct {
		Output string
		Error  string
	}, error)
}) ExecutorOption {
	return func(e *Executor) {
		e.toolRegistry = tr
	}
}

// WithWorkingDir 设置工作目录
// 参数:
//   - dir: 工作目录路径
//
// 返回:
//   - ExecutorOption: 配置选项
func WithWorkingDir(dir string) ExecutorOption {
	return func(e *Executor) {
		e.workingDir = dir
	}
}

// WithTimeout 设置脚本执行超时时间
// 参数:
//   - timeout: 超时时间
//
// 返回:
//   - ExecutorOption: 配置选项
func WithTimeout(timeout time.Duration) ExecutorOption {
	return func(e *Executor) {
		e.timeout = timeout
	}
}

// NewExecutor 创建新的Executor实例
// 参数:
//   - registry: Skill注册表
//   - opts: 配置选项
//
// 返回:
//   - *Executor: Executor实例
func NewExecutor(registry *SkillRegistry, opts ...ExecutorOption) *Executor {
	e := &Executor{
		registry:   registry,
		workingDir: ".",
		timeout:    30 * time.Second,
	}

	for _, opt := range opts {
		opt(e)
	}

	return e
}

// ExecutionContext Skill执行上下文
// 在执行过程中维护状态信息
type ExecutionContext struct {
	Skill      *Skill                 // 当前执行的Skill
	WorkingDir string                 // 工作目录
	Variables  map[string]interface{} // 变量映射
	StartTime  time.Time              // 开始时间
	EndTime    time.Time              // 结束时间
	Output     string                 // 输出内容
	Error      error                  // 错误信息
	Logs       []string               // 执行日志
	ScriptsRun []string               // 已执行的脚本列表
	ToolsUsed  []string               // 已使用的工具列表
}

// AddLog 添加执行日志
// 参数:
//   - format: 格式化字符串
//   - args: 参数列表
func (ec *ExecutionContext) AddLog(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	ec.Logs = append(ec.Logs, msg)
}

// Duration 获取执行耗时
// 参数:
//   - 无
//
// 返回:
//   - time.Duration: 执行耗时
func (ec *ExecutionContext) Duration() time.Duration {
	if ec.EndTime.IsZero() {
		return time.Since(ec.StartTime)
	}
	return ec.EndTime.Sub(ec.StartTime)
}

// NewExecutionContext 创建新的执行上下文
// 参数:
//   - skill: Skill实例
//   - workingDir: 工作目录
//
// 返回:
//   - *ExecutionContext: 执行上下文实例
func NewExecutionContext(skill *Skill, workingDir string) *ExecutionContext {
	return &ExecutionContext{
		Skill:      skill,
		WorkingDir: workingDir,
		Variables:  make(map[string]interface{}),
		StartTime:  time.Now(),
		Logs:       []string{},
		ScriptsRun: []string{},
		ToolsUsed:  []string{},
	}
}

// Activate 激活指定的Skill
// 检查Skill是否存在，验证依赖是否满足
// 参数:
//   - skillName: Skill名称
//
// 返回:
//   - *ExecutionContext: 执行上下文
//   - error: 激活错误
func (e *Executor) Activate(skillName string) (*ExecutionContext, error) {
	skill, err := e.registry.Get(skillName)
	if err != nil {
		return nil, fmt.Errorf("获取Skill失败: %w", err)
	}

	ctx := NewExecutionContext(skill, e.workingDir)
	ctx.AddLog("激活Skill: %s", skillName)

	// 检查依赖
	if len(skill.Requires()) > 0 {
		ctx.AddLog("检查依赖: %v", skill.Requires())
		for _, req := range skill.Requires() {
			if !e.registry.HasSkill(req) {
				return nil, fmt.Errorf("缺少依赖Skill: %s", req)
			}
		}
	}

	return ctx, nil
}

// GetInstructions 获取Skill的指令内容
// 参数:
//   - skillName: Skill名称
//
// 返回:
//   - string: 指令内容
//   - error: 获取错误
func (e *Executor) GetInstructions(skillName string) (string, error) {
	skill, err := e.registry.Get(skillName)
	if err != nil {
		return "", fmt.Errorf("获取Skill失败: %w", err)
	}
	return skill.Instructions(), nil
}

// ExecuteScript 执行Skill中的脚本
// 参数:
//   - ctx: 执行上下文
//   - scriptName: 脚本名称
//   - args: 脚本参数
//
// 返回:
//   - string: 脚本输出
//   - error: 执行错误
func (e *Executor) ExecuteScript(ctx *ExecutionContext, scriptName string, args ...string) (string, error) {
	if ctx.Skill == nil {
		return "", fmt.Errorf("未设置Skill上下文")
	}

	scripts, err := ctx.Skill.ListScripts()
	if err != nil {
		return "", fmt.Errorf("列出脚本失败: %w", err)
	}

	// 查找匹配的脚本
	var scriptPath string
	for _, script := range scripts {
		// 完整路径匹配
		if strings.HasSuffix(script, scriptName) || filepath.Base(script) == scriptName {
			scriptPath = script
			break
		}
		// 处理 scripts/xxx 或 xxx 格式
		baseName := filepath.Base(script)
		targetBase := filepath.Base(scriptName)
		if baseName == targetBase {
			scriptPath = script
			break
		}
	}

	if scriptPath == "" {
		return "", fmt.Errorf("未找到脚本: %s", scriptName)
	}

	ctx.AddLog("执行脚本: %s", scriptPath)
	ctx.ScriptsRun = append(ctx.ScriptsRun, scriptName)

	return e.runScript(scriptPath, args...)
}

// runScript 执行指定路径的脚本
// 参数:
//   - scriptPath: 脚本完整路径
//   - args: 脚本参数
//
// 返回:
//   - string: 脚本输出
//   - error: 执行错误
func (e *Executor) runScript(scriptPath string, args ...string) (string, error) {
	ext := strings.ToLower(filepath.Ext(scriptPath))

	// 根据脚本类型选择执行命令
	var cmd *exec.Cmd
	switch ext {
	case ".py":
		cmd = exec.Command("python", append([]string{scriptPath}, args...)...)
	case ".js":
		cmd = exec.Command("node", append([]string{scriptPath}, args...)...)
	case ".sh":
		cmd = exec.Command("bash", append([]string{scriptPath}, args...)...)
	case ".ps1":
		cmd = exec.Command("powershell", append([]string{"-File", scriptPath}, args...)...)
	case ".bat", ".cmd":
		cmd = exec.Command("cmd", append([]string{"/c", scriptPath}, args...)...)
	case ".go":
		cmd = exec.Command("go", append([]string{"run", scriptPath}, args...)...)
	default:
		return "", fmt.Errorf("不支持的脚本类型: %s", ext)
	}

	cmd.Dir = e.workingDir

	// 捕获标准输出和错误
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// 设置超时
	timeout := e.timeout
	done := make(chan error, 1)

	go func() {
		done <- cmd.Run()
	}()

	// 等待执行结果或超时
	select {
	case err := <-done:
		if err != nil {
			return "", fmt.Errorf("脚本执行失败: %w\n%s", err, stderr.String())
		}
		return stdout.String(), nil
	case <-time.After(timeout):
		cmd.Process.Kill()
		return "", fmt.Errorf("脚本执行超时 (%v)", timeout)
	}
}

// ExecuteTool 执行工具
// 参数:
//   - ctx: 执行上下文
//   - toolName: 工具名称
//   - params: 工具参数
//
// 返回:
//   - string: 工具输出
//   - error: 执行错误
func (e *Executor) ExecuteTool(ctx *ExecutionContext, toolName string, params map[string]interface{}) (string, error) {
	if e.toolRegistry == nil {
		return "", fmt.Errorf("未配置工具注册表")
	}

	ctx.AddLog("执行工具: %s", toolName)
	ctx.ToolsUsed = append(ctx.ToolsUsed, toolName)

	result, err := e.toolRegistry.Execute(toolName, params)
	if err != nil {
		return "", fmt.Errorf("工具执行失败: %w", err)
	}

	// 检查工具返回的错误
	if result != nil && result.Error != "" {
		return result.Output, fmt.Errorf("%s", result.Error)
	}

	return result.Output, nil
}

// LoadReferences 加载Skill的参考资料
// 参数:
//   - ctx: 执行上下文
//
// 返回:
//   - map[string]string: 文件名到内容的映射
//   - error: 加载错误
func (e *Executor) LoadReferences(ctx *ExecutionContext) (map[string]string, error) {
	if ctx.Skill == nil {
		return nil, fmt.Errorf("未设置Skill上下文")
	}

	// 检查是否有参考资料目录
	if !ctx.Skill.HasReferences() {
		return nil, nil
	}

	refs, err := ctx.Skill.ListReferences()
	if err != nil {
		return nil, fmt.Errorf("列出参考资料失败: %w", err)
	}

	// 读取所有参考资料内容
	result := make(map[string]string)
	for _, ref := range refs {
		content, err := os.ReadFile(ref)
		if err != nil {
			return nil, fmt.Errorf("读取参考资料失败 %s: %w", ref, err)
		}
		result[filepath.Base(ref)] = string(content)
	}

	ctx.AddLog("加载了 %d 个参考资料", len(result))
	return result, nil
}

// LoadAsset 加载Skill的资源文件
// 参数:
//   - ctx: 执行上下文
//   - assetName: 资源文件名
//
// 返回:
//   - []byte: 资源内容
//   - error: 加载错误
func (e *Executor) LoadAsset(ctx *ExecutionContext, assetName string) ([]byte, error) {
	if ctx.Skill == nil {
		return nil, fmt.Errorf("未设置Skill上下文")
	}

	// 检查是否有资源目录
	if !ctx.Skill.HasAssets() {
		return nil, fmt.Errorf("Skill没有assets目录")
	}

	assets, err := ctx.Skill.ListAssets()
	if err != nil {
		return nil, fmt.Errorf("列出资源失败: %w", err)
	}

	// 查找匹配的资源文件
	var assetPath string
	for _, asset := range assets {
		if strings.HasSuffix(asset, assetName) || filepath.Base(asset) == assetName {
			assetPath = asset
			break
		}
	}

	if assetPath == "" {
		return nil, fmt.Errorf("未找到资源: %s", assetName)
	}

	return os.ReadFile(assetPath)
}

// ListAvailableSkills 列出所有可用的Skill
// 参数:
//   - 无
//
// 返回:
//   - []*Skill: Skill列表
func (e *Executor) ListAvailableSkills() []*Skill {
	return e.registry.List()
}

// FindSkillByTag 根据标签查找Skill
// 参数:
//   - tag: 标签名称
//
// 返回:
//   - []*Skill: 匹配的Skill列表
func (e *Executor) FindSkillByTag(tag string) []*Skill {
	return e.registry.FindByTag(tag)
}

// GetUserInvocableSkills 获取所有用户可调用的Skill
// 参数:
//   - 无
//
// 返回:
//   - []*Skill: 可用户调用的Skill列表
func (e *Executor) GetUserInvocableSkills() []*Skill {
	return e.registry.GetUserInvocableSkills()
}

// CompleteExecution 完成执行，记录结束时间
// 参数:
//   - ctx: 执行上下文
func (e *Executor) CompleteExecution(ctx *ExecutionContext) {
	ctx.EndTime = time.Now()
}

// SkillResult Skill执行结果
// 用于记录和展示执行结果
type SkillResult struct {
	Success    bool          // 是否成功
	SkillName  string        // Skill名称
	Output     string        // 输出内容
	Error      error         // 错误信息
	Duration   time.Duration // 执行耗时
	Logs       []string      // 执行日志
	ScriptsRun []string      // 执行的脚本列表
	ToolsUsed  []string      // 使用的工具列表
}

// String 将结果转换为字符串
// 参数:
//   - 无
//
// 返回:
//   - string: 格式化的结果字符串
func (sr *SkillResult) String() string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("Skill: %s\n", sr.SkillName))
	b.WriteString(fmt.Sprintf("成功: %v\n", sr.Success))
	if sr.Output != "" {
		b.WriteString(fmt.Sprintf("输出:\n%s\n", sr.Output))
	}
	if sr.Error != nil {
		b.WriteString(fmt.Sprintf("错误: %v\n", sr.Error))
	}
	b.WriteString(fmt.Sprintf("耗时: %v\n", sr.Duration))
	if len(sr.ScriptsRun) > 0 {
		b.WriteString(fmt.Sprintf("执行的脚本: %v\n", sr.ScriptsRun))
	}
	if len(sr.ToolsUsed) > 0 {
		b.WriteString(fmt.Sprintf("使用的工具: %v\n", sr.ToolsUsed))
	}
	return b.String()
}
