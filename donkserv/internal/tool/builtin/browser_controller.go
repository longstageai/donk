package builtin

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/longstageai/donk/donk/internal/tool"

	"github.com/chromedp/chromedp"
)

type BrowserController struct {
	mu          sync.Mutex
	allocCtx    context.Context
	cancel      context.CancelFunc
	tabCtx      context.Context
	tabCancel   context.CancelFunc
	chromePath  string
	userDataDir string
	debugPort   int
}

type BrowserAction string

const (
	ActionNavigate   BrowserAction = "navigate"
	ActionClick      BrowserAction = "click"
	ActionInput      BrowserAction = "input"
	ActionGetText    BrowserAction = "get_text"
	ActionGetHTML    BrowserAction = "get_html"
	ActionScreenshot BrowserAction = "screenshot"
	ActionExecuteJS  BrowserAction = "execute_js"
	ActionWait       BrowserAction = "wait"
	ActionScroll     BrowserAction = "scroll"
)

const (
	browserDefaultPort    = 9222
	browserDefaultTimeout = 10
	browserMaxTimeout     = 120
	browserLaunchRetries  = 20
)

var browserSafeSelectorPattern = regexp.MustCompile(`^[a-zA-Z0-9_\-\.\#\[\]\=\s\:\>\+\~\*\']+$`)

func NewBrowserController() *BrowserController {
	return &BrowserController{
		debugPort: browserDefaultPort,
	}
}

func (b *BrowserController) Name() string {
	return "browser_controller"
}

func (b *BrowserController) Description() string {
	return "自动化控制Chrome或Edge浏览器执行网页操作。支持导航、点击、输入、截图等功能，会自动查找并启动浏览器。"
}

func (b *BrowserController) Version() string {
	return "1.0.0"
}

func (b *BrowserController) Category() string {
	return string(tool.CategoryUtility)
}

func (b *BrowserController) Parameters() *tool.Schema {
	schema := tool.NewSchema()
	schema.Properties = map[string]*tool.Property{
		"action": {
			Type:        "string",
			Description: "操作类型: navigate(导航), click(点击), input(输入), get_text(获取文本), get_html(获取HTML), screenshot(截图), execute_js(执行JS), wait(等待元素), scroll(滚动)",
			Enum:        []interface{}{"navigate", "click", "input", "get_text", "get_html", "screenshot", "execute_js", "wait", "scroll"},
		},
		"url": {
			Type:        "string",
			Description: "URL地址（navigate操作必需）",
		},
		"selector": {
			Type:        "string",
			Description: "CSS选择器（click/input/get_text/get_html/wait操作需要）",
		},
		"text": {
			Type:        "string",
			Description: "要输入的文本（input操作需要）",
		},
		"script": {
			Type:        "string",
			Description: "JavaScript代码（execute_js操作需要）",
		},
		"wait_time": {
			Type:        "integer",
			Description: "等待时间（秒），默认10秒，最大120秒",
			Default:     browserDefaultTimeout,
		},
		"full_page": {
			Type:        "boolean",
			Description: "是否截取整个页面（screenshot操作），默认false只截取可视区域",
			Default:     false,
		},
		"x": {
			Type:        "integer",
			Description: "水平滚动位置（scroll操作）",
		},
		"y": {
			Type:        "integer",
			Description: "垂直滚动位置（scroll操作）",
		},
	}
	schema.Required = []string{"action"}
	return schema
}

