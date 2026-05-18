package skill_test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/longstageai/donk/donk/internal/skill"
	"github.com/longstageai/donk/donk/internal/tool"
	"github.com/longstageai/donk/donk/internal/tool/builtin"
)

func TestSkillExample(t *testing.T) {
	t.Log("=== Skill 系统使用示例 ===\n")

	skillDir := "../../data/skills"

	loader := skill.NewSkillLoader(skillDir)
	t.Logf("1. 创建 SkillLoader，目录: %s\n", skillDir)

	skills, err := loader.Load()
	if err != nil {
		t.Fatalf("加载Skill失败: %v", err)
	}
	t.Logf("2. 加载了 %d 个Skill\n", len(skills))

	for _, s := range skills {
		t.Logf("\n--- Skill: %s ---", s.Name())
		t.Logf("描述: %s", s.Description())
		t.Logf("版本: %s", s.Version())
		t.Logf("作者: %s", s.Author())
		t.Logf("用户可调用: %v", s.IsUserInvocable())
		t.Logf("禁用自动触发: %v", s.DisableModelInvocation())
		t.Logf("标签: %v", s.Tags())
		t.Logf("允许工具: %v", s.AllowedTools())

		if s.License() != "" {
			t.Logf("许可证: %s", s.License())
		}
		if s.Compatibility() != "" {
			t.Logf("环境要求: %s", s.Compatibility())
		}

		if len(s.Examples()) > 0 {
			t.Logf("示例: %v", s.Examples())
		}

		if len(s.Requires()) > 0 {
			t.Logf("依赖: %v", s.Requires())
		}

		t.Logf("基础目录: %s", s.BaseDir())
		t.Logf("加载时间: %s", s.Loaded())

		if s.HasScripts() {
			scripts, _ := s.ListScripts()
			t.Logf("Scripts目录文件: %v", scripts)
		}
		if s.HasReferences() {
			refs, _ := s.ListReferences()
			t.Logf("References目录文件: %v", refs)
		}
		if s.HasAssets() {
			assets, _ := s.ListAssets()
			t.Logf("Assets目录文件: %v", assets)
		}
	}

	registry := skill.NewSkillRegistryWithLoader(loader)
	if err := registry.LoadAndRegister(); err != nil {
		t.Fatalf("注册Skill失败: %v", err)
	}
	t.Logf("\n3. 注册了 %d 个Skill到Registry\n", registry.Count())

	userSkills := registry.GetUserInvocableSkills()
	t.Logf("4. 用户可调用的Skill: %d 个\n", len(userSkills))
	for _, s := range userSkills {
		t.Logf("   - %s", s.Name())
	}

	autoSkills := registry.GetAutoInvocableSkills()
	t.Logf("5. 可自动触发的Skill: %d 个\n", len(autoSkills))

	converter := skill.NewSkillConverter()
	metadataPrompt := converter.GenerateSystemPromptMetadata(skills)
	t.Logf("\n6. 生成的元数据提示词:\n%s\n", metadataPrompt)

	fullPrompt := converter.GenerateFullSystemPrompt(skills)
	t.Logf("7. 生成的完整提示词:\n%s\n", fullPrompt)
}

func TestSkillProgressiveLoading(t *testing.T) {
	t.Log("\n=== 渐进式加载示例 ===\n")

	skillDir := "../../data/skills"
	loader := skill.NewSkillLoader(skillDir)

	skills, err := loader.Load()
	if err != nil {
		t.Fatalf("加载Skill失败: %v", err)
	}

	converter := skill.NewSkillConverter()

	t.Log("Level 1: 发现阶段 - 只加载元数据")
	metadata := converter.GenerateSystemPromptMetadata(skills)
	t.Logf("元数据 (约 %d tokens):\n%s\n", len(metadata)/4, metadata)

	if len(skills) > 0 {
		s := skills[0]

		t.Log("Level 2: 激活阶段 - 加载完整指令")
		instructions := converter.GetSkillInstructionsForAgent(s, true)
		t.Logf("完整指令 (约 %d tokens):\n%s\n", len(instructions)/4, instructions)

		if s.HasReferences() {
			t.Log("Level 3: 资源阶段 - 按需加载参考资料")
			refs, _ := s.ListReferences()
			t.Logf("参考资料: %v\n", refs)
		}
	}
}

