package agent

import (
	"context"
	"fmt"
	"path"
	"time"

	"github.com/longstageai/donk/donk/internal/creative"
	"github.com/longstageai/donk/donk/internal/memory"
	"github.com/longstageai/donk/donk/internal/scheduler"
	"github.com/longstageai/donk/donk/internal/tool"
	"github.com/longstageai/donk/donk/internal/tool/builtin"
)

// ExecutionAgentDeps 任务执行 Agent 的依赖配置
type ExecutionAgentDeps struct {
	HistoryStore *memory.HistoryStore // 历史记录存储（用于获取最近对话）
	Scheduler    *scheduler.Scheduler // 任务调度器（用于 task_manager）
	SkillsDir    string               // 技能目录（用于 skill_creator）
	WorkDir      string               // 工作目录（用于 file_reader/file_writer）
}

// NewExecutionAgent 创建任务执行 Agent，负责基于计划产出执行结果。
// 该 Agent 配置了 http、file_reader、file_writer、skill_creator、task_manager 工具。
// deps 参数为可选，如果为 nil 或部分字段为空，则对应工具将使用零值初始化（可能无法正常工作）。
func NewExecutionAgent(llm CreativeLLMClient, deps *ExecutionAgentDeps) creative.Agent {
	// 创建工具注册表并注册任务执行 Agent 所需的工具
	tools := tool.NewRegistry()

	// 注册 HTTP 工具
	tools.Register(builtin.NewHTTP())

	// 注册文件读取工具
	workDir := ""
	if deps != nil {
		workDir = deps.WorkDir
	}

	join := path.Join(workDir, "data", "doc")

	tools.Register(builtin.NewFileReader(builtin.WithWorkingDir(join)))

	// 注册文件写入工具
	tools.Register(builtin.NewFileWriter(builtin.WithWorkingDirWriter(join)))

	// 注册 Skill 创建工具
	if deps != nil && deps.SkillsDir != "" {
		tools.Register(builtin.NewSkillCreator(deps.SkillsDir))
	}

	// 注册任务管理工具
	if deps != nil && deps.Scheduler != nil {
		tools.Register(builtin.NewTaskManager(deps.Scheduler))
	}

	promptBuilder := func(input creative.AgentInput) PromptSpec {
		return buildExecutionPrompt(input)
	}

	opts := []LLMAgentOption{
		WithTools(tools),
	}

	if deps != nil && deps.HistoryStore != nil {
		opts = append(opts, WithHistoryStore(deps.HistoryStore))
	}

	return NewLLMAgentWithDynamicPrompt("execution", "任务执行 Agent", creative.RoleExecution, []creative.EventType{creative.EventExecutionRequested, creative.EventExecutionRevisionRequested}, promptBuilder, llm, executionOutput, opts...)
}

// buildExecutionPrompt 构建任务执行 Agent 的完整提示词
func buildExecutionPrompt(input creative.AgentInput) PromptSpec {
	// 从 input 中提取可执行计划和最终目标
	var planDescription string
	var goalTitle, goalDescription string

	for _, artifact := range input.Artifacts {
		switch artifact.Type {
		case creative.ArtifactExecutablePlan:
			if plan, ok := artifact.Data.(creative.ExecutablePlan); ok && len(plan.Steps) > 0 {
				// 合并所有步骤描述
				for _, step := range plan.Steps {
					planDescription += fmt.Sprintf("- %s: %s\n", step.Title, step.Description)
				}
			}
		case creative.ArtifactFinalExecutableGoal:
			if goal, ok := artifact.Data.(creative.FinalExecutableGoal); ok {
				goalTitle = goal.Title
				goalDescription = goal.Description
			}
		}
	}

	// 系统提示词：使用 prompts.go 中的模板
	systemPrompt := fmt.Sprintf(executionPromptTemplate, goalTitle, goalDescription, planDescription)

	// 用户提示词：强化按步骤执行的要求
	userPrompt := fmt.Sprintf(`【执行任务指令】

当前事件类型：%s
当前阶段：%s

执行要求：
1. 首先，仔细分析系统提示词中的"可执行计划"，识别出所有需要执行的步骤
2. 严格按照步骤顺序，从第1步开始逐一执行
3. 每完成一个步骤，必须立即输出该步骤的执行结果（使用工具、执行状态、详细输出）
4. 如果某个步骤执行失败，记录失败原因后继续尝试执行后续步骤（如果可能）
5. 所有步骤执行完毕后，输出"执行汇总"部分

重要提醒：
- 不要跳过任何步骤，也不要合并多个步骤的输出
- 每个步骤必须独立输出，包含：执行动作、使用工具、执行结果、详细输出
- 优先使用可用工具（http/file_reader/file_writer/skill_creator/task_manager）完成实际任务
- 不要只给出文本描述，必须实际执行计划中的每个步骤

输出要求：
请严格按照系统提示词中的"输出格式"部分，按步骤输出执行结果。`, input.Event.Type, input.Session.CurrentPhase)

	return PromptSpec{
		SystemPrompt: systemPrompt,
		UserPrompt:   userPrompt,
		OutputFormat: "请严格按照系统提示词中的输出格式，按步骤逐一输出执行结果。",
	}
}

func executionOutput(ctx context.Context, input creative.AgentInput, content string, usage creative.TokenUsage) creative.AgentOutput {
	result := creative.ExecutionResult{ID: creative.NextID("execution"), PlanID: input.Session.PlanID, StepResults: []creative.StepResult{{Status: "done", Output: content}}, CreatedAt: time.Now()}
	return creative.AgentOutput{Status: creative.AgentRunSucceeded, Decision: creative.DecisionSucceeded, TokenUsage: usage, Messages: []creative.MessageDraft{{Role: creative.MessageRoleAgent, Content: content}}, Artifacts: []creative.ArtifactDraft{{Type: creative.ArtifactExecutionResult, Data: result}}, Events: []creative.EventDraft{{Type: creative.EventResultReviewRequested, DispatchMode: creative.DispatchExclusive, Priority: 20}}}
}