func (b *BrowserController) Execute(ctx *tool.Context) (*tool.Result, error) {
	if runtime.GOOS != "windows" {
		return tool.NewErrorResultWithMsg(tool.ErrCodeExecution, "此工具仅支持Windows系统"), nil
	}

	action, ok := ctx.Params["action"].(string)
	if !ok || strings.TrimSpace(action) == "" {
		return tool.NewErrorResultWithMsg(tool.ErrCodeInvalidParams, "操作类型不能为空"), nil
	}

	browserAction := BrowserAction(strings.TrimSpace(action))
	if err := b.validateActionParams(browserAction, ctx.Params); err != nil {
		return tool.NewErrorResultWithMsg(tool.ErrCodeInvalidParams, err.Error()), nil
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	if err := b.ensureChrome(); err != nil {
		return tool.NewErrorResultWithMsg(tool.ErrCodeExecution, fmt.Sprintf("Chrome连接失败: %v", err)), nil
	}

	if err := b.ensureTab(); err != nil {
		return tool.NewErrorResultWithMsg(tool.ErrCodeExecution, fmt.Sprintf("创建标签页失败: %v", err)), nil
	}

	waitTime := b.getWaitTime(ctx.Params)

	switch browserAction {
	case ActionNavigate:
		return b.navigate(waitTime, ctx.Params)
	case ActionClick:
		return b.click(waitTime, ctx.Params)
	case ActionInput:
		return b.input(waitTime, ctx.Params)
	case ActionGetText:
		return b.getText(waitTime, ctx.Params)
	case ActionGetHTML:
		return b.getHTML(waitTime, ctx.Params)
	case ActionScreenshot:
		return b.screenshot(waitTime, ctx.Params)
	case ActionExecuteJS:
		return b.executeJS(waitTime, ctx.Params)
	case ActionWait:
		return b.wait(waitTime, ctx.Params)
	case ActionScroll:
		return b.scroll(waitTime, ctx.Params)
	default:
		return tool.NewErrorResultWithMsg(tool.ErrCodeInvalidParams, fmt.Sprintf("不支持的操作类型: %s", action)), nil
	}
}

func (b *BrowserController) validateActionParams(action BrowserAction, params map[string]any) error {
	switch action {
	case ActionNavigate:
		url, ok := params["url"].(string)
		if !ok || strings.TrimSpace(url) == "" {
			return fmt.Errorf("URL不能为空")
		}
		if !b.isSafeURL(url) {
			return fmt.Errorf("不安全的URL")
		}
	case ActionClick, ActionGetText, ActionWait:
		selector, ok := params["selector"].(string)
		if !ok || strings.TrimSpace(selector) == "" {
			return fmt.Errorf("选择器不能为空")
		}
		if !b.isSafeSelector(selector) {
			return fmt.Errorf("不安全的选择器")
		}
	case ActionInput:
		selector, ok := params["selector"].(string)
		if !ok || strings.TrimSpace(selector) == "" {
			return fmt.Errorf("选择器不能为空")
		}
		if !b.isSafeSelector(selector) {
			return fmt.Errorf("不安全的选择器")
		}
		if _, ok := params["text"].(string); !ok {
			return fmt.Errorf("文本不能为空")
		}
	case ActionGetHTML:
		if selector, ok := params["selector"].(string); ok && strings.TrimSpace(selector) != "" && !b.isSafeSelector(selector) {
			return fmt.Errorf("不安全的选择器")
		}
	case ActionExecuteJS:
		script, ok := params["script"].(string)
		if !ok || strings.TrimSpace(script) == "" {
			return fmt.Errorf("脚本不能为空")
		}
		if !b.isSafeScript(script) {
			return fmt.Errorf("不安全的脚本")
		}
	case ActionScreenshot, ActionScroll:
		return nil
	default:
		return fmt.Errorf("不支持的操作类型: %s", action)
	}
	return nil
}

func (b *BrowserController) ensureChrome() error {
	if b.allocCtx != nil && b.cancel != nil && b.isAllocatorAlive() {
		return nil
	}

	b.resetContexts()

	chromePath, err := b.findChrome()
	if err != nil {
		return err
	}
	b.chromePath = chromePath

	userDataDir, err := b.defaultUserDataDir()
	if err != nil {
		return err
	}
	b.userDataDir = userDataDir

	if err := os.MkdirAll(b.userDataDir, 0755); err != nil {
		return fmt.Errorf("创建用户数据目录失败: %w", err)
	}

	if b.tryConnectExisting() {
		return nil
	}

	return b.launchChrome()
}

func (b *BrowserController) isAllocatorAlive() bool {
	if b.tabCtx != nil {
		testCtx, cancel := context.WithTimeout(b.tabCtx, 2*time.Second)
		defer cancel()
		var title string
		return chromedp.Run(testCtx, chromedp.Title(&title)) == nil
	}

	if !b.debugEndpointReady(2 * time.Second) {
		return false
	}

	return true
}

func (b *BrowserController) resetContexts() {
	if b.tabCancel != nil {
		b.tabCancel()
	}
	if b.cancel != nil {
		b.cancel()
	}
	b.allocCtx = nil
	b.cancel = nil
	b.tabCtx = nil
	b.tabCancel = nil
}

func (b *BrowserController) defaultUserDataDir() (string, error) {
	execPath, err := os.Executable()
	if err != nil {
		wd, wdErr := os.Getwd()
		if wdErr != nil {
			return "", err
		}
		return filepath.Join(wd, "data", "chrome-debug"), nil
	}
	return filepath.Join(filepath.Dir(execPath), "data", "chrome-debug"), nil
}

func (b *BrowserController) findChrome() (string, error) {
	if b.chromePath != "" {
		if _, err := os.Stat(b.chromePath); err == nil {
			return b.chromePath, nil
		}
	}

	if chromePath := b.findChromeFromRegistry(); chromePath != "" {
		return chromePath, nil
	}

	possiblePaths := []string{
		`C:\Program Files\Google\Chrome\Application\chrome.exe`,
		`C:\Program Files (x86)\Google\Chrome\Application\chrome.exe`,
		filepath.Join(os.Getenv("LOCALAPPDATA"), `Google\Chrome\Application\chrome.exe`),
		filepath.Join(os.Getenv("PROGRAMFILES"), `Google\Chrome\Application\chrome.exe`),
		filepath.Join(os.Getenv("PROGRAMFILES(X86)"), `Google\Chrome\Application\chrome.exe`),
		`C:\Program Files (x86)\Microsoft\Edge\Application\msedge.exe`,
		`C:\Program Files\Microsoft\Edge\Application\msedge.exe`,
		filepath.Join(os.Getenv("PROGRAMFILES(X86)"), `Microsoft\Edge\Application\msedge.exe`),
		filepath.Join(os.Getenv("PROGRAMFILES"), `Microsoft\Edge\Application\msedge.exe`),
	}

	for _, path := range possiblePaths {
		if path == "" {
			continue
		}
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}

	if path, err := exec.LookPath("chrome.exe"); err == nil {
		return path, nil
	}
	if path, err := exec.LookPath("msedge.exe"); err == nil {
		return path, nil
	}

	return "", fmt.Errorf("未找到Chrome或Edge浏览器，请安装Chrome后再试")
}

func (b *BrowserController) findChromeFromRegistry() string {
	keys := []string{
		`HKEY_LOCAL_MACHINE\SOFTWARE\Microsoft\Windows\CurrentVersion\App Paths\chrome.exe`,
		`HKEY_CURRENT_USER\SOFTWARE\Microsoft\Windows\CurrentVersion\App Paths\chrome.exe`,
		`HKEY_LOCAL_MACHINE\SOFTWARE\Microsoft\Windows\CurrentVersion\App Paths\msedge.exe`,
		`HKEY_CURRENT_USER\SOFTWARE\Microsoft\Windows\CurrentVersion\App Paths\msedge.exe`,
	}

	for _, key := range keys {
		cmd := exec.Command("reg", "query", key, "/ve")
		output, err := cmd.Output()
		if err != nil {
			continue
		}
		if path := parseRegistryDefaultPath(string(output)); path != "" {
			return path
		}
	}

	return ""
}

func parseRegistryDefaultPath(output string) string {
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) < 3 {
			continue
		}
		if fields[1] != "REG_SZ" && fields[1] != "REG_EXPAND_SZ" {
			continue
		}
		path := strings.Join(fields[2:], " ")
		path = os.ExpandEnv(strings.Trim(path, " \t\r\n\""))
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}
	return ""
}