func TestSkillFindByTag(t *testing.T) {
	t.Log("\n=== 按标签查找示例 ===\n")

	skillDir := "../../data/skills"
	loader := skill.NewSkillLoader(skillDir)
	registry := skill.NewSkillRegistryWithLoader(loader)

	if err := registry.LoadAndRegister(); err != nil {
		t.Fatalf("注册Skill失败: %v", err)
	}

	allSkills := registry.List()
	for _, s := range allSkills {
		tags := s.Tags()
		if len(tags) > 0 {
			t.Logf("Skill '%s' 的标签: %v", s.Name(), tags)
		}
	}
}

func TestSkillFileOperations(t *testing.T) {
	t.Log("\n=== 文件操作示例 ===\n")

	tmpDir := t.TempDir()
	skillFile := filepath.Join(tmpDir, "SKILL.md")

	content := `---
name: test-skill
description: 这是一个测试技能，用于验证文件解析功能
version: 1.0.0
author: test
tags:
  - test
  - demo
user-invocable: true
allowed-tools:
  - echo
license: MIT
compatibility: go1.21+
metadata:
  category: utility
  difficulty: easy
requires:
  - hello
examples:
  - "说你好"
  - "打个招呼"
---

# Test Skill

这是一个测试技能。

## 何时使用

当需要测试时使用此技能。

## 指令

1. 测试步骤1
2. 测试步骤2
`

	if err := os.WriteFile(skillFile, []byte(content), 0644); err != nil {
		t.Fatalf("写入测试文件失败: %v", err)
	}
	t.Logf("创建测试文件: %s\n", skillFile)

	parser := skill.NewSkillParser()
	s, err := parser.ParseFile(skillFile)
	if err != nil {
		t.Fatalf("解析Skill失败: %v", err)
	}

	t.Logf("解析结果:")
	t.Logf("  名称: %s", s.Name())
	t.Logf("  描述: %s", s.Description())
	t.Logf("  版本: %s", s.Version())
	t.Logf("  作者: %s", s.Author())
	t.Logf("  标签: %v", s.Tags())
	t.Logf("  用户可调用: %v", s.IsUserInvocable())
	t.Logf("  允许工具: %v", s.AllowedTools())
	t.Logf("  许可证: %s", s.License())
	t.Logf("  环境要求: %s", s.Compatibility())
	t.Logf("  自定义元数据: %v", s.CustomMetadata())
	t.Logf("  依赖: %v", s.Requires())
	t.Logf("  示例: %v", s.Examples())
	t.Logf("  指令: %s", s.Instructions())

	dir := filepath.Dir(skillFile)
	subDirs := []string{"scripts", "references", "assets"}
	for _, subDir := range subDirs {
		subPath := filepath.Join(dir, subDir)
		if err := os.MkdirAll(subPath, 0755); err != nil {
			t.Fatalf("创建子目录失败: %v", err)
		}
	}

	t.Logf("\n创建子目录后:")
	t.Logf("  HasScripts: %v", s.HasScripts())
	t.Logf("  HasReferences: %v", s.HasReferences())
	t.Logf("  HasAssets: %v", s.HasAssets())

	registry := skill.NewSkillRegistry()
	if err := registry.Register(s); err != nil {
		t.Fatalf("注册Skill失败: %v", err)
	}
	t.Logf("\n注册成功，Registry中共有 %d 个Skill", registry.Count())
}

func TestSkillConverter(t *testing.T) {
	t.Log("\n=== Converter 使用示例 ===\n")

	s := skill.NewSkill("converter-test", "测试Converter功能")

	converter := skill.NewSkillConverter()

	component := converter.ToPromptComponent(s)
	t.Logf("提示词组件名称: %s", component.Name())
	t.Logf("提示词组件优先级: %d", component.Priority())

	content, err := component.Content(nil)
	if err != nil {
		t.Fatalf("获取组件内容失败: %v", err)
	}
	t.Logf("组件内容:\n%s\n", content)

	summary := converter.ToSkillSummary(s)
	t.Logf("Skill摘要: %v\n", summary)

	allowedTools := converter.GetAllowedTools(s)
	t.Logf("允许的工具: %v\n", allowedTools)
}

