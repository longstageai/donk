package agent

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/longstageai/donk/donk/internal/skill"
	"github.com/longstageai/donk/donk/pkg/logger"
)

// buildSystemMessage 构建带上下文的系统提示词
// 在系统提示词末尾添加当前上下文信息（时间、时区、工作空间、操作系统）
func (a *Agent) buildSystemMessage() string {
	var sb strings.Builder

	// 添加基础系统提示词（如果存在）
	if a.systemPrompt != "" {
		sb.WriteString(a.systemPrompt)
	}

	// 添加用户画像（如果存在）
	if a.profileManager != nil {
		profileContent := a.profileManager.GetProfilePrompt()
		if profileContent != "" {
			sb.WriteString("\n\n## 用户画像\n\n")
			sb.WriteString(profileContent)
			sb.WriteString("\n")
		}
	}

	// 添加Skill指令（每次对话从数据库重新加载启用的技能）
	skills := a.loadEnabledSkills()
	if len(skills) > 0 {
		sb.WriteString("\n\n## 可用技能\n\n")
		sb.WriteString("你可以通过调用 skill 工具来使用以下技能：\n\n")
		for _, s := range skills {
			sb.WriteString(fmt.Sprintf("### %s\n", s.Name()))
			sb.WriteString(fmt.Sprintf("%s\n\n", s.Description()))
			sb.WriteString(fmt.Sprintf("使用示例：\n```json\n{\"skill_name\": \"%s\", \"action\": \"load_instructions\"}\n```\n\n", s.Name()))
		}
	}

	// 添加当前上下文
	contextInfo := fmt.Sprintf("\n\n- **时间**: %s\n- **时区**: %s\n- **工作空间**: %s\n- **操作系统**: %s\n",
		time.Now().Format("2006-01-02 15:04:05"),
		time.Now().Location().String(),
		a.workspace,
		getOSInfo(),
	)
	sb.WriteString(contextInfo)

	return sb.String()
}

// getOSInfo 获取操作系统详细信息
// 返回格式：操作系统类型 (主机名, CPU核心数)
func getOSInfo() string {
	os := runtime.GOOS
	arch := runtime.GOARCH

	info := fmt.Sprintf("%s/%s", os, arch)

	switch os {
	case "windows":
		info = getWindowsInfo()
	case "linux":
		info = getLinuxInfo()
	case "darwin":
		info = fmt.Sprintf("macOS %s", arch)
	}

	return info
}

// getWindowsInfo 获取 Windows 详细信息
func getWindowsInfo() string {
	hostname, _ := getHostname()
	info := fmt.Sprintf("Windows (Host: %s)", hostname)
	return info
}

// getLinuxInfo 获取 Linux 详细信息
func getLinuxInfo() string {
	hostname, _ := getHostname()
	info := fmt.Sprintf("Linux (Host: %s)", hostname)
	return info
}

// getHostname 获取主机名
func getHostname() (string, error) {
	return "localhost", nil
}

// loadEnabledSkills 从数据库加载启用的技能
// 每次对话时调用，确保获取最新的技能状态
// 返回:
//   - []*skill.Skill: 启用的技能列表
func (a *Agent) loadEnabledSkills() []*skill.Skill {
	// 如果没有数据库连接或技能目录，使用已有的 skillRegistry
	if a.db == nil || a.skillDir == "" {
		if a.skillRegistry != nil {
			return a.skillRegistry.List()
		}
		return nil
	}

	// 检查技能目录是否存在
	if _, err := os.Stat(a.skillDir); os.IsNotExist(err) {
		return nil
	}

	// 从数据库获取启用的技能列表
	stateRepo := skill.NewStateRepository(a.db)
	enabledStates, err := stateRepo.GetEnabled()
	if err != nil {
		logger.Warn("从数据库获取启用的Skill失败", map[string]interface{}{"error": err.Error()})
		// 数据库查询失败时，回退到使用已有的 skillRegistry
		if a.skillRegistry != nil {
			return a.skillRegistry.List()
		}
		return nil
	}

	// 加载启用的技能
	loader := skill.NewSkillLoader(a.skillDir)
	var skills []*skill.Skill
	for _, state := range enabledStates {
		skillPath := filepath.Join(a.skillDir, state.Name)
		s, err := loader.LoadFromDir(skillPath)
		if err != nil {
			logger.Warn("加载Skill失败", map[string]interface{}{"skill": state.Name, "error": err.Error()})
			continue
		}
		skills = append(skills, s)
	}

	logger.Debug("动态加载启用的Skill", map[string]interface{}{"count": len(skills)})
	return skills
}
