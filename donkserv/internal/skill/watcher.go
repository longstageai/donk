package skill

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/longstageai/donk/donk/pkg/logger"

	"github.com/fsnotify/fsnotify"
)

// Watcher Skill 文件监听器
type Watcher struct {
	watcher    *fsnotify.Watcher
	loader     *SkillLoader
	repo       *StateRepository
	registry   *SkillRegistry
	skillDir   string
	stopCh     chan struct{}
	syncTimers map[string]*time.Timer
	timersMu   sync.Mutex
}

// NewWatcher 创建 Skill 文件监听器
// 参数:
//   - skillDir: Skill 根目录
//   - loader: Skill 加载器
//   - repo: 状态仓库
//   - registry: Skill 注册表（可选）
//
// 返回:
//   - *Watcher: 监听器实例
//   - error: 创建错误
func NewWatcher(skillDir string, loader *SkillLoader, repo *StateRepository, registry *SkillRegistry) (*Watcher, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	return &Watcher{
		watcher:    watcher,
		loader:     loader,
		repo:       repo,
		registry:   registry,
		skillDir:   skillDir,
		stopCh:     make(chan struct{}),
		syncTimers: make(map[string]*time.Timer),
	}, nil
}

// Start 开始监听
// 监听 skills 根目录和所有子目录
// 返回:
//   - error: 启动错误
func (w *Watcher) Start() error {
	// 检查目录是否存在
	if _, err := os.Stat(w.skillDir); os.IsNotExist(err) {
		return nil // 目录不存在，静默返回
	}

	// 监听 skills 根目录
	if err := w.watcher.Add(w.skillDir); err != nil {
		return err
	}

	// 递归监听所有已有的 Skill 目录
	entries, err := os.ReadDir(w.skillDir)
	if err == nil {
		for _, entry := range entries {
			if entry.IsDir() {
				skillPath := filepath.Join(w.skillDir, entry.Name())
				w.watcher.Add(skillPath)

				// 监听 scripts, references, assets 子目录
				w.addSubDirs(skillPath)
			}
		}
	}

	go w.run()
	return nil
}

// addSubDirs 添加子目录监听
func (w *Watcher) addSubDirs(skillPath string) {
	subDirs := []string{"scripts", "references", "assets"}
	for _, dir := range subDirs {
		path := filepath.Join(skillPath, dir)
		if info, err := os.Stat(path); err == nil && info.IsDir() {
			w.watcher.Add(path)
		}
	}
}

// run 监听循环
func (w *Watcher) run() {
	for {
		select {
		case event, ok := <-w.watcher.Events:
			if !ok {
				return
			}
			w.handleEvent(event)

		case err, ok := <-w.watcher.Errors:
			if !ok {
				return
			}
			logger.Error("SkillWatcher 监听错误", map[string]interface{}{"error": err.Error()})

		case <-w.stopCh:
			return
		}
	}
}

// handleEvent 处理文件事件
func (w *Watcher) handleEvent(event fsnotify.Event) {
	// 获取相对路径
	relPath, _ := filepath.Rel(w.skillDir, event.Name)
	parts := strings.Split(relPath, string(filepath.Separator))

	if len(parts) == 0 {
		return
	}

	skillName := parts[0]

	// 处理不同事件
	switch {
	case event.Op&fsnotify.Write == fsnotify.Write:
		// 文件被修改
		w.handleWrite(event.Name, skillName)

	case event.Op&fsnotify.Create == fsnotify.Create:
		// 文件或目录被创建
		w.handleCreate(event.Name, skillName)

	case event.Op&fsnotify.Remove == fsnotify.Remove:
		// 文件或目录被删除
		w.handleRemove(event.Name, skillName)

	case event.Op&fsnotify.Rename == fsnotify.Rename:
		// 文件被重命名
		w.handleRemove(event.Name, skillName)
	}
}

// handleWrite 处理文件修改（带防抖）
func (w *Watcher) handleWrite(path, skillName string) {
	// 忽略以 . 开头的隐藏目录（如安装器的临时目录 .install-xxx）
	if strings.HasPrefix(skillName, ".") {
		return
	}

	// 只关注 SKILL.md 的修改
	if filepath.Base(path) != "SKILL.md" {
		return
	}

	logger.Infof("SkillWatcher SKILL.md 被修改: %s，等待 3 秒后同步", skillName)

	// 防抖：等待 3 秒，期间如果有新事件则重置定时器
	w.debounceSync(skillName)
}

