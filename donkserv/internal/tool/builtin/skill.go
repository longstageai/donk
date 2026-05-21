package builtin

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/longstageai/donk/donk/internal/skill"
	"github.com/longstageai/donk/donk/internal/tool"
)

// SkillToolName Skill工具名称
const SkillToolName = "skill"

// SkillAction Skill操作类型
type SkillAction string

const (
	ActionLoadInstructions SkillAction = "load_instructions" // 加载Skill完整指令
	ActionLoadReferences   SkillAction = "load_references"   // 加载参考资料
	ActionLoadAssets       SkillAction = "load_assets"       // 加载资源文件
	ActionExecuteScript    SkillAction = "execute_script"    // 执行脚本
	ActionInfo             SkillAction = "info"              // 获取Skill元信息
)

// SkillTool 技能工具
// 统一入口工具，通过参数区分不同的Skill和操作
type SkillTool struct {
	registry     *skill.SkillRegistry // Skill注册表
	executor     *skill.Executor      // 执行器
	workingDir   string               // 工作目录
	scriptRunner *ScriptRunner        // Donk脚本执行器
}

// NewSkillTool 创建新的技能工具
// 参数:
//   - registry: Skill注册表
//   - executor: 执行器（可选，如果为nil则自动创建）
//   - workingDir: 工作目录
//
// 返回:
//   - *SkillTool: 技能工具实例
func NewSkillTool(registry *skill.SkillRegistry, executor *skill.Executor, workingDir string) *SkillTool {
	// 如果executor为nil，自动创建
	if executor == nil {
		executor = skill.NewExecutor(registry, skill.WithWorkingDir(workingDir))
	}
	return &SkillTool{
		registry:     registry,
		executor:     executor,
		workingDir:   workingDir,
		scriptRunner: NewScriptRunner(WithScriptRunnerBaseDir(filepath.Join(workingDir, "script_runtime"))),
	}
}

// Name 返回工具名称
// 参数:
//   - 无
//
// 返回:
//   - string: 工具名称 "skill"
func (t *SkillTool) Name() string {
	return SkillToolName
}

// Description 返回工具描述
// 参数:
//   - 无
//
// 返回:
//   - string: 工具描述信息（动态生成）
func (t *SkillTool) Description() string {
	var sb strings.Builder
	sb.WriteString("技能系统 (Skill Tool)\n\n")
	sb.WriteString("当用户请求需要特定技能处理时使用此工具。\n\n")
	sb.WriteString("使用方式:\n")
	sb.WriteString("1. 根据用户意图选择最合适的技能\n")
	sb.WriteString("2. 指定 skill_name 为技能名称\n")
	sb.WriteString("3. 通过 action 参数指定操作类型（load_instructions, execute_script, load_references, load_assets, info）\n")
	sb.WriteString("4. execute_script 如未指定 params.script 且技能只有一个脚本，会自动执行该脚本；多个脚本时必须在 params.script 指定脚本名\n")
	sb.WriteString("5. Python 脚本会使用 Donk script_runner 执行，不依赖系统全局 python；第三方依赖由 python_dependency_manager 管理\n\n")
	sb.WriteString("可用的技能列表:\n")

	skills := t.registry.List()
	if len(skills) == 0 {
		sb.WriteString("- 暂无可用技能\n")
	} else {
		for _, s := range skills {
			sb.WriteString(fmt.Sprintf("- %s: %s\n", s.Name(), s.Description()))
		}
	}

	return sb.String()
}

// Version 返回工具版本
// 参数:
//   - 无
//
// 返回:
//   - string: 版本号
func (t *SkillTool) Version() string {
	return "1.0.0"
}

// Category 返回工具类别
// 参数:
//   - 无
//
// 返回:
//   - string: 类别名称
func (t *SkillTool) Category() string {
	return "skill"
}

