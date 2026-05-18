package setting

import (
	"database/sql"
	"fmt"
	"strconv"
	"sync"
	"syscall"
)

var (
	kernel32               = syscall.NewLazyDLL("kernel32.dll")
	procSetThreadExecState = kernel32.NewProc("SetThreadExecutionState")
)

// 执行状态标志
const (
	ES_AWAYMODE_REQUIRED = 0x00000040 // 离开模式（允许系统假眠但保持运行）
	ES_CONTINUOUS        = 0x80000000 // 持续生效，直到被清除
	ES_DISPLAY_REQUIRED  = 0x00000002 // 保持显示器开启
	ES_SYSTEM_REQUIRED   = 0x00000001 // 保持系统运行（阻止睡眠）
)

// 系统状态键名常量
const (
	// SleepPreventedKey 睡眠阻止状态键名
	SleepPreventedKey = "sleep_prevented"
	// SleepKeepDisplayKey 显示器保持状态键名
	SleepKeepDisplayKey = "sleep_keep_display"
)

// SleepManager 系统睡眠管理器
// 用于控制阻止或允许系统进入睡眠状态
// 支持状态持久化，程序重启后可恢复之前的设置
// 注意：程序退出时，Windows 会自动恢复系统睡眠状态
type SleepManager struct {
	mu          sync.RWMutex
	isActive    bool    // 当前是否阻止睡眠
	keepDisplay bool    // 是否保持显示器开启
	db          *sql.DB // 数据库连接，用于持久化状态
}

// NewSleepManager 创建睡眠管理器
//
// 参数:
//   - db: 数据库连接，用于持久化状态，可为 nil
//
// 返回:
//   - *SleepManager: 睡眠管理器实例
func NewSleepManager(db *sql.DB) *SleepManager {
	return &SleepManager{
		isActive:    false,
		keepDisplay: false,
		db:          db,
	}
}

// SetDB 设置数据库连接
// 用于在初始化后设置数据库连接
//
// 参数:
//   - db: 数据库连接
func (m *SleepManager) SetDB(db *sql.DB) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.db = db
}

// Prevent 阻止系统进入睡眠
// 使用 Windows API SetThreadExecutionState 临时阻止系统睡眠
// 程序退出时，Windows 会自动恢复睡眠状态
// 同时会将状态保存到数据库，便于程序重启后恢复
//
// 参数:
//   - keepDisplay: 是否保持显示器开启，true 则显示器不会关闭
//
// 返回:
//   - error: 错误信息，成功返回 nil
//
// 示例:
//
//	sm := setting.NewSleepManager(db)
//	err := sm.Prevent(true)  // 阻止睡眠，保持显示器开启
func (m *SleepManager) Prevent(keepDisplay bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 调用 Windows API 设置执行状态
	var flags uintptr = ES_CONTINUOUS | ES_SYSTEM_REQUIRED
	if keepDisplay {
		flags |= ES_DISPLAY_REQUIRED
	}

	ret, _, err := procSetThreadExecState.Call(flags)
	if ret == 0 {
		return fmt.Errorf("设置执行状态失败: %v", err)
	}

	m.isActive = true
	m.keepDisplay = keepDisplay

	// 持久化状态到数据库
	if m.db != nil {
		m.saveStateToDB(true, keepDisplay)
	}

	return nil
}

// Allow 允许系统进入睡眠
// 清除之前设置的阻止睡眠状态，恢复系统默认行为
// 程序退出时会自动调用，也可以手动调用提前恢复
// 同时会清除数据库中的状态记录
//
// 返回:
//   - error: 错误信息，成功返回 nil
//
// 示例:
//
//	sm := setting.NewSleepManager(db)
//	sm.Prevent(true)
//	defer sm.Allow()  // 提前恢复睡眠
func (m *SleepManager) Allow() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 调用 Windows API 清除执行状态
	ret, _, err := procSetThreadExecState.Call(uintptr(ES_CONTINUOUS))
	if ret == 0 {
		return fmt.Errorf("清除执行状态失败: %v", err)
	}

	m.isActive = false
	m.keepDisplay = false

	// 清除数据库中的状态
	if m.db != nil {
		m.saveStateToDB(false, false)
	}

	return nil
}

// saveStateToDB 将睡眠状态保存到数据库
//
// 参数:
//   - isActive: 是否阻止睡眠
//   - keepDisplay: 是否保持显示器开启
func (m *SleepManager) saveStateToDB(isActive, keepDisplay bool) {
	// 保存阻止状态
	_, _ = m.db.Exec(
		`INSERT INTO system_state (key, value, updated_at) VALUES (?, ?, datetime('now'))
		ON CONFLICT(key) DO UPDATE SET value = ?, updated_at = datetime('now')`,
		SleepPreventedKey, strconv.FormatBool(isActive), strconv.FormatBool(isActive),
	)

	// 保存显示器状态
	_, _ = m.db.Exec(
		`INSERT INTO system_state (key, value, updated_at) VALUES (?, ?, datetime('now'))
		ON CONFLICT(key) DO UPDATE SET value = ?, updated_at = datetime('now')`,
		SleepKeepDisplayKey, strconv.FormatBool(keepDisplay), strconv.FormatBool(keepDisplay),
	)
}