func TestSkillLoaderAdvanced(t *testing.T) {
	t.Log("\n=== Loader 高级功能示例 ===\n")

	skillDir := "../../data/skills"
	loader := skill.NewSkillLoader(skillDir)

	names, err := loader.ListSkillNames()
	if err != nil {
		t.Fatalf("列出Skill名称失败: %v", err)
	}
	t.Logf("所有Skill名称: %v\n", names)

	userSkills, err := loader.FindUserInvocableSkills()
	if err != nil {
		t.Fatalf("查找用户可调用Skill失败: %v", err)
	}
	t.Logf("用户可调用的Skill: %v\n", userSkills)

	autoSkills, err := loader.FindAutoInvocableSkills()
	if err != nil {
		t.Fatalf("查找自动可调用Skill失败: %v", err)
	}
	t.Logf("自动可调用的Skill: %v\n", autoSkills)

	loadedSkills, err := loader.Reload()
	if err != nil {
		t.Fatalf("重新加载失败: %v", err)
	}
	t.Logf("重新加载了 %d 个Skill\n", len(loadedSkills))

	for _, s := range loadedSkills {
		fmt.Printf("Skill: %s\n", s.Name())
	}
}

func TestExecutorBasic(t *testing.T) {
	t.Log("\n=== Executor 基本使用示例 ===\n")

	skillDir := "../../data/skills"
	loader := skill.NewSkillLoader(skillDir)
	registry := skill.NewSkillRegistryWithLoader(loader)

	if err := registry.LoadAndRegister(); err != nil {
		t.Fatalf("注册Skill失败: %v", err)
	}

	exec := skill.NewExecutor(registry)
	t.Logf("1. 创建Executor\n")

	availableSkills := exec.ListAvailableSkills()
	t.Logf("2. 可用的Skill: %d 个\n", len(availableSkills))
	for _, s := range availableSkills {
		t.Logf("   - %s: %s", s.Name(), s.Description())
	}

	userSkills := exec.GetUserInvocableSkills()
	t.Logf("3. 用户可调用的Skill: %d 个\n", len(userSkills))
}

func TestExecutorActivate(t *testing.T) {
	t.Log("\n=== Executor 激活Skill ===\n")

	skillDir := "../../data/skills"
	loader := skill.NewSkillLoader(skillDir)
	registry := skill.NewSkillRegistryWithLoader(loader)

	if err := registry.LoadAndRegister(); err != nil {
		t.Fatalf("注册Skill失败: %v", err)
	}

	exec := skill.NewExecutor(registry)

	ctx, err := exec.Activate("hello")
	if err != nil {
		t.Fatalf("激活Skill失败: %v", err)
	}
	t.Logf("激活成功: %s", ctx.Skill.Name())
	t.Logf("描述: %s", ctx.Skill.Description())
	t.Logf("依赖: %v", ctx.Skill.Requires())

	instructions, err := exec.GetInstructions("hello")
	if err != nil {
		t.Fatalf("获取指令失败: %v", err)
	}
	t.Logf("指令:\n%s\n", instructions)
}

func TestExecutorReferences(t *testing.T) {
	t.Log("\n=== Executor 加载参考资料 ===\n")

	skillDir := "../../data/skills"
	loader := skill.NewSkillLoader(skillDir)
	registry := skill.NewSkillRegistryWithLoader(loader)

	if err := registry.LoadAndRegister(); err != nil {
		t.Fatalf("注册Skill失败: %v", err)
	}

	exec := skill.NewExecutor(registry)

	ctx, err := exec.Activate("calculator")
	if err != nil {
		t.Fatalf("激活Skill失败: %v", err)
	}

	refs, err := exec.LoadReferences(ctx)
	if err != nil {
		t.Fatalf("加载参考资料失败: %v", err)
	}

	if refs != nil {
		t.Logf("加载了 %d 个参考资料:", len(refs))
		for name, content := range refs {
			t.Logf("   - %s (%d 字符)", name, len(content))
		}
	} else {
		t.Log("无参考资料")
	}
}

func TestExecutorScripts(t *testing.T) {
	t.Log("\n=== Executor 列出脚本 ===\n")

	skillDir := "../../data/skills"
	loader := skill.NewSkillLoader(skillDir)
	registry := skill.NewSkillRegistryWithLoader(loader)

	if err := registry.LoadAndRegister(); err != nil {
		t.Fatalf("注册Skill失败: %v", err)
	}

	exec := skill.NewExecutor(registry)

	ctx, err := exec.Activate("hello")
	if err != nil {
		t.Fatalf("激活Skill失败: %v", err)
	}

	if ctx.Skill.HasScripts() {
		scripts, err := ctx.Skill.ListScripts()
		if err != nil {
			t.Fatalf("列出脚本失败: %v", err)
		}
		t.Logf("Skill '%s' 有 %d 个脚本:", ctx.Skill.Name(), len(scripts))
		for _, script := range scripts {
			t.Logf("   - %s", filepath.Base(script))
		}
	} else {
		t.Logf("Skill '%s' 没有scripts目录", ctx.Skill.Name())
	}
}