func (b *BrowserController) tryConnectExisting() bool {
	if !b.debugEndpointReady(3 * time.Second) {
		return false
	}

	allocatorCtx, cancel := chromedp.NewRemoteAllocator(context.Background(), fmt.Sprintf("http://127.0.0.1:%d", b.debugPort))
	b.allocCtx = allocatorCtx
	b.cancel = cancel
	return true
}

func (b *BrowserController) debugEndpointReady(timeout time.Duration) bool {
	client := http.Client{Timeout: timeout}
	resp, err := client.Get(fmt.Sprintf("http://127.0.0.1:%d/json/version", b.debugPort))
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode >= 200 && resp.StatusCode < 300
}

func (b *BrowserController) launchChrome() error {
	args := []string{
		fmt.Sprintf("--remote-debugging-port=%d", b.debugPort),
		fmt.Sprintf("--user-data-dir=%s", b.userDataDir),
		"--no-first-run",
		"--no-default-browser-check",
		"--disable-default-apps",
		"--disable-popup-blocking",
		"--disable-infobars",
		"--window-size=1280,720",
	}

	cmd := exec.Command(b.chromePath, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: false}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("启动Chrome失败: %w", err)
	}

	for i := 0; i < browserLaunchRetries; i++ {
		if b.tryConnectExisting() {
			return nil
		}
		time.Sleep(500 * time.Millisecond)
	}

	return fmt.Errorf("Chrome已启动，但无法连接到调试端口 %d", b.debugPort)
}

