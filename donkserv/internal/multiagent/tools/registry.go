// tools 工具注册模块
// 提供动态工具注册和管理功能
package tools

import (
	"encoding/json"
	"fmt"
	"sync"

	"github.com/longstageai/donk/donk/internal/multiagent/types"
)

// Handler 工具处理函数类型
type Handler func(params map[string]interface{}) (map[string]interface{}, error)

// Tool 注册的工具信息
type Tool struct {
	Name        string                 `json:"name"`        // 工具名称
	Description string                 `json:"description"` // 工具描述
	Parameters  map[string]interface{} `json:"parameters"`  // 参数定义(JSON Schema)
	Handler     Handler                `json:"-"`           // 处理函数
}

// Definition 工具定义(用于LLM)
type Definition struct {
	Type     string `json:"type"` // 类型，通常为"function"
	Function struct {
		Name        string                 `json:"name"`        // 函数名
		Description string                 `json:"description"` // 函数描述
		Parameters  map[string]interface{} `json:"parameters"`  // 参数定义
	} `json:"function"`
}

// Registry 工具注册表
type Registry struct {
	mu    sync.RWMutex
	tools map[string]*Tool
}

// NewRegistry 创建工具注册表
func NewRegistry() *Registry {
	return &Registry{
		tools: make(map[string]*Tool),
	}
}

// Register 注册工具
func (r *Registry) Register(name, description string, parameters map[string]interface{}, handler Handler) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.tools[name]; exists {
		return fmt.Errorf("工具 '%s' 已存在", name)
	}

	r.tools[name] = &Tool{
		Name:        name,
		Description: description,
		Parameters:  parameters,
		Handler:     handler,
	}

	return nil
}

// Unregister 注销工具
func (r *Registry) Unregister(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.tools[name]; !exists {
		return fmt.Errorf("工具 '%s' 不存在", name)
	}

	delete(r.tools, name)
	return nil
}

// Get 获取工具
func (r *Registry) Get(name string) (*Tool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	tool, exists := r.tools[name]
	if !exists {
		return nil, fmt.Errorf("工具 '%s' 不存在", name)
	}

	return tool, nil
}

// GetAllDefinitions 获取所有工具定义(用于LLM)
func (r *Registry) GetAllDefinitions() []Definition {
	r.mu.RLock()
	defer r.mu.RUnlock()

	definitions := make([]Definition, 0, len(r.tools))
	for _, tool := range r.tools {
		def := Definition{
			Type: "function",
		}
		def.Function.Name = tool.Name
		def.Function.Description = tool.Description
		def.Function.Parameters = tool.Parameters
		definitions = append(definitions, def)
	}

	return definitions
}

// GetAllToolDefinitions 获取所有工具定义(返回types.ToolDefinition格式)
func (r *Registry) GetAllToolDefinitions() []types.ToolDefinition {
	r.mu.RLock()
	defer r.mu.RUnlock()

	definitions := make([]types.ToolDefinition, 0, len(r.tools))
	for _, tool := range r.tools {
		def := types.ToolDefinition{
			Type: "function",
			Function: types.FunctionInfo{
				Name:        tool.Name,
				Description: tool.Description,
				Parameters:  tool.Parameters,
			},
		}
		definitions = append(definitions, def)
	}

	return definitions
}

// Execute 执行工具
func (r *Registry) Execute(name string, arguments string) (map[string]interface{}, error) {
	r.mu.RLock()
	tool, exists := r.tools[name]
	r.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("工具 '%s' 不存在", name)
	}

	// 解析参数
	var params map[string]interface{}
	if err := json.Unmarshal([]byte(arguments), &params); err != nil {
		return nil, fmt.Errorf("解析参数失败: %w", err)
	}

	// 执行工具
	return tool.Handler(params)
}

// List 列出所有工具名称
func (r *Registry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.tools))
	for name := range r.tools {
		names = append(names, name)
	}

	return names
}

// Count 获取工具数量
func (r *Registry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.tools)
}

