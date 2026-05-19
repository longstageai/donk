package builtin

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/longstageai/donk/donk/internal/tool"
)

// HTTPMethod HTTP请求方法
type HTTPMethod string

const (
	MethodGET     HTTPMethod = "GET"
	MethodPOST    HTTPMethod = "POST"
	MethodPUT     HTTPMethod = "PUT"
	MethodDELETE  HTTPMethod = "DELETE"
	MethodPATCH   HTTPMethod = "PATCH"
	MethodHEAD    HTTPMethod = "HEAD"
	MethodOPTIONS HTTPMethod = "OPTIONS"
)

// HTTPConfig HTTP工具配置
type HTTPConfig struct {
	Timeout        time.Duration     // 请求超时时间
	DefaultHeaders map[string]string // 默认请求头
	AllowedHosts   []string          // 允许访问的域名
	BlockedHosts   []string          // 禁止访问的域名
	MaxRetries     int               // 最大重试次数
	RetryDelay     time.Duration     // 重试间隔
}

// DefaultHTTPConfig 默认HTTP配置
var DefaultHTTPConfig = HTTPConfig{
	Timeout:    30 * time.Second,
	MaxRetries: 3,
	RetryDelay: 1 * time.Second,
	DefaultHeaders: map[string]string{
		"User-Agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
	},
}

var (
	htmlScriptRegex     = regexp.MustCompile(`(?is)<script[^>]*>.*?</script>`)
	htmlStyleRegex      = regexp.MustCompile(`(?is)<style[^>]*>.*?</style>`)
	htmlCommentRegex    = regexp.MustCompile(`(?s)<!--.*?-->`)
	htmlTagRegex        = regexp.MustCompile(`(?s)<[^>]+>`)
	htmlWhitespaceRegex = regexp.MustCompile(`\s+`)
)

// HTTP 工具
// 用于发送HTTP请求，支持重试机制、超时控制、安全检查
type HTTP struct {
	client *http.Client
	config HTTPConfig
}

// NewHTTP 创建HTTP工具
func NewHTTP(config ...HTTPConfig) *HTTP {
	cfg := DefaultHTTPConfig
	if len(config) > 0 {
		cfg = config[0]
	}

	return &HTTP{
		client: &http.Client{
			Timeout: cfg.Timeout,
		},
		config: cfg,
	}
}

// Name 返回工具名称
func (h *HTTP) Name() string {
	return "http"
}

// Description 返回工具描述
func (h *HTTP) Description() string {
	return "发送HTTP请求，获取网页内容或调用API。支持重试机制、超时控制、安全检查。"
}

// Version 返回版本
func (h *HTTP) Version() string {
	return "1.1.0"
}

// Category 返回分类
func (h *HTTP) Category() string {
	return string(tool.CategoryNetwork)
}

// Parameters 返回参数定义
func (h *HTTP) Parameters() *tool.Schema {
	schema := tool.NewSchema()
	schema.Properties = map[string]*tool.Property{
		"url": {
			Type:        "string",
			Description: "请求URL",
		},
		"method": {
			Type:        "string",
			Description: "请求方法(GET/POST/PUT/DELETE/PATCH/HEAD/OPTIONS)",
			Default:     "GET",
			Enum:        []interface{}{"GET", "POST", "PUT", "DELETE", "PATCH", "HEAD", "OPTIONS"},
		},
		"headers": {
			Type:        "object",
			Description: "请求头，例如: {\"Content-Type\": \"application/json\"}",
		},
		"body": {
			Type:        "object",
			Description: "请求体（对象或字符串）",
		},
		"timeout": {
			Type:        "integer",
			Description: "超时时间(秒)，默认30秒",
			Default:     30,
		},
		"retries": {
			Type:        "integer",
			Description: "重试次数，默认3次",
			Default:     3,
		},
		"follow_redirects": {
			Type:        "boolean",
			Description: "是否跟随重定向，默认true",
			Default:     true,
		},
	}
	schema.Required = []string{"url"}
	return schema
}