// LoadStateFromDB 从数据库加载睡眠状态并应用
// 通常在程序启动时调用，恢复之前的设置
//
// 返回:
//   - error: 错误信息，成功返回 nil
func (m *SleepManager) LoadStateFromDB() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.db == nil {
		return fmt.Errorf("数据库连接未设置")
	}

	// 查询阻止状态
	var preventedValue string
	err := m.db.QueryRow("SELECT value FROM system_state WHERE key = ?", SleepPreventedKey).Scan(&preventedValue)
	if err == sql.ErrNoRows {
		// 没有记录，使用默认状态（不阻止）
		return nil
	}
	if err != nil {
		return fmt.Errorf("查询睡眠状态失败: %w", err)
	}

	prevented, _ := strconv.ParseBool(preventedValue)
	if !prevented {
		// 之前未阻止睡眠，无需恢复
		return nil
	}

	// 查询显示器状态
	var keepDisplayValue string
	_ = m.db.QueryRow("SELECT value FROM system_state WHERE key = ?", SleepKeepDisplayKey).Scan(&keepDisplayValue)
	keepDisplay, _ := strconv.ParseBool(keepDisplayValue)

	// 应用状态：阻止睡眠
	var flags uintptr = ES_CONTINUOUS | ES_SYSTEM_REQUIRED
	if keepDisplay {
		flags |= ES_DISPLAY_REQUIRED
	}

	ret, _, err := procSetThreadExecState.Call(flags)
	if ret == 0 {
		return fmt.Errorf("恢复睡眠状态失败: %v", err)
	}

	m.isActive = true
	m.keepDisplay = keepDisplay

	return nil
}

// IsActive 返回当前是否阻止睡眠
//
// 返回:
//   - bool: true 表示正在阻止睡眠，false 表示允许睡眠
func (m *SleepManager) IsActive() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.isActive
}

// IsKeepDisplay 返回是否保持显示器开启
//
// 返回:
//   - bool: true 表示保持显示器开启
func (m *SleepManager) IsKeepDisplay() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.keepDisplay
}

// Status 返回睡眠管理器的完整状态
//
// 返回:
//   - SleepStatus: 包含当前状态的结构体
func (m *SleepManager) Status() SleepStatus {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return SleepStatus{
		IsActive:    m.isActive,
		KeepDisplay: m.keepDisplay,
	}
}

// SleepStatus 睡眠管理器状态
type SleepStatus struct {
	IsActive    bool `json:"is_active"`    // 是否阻止睡眠
	KeepDisplay bool `json:"keep_display"` // 是否保持显示器开启
}

// globalSleepManager 全局睡眠管理器实例
var globalSleepManager *SleepManager
var globalSleepManagerOnce sync.Once

// GetSleepManager 获取全局睡眠管理器（单例）
// 首次调用时可传入数据库连接进行初始化
//
// 参数:
//   - db: 数据库连接，可选，仅在首次调用时有效
//
// 返回:
//   - *SleepManager: 全局睡眠管理器实例
//
// 示例:
//
//	sm := setting.GetSleepManager(db)  // 首次调用，设置数据库
//	sm := setting.GetSleepManager(nil) // 后续调用，获取实例
func GetSleepManager(db ...*sql.DB) *SleepManager {
	globalSleepManagerOnce.Do(func() {
		var database *sql.DB
		if len(db) > 0 {
			database = db[0]
		}
		globalSleepManager = NewSleepManager(database)
	})
	return globalSleepManager
}

// PreventSleep 便捷函数：阻止系统睡眠（使用全局管理器）
//
// 参数:
//   - keepDisplay: 是否保持显示器开启
//
// 返回:
//   - error: 错误信息
func PreventSleep(keepDisplay bool) error {
	return GetSleepManager().Prevent(keepDisplay)
}

// AllowSleep 便捷函数：允许系统睡眠（使用全局管理器）
//
// 返回:
//   - error: 错误信息
func AllowSleep() error {
	return GetSleepManager().Allow()
}

// IsSleepPrevented 便捷函数：检查是否正在阻止睡眠（使用全局管理器）
//
// 返回:
//   - bool: true 表示正在阻止睡眠
func IsSleepPrevented() bool {
	return GetSleepManager().IsActive()
}