// CreateStandardRegistry 创建标准工具注册表
// 包含常用的内置工具
func CreateStandardRegistry() *Registry {
	registry := NewRegistry()
	//
	//// 注册获取用户画像工具
	//registry.Register(
	//	"get_user_profile",
	//	"获取用户基本信息和偏好，如果用户不存在则返回found=false",
	//	map[string]interface{}{
	//		"type": "object",
	//		"properties": map[string]interface{}{
	//			"user_id": map[string]interface{}{
	//				"type":        "string",
	//				"description": "用户ID",
	//			},
	//		},
	//		"required": []string{"user_id"},
	//	},
	//	func(params map[string]interface{}) (map[string]interface{}, error) {
	//		userID := params["user_id"].(string)
	//
	//		// TODO: 实际应该查询数据库/用户服务
	//		// 这里模拟：只有特定用户ID才返回画像
	//		if userID == "" || userID == "unknown" || userID == "guest" {
	//			return map[string]interface{}{
	//				"found":   false,
	//				"message": "未找到该用户的画像信息",
	//			}, nil
	//		}
	//
	//		// 模拟返回用户画像（实际应从数据库查询）
	//		return map[string]interface{}{
	//			"found":      true,
	//			"user_id":    userID,
	//			"name":       "小明",
	//			"gender":     "male",
	//			"age":        25,
	//			"occupation": "工程师",
	//			"hobbies":    []string{"读书", "旅游", "摄影"},
	//			"preferences": map[string]string{
	//				"color": "蓝色",
	//				"style": "简约",
	//			},
	//		}, nil
	//	},
	//)
	//
	//// 注册获取星座工具
	//registry.Register(
	//	"get_zodiac",
	//	"根据日期计算星座",
	//	map[string]interface{}{
	//		"type": "object",
	//		"properties": map[string]interface{}{
	//			"month": map[string]interface{}{
	//				"type":        "integer",
	//				"description": "月份(1-12)",
	//			},
	//			"day": map[string]interface{}{
	//				"type":        "integer",
	//				"description": "日期(1-31)",
	//			},
	//		},
	//		"required": []string{"month", "day"},
	//	},
	//	func(params map[string]interface{}) (map[string]interface{}, error) {
	//		month := int(params["month"].(float64))
	//		day := int(params["day"].(float64))
	//		zodiac := getZodiac(month, day)
	//		return map[string]interface{}{
	//			"zodiac": zodiac,
	//		}, nil
	//	},
	//)
	//
	//// 注册生成祝福语工具
	//registry.Register(
	//	"generate_blessing",
	//	"生成个性化祝福语",
	//	map[string]interface{}{
	//		"type": "object",
	//		"properties": map[string]interface{}{
	//			"occasion": map[string]interface{}{
	//				"type":        "string",
	//				"description": "场合，如birthday/festival/morning",
	//			},
	//			"style": map[string]interface{}{
	//				"type":        "string",
	//				"description": "风格，如warm/funny/serious",
	//			},
	//		},
	//		"required": []string{"occasion"},
	//	},
	//	func(params map[string]interface{}) (map[string]interface{}, error) {
	//		occasion := params["occasion"].(string)
	//		blessing := generateBlessing(occasion)
	//		return map[string]interface{}{
	//			"blessing": blessing,
	//		}, nil
	//	},
	//)
	//
	//// 注册搜索内容工具
	//registry.Register(
	//	"search_content",
	//	"搜索相关内容",
	//	map[string]interface{}{
	//		"type": "object",
	//		"properties": map[string]interface{}{
	//			"query": map[string]interface{}{
	//				"type":        "string",
	//				"description": "搜索关键词",
	//			},
	//			"limit": map[string]interface{}{
	//				"type":        "integer",
	//				"description": "返回结果数量限制",
	//			},
	//		},
	//		"required": []string{"query"},
	//	},
	//	func(params map[string]interface{}) (map[string]interface{}, error) {
	//		query := params["query"].(string)
	//		return map[string]interface{}{
	//			"results": []string{
	//				"关于 " + query + " 的搜索结果1",
	//				"关于 " + query + " 的搜索结果2",
	//			},
	//		}, nil
	//	},
	//)

	return registry
}

// getZodiac 根据月份和日期获取星座
func getZodiac(month, day int) string {
	zodiacSigns := []string{
		"摩羯座", "水瓶座", "双鱼座", "白羊座",
		"金牛座", "双子座", "巨蟹座", "狮子座",
		"处女座", "天秤座", "天蝎座", "射手座", "摩羯座",
	}
	cutoffDates := []int{20, 19, 21, 20, 21, 22, 23, 23, 23, 24, 23, 22}

	if day < cutoffDates[month-1] {
		return zodiacSigns[month-1]
	}
	return zodiacSigns[month]
}

// generateBlessing 生成祝福语
func generateBlessing(occasion string) string {
	blessings := map[string]string{
		"birthday": "祝你生日快乐，愿你的每一天都充满阳光和欢笑！",
		"festival": "节日快乐！愿你在这个特别的日子里收获满满的幸福！",
		"morning":  "早安！愿你今天充满活力，一切顺利！",
		"night":    "晚安！愿你有个甜美的梦，明天见！",
	}

	if blessing, ok := blessings[occasion]; ok {
		return blessing
	}
	return "祝你一切顺利，天天开心！"
}