// Parameters 返回参数定义
// 参数:
//   - 无
//
// 返回:
//   - *tool.Schema: 参数Schema定义
func (t *SkillTool) Parameters() *tool.Schema {
	// 获取所有可用的Skill名称
	skills := t.registry.List()
	skillNames := make([]interface{}, len(skills))
	for i, s := range skills {
		skillNames[i] = s.Name()
	}

	schema := tool.NewSchema()
	schema.Type = "object"
	schema.Properties = map[string]*tool.Property{
		"skill_name": {
			Type:        "string",
			Description: "要调用的技能名称",
			Enum:        skillNames,
		},
		"action": {
			Type:        "string",
			Description: "要执行的操作类型",
			Enum:        []interface{}{"load_instructions", "load_references", "load_assets", "execute_script", "info"},
		},
		"params": {
			Type:        "object",
			Description: "操作参数，根据action类型不同而不同。execute_script可传script和args；如果不传script且Skill只有一个脚本，会自动执行该脚本",
		},
	}
	schema.Required = []string{"skill_name", "action"}

	return schema
}

// Execute 执行技能工具
// 根据action参数执行不同的操作
// 参数:
//   - ctx: 工具执行上下文
//
// 返回:
//   - *tool.Result: 执行结果
//   - error: 执行错误
func (t *SkillTool) Execute(ctx *tool.Context) (*tool.Result, error) {
	startTime := time.Now()

	// 获取参数
	skillName, _ := ctx.Params["skill_name"].(string)
	action, _ := ctx.Params["action"].(string)
	params, _ := ctx.Params["params"].(map[string]any)

	// 验证必填参数
	if skillName == "" {
		return t.errorResult("MISSING_PARAM", "缺少必填参数skill_name", map[string]any{"action": action}), nil
	}
	if action == "" {
		return t.errorResult("MISSING_PARAM", "缺少必填参数action", map[string]any{"skill_name": skillName}), nil
	}

	// 获取Skill实例
	s, err := t.registry.Get(skillName)
	if err != nil {
		return t.errorResult("SKILL_NOT_FOUND", fmt.Sprintf("技能不存在: %s", skillName), map[string]any{
			"skill_name":       skillName,
			"available_skills": t.availableSkillNames(),
		}), nil
	}

	// 创建执行上下文
	execCtx := skill.NewExecutionContext(s, t.workingDir)

	// 根据action执行不同操作
	var output string
	switch SkillAction(action) {
	case ActionLoadInstructions:
		// 加载完整指令
		output = t.handleLoadInstructions(s)

	case ActionLoadReferences:
		// 加载参考资料
		output, err = t.handleLoadReferences(execCtx, params)

	case ActionLoadAssets:
		// 加载资源文件
		output, err = t.handleLoadAssets(execCtx, params)

	case ActionExecuteScript:
		// 执行脚本
		output, err = t.handleExecuteScript(execCtx, params)

	case ActionInfo:
		// 获取Skill元信息
		output = t.handleInfo(s)

	default:
		return t.errorResult("UNKNOWN_ACTION", fmt.Sprintf("未知操作: %s", action), map[string]any{
			"skill_name": skillName,
			"action":     action,
		}), nil
	}

	if err != nil {
		return t.errorResult("EXECUTE_ERROR", err.Error(), map[string]any{
			"skill_name": skillName,
			"action":     action,
			"params":     params,
		}), nil
	}

	// 返回成功结果
	result := tool.NewResult(output)
	result.ExecutionTime = time.Since(startTime)
	result.Metadata = map[string]any{
		"skill_name": skillName,
		"action":     action,
	}

	return result, nil
}

// handleLoadInstructions 处理加载指令操作
// 参数:
//   - s: Skill实例
//
// 返回:
//   - string: 指令内容
func (t *SkillTool) handleLoadInstructions(s *skill.Skill) string {
	return s.Instructions()
}