func (b *BrowserController) ensureTab() error {
	if b.tabCtx != nil && b.tabCancel != nil {
		if b.isTabAlive() {
			return nil
		}

		b.tabCancel()
		b.tabCtx = nil
		b.tabCancel = nil
	}

	tabCtx, tabCancel := chromedp.NewContext(b.allocCtx)
	b.tabCtx = tabCtx
	b.tabCancel = tabCancel

	if err := chromedp.Run(b.tabCtx); err != nil {
		b.tabCancel()
		b.tabCtx = nil
		b.tabCancel = nil
		return err
	}

	return nil
}

func (b *BrowserController) isTabAlive() bool {
	if b.tabCtx == nil {
		return false
	}

	testCtx, cancel := context.WithTimeout(b.tabCtx, 2*time.Second)
	defer cancel()

	var title string
	return chromedp.Run(testCtx, chromedp.Title(&title)) == nil
}

func (b *BrowserController) withTimeout(waitTime int) (context.Context, context.CancelFunc) {
	return context.WithTimeout(b.tabCtx, time.Duration(waitTime)*time.Second)
}

func (b *BrowserController) getWaitTime(params map[string]any) int {
	waitTime := browserDefaultTimeout
	if wt, ok := params["wait_time"].(float64); ok && wt > 0 {
		waitTime = int(wt)
	}
	if wt, ok := params["wait_time"].(int); ok && wt > 0 {
		waitTime = wt
	}
	if waitTime > browserMaxTimeout {
		return browserMaxTimeout
	}
	return waitTime
}

func (b *BrowserController) navigate(waitTime int, params map[string]any) (*tool.Result, error) {
	url := strings.TrimSpace(params["url"].(string))
	ctx, cancel := b.withTimeout(waitTime)
	defer cancel()

	var title string
	var currentURL string
	if err := chromedp.Run(ctx,
		chromedp.Navigate(url),
		chromedp.WaitReady("body", chromedp.ByQuery),
		chromedp.Title(&title),
		chromedp.Location(&currentURL),
	); err != nil {
		return tool.NewErrorResultWithMsg(tool.ErrCodeExecution, fmt.Sprintf("导航失败: %v", err)), nil
	}

	return tool.NewResult(map[string]interface{}{
		"success": true,
		"title":   title,
		"url":     currentURL,
	}), nil
}

func (b *BrowserController) click(waitTime int, params map[string]any) (*tool.Result, error) {
	selector := strings.TrimSpace(params["selector"].(string))
	ctx, cancel := b.withTimeout(waitTime)
	defer cancel()

	if err := chromedp.Run(ctx,
		chromedp.WaitReady(selector, chromedp.ByQuery),
		chromedp.Click(selector, chromedp.ByQuery),
	); err != nil {
		return tool.NewErrorResultWithMsg(tool.ErrCodeExecution, fmt.Sprintf("点击失败: %v", err)), nil
	}

	return tool.NewResult(map[string]interface{}{
		"success":  true,
		"action":   "click",
		"selector": selector,
	}), nil
}