func TestExecutorAssets(t *testing.T) {
	t.Log("\n=== Executor 加载资源文件 ===\n")

	skillDir := "../../data/skills"
	loader := skill.NewSkillLoader(skillDir)
	registry := skill.NewSkillRegistryWithLoader(loader)

	if err := registry.LoadAndRegister(); err != nil {
		t.Fatalf("注册Skill失败: %v", err)
	}

	exec := skill.NewExecutor(registry)

	ctx, err := exec.Activate("file-manager")
	if err != nil {
		t.Fatalf("激活Skill失败: %v", err)
	}

	if ctx.Skill.HasAssets() {
		assets, err := ctx.Skill.ListAssets()
		if err != nil {
			t.Fatalf("列出资源失败: %v", err)
		}
		t.Logf("Skill '%s' 有 %d 个资源:", ctx.Skill.Name(), len(assets))
		for _, asset := range assets {
			t.Logf("   - %s", filepath.Base(asset))
		}

		data, err := exec.LoadAsset(ctx, "config.json")
		if err != nil {
			t.Logf("加载资源失败: %v", err)
		} else {
			t.Logf("加载的资源内容: %s", string(data))
		}
	} else {
		t.Logf("Skill '%s' 没有assets目录", ctx.Skill.Name())
	}
}

func TestExecutorExecuteScript(t *testing.T) {
	t.Log("\n=== Executor 执行脚本 ===\n")

	skillDir := "../../data/skills"
	loader := skill.NewSkillLoader(skillDir)
	registry := skill.NewSkillRegistryWithLoader(loader)

	if err := registry.LoadAndRegister(); err != nil {
		t.Fatalf("注册Skill失败: %v", err)
	}

	exec := skill.NewExecutor(registry)

	ctx, err := exec.Activate("hello")
	if err != nil {
		t.Fatalf("激活Skill失败: %v", err)
	}

	scripts, err := ctx.Skill.ListScripts()
	if err != nil {
		t.Fatalf("列出脚本失败: %v", err)
	}

	if len(scripts) == 0 {
		t.Skip("Skill没有脚本，跳过测试")
	}

	output, err := exec.ExecuteScript(ctx, "echo_test.py", "Hello", "World")
	if err != nil {
		t.Fatalf("执行脚本失败: %v", err)
	}

	t.Logf("脚本输出: %s", output)

	for _, log := range ctx.Logs {
		t.Logf("日志: %s", log)
	}
}

func TestSkillToolBasic(t *testing.T) {
	t.Log("\n=== SkillTool 基本使用示例 ===\n")

	skillDir := "../../data/skills"
	loader := skill.NewSkillLoader(skillDir)
	registry := skill.NewSkillRegistryWithLoader(loader)

	if err := registry.LoadAndRegister(); err != nil {
		t.Fatalf("注册Skill失败: %v", err)
	}

	exec := skill.NewExecutor(registry)
	skillTool := builtin.NewSkillTool(registry, exec, ".")

	t.Logf("工具名称: %s", skillTool.Name())
	t.Logf("工具类别: %s", skillTool.Category())
	t.Logf("工具版本: %s", skillTool.Version())
	t.Logf("描述: %s", skillTool.Description())

	params := skillTool.Parameters()
	t.Logf("参数类型: %s", params.Type)
	t.Logf("必填参数: %v", params.Required)
}

func TestSkillToolInfo(t *testing.T) {
	t.Log("\n=== SkillTool Info 操作 ===\n")

	skillDir := "../../data/skills"
	loader := skill.NewSkillLoader(skillDir)
	registry := skill.NewSkillRegistryWithLoader(loader)

	if err := registry.LoadAndRegister(); err != nil {
		t.Fatalf("注册Skill失败: %v", err)
	}

	exec := skill.NewExecutor(registry)
	skillTool := builtin.NewSkillTool(registry, exec, ".")

	ctx := tool.NewContext("skill", map[string]any{
		"skill_name": "hello",
		"action":     "info",
	})

	result, err := skillTool.Execute(ctx)
	if err != nil {
		t.Fatalf("执行失败: %v", err)
	}

	if result.Success {
		t.Logf("Skill 信息:\n%s", result.Data)
	} else {
		t.Fatalf("执行失败: %v", result.Error)
	}
}