// handleLoadReferences 处理加载参考资料操作
// 参数:
//   - execCtx: 执行上下文
//   - params: 操作参数
//
// 返回:
//   - string: 参考资料内容
//   - error: 加载错误
func (t *SkillTool) handleLoadReferences(execCtx *skill.ExecutionContext, params map[string]any) (string, error) {
	refs, err := t.executor.LoadReferences(execCtx)
	if err != nil {
		return "", fmt.Errorf("加载参考资料失败: %w", err)
	}

	if refs == nil {
		return "该技能没有参考资料", nil
	}

	// 如果指定了文件名，返回指定文件内容
	if file, ok := params["file"].(string); ok {
		if content, exists := refs[file]; exists {
			return content, nil
		}
		return "", fmt.Errorf("文件不存在: %s", file)
	}

	// 否则返回所有参考资料
	var sb strings.Builder
	for name, content := range refs {
		sb.WriteString(fmt.Sprintf("=== %s ===\n%s\n\n", name, content))
	}

	return sb.String(), nil
}

// handleLoadAssets 处理加载资源文件操作
// 参数:
//   - execCtx: 执行上下文
//   - params: 操作参数
//
// 返回:
//   - string: 资源文件内容
//   - error: 加载错误
func (t *SkillTool) handleLoadAssets(execCtx *skill.ExecutionContext, params map[string]any) (string, error) {
	file, ok := params["file"].(string)
	if !ok || file == "" {
		return "", fmt.Errorf("缺少参数file")
	}

	data, err := t.executor.LoadAsset(execCtx, file)
	if err != nil {
		return "", fmt.Errorf("加载资源文件失败: %w", err)
	}

	return string(data), nil
}

// handleExecuteScript 处理执行脚本操作
// 参数:
//   - execCtx: 执行上下文
//   - params: 操作参数
//
// 返回:
//   - string: 脚本执行输出
//   - error: 执行错误
func (t *SkillTool) handleExecuteScript(execCtx *skill.ExecutionContext, params map[string]any) (string, error) {
	if params == nil {
		params = map[string]any{}
	}

	script, _ := params["script"].(string)
	script = strings.TrimSpace(script)
	if script == "" {
		defaultScript, err := t.resolveDefaultScript(execCtx)
		if err != nil {
			return "", err
		}
		script = defaultScript
	}

	// 获取脚本参数
	var args []string
	if argsVal, ok := params["args"].([]any); ok {
		for _, arg := range argsVal {
			if str, ok := arg.(string); ok {
				args = append(args, str)
			} else if m, ok := arg.(map[string]any); ok {
				jsonBytes, err := json.Marshal(m)
				if err == nil {
					args = append(args, string(jsonBytes))
				}
			}
		}
	} else if argsVal, ok := params["args"].(map[string]any); ok {
		jsonBytes, err := json.Marshal(argsVal)
		if err == nil {
			args = append(args, string(jsonBytes))
		}
	}

	scriptPath, err := t.resolveScriptPath(execCtx, script)
	if err != nil {
		return "", err
	}

	if t.canRunWithScriptRunner(scriptPath) {
		return t.executeScriptWithRunner(execCtx, scriptPath, args, params)
	}

	output, err := t.executor.ExecuteScript(execCtx, script, args...)
	if err != nil {
		return "", fmt.Errorf("执行脚本失败: %w", err)
	}

	return output, nil
}

func (t *SkillTool) resolveDefaultScript(execCtx *skill.ExecutionContext) (string, error) {
	scripts, err := execCtx.Skill.ListScripts()
	if err != nil {
		return "", fmt.Errorf("列出脚本失败: %w", err)
	}
	if len(scripts) == 0 {
		return "", fmt.Errorf("该技能没有可执行脚本，请改用load_instructions读取技能指令")
	}
	if len(scripts) > 1 {
		names := make([]string, 0, len(scripts))
		for _, script := range scripts {
			names = append(names, filepath.Base(script))
		}
		return "", fmt.Errorf("该技能包含多个脚本，请在params.script中指定要执行的脚本: %s", strings.Join(names, ", "))
	}
	return filepath.Base(scripts[0]), nil
}

