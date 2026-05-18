package setting

import "errors"

// Service 业务逻辑层结构体
// 封装了存储层的操作，提供业务逻辑处理和参数验证
type Service struct {
	storage *Storage
}

// NewService 创建业务逻辑层实例
func NewService(storage *Storage) *Service {
	return &Service{storage: storage}
}

// GetConfig 获取完整配置
func (s *Service) GetConfig() (*Config, error) {
	return s.storage.GetConfig()
}

// UpdateConfig 更新完整配置
func (s *Service) UpdateConfig(req *ConfigRequest) error {
	if req == nil {
		return errors.New("请求参数不能为空")
	}
	return s.storage.UpdateConfig(req)
}

// UpdateConfigPartial 部分更新配置（只更新传入的字段）
func (s *Service) UpdateConfigPartial(req *ConfigUpdateRequest) error {
	if req == nil {
		return errors.New("请求参数不能为空")
	}
	return s.storage.UpdateConfigPartial(req)
}

// GetLLMConfig 获取LLM配置
func (s *Service) GetLLMConfig() (*LLMConfigRequest, error) {
	return s.storage.GetLLMConfig()
}

// UpdateLLMConfig 更新LLM配置
func (s *Service) UpdateLLMConfig(req *LLMConfigRequest) error {
	if req == nil {
		return errors.New("请求参数不能为空")
	}
	if req.Provider == "" {
		return errors.New("provider 不能为空")
	}
	if req.Model == "" {
		return errors.New("model 不能为空")
	}
	return s.storage.UpdateLLMConfig(req)
}

// GetEmbeddingConfig 获取Embedding配置
func (s *Service) GetEmbeddingConfig() (*EmbeddingConfigRequest, error) {
	return s.storage.GetEmbeddingConfig()
}

// UpdateEmbeddingConfig 更新Embedding配置
func (s *Service) UpdateEmbeddingConfig(req *EmbeddingConfigRequest) error {
	if req == nil {
		return errors.New("请求参数不能为空")
	}
	if req.Provider == "" {
		return errors.New("provider 不能为空")
	}
	if req.Model == "" {
		return errors.New("model 不能为空")
	}
	return s.storage.UpdateEmbeddingConfig(req)
}

// GetAgentConfig 获取Agent配置
func (s *Service) GetAgentConfig() (*AgentConfigRequest, error) {
	return s.storage.GetAgentConfig()
}

// UpdateAgentConfig 更新Agent配置
func (s *Service) UpdateAgentConfig(req *AgentConfigRequest) error {
	if req == nil {
		return errors.New("请求参数不能为空")
	}
	if req.Name == "" {
		return errors.New("name 不能为空")
	}
	return s.storage.UpdateAgentConfig(req)
}

// GetKnowledgeConfig 获取知识库配置
func (s *Service) GetKnowledgeConfig() (*KnowledgeConfigRequest, error) {
	return s.storage.GetKnowledgeConfig()
}

// UpdateKnowledgeConfig 更新知识库配置
func (s *Service) UpdateKnowledgeConfig(req *KnowledgeConfigRequest) error {
	if req == nil {
		return errors.New("请求参数不能为空")
	}
	return s.storage.UpdateKnowledgeConfig(req)
}