func TestSkillToolLoadInstructions(t *testing.T) {
	t.Log("\n=== SkillTool 加载指令 ===\n")

	skillDir := "../../data/skills"
	loader := skill.NewSkillLoader(skillDir)
	registry := skill.NewSkillRegistryWithLoader(loader)

	if err := registry.LoadAndRegister(); err != nil {
		t.Fatalf("注册Skill失败: %v", err)
	}

	exec := skill.NewExecutor(registry)
	skillTool := builtin.NewSkillTool(registry, exec, ".")

	ctx := tool.NewContext("skill", map[string]any{
		"skill_name": "hello",
		"action":     "load_instructions",
	})

	result, err := skillTool.Execute(ctx)
	if err != nil {
		t.Fatalf("执行失败: %v", err)
	}

	if result.Success {
		instructions := result.Data.(string)
		t.Logf("加载的指令长度: %d 字符", len(instructions))
		t.Logf("指令前100字符: %s", instructions[:min(100, len(instructions))])
	} else {
		t.Fatalf("执行失败: %v", result.Error)
	}
}

func TestSkillToolLoadReferences(t *testing.T) {
	t.Log("\n=== SkillTool 加载参考资料 ===\n")

	skillDir := "../../data/skills"
	loader := skill.NewSkillLoader(skillDir)
	registry := skill.NewSkillRegistryWithLoader(loader)

	if err := registry.LoadAndRegister(); err != nil {
		t.Fatalf("注册Skill失败: %v", err)
	}

	exec := skill.NewExecutor(registry)
	skillTool := builtin.NewSkillTool(registry, exec, ".")

	ctx := tool.NewContext("skill", map[string]any{
		"skill_name": "calculator",
		"action":     "load_references",
	})

	result, err := skillTool.Execute(ctx)
	if err != nil {
		t.Fatalf("执行失败: %v", err)
	}

	if result.Success {
		t.Logf("参考资料加载成功: %s", result.Data)
	} else {
		t.Logf("执行结果: 成功=%v, 错误=%v", result.Success, result.Error)
	}
}

func TestSkillToolExecuteScript(t *testing.T) {
	t.Log("\n=== SkillTool 执行脚本 ===\n")

	skillDir := "../../data/skills"
	loader := skill.NewSkillLoader(skillDir)
	registry := skill.NewSkillRegistryWithLoader(loader)

	if err := registry.LoadAndRegister(); err != nil {
		t.Fatalf("注册Skill失败: %v", err)
	}

	exec := skill.NewExecutor(registry)
	skillTool := builtin.NewSkillTool(registry, exec, ".")

	ctx := tool.NewContext("skill", map[string]any{
		"skill_name": "hello",
		"action":     "execute_script",
		"params": map[string]any{
			"script": "echo_test.py",
			"args":   []string{"Hello", "World"},
		},
	})

	result, err := skillTool.Execute(ctx)
	if err != nil {
		t.Fatalf("执行失败: %v", err)
	}

	if result.Success {
		t.Logf("脚本执行成功!")
		t.Logf("输出: %s", result.Data)
		t.Logf("执行时间: %v", result.ExecutionTime)
	} else {
		t.Fatalf("执行失败: %v", result.Error)
	}
}

func TestSkillToolExecuteScriptWithJSONArgs(t *testing.T) {
	t.Log("\n=== SkillTool 执行脚本（JSON参数）===\n")

	skillDir := "../../data/skills"
	loader := skill.NewSkillLoader(skillDir)
	registry := skill.NewSkillRegistryWithLoader(loader)

	if err := registry.LoadAndRegister(); err != nil {
		t.Fatalf("注册Skill失败: %v", err)
	}

	exec := skill.NewExecutor(registry)
	skillTool := builtin.NewSkillTool(registry, exec, ".")

	ctx := tool.NewContext("skill", map[string]any{
		"skill_name": "hello",
		"action":     "execute_script",
		"params": map[string]any{
			"script": "echo_test.py",
			"args": map[string]any{
				"message": "张三",
			},
		},
	})

	result, err := skillTool.Execute(ctx)
	if err != nil {
		t.Fatalf("执行失败: %v", err)
	}

	if result.Success {
		t.Logf("脚本执行成功!")
		t.Logf("输出: %s", result.Data)
		t.Logf("执行时间: %v", result.ExecutionTime)
	} else {
		t.Fatalf("执行失败: %v", result.Error)
	}
}