// handleCreate 处理创建事件（带防抖）
func (w *Watcher) handleCreate(path, skillName string) {
	// 忽略以 . 开头的隐藏目录（如安装器的临时目录 .install-xxx）
	if strings.HasPrefix(skillName, ".") {
		return
	}

	info, err := os.Stat(path)
	if err != nil {
		return
	}

	if info.IsDir() {
		// 新目录被创建，添加到监听
		w.watcher.Add(path)

		// 检查是否是新的 Skill 目录
		if filepath.Dir(path) == w.skillDir {
			logger.Infof("SkillWatcher 新 Skill 被创建: %s，等待 3 秒后同步", skillName)
			w.debounceSync(skillName)
		}
	} else if filepath.Base(path) == "SKILL.md" {
		// 新的 SKILL.md 被创建
		logger.Infof("SkillWatcher SKILL.md 被创建: %s，等待 3 秒后同步", skillName)
		w.debounceSync(skillName)
	}
}

// debounceSync 防抖同步
// 等待 10 秒，期间如果同一 Skill 有新事件则重置定时器
func (w *Watcher) debounceSync(skillName string) {
	if !isValidSkillName(skillName) {
		return
	}

	w.timersMu.Lock()
	defer w.timersMu.Unlock()

	// 如果已有定时器，停止并删除
	if timer, exists := w.syncTimers[skillName]; exists {
		timer.Stop()
		delete(w.syncTimers, skillName)
	}

	// 创建新定时器
	timer := time.AfterFunc(3*time.Second, func() {
		skillPath := filepath.Join(w.skillDir, skillName)
		w.syncSkill(skillName, skillPath)

		// 同步完成后删除定时器
		w.timersMu.Lock()
		delete(w.syncTimers, skillName)
		w.timersMu.Unlock()
	})

	w.syncTimers[skillName] = timer
}

// handleRemove 处理删除事件（立即执行，不防抖）
func (w *Watcher) handleRemove(path, skillName string) {
	// 忽略以 . 开头的隐藏目录（如安装器的临时目录 .install-xxx）
	if strings.HasPrefix(skillName, ".") {
		return
	}

	// 检查是否是 Skill 目录被删除
	if filepath.Dir(path) == w.skillDir {
		logger.Infof("SkillWatcher Skill 被删除: %s", skillName)

		// 取消该 Skill 的防抖定时器（如果有）
		w.timersMu.Lock()
		if timer, exists := w.syncTimers[skillName]; exists {
			timer.Stop()
			delete(w.syncTimers, skillName)
		}
		w.timersMu.Unlock()

		// 从数据库删除记录
		if err := w.repo.Delete(skillName); err != nil {
			logger.Error("SkillWatcher 删除 Skill 记录失败", map[string]interface{}{"skill": skillName, "error": err.Error()})
		}
	}
}

// syncSkill 同步单个 Skill 到数据库
func (w *Watcher) syncSkill(name, dir string) {
	// 检查 SKILL.md 是否存在
	skillFile := filepath.Join(dir, "SKILL.md")
	if _, err := os.Stat(skillFile); err != nil {
		if os.IsNotExist(err) {
			if err := w.repo.Delete(name); err != nil {
				logger.Error("SkillWatcher 删除缺失 Skill 记录失败", map[string]interface{}{"skill": name, "error": err.Error()})
			} else {
				logger.Infof("SkillWatcher 缺失 Skill 已从数据库删除: %s", name)
			}
			return
		}
		logger.Error("SkillWatcher 检查 SKILL.md 失败", map[string]interface{}{"skill": name, "error": err.Error()})
		return
	}

	// 加载 Skill
	skill, err := w.loader.LoadFromDir(dir)
	if err != nil {
		logger.Error("SkillWatcher 加载 Skill 失败", map[string]interface{}{"skill": name, "error": err.Error()})
		return
	}

	// 保存到数据库
	if err := w.repo.Save(skill.Name(), skill.Description(), true); err != nil {
		logger.Error("SkillWatcher 同步 Skill 失败", map[string]interface{}{"skill": name, "error": err.Error()})
		return
	}
	logger.Infof("SkillWatcher Skill 已同步: %s", name)

	// 注册到 SkillRegistry（如果提供了 registry）
	if w.registry != nil {
		if err := w.registry.Register(skill); err != nil {
			logger.Warn("SkillWatcher 注册 Skill 到 registry 失败", map[string]interface{}{"skill": name, "error": err.Error()})
		} else {
			logger.Infof("SkillWatcher Skill 已注册到 registry: %s", name)
		}
	}

	// 添加子目录监听
	w.addSubDirs(dir)
}

// Stop 停止监听
func (w *Watcher) Stop() {
	// 停止所有定时器
	w.timersMu.Lock()
	for _, timer := range w.syncTimers {
		timer.Stop()
	}
	w.syncTimers = make(map[string]*time.Timer)
	w.timersMu.Unlock()

	close(w.stopCh)
	w.watcher.Close()
}