// Execute 执行HTTP请求
func (h *HTTP) Execute(ctx *tool.Context) (*tool.Result, error) {
	// 获取URL
	reqURL, ok := ctx.Params["url"].(string)
	if !ok || reqURL == "" {
		return tool.NewErrorResultWithMsg(tool.ErrCodeInvalidParams, "URL不能为空"), nil
	}

	// 验证URL
	if err := h.validateURL(reqURL); err != nil {
		return tool.NewErrorResultWithMsg(tool.ErrCodeInvalidParams, fmt.Sprintf("URL验证失败: %v", err)), nil
	}

	// 获取请求方法
	method := "GET"
	if m, ok := ctx.Params["method"].(string); ok && m != "" {
		method = strings.ToUpper(m)
	}

	// 验证方法
	if !h.isValidMethod(method) {
		return tool.NewErrorResultWithMsg(tool.ErrCodeInvalidParams, fmt.Sprintf("不支持的HTTP方法: %s", method)), nil
	}

	// 获取超时
	timeout := h.config.Timeout
	if t, ok := parsePositiveDuration(ctx.Params["timeout"]); ok {
		timeout = t
	}

	// 获取重试次数
	retries := h.config.MaxRetries
	if r, ok := parseNonNegativeInt(ctx.Params["retries"]); ok {
		retries = r
	}

	// 获取是否跟随重定向
	followRedirects := true
	if fr, ok := ctx.Params["follow_redirects"].(bool); ok {
		followRedirects = fr
	}

	// 构建请求头
	headers := make(map[string]string)
	for k, v := range h.config.DefaultHeaders {
		headers[k] = v
	}
	if hdrs, ok := ctx.Params["headers"].(map[string]any); ok {
		for k, v := range hdrs {
			if sv, ok := v.(string); ok {
				headers[k] = sv
			}
		}
	}

	// 构建请求体
	var bodyData []byte
	contentType := ""
	if b, ok := ctx.Params["body"].(string); ok && b != "" {
		bodyData = []byte(b)
		contentType = "text/plain"
	} else if b, ok := ctx.Params["body"].(map[string]any); ok {
		jsonData, err := json.Marshal(b)
		if err != nil {
			return tool.NewErrorResultWithMsg(tool.ErrCodeInvalidParams, fmt.Sprintf("请求体JSON序列化失败: %v", err)), nil
		}
		bodyData = jsonData
		contentType = "application/json"
	}

	if contentType != "" && !hasHeader(headers, "Content-Type") {
		headers["Content-Type"] = contentType
	}

	// 创建HTTP客户端
	client := &http.Client{
		Timeout: timeout,
	}
	if !followRedirects {
		client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		}
	}

	// 执行请求（带重试）
	startTime := time.Now()
	var lastErr error
	var resp *http.Response
	attempts := 0

	for attempt := 0; attempt <= retries; attempt++ {
		attempts = attempt + 1
		if attempt > 0 {
			time.Sleep(h.config.RetryDelay * time.Duration(attempt))
		}

		resp, lastErr = h.doRequest(client, ctx.Values, method, reqURL, headers, bodyData)
		if lastErr == nil && resp != nil && resp.StatusCode < 500 {
			break
		}

		if resp != nil {
			resp.Body.Close()
		}
	}

	if lastErr != nil {
		return tool.NewErrorResultWithMsg(tool.ErrCodeExecution, fmt.Sprintf("请求失败: %v", lastErr)), nil
	}

	defer resp.Body.Close()

	// 读取响应体
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return tool.NewErrorResultWithMsg(tool.ErrCodeExecution, fmt.Sprintf("读取响应失败: %v", err)), nil
	}

	// 构建响应
	responseHeaders := make(map[string]string)
	for k, v := range resp.Header {
		if len(v) > 0 {
			responseHeaders[k] = v[0]
		}
	}

	responseText := string(respBody)

	// 清理HTML标签，提取纯文本
	cleanText := stripHTML(responseText)

	result := tool.NewResult(map[string]any{
		"status_code": resp.StatusCode,
		"status":      resp.Status,
		"headers":     responseHeaders,
		"body":        cleanText,
		"duration_ms": time.Since(startTime).Milliseconds(),
		"retries":     attempts - 1,
	})
	result.SetExecutionTime(time.Since(startTime))

	return result, nil
}

// doRequest 执行单次HTTP请求
func (h *HTTP) doRequest(client *http.Client, ctx context.Context, method, url string, headers map[string]string, bodyData []byte) (*http.Response, error) {
	var body io.Reader
	if len(bodyData) > 0 {
		body = bytes.NewReader(bodyData)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, err
	}

	for k, v := range headers {
		req.Header.Set(k, v)
	}

	return client.Do(req)
}