func (b *BrowserController) input(waitTime int, params map[string]any) (*tool.Result, error) {
	selector := strings.TrimSpace(params["selector"].(string))
	text := params["text"].(string)
	ctx, cancel := b.withTimeout(waitTime)
	defer cancel()

	selectorJSON, _ := json.Marshal(selector)
	textJSON, _ := json.Marshal(text)
	script := fmt.Sprintf(`(() => {
	const element = document.querySelector(%s);
	if (!element) {
		throw new Error("元素不存在");
	}
	element.focus();
	element.value = %s;
	element.dispatchEvent(new Event("input", { bubbles: true }));
	element.dispatchEvent(new Event("change", { bubbles: true }));
	return true;
})()`, selectorJSON, textJSON)

	if err := chromedp.Run(ctx,
		chromedp.WaitReady(selector, chromedp.ByQuery),
		chromedp.Evaluate(script, nil),
	); err != nil {
		return tool.NewErrorResultWithMsg(tool.ErrCodeExecution, fmt.Sprintf("输入失败: %v", err)), nil
	}

	return tool.NewResult(map[string]interface{}{
		"success":  true,
		"action":   "input",
		"selector": selector,
	}), nil
}

func (b *BrowserController) getText(waitTime int, params map[string]any) (*tool.Result, error) {
	selector := strings.TrimSpace(params["selector"].(string))
	ctx, cancel := b.withTimeout(waitTime)
	defer cancel()

	var text string
	if selector == "title" {
		if err := chromedp.Run(ctx, chromedp.Title(&text)); err != nil {
			return tool.NewErrorResultWithMsg(tool.ErrCodeExecution, fmt.Sprintf("获取文本失败: %v", err)), nil
		}
	} else if err := chromedp.Run(ctx,
		chromedp.WaitReady(selector, chromedp.ByQuery),
		chromedp.Text(selector, &text, chromedp.ByQuery),
	); err != nil {
		return tool.NewErrorResultWithMsg(tool.ErrCodeExecution, fmt.Sprintf("获取文本失败: %v", err)), nil
	}

	// 限制文本内容大小，避免超出LLM上下文限制
	const maxTextLength = 10000
	truncated := false
	if len(text) > maxTextLength {
		text = text[:maxTextLength] + "\n... [内容已截断，原长度: " + fmt.Sprintf("%d", len(text)) + " 字符]"
		truncated = true
	}

	return tool.NewResult(map[string]interface{}{
		"success":   true,
		"selector":  selector,
		"text":      text,
		"truncated": truncated,
	}), nil
}

func (b *BrowserController) getHTML(waitTime int, params map[string]any) (*tool.Result, error) {
	selector, _ := params["selector"].(string)
	selector = strings.TrimSpace(selector)
	if selector == "" {
		selector = "html"
	}

	ctx, cancel := b.withTimeout(waitTime)
	defer cancel()

	var html string
	if err := chromedp.Run(ctx,
		chromedp.WaitReady(selector, chromedp.ByQuery),
		chromedp.OuterHTML(selector, &html, chromedp.ByQuery),
	); err != nil {
		return tool.NewErrorResultWithMsg(tool.ErrCodeExecution, fmt.Sprintf("获取HTML失败: %v", err)), nil
	}

	// 限制HTML内容大小，避免超出LLM上下文限制
	const maxHTMLLength = 50000
	truncated := false
	if len(html) > maxHTMLLength {
		html = html[:maxHTMLLength] + "\n... [内容已截断，原长度: " + fmt.Sprintf("%d", len(html)) + " 字符]"
		truncated = true
	}

	return tool.NewResult(map[string]interface{}{
		"success":   true,
		"selector":  selector,
		"html":      html,
		"truncated": truncated,
	}), nil
}

