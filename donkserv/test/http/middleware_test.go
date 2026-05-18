package http_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	httpmodule "github.com/longstageai/donk/donk/internal/http"
)

func TestAuthMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("no auth header", func(t *testing.T) {
		auth := httpmodule.NewAuthMiddleware("valid-key")

		router := gin.New()
		router.Use(auth.GinHandler())
		router.GET("/test", func(c *gin.Context) {
			c.JSON(200, gin.H{"status": "ok"})
		})

		req, _ := http.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("Expected status 401, got %d", w.Code)
		}
	})

	t.Run("invalid auth format", func(t *testing.T) {
		auth := httpmodule.NewAuthMiddleware("valid-key")

		router := gin.New()
		router.Use(auth.GinHandler())
		router.GET("/test", func(c *gin.Context) {
			c.JSON(200, gin.H{"status": "ok"})
		})

		req, _ := http.NewRequest("GET", "/test", nil)
		req.Header.Set("Authorization", "InvalidFormat")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("Expected status 401, got %d", w.Code)
		}
	})

	t.Run("invalid api key", func(t *testing.T) {
		auth := httpmodule.NewAuthMiddleware("valid-key")

		router := gin.New()
		router.Use(auth.GinHandler())
		router.GET("/test", func(c *gin.Context) {
			c.JSON(200, gin.H{"status": "ok"})
		})

		req, _ := http.NewRequest("GET", "/test", nil)
		req.Header.Set("Authorization", "Bearer invalid-key")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("Expected status 401, got %d", w.Code)
		}
	})

	t.Run("valid api key", func(t *testing.T) {
		auth := httpmodule.NewAuthMiddleware("valid-key")

		router := gin.New()
		router.Use(auth.GinHandler())
		router.GET("/test", func(c *gin.Context) {
			c.JSON(200, gin.H{"status": "ok"})
		})

		req, _ := http.NewRequest("GET", "/test", nil)
		req.Header.Set("Authorization", "Bearer valid-key")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
	})

	t.Run("multiple valid keys", func(t *testing.T) {
		auth := httpmodule.NewAuthMiddleware("key1", "key2", "key3")

		router := gin.New()
		router.Use(auth.GinHandler())
		router.GET("/test", func(c *gin.Context) {
			c.JSON(200, gin.H{"status": "ok"})
		})

		req, _ := http.NewRequest("GET", "/test", nil)
		req.Header.Set("Authorization", "Bearer key2")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
	})

	t.Run("empty key should be ignored", func(t *testing.T) {
		auth := httpmodule.NewAuthMiddleware("valid-key", "")

		router := gin.New()
		router.Use(auth.GinHandler())
		router.GET("/test", func(c *gin.Context) {
			c.JSON(200, gin.H{"status": "ok"})
		})

		req, _ := http.NewRequest("GET", "/test", nil)
		req.Header.Set("Authorization", "Bearer ")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("Expected status 401, got %d", w.Code)
		}
	})

	t.Run("no auth middleware enabled", func(t *testing.T) {
		auth := httpmodule.NewAuthMiddleware()

		router := gin.New()
		if auth.IsEnabled() {
			router.Use(auth.GinHandler())
		}
		router.GET("/test", func(c *gin.Context) {
			c.JSON(200, gin.H{"status": "ok"})
		})

		req, _ := http.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
	})

	t.Run("IsEnabled check", func(t *testing.T) {
		authWithKeys := httpmodule.NewAuthMiddleware("key1")
		if !authWithKeys.IsEnabled() {
			t.Error("Expected auth to be enabled")
		}

		authWithoutKeys := httpmodule.NewAuthMiddleware()
		if authWithoutKeys.IsEnabled() {
			t.Error("Expected auth to be disabled")
		}

		authWithEmptyKeys := httpmodule.NewAuthMiddleware("", "")
		if authWithEmptyKeys.IsEnabled() {
			t.Error("Expected auth to be disabled with empty keys")
		}
	})
}
