package agent_test

import (
	"testing"

	"github.com/longstageai/donk/donk/internal/agent"
	"github.com/longstageai/donk/donk/internal/skill"
	"github.com/longstageai/donk/donk/internal/tool"
	"github.com/longstageai/donk/donk/internal/tool/builtin"
)

func TestAgentWithSkillTool(t *testing.T) {
	t.Log("\n=== Agent 集成 Skill 工具测试 ===\n")

	skillDir := "../../data/skills"
	loader := skill.NewSkillLoader(skillDir)
	registry := skill.NewSkillRegistryWithLoader(loader)

	if err := registry.LoadAndRegister(); err != nil {
		t.Fatalf("注册Skill失败: %v", err)
	}

	tools := tool.NewRegistry()
	executor := skill.NewExecutor(registry, skill.WithWorkingDir("."))
	skillTool := builtin.NewSkillTool(registry, executor, ".")
	tools.Register(skillTool)

	registeredTools := tools.List()
	t.Logf("已注册的工具: %v", registeredTools)

	if len(registeredTools) == 0 {
		t.Fatal("没有工具被注册")
	}

	if registeredTools[0] != "skill" {
		t.Fatalf("预期工具名为 skill，实际为 %s", registeredTools[0])
	}

	t.Log("Skill 工具注册成功!")
}

func TestWithSkillRegistryOption(t *testing.T) {
	t.Log("\n=== WithSkillRegistry Option 功能测试 ===\n")

	skillDir := "../../data/skills"
	loader := skill.NewSkillLoader(skillDir)
	registry := skill.NewSkillRegistryWithLoader(loader)

	if err := registry.LoadAndRegister(); err != nil {
		t.Fatalf("注册Skill失败: %v", err)
	}

	option := agent.WithSkillRegistry(registry, ".")
	t.Logf("Option 创建成功: %v", option != nil)

	t.Log("WithSkillRegistry Option 功能验证通过!")
}