func (t *SkillTool) resolveScriptPath(execCtx *skill.ExecutionContext, scriptName string) (string, error) {
	scripts, err := execCtx.Skill.ListScripts()
	if err != nil {
		return "", fmt.Errorf("列出脚本失败: %w", err)
	}
	for _, script := range scripts {
		if strings.HasSuffix(script, scriptName) || filepath.Base(script) == scriptName || filepath.Base(script) == filepath.Base(scriptName) {
			return script, nil
		}
	}
	return "", fmt.Errorf("未找到脚本: %s", scriptName)
}

func (t *SkillTool) canRunWithScriptRunner(scriptPath string) bool {
	ext := strings.ToLower(filepath.Ext(scriptPath))
	return ext == ".py"
}

func (t *SkillTool) executeScriptWithRunner(execCtx *skill.ExecutionContext, scriptPath string, args []string, params map[string]any) (string, error) {
	if t.scriptRunner == nil {
		return "", fmt.Errorf("Donk脚本执行器未初始化")
	}
	code, err := os.ReadFile(scriptPath)
	if err != nil {
		return "", fmt.Errorf("读取脚本失败 %s: %w", filepath.Base(scriptPath), err)
	}
	language, err := skillScriptLanguage(scriptPath)
	if err != nil {
		return "", err
	}
	stdin := strings.Join(args, "\n")
	if explicitStdin := stringParam(params["stdin"]); explicitStdin != "" {
		stdin = explicitStdin
	}
	codeText := t.prepareScriptRunnerCode(string(code), scriptPath, args)
	runnerParams := map[string]any{
		"language":        language,
		"code":            codeText,
		"runtime_version": t.resolveSkillRuntimeVersion(execCtx.Skill, scriptPath, params),
		"timeout":         params["timeout"],
		"stdin":           stdin,
		"env":             t.resolveSkillScriptEnv(language, params),
	}
	runnerCtx := tool.NewContext(t.scriptRunner.Name(), runnerParams)
	result, err := t.scriptRunner.Execute(runnerCtx)
	if err != nil {
		return "", err
	}
	if result == nil {
		return "", fmt.Errorf("脚本执行器无返回结果")
	}
	output := formatScriptRunnerResult(result.Data)
	if !result.Success || scriptRunnerDataFailed(result.Data) {
		return "", fmt.Errorf("脚本执行失败: script=%s output=%s", filepath.Base(scriptPath), output)
	}
	return output, nil
}

func (t *SkillTool) resolveSkillScriptEnv(language string, params map[string]any) map[string]string {
	env := stringMapParam(params["env"])
	if language == "python" {
		env["PYTHONIOENCODING"] = "utf-8"
		env["PYTHONUTF8"] = "1"
	}
	return env
}

func (t *SkillTool) prepareScriptRunnerCode(code, scriptPath string, args []string) string {
	ext := strings.ToLower(filepath.Ext(scriptPath))
	if len(args) == 0 {
		return code
	}
	encodedArgs, err := json.Marshal(args)
	if err != nil {
		return code
	}
	switch ext {
	case ".py":
		return "import sys\nsys.argv = [" + strconv.Quote(filepath.Base(scriptPath)) + "] + " + string(encodedArgs) + "\n" + code
	default:
		return code
	}
}

func (t *SkillTool) resolveSkillRuntimeVersion(s *skill.Skill, scriptPath string, params map[string]any) string {
	if version := strings.TrimSpace(stringParam(params["runtime_version"])); version != "" {
		return version
	}
	return s.ScriptConfig(filepath.Base(scriptPath)).RuntimeVersion
}

func skillScriptLanguage(scriptPath string) (string, error) {
	switch strings.ToLower(filepath.Ext(scriptPath)) {
	case ".py":
		return "python", nil
	default:
		return "", fmt.Errorf("不支持Donk脚本执行器执行的脚本类型: %s", filepath.Ext(scriptPath))
	}
}