func (b *BrowserController) screenshot(waitTime int, params map[string]any) (*tool.Result, error) {
	fullPage := false
	if fp, ok := params["full_page"].(bool); ok {
		fullPage = fp
	}

	ctx, cancel := b.withTimeout(waitTime)
	defer cancel()

	var buf []byte
	var err error
	if fullPage {
		err = chromedp.Run(ctx, chromedp.FullScreenshot(&buf, 90))
	} else {
		err = chromedp.Run(ctx, chromedp.CaptureScreenshot(&buf))
	}
	if err != nil {
		return tool.NewErrorResultWithMsg(tool.ErrCodeExecution, fmt.Sprintf("截图失败: %v", err)), nil
	}

	return tool.NewResult(map[string]interface{}{
		"success":   true,
		"full_page": fullPage,
		"image":     base64.StdEncoding.EncodeToString(buf),
		"format":    "png",
	}), nil
}

func (b *BrowserController) executeJS(waitTime int, params map[string]any) (*tool.Result, error) {
	script := strings.TrimSpace(params["script"].(string))
	ctx, cancel := b.withTimeout(waitTime)
	defer cancel()

	var result interface{}
	if err := chromedp.Run(ctx, chromedp.Evaluate(script, &result)); err != nil {
		return tool.NewErrorResultWithMsg(tool.ErrCodeExecution, fmt.Sprintf("执行脚本失败: %v", err)), nil
	}

	return tool.NewResult(map[string]interface{}{
		"success": true,
		"result":  result,
	}), nil
}

func (b *BrowserController) wait(waitTime int, params map[string]any) (*tool.Result, error) {
	selector := strings.TrimSpace(params["selector"].(string))
	ctx, cancel := b.withTimeout(waitTime)
	defer cancel()

	if err := chromedp.Run(ctx, chromedp.WaitVisible(selector, chromedp.ByQuery)); err != nil {
		return tool.NewErrorResultWithMsg(tool.ErrCodeExecution, fmt.Sprintf("等待元素失败: %v", err)), nil
	}

	return tool.NewResult(map[string]interface{}{
		"success":  true,
		"selector": selector,
	}), nil
}

func (b *BrowserController) scroll(waitTime int, params map[string]any) (*tool.Result, error) {
	x := getIntParam(params, "x", 0)
	y := getIntParam(params, "y", 0)
	ctx, cancel := b.withTimeout(waitTime)
	defer cancel()

	script := fmt.Sprintf("window.scrollTo(%d, %d);", x, y)
	if err := chromedp.Run(ctx, chromedp.Evaluate(script, nil)); err != nil {
		return tool.NewErrorResultWithMsg(tool.ErrCodeExecution, fmt.Sprintf("滚动失败: %v", err)), nil
	}

	return tool.NewResult(map[string]interface{}{
		"success": true,
		"x":       x,
		"y":       y,
	}), nil
}

func getIntParam(params map[string]any, key string, defaultValue int) int {
	if value, ok := params[key].(float64); ok {
		return int(value)
	}
	if value, ok := params[key].(int); ok {
		return value
	}
	return defaultValue
}

func (b *BrowserController) isSafeURL(url string) bool {
	lowerURL := strings.ToLower(strings.TrimSpace(url))
	dangerousProtocols := []string{"file://", "javascript:", "data:", "vbscript:"}
	for _, proto := range dangerousProtocols {
		if strings.HasPrefix(lowerURL, proto) {
			return false
		}
	}
	return strings.HasPrefix(lowerURL, "http://") || strings.HasPrefix(lowerURL, "https://")
}

func (b *BrowserController) isSafeSelector(selector string) bool {
	selector = strings.TrimSpace(selector)
	if selector == "" || len(selector) > 500 {
		return false
	}
	dangerous := []string{"<", ";", "&", "|", "`", "$", "\""}
	for _, char := range dangerous {
		if strings.Contains(selector, char) {
			return false
		}
	}
	return browserSafeSelectorPattern.MatchString(selector)
}

func (b *BrowserController) isSafeScript(script string) bool {
	lowerScript := strings.ToLower(strings.TrimSpace(script))
	if lowerScript == "" || len(lowerScript) > 5000 {
		return false
	}
	dangerous := []string{
		"eval(", "function", "settimeout", "setinterval",
		"xmlhttprequest", "fetch(", "websocket",
		"document.write", "document.writeln",
		"localstorage", "sessionstorage", "indexeddb",
		"cookie", "document.cookie",
	}
	for _, d := range dangerous {
		if strings.Contains(lowerScript, d) {
			return false
		}
	}
	return true
}
