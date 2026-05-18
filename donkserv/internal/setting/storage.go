package setting

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

// Storage 存储层结构体
// 封装了数据库操作，提供配置数据的 CRUD 功能
type Storage struct {
	db *sql.DB
}

// NewStorage 创建存储层实例
func NewStorage(db *sql.DB) *Storage {
	return &Storage{db: db}
}

// GetConfig 获取完整配置
// 从 config 表获取第一条记录
func (s *Storage) GetConfig() (*Config, error) {
	query := `
	SELECT id, llm_provider, llm_model, llm_api_key, llm_base_url, llm_temperature, llm_max_tokens,
	       embedding_provider, embedding_model, embedding_api_key, embedding_base_url, embedding_dimension,
	       agent_name, agent_max_loop, agent_converge_after, agent_timeout, agent_daily_token_limit,
	       agent_history_max_entries, agent_history_max_days, knowledge_enabled,
	       created_at, updated_at
	FROM config
	ORDER BY id DESC
	LIMIT 1`

	var cfg Config
	err := s.db.QueryRow(query).Scan(
		&cfg.ID, &cfg.LLMProvider, &cfg.LLMModel, &cfg.LLMAPISKey, &cfg.LLMBaseURL,
		&cfg.LLMTemperature, &cfg.LLMMaxTokens,
		&cfg.EmbeddingProvider, &cfg.EmbeddingModel, &cfg.EmbeddingAPISKey, &cfg.EmbeddingBaseURL,
		&cfg.EmbeddingDimension,
		&cfg.AgentName, &cfg.AgentMaxLoop, &cfg.AgentConvergeAfter, &cfg.AgentTimeout, &cfg.AgentDailyTokenLimit,
		&cfg.AgentHistoryMaxEntries, &cfg.AgentHistoryMaxDays, &cfg.KnowledgeEnabled,
		&cfg.CreatedAt, &cfg.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &cfg, nil
}

// UpdateConfig 更新完整配置
// 如果不存在则插入，存在则更新
func (s *Storage) UpdateConfig(req *ConfigRequest) error {
	existing, err := s.GetConfig()
	if err != nil {
		return err
	}

	now := time.Now()

	if existing == nil {
		// 不存在则插入
		query := `
		INSERT INTO config (llm_provider, llm_model, llm_api_key, llm_base_url, llm_temperature, llm_max_tokens,
		                    embedding_provider, embedding_model, embedding_api_key, embedding_base_url, embedding_dimension,
		                    agent_name, agent_max_loop, agent_converge_after, agent_timeout, agent_daily_token_limit,
		                    agent_history_max_entries, agent_history_max_days, knowledge_enabled,
		                    created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
		_, err = s.db.Exec(query,
			req.LLMProvider, req.LLMModel, req.LLMAPISKey, req.LLMBaseURL,
			req.LLMTemperature, req.LLMMaxTokens,
			req.EmbeddingProvider, req.EmbeddingModel, req.EmbeddingAPISKey, req.EmbeddingBaseURL,
			req.EmbeddingDimension,
			req.AgentName, req.AgentMaxLoop, req.AgentConvergeAfter, req.AgentTimeout, req.AgentDailyTokenLimit,
			req.AgentHistoryMaxEntries, req.AgentHistoryMaxDays, req.KnowledgeEnabled,
			now, now,
		)
	} else {
		// 存在则更新
		query := `
		UPDATE config
		SET llm_provider = ?, llm_model = ?, llm_api_key = ?, llm_base_url = ?, llm_temperature = ?, llm_max_tokens = ?,
		    embedding_provider = ?, embedding_model = ?, embedding_api_key = ?, embedding_base_url = ?, embedding_dimension = ?,
		    agent_name = ?, agent_max_loop = ?, agent_converge_after = ?, agent_timeout = ?, agent_daily_token_limit = ?,
		    agent_history_max_entries = ?, agent_history_max_days = ?, knowledge_enabled = ?,
		    updated_at = ?
		WHERE id = ?`
		_, err = s.db.Exec(query,
			req.LLMProvider, req.LLMModel, req.LLMAPISKey, req.LLMBaseURL,
			req.LLMTemperature, req.LLMMaxTokens,
			req.EmbeddingProvider, req.EmbeddingModel, req.EmbeddingAPISKey, req.EmbeddingBaseURL,
			req.EmbeddingDimension,
			req.AgentName, req.AgentMaxLoop, req.AgentConvergeAfter, req.AgentTimeout, req.AgentDailyTokenLimit,
			req.AgentHistoryMaxEntries, req.AgentHistoryMaxDays, req.KnowledgeEnabled,
			now, existing.ID,
		)
	}
	return err
}

// UpdateConfigPartial 部分更新配置（只更新传入的字段）
// 传入 nil 的字段不会被更新，保持数据库原有值
func (s *Storage) UpdateConfigPartial(req *ConfigUpdateRequest) error {
	existing, err := s.GetConfig()
	if err != nil {
		return err
	}
	if existing == nil {
		return errors.New("配置不存在，请先创建配置")
	}

	setClauses := []string{}
	args := []interface{}{}

	if req.LLMProvider != nil {
		setClauses = append(setClauses, "llm_provider = ?")
		args = append(args, *req.LLMProvider)
	}
	if req.LLMModel != nil {
		setClauses = append(setClauses, "llm_model = ?")
		args = append(args, *req.LLMModel)
	}
	if req.LLMAPISKey != nil {
		setClauses = append(setClauses, "llm_api_key = ?")
		args = append(args, *req.LLMAPISKey)
	}
	if req.LLMBaseURL != nil {
		setClauses = append(setClauses, "llm_base_url = ?")
		args = append(args, *req.LLMBaseURL)
	}
	if req.LLMTemperature != nil {
		setClauses = append(setClauses, "llm_temperature = ?")
		args = append(args, *req.LLMTemperature)
	}
	if req.LLMMaxTokens != nil {
		setClauses = append(setClauses, "llm_max_tokens = ?")
		args = append(args, *req.LLMMaxTokens)
	}
	if req.EmbeddingProvider != nil {
		setClauses = append(setClauses, "embedding_provider = ?")
		args = append(args, *req.EmbeddingProvider)
	}
	if req.EmbeddingModel != nil {
		setClauses = append(setClauses, "embedding_model = ?")
		args = append(args, *req.EmbeddingModel)
	}
	if req.EmbeddingAPISKey != nil {
		setClauses = append(setClauses, "embedding_api_key = ?")
		args = append(args, *req.EmbeddingAPISKey)
	}
	if req.EmbeddingBaseURL != nil {
		setClauses = append(setClauses, "embedding_base_url = ?")
		args = append(args, *req.EmbeddingBaseURL)
	}
	if req.EmbeddingDimension != nil {
		setClauses = append(setClauses, "embedding_dimension = ?")
		args = append(args, *req.EmbeddingDimension)
	}
	if req.AgentName != nil {
		setClauses = append(setClauses, "agent_name = ?")
		args = append(args, *req.AgentName)
	}
	if req.AgentMaxLoop != nil {
		setClauses = append(setClauses, "agent_max_loop = ?")
		args = append(args, *req.AgentMaxLoop)
	}
	if req.AgentConvergeAfter != nil {
		setClauses = append(setClauses, "agent_converge_after = ?")
		args = append(args, *req.AgentConvergeAfter)
	}
	if req.AgentTimeout != nil {
		setClauses = append(setClauses, "agent_timeout = ?")
		args = append(args, *req.AgentTimeout)
	}
	if req.AgentDailyTokenLimit != nil {
		setClauses = append(setClauses, "agent_daily_token_limit = ?")
		args = append(args, *req.AgentDailyTokenLimit)
	}
	if req.AgentHistoryMaxEntries != nil {
		setClauses = append(setClauses, "agent_history_max_entries = ?")
		args = append(args, *req.AgentHistoryMaxEntries)
	}
	if req.AgentHistoryMaxDays != nil {
		setClauses = append(setClauses, "agent_history_max_days = ?")
		args = append(args, *req.AgentHistoryMaxDays)
	}
	if req.KnowledgeEnabled != nil {
		setClauses = append(setClauses, "knowledge_enabled = ?")
		args = append(args, *req.KnowledgeEnabled)
	}

	if len(setClauses) == 0 {
		return nil
	}

	setClauses = append(setClauses, "updated_at = ?")
	args = append(args, time.Now())
	args = append(args, existing.ID)

	query := fmt.Sprintf("UPDATE config SET %s WHERE id = ?", strings.Join(setClauses, ", "))
	_, err = s.db.Exec(query, args...)
	return err
}

// GetLLMConfig 获取LLM配置
// 从完整的 config 记录中提取 LLM 相关字段
func (s *Storage) GetLLMConfig() (*LLMConfigRequest, error) {
	cfg, err := s.GetConfig()
	if err != nil || cfg == nil {
		return nil, err
	}
	return &LLMConfigRequest{
		Provider:    cfg.LLMProvider,
		Model:       cfg.LLMModel,
		APIKey:      cfg.LLMAPISKey,
		BaseURL:     cfg.LLMBaseURL,
		Temperature: cfg.LLMTemperature,
		MaxTokens:   cfg.LLMMaxTokens,
	}, nil
}

// GetEmbeddingConfig 获取Embedding配置
// 从完整的 config 记录中提取 Embedding 相关字段
func (s *Storage) GetEmbeddingConfig() (*EmbeddingConfigRequest, error) {
	cfg, err := s.GetConfig()
	if err != nil || cfg == nil {
		return nil, err
	}
	return &EmbeddingConfigRequest{
		Provider:  cfg.EmbeddingProvider,
		Model:     cfg.EmbeddingModel,
		APIKey:    cfg.EmbeddingAPISKey,
		BaseURL:   cfg.EmbeddingBaseURL,
		Dimension: cfg.EmbeddingDimension,
	}, nil
}

// GetAgentConfig 获取Agent配置
// 从完整的 config 记录中提取 Agent 相关字段
func (s *Storage) GetAgentConfig() (*AgentConfigRequest, error) {
	cfg, err := s.GetConfig()
	if err != nil || cfg == nil {
		return nil, err
	}
	return &AgentConfigRequest{
		Name:              cfg.AgentName,
		MaxLoop:           cfg.AgentMaxLoop,
		ConvergeAfter:     cfg.AgentConvergeAfter,
		Timeout:           cfg.AgentTimeout,
		DailyTokenLimit:   cfg.AgentDailyTokenLimit,
		HistoryMaxEntries: cfg.AgentHistoryMaxEntries,
		HistoryMaxDays:    cfg.AgentHistoryMaxDays,
	}, nil
}

// configRequestFromConfig 将数据库配置转换为完整更新请求
// 用于模块级配置更新时保留其他模块的现有配置，避免未更新字段被零值覆盖
func configRequestFromConfig(cfg *Config) *ConfigRequest {
	if cfg == nil {
		return &ConfigRequest{}
	}
	return &ConfigRequest{
		LLMProvider:            cfg.LLMProvider,
		LLMModel:               cfg.LLMModel,
		LLMAPISKey:             cfg.LLMAPISKey,
		LLMBaseURL:             cfg.LLMBaseURL,
		LLMTemperature:         cfg.LLMTemperature,
		LLMMaxTokens:           cfg.LLMMaxTokens,
		EmbeddingProvider:      cfg.EmbeddingProvider,
		EmbeddingModel:         cfg.EmbeddingModel,
		EmbeddingAPISKey:       cfg.EmbeddingAPISKey,
		EmbeddingBaseURL:       cfg.EmbeddingBaseURL,
		EmbeddingDimension:     cfg.EmbeddingDimension,
		AgentName:              cfg.AgentName,
		AgentMaxLoop:           cfg.AgentMaxLoop,
		AgentConvergeAfter:     cfg.AgentConvergeAfter,
		AgentTimeout:           cfg.AgentTimeout,
		AgentDailyTokenLimit:   cfg.AgentDailyTokenLimit,
		AgentHistoryMaxEntries: cfg.AgentHistoryMaxEntries,
		AgentHistoryMaxDays:    cfg.AgentHistoryMaxDays,
		KnowledgeEnabled:       cfg.KnowledgeEnabled,
	}
}

// UpdateLLMConfig 更新LLM配置
// 仅更新 config 表中的 LLM 相关字段，保留其他字段不变
func (s *Storage) UpdateLLMConfig(req *LLMConfigRequest) error {
	fullCfg, err := s.GetConfig()
	if err != nil {
		return err
	}

	updateReq := configRequestFromConfig(fullCfg)

	updateReq.LLMProvider = req.Provider
	updateReq.LLMModel = req.Model
	updateReq.LLMAPISKey = req.APIKey
	updateReq.LLMBaseURL = req.BaseURL
	updateReq.LLMTemperature = req.Temperature
	updateReq.LLMMaxTokens = req.MaxTokens

	return s.UpdateConfig(updateReq)
}

// UpdateEmbeddingConfig 更新Embedding配置
// 仅更新 config 表中的 Embedding 相关字段，保留其他字段不变
func (s *Storage) UpdateEmbeddingConfig(req *EmbeddingConfigRequest) error {
	fullCfg, err := s.GetConfig()
	if err != nil {
		return err
	}

	updateReq := configRequestFromConfig(fullCfg)

	updateReq.EmbeddingProvider = req.Provider
	updateReq.EmbeddingModel = req.Model
	updateReq.EmbeddingAPISKey = req.APIKey
	updateReq.EmbeddingBaseURL = req.BaseURL
	updateReq.EmbeddingDimension = req.Dimension

	return s.UpdateConfig(updateReq)
}

// UpdateAgentConfig 更新Agent配置
// 仅更新 config 表中的 Agent 相关字段，保留其他字段不变
func (s *Storage) UpdateAgentConfig(req *AgentConfigRequest) error {
	fullCfg, err := s.GetConfig()
	if err != nil {
		return err
	}

	updateReq := configRequestFromConfig(fullCfg)

	updateReq.AgentName = req.Name
	updateReq.AgentMaxLoop = req.MaxLoop
	updateReq.AgentConvergeAfter = req.ConvergeAfter
	updateReq.AgentTimeout = req.Timeout
	updateReq.AgentDailyTokenLimit = req.DailyTokenLimit
	updateReq.AgentHistoryMaxEntries = req.HistoryMaxEntries
	updateReq.AgentHistoryMaxDays = req.HistoryMaxDays

	return s.UpdateConfig(updateReq)
}

// GetKnowledgeConfig 获取知识库配置
// 从完整的 config 记录中提取知识库相关字段
func (s *Storage) GetKnowledgeConfig() (*KnowledgeConfigRequest, error) {
	cfg, err := s.GetConfig()
	if err != nil || cfg == nil {
		return nil, err
	}
	return &KnowledgeConfigRequest{
		Enabled: cfg.KnowledgeEnabled,
	}, nil
}

// UpdateKnowledgeConfig 更新知识库配置
// 仅更新 config 表中的知识库相关字段，保留其他字段不变
func (s *Storage) UpdateKnowledgeConfig(req *KnowledgeConfigRequest) error {
	fullCfg, err := s.GetConfig()
	if err != nil {
		return err
	}

	updateReq := configRequestFromConfig(fullCfg)

	updateReq.KnowledgeEnabled = req.Enabled

	return s.UpdateConfig(updateReq)
}