// validateURL 验证URL安全
func (h *HTTP) validateURL(reqURL string) error {
	parsedURL, err := url.Parse(reqURL)
	if err != nil {
		return fmt.Errorf("无效的URL: %v", err)
	}

	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return fmt.Errorf("只支持http和https协议")
	}

	host := parsedURL.Hostname()

	// 检查禁止的域名
	for _, blocked := range h.config.BlockedHosts {
		if strings.Contains(host, blocked) {
			return fmt.Errorf("禁止访问的域名: %s", host)
		}
	}

	// 检查允许的域名（如果设置了白名单）
	if len(h.config.AllowedHosts) > 0 {
		allowed := false
		for _, allowedHost := range h.config.AllowedHosts {
			if strings.Contains(host, allowedHost) {
				allowed = true
				break
			}
		}
		if !allowed {
			return fmt.Errorf("域名不在允许列表中: %s", host)
		}
	}

	return nil
}

// isValidMethod 验证HTTP方法
func (h *HTTP) isValidMethod(method string) bool {
	validMethods := []string{"GET", "POST", "PUT", "DELETE", "PATCH", "HEAD", "OPTIONS"}
	for _, m := range validMethods {
		if method == m {
			return true
		}
	}
	return false
}

func parsePositiveDuration(value any) (time.Duration, bool) {
	seconds, ok := parsePositiveFloat(value)
	if !ok {
		return 0, false
	}
	return time.Duration(seconds * float64(time.Second)), true
}

func parsePositiveFloat(value any) (float64, bool) {
	switch v := value.(type) {
	case float64:
		return v, v > 0
	case float32:
		return float64(v), v > 0
	case int:
		return float64(v), v > 0
	case int64:
		return float64(v), v > 0
	case int32:
		return float64(v), v > 0
	case json.Number:
		f, err := v.Float64()
		return f, err == nil && f > 0
	default:
		return 0, false
	}
}

func parseNonNegativeInt(value any) (int, bool) {
	switch v := value.(type) {
	case float64:
		return int(v), v >= 0
	case float32:
		return int(v), v >= 0
	case int:
		return v, v >= 0
	case int64:
		return int(v), v >= 0
	case int32:
		return int(v), v >= 0
	case json.Number:
		i, err := v.Int64()
		return int(i), err == nil && i >= 0
	default:
		return 0, false
	}
}

func hasHeader(headers map[string]string, name string) bool {
	for k := range headers {
		if strings.EqualFold(k, name) {
			return true
		}
	}
	return false
}

// HTTPRequest HTTP请求参数（用于程序化调用）
type HTTPRequest struct {
	URL         string            `json:"url"`          // 请求URL
	Method      string            `json:"method"`       // 请求方法
	Headers     map[string]string `json:"headers"`      // 请求头
	Body        any               `json:"body"`         // 请求体
	Timeout     int               `json:"timeout"`      // 超时时间(秒)
	ContentType string            `json:"content_type"` // Content-Type
}

// HTTPResponse HTTP响应结果
type HTTPResponse struct {
	StatusCode int               `json:"status_code"` // 状态码
	Status     string            `json:"status"`      // 状态描述
	Headers    map[string]string `json:"headers"`     // 响应头
	Body       string            `json:"body"`        // 响应体
	Duration   int64             `json:"duration"`    // 请求耗时(毫秒)
}

// NewHTTPRequest 创建HTTP请求
func NewHTTPRequest(url string, method HTTPMethod) *HTTPRequest {
	return &HTTPRequest{
		URL:     url,
		Method:  string(method),
		Headers: make(map[string]string),
	}
}

// NewHTTPRequestWithBody 创建带请求体的HTTP请求
func NewHTTPRequestWithBody(url string, method HTTPMethod, body any) *HTTPRequest {
	req := NewHTTPRequest(url, method)
	req.Body = body
	return req
}

// NewHTTPWithConfig 创建带配置的HTTP工具
func NewHTTPWithConfig(config HTTPConfig) *HTTP {
	return NewHTTP(config)
}

// SimpleHTTP 简单HTTP请求工具
// 创建一个简单的HTTP GET请求工具（向后兼容）
func SimpleHTTP() tool.Tool {
	return NewHTTP()
}

// stripHTML 去除HTML标签，提取纯文本
func stripHTML(rawHTML string) string {
	text := htmlScriptRegex.ReplaceAllString(rawHTML, "")
	text = htmlStyleRegex.ReplaceAllString(text, "")
	text = htmlCommentRegex.ReplaceAllString(text, "")
	text = htmlTagRegex.ReplaceAllString(text, "")
	text = html.UnescapeString(text)
	text = htmlWhitespaceRegex.ReplaceAllString(text, " ")
	return strings.TrimSpace(text)
}
