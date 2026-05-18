package setting_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/longstageai/donk/donk/internal/setting"
	"github.com/longstageai/donk/donk/internal/sql"
)

func TestIntegration(t *testing.T) {
	dbPath := "./data/setting/test_integration.db"

	os.Remove(dbPath)
	defer os.Remove(dbPath)

	gin.SetMode(gin.TestMode)
	engine := gin.New()

	db, err := sql.Open(dbPath)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	engine, err = setting.Setup(db.DB, engine, true)
	if err != nil {
		t.Fatalf("Failed to setup: %v", err)
	}

	t.Run("Health Check", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/health", nil)
		w := httptest.NewRecorder()
		engine.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
	})

	t.Run("Get Default LLM Config", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/v1/config/llm", nil)
		w := httptest.NewRecorder()
		engine.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		var cfg setting.LLMConfigRequest
		if err := json.Unmarshal(w.Body.Bytes(), &cfg); err != nil {
			t.Errorf("Failed to parse response: %v", err)
		}

		if cfg.Provider != "openai" {
			t.Errorf("Expected provider 'openai', got '%s'", cfg.Provider)
		}
	})

	t.Run("Update LLM Config", func(t *testing.T) {
		reqBody := setting.LLMConfigRequest{
			Provider:    "deepseek",
			Model:       "deepseek-chat",
			APIKey:      "sk-test-key",
			BaseURL:     "https://api.deepseek.com",
			Temperature: 0.8,
			MaxTokens:   8192,
		}
		body, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest("PUT", "/api/v1/config/llm", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		engine.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
	})

	t.Run("Verify Updated LLM Config", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/v1/config/llm", nil)
		w := httptest.NewRecorder()
		engine.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		var cfg setting.LLMConfigRequest
		if err := json.Unmarshal(w.Body.Bytes(), &cfg); err != nil {
			t.Errorf("Failed to parse response: %v", err)
		}

		if cfg.Provider != "deepseek" {
			t.Errorf("Expected provider 'deepseek', got '%s'", cfg.Provider)
		}
		if cfg.Model != "deepseek-chat" {
			t.Errorf("Expected model 'deepseek-chat', got '%s'", cfg.Model)
		}
	})

	t.Run("Update Embedding Config", func(t *testing.T) {
		reqBody := setting.EmbeddingConfigRequest{
			Provider:  "openai",
			Model:     "text-embedding-3-large",
			APIKey:    "sk-embed-key",
			BaseURL:   "",
			Dimension: 3072,
		}
		body, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest("PUT", "/api/v1/config/embedding", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		engine.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
	})

	t.Run("Update Agent Config", func(t *testing.T) {
		reqBody := setting.AgentConfigRequest{
			Name:            "TestAgent",
			MaxLoop:         20,
			ConvergeAfter:   5,
			Timeout:         600,
			DailyTokenLimit: 100000,
		}
		body, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest("PUT", "/api/v1/config/agent", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		engine.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
	})

	t.Run("Verify Updated Agent Config", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/v1/config/agent", nil)
		w := httptest.NewRecorder()
		engine.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		var cfg setting.AgentConfigRequest
		if err := json.Unmarshal(w.Body.Bytes(), &cfg); err != nil {
			t.Errorf("Failed to parse response: %v", err)
		}

		if cfg.Name != "TestAgent" {
			t.Errorf("Expected name 'TestAgent', got '%s'", cfg.Name)
		}
		if cfg.MaxLoop != 20 {
			t.Errorf("Expected max_loop 20, got %d", cfg.MaxLoop)
		}
	})
}