func (t *SkillTool) errorResult(errorType, message string, details map[string]any) *tool.Result {
	data := map[string]any{
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

func (t *SkillTool) availableSkillNames() []string {
	skills := t.registry.List()
	names := make([]string, 0, len(skills))
	for _, s := range skills {
		names = append(names, s.Name())
	}
	sort.Strings(names)
	return names
}

func scriptRunnerDataFailed(data any) bool {
	result, ok := data.(map[string]interface{})
	if !ok {
		return false
	}
	success, ok := result["success"].(bool)
	return ok && !success
}

func formatScriptRunnerResult(data any) string {
	result, ok := data.(map[string]interface{})
	if !ok {
		return fmt.Sprintf("%v", data)
	}
	stdout := stringParam(result["stdout"])
	stderr := stringParam(result["stderr"])
	if success, _ := result["success"].(bool); !success {
		parts := []string{}
		if message := stringParam(result["message"]); message != "" {
			parts = append(parts, message)
		}
		if errorType := stringParam(result["error_type"]); errorType != "" {
			parts = append(parts, "错误类型: "+errorType)
		}
		if status := stringParam(result["status"]); status != "" {
			parts = append(parts, "状态: "+status)
		}
		if stderr != "" {
			parts = append(parts, "stderr: "+stderr)
		}
		if stdout != "" {
			parts = append(parts, "stdout: "+stdout)
		}
		if details, ok := result["details"]; ok && details != nil {
			parts = append(parts, fmt.Sprintf("details: %v", details))
		}
		if len(parts) == 0 {
			return fmt.Sprintf("执行失败: %v", data)
		}
		return "执行失败: " + strings.Join(parts, "；")
	}
	if stderr != "" {
		return strings.TrimSpace(stdout + "\n" + stderr)
	}
	return stdout
}

func stringParamDefault(value interface{}, defaultValue string) string {
	valueString := stringParam(value)
	if valueString == "" {
		return defaultValue
	}
	return valueString
}

// handleInfo 处理获取元信息操作
// 参数:
//   - s: Skill实例
//
// 返回:
//   - string: 元信息内容
func (t *SkillTool) handleInfo(s *skill.Skill) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("技能名称: %s\n", s.Name()))
	sb.WriteString(fmt.Sprintf("版本: %s\n", s.Version()))
	sb.WriteString(fmt.Sprintf("描述: %s\n", s.Description()))
	sb.WriteString(fmt.Sprintf("作者: %s\n", s.Author()))
	sb.WriteString(fmt.Sprintf("标签: %v\n", s.Tags()))

	if hint := s.ArgumentHint(); hint != "" {
		sb.WriteString(fmt.Sprintf("参数提示: %s\n", hint))
	}

	if requires := s.Requires(); len(requires) > 0 {
		sb.WriteString(fmt.Sprintf("依赖技能: %v\n", requires))
	}

	sb.WriteString(fmt.Sprintf("用户可调用: %v\n", s.IsUserInvocable()))

	if s.HasScripts() {
		scripts, _ := s.ListScripts()
		sb.WriteString(fmt.Sprintf("可用脚本: %v\n", scripts))
	}

	if s.HasReferences() {
		refs, _ := s.ListReferences()
		sb.WriteString(fmt.Sprintf("参考资料: %v\n", refs))
	}

	if s.HasAssets() {
		assets, _ := s.ListAssets()
		sb.WriteString(fmt.Sprintf("资源文件: %v\n", assets))
	}

	return sb.String()
}

// MustNewSkillTool 创建技能工具（ panic如果失败）
// 参数:
//   - registry: Skill注册表
//   - workingDir: 工作目录
//
// 返回:
//   - *SkillTool: 技能工具实例
func MustNewSkillTool(registry *skill.SkillRegistry, workingDir string) *SkillTool {
	executor := skill.NewExecutor(registry, skill.WithWorkingDir(workingDir))
	if executor == nil {
		panic("创建Executor失败")
	}
	return NewSkillTool(registry, executor, workingDir)
}
