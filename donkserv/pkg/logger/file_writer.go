package logger

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// FileConfig 文件写入器配置结构体
type FileConfig struct {
	Dir           string // 日志文件目录
	Level         Level  // 日志级别
	MaxSize       int64  // 单个日志文件最大大小（字节）
	MaxAge        int    // 日志文件保留天数
	MaxBackups    int    // 保留的备份文件数量
	Compress      bool   // 是否压缩备份文件
	RotateEnabled bool   // 是否启用日志轮转
}

// fileWriter 文件日志写入器
type fileWriter struct {
	mu          sync.Mutex // 互斥锁，保证线程安全
	config      FileConfig // 配置信息
	file        *os.File   // 当前打开的日志文件
	currentDate string     // 当前日志文件的日期
	filename    string     // 当前日志文件路径
}

// NewFileWriter 创建文件写入器
// config: 文件写入器配置
// 返回: 文件写入器实例和可能的错误
func NewFileWriter(config FileConfig) (*fileWriter, error) {
	// 设置默认日志目录
	if config.Dir == "" {
		config.Dir = "./logs"
	}

	// 创建日志根目录
	if err := os.MkdirAll(config.Dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create log directory: %w", err)
	}

	w := &fileWriter{
		config:      config,
		currentDate: time.Now().Format("2006-01-02"),
	}

	// 初始化时创建日志文件
	if err := w.rotateFile(); err != nil {
		return nil, err
	}

	return w, nil
}

// getFilePath 获取日志文件路径
// 路径格式: {目录}/{日期}/{级别}.log
// 例如: logs/2026-03-18/info.log
func (w *fileWriter) getFilePath() string {
	levelName := strings.ToLower(w.config.Level.String())
	dateStr := time.Now().Format("2006-01-02")
	return filepath.Join(w.config.Dir, dateStr, fmt.Sprintf("%s.log", levelName))
}

// rotateFile 执行日志文件轮转
// 按日期轮转，或者当文件大小超过MaxSize时创建新文件
func (w *fileWriter) rotateFile() error {
	newPath := w.getFilePath()
	newDate := time.Now().Format("2006-01-02")

	// 获取日志文件所在目录并创建
	levelDir := filepath.Dir(newPath)
	if err := os.MkdirAll(levelDir, 0755); err != nil {
		return fmt.Errorf("failed to create level directory: %w", err)
	}

	if w.file != nil {
		// 检查是否需要按日期轮转
		if w.currentDate != newDate {
			w.file.Close()
			w.file = nil
			// 清理旧文件
			w.cleanupOldFiles(levelDir)
		} else if w.config.RotateEnabled && w.config.MaxSize > 0 {
			// 检查是否需要按大小轮转
			stat, err := w.file.Stat()
			if err == nil && stat.Size() >= w.config.MaxSize {
				w.file.Close()
				w.file = nil
				// 移动当前文件为备份
				backupPath := newPath + "." + time.Now().Format("2006-01-02-15-04-05") + ".bak"
				os.Rename(newPath, backupPath)
			}
		}
	}

	// 如果文件未打开，则打开或创建
	if w.file == nil {
		flag := os.O_APPEND | os.O_CREATE | os.O_WRONLY
		perm := os.FileMode(0644)

		file, err := os.OpenFile(newPath, flag, perm)
		if err != nil {
			return fmt.Errorf("failed to open log file: %w", err)
		}
		w.file = file
		w.filename = newPath
		w.currentDate = newDate
	}

	return nil
}

// cleanupOldFiles 清理过期的日志文件
// 根据MaxAge和MaxBackups配置删除旧文件
func (w *fileWriter) cleanupOldFiles(dir string) {
	// 如果没有配置清理规则，直接返回
	if w.config.MaxBackups <= 0 && w.config.MaxAge <= 0 {
		return
	}

	// 读取目录下的所有文件
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}

	var files []os.FileInfo
	for _, entry := range entries {
		if info, err := entry.Info(); err == nil {
			files = append(files, info)
		}
	}

	// 计算过期时间点
	cutoff := time.Now().AddDate(0, 0, -w.config.MaxAge)

	// 遍历文件，判断是否需要删除
	for _, file := range files {
		if file.IsDir() {
			continue
		}

		shouldDelete := false

		// 根据MaxAge判断是否过期
		if w.config.MaxAge > 0 && file.ModTime().Before(cutoff) {
			shouldDelete = true
		}

		// 根据MaxBackups判断是否超过保留数量
		if w.config.MaxBackups > 0 {
			count := 0
			for _, f := range files {
				if !f.IsDir() && strings.HasPrefix(f.Name(), filepath.Base(w.filename)) {
					count++
				}
			}
			if count > w.config.MaxBackups {
				shouldDelete = true
			}
		}

		// 删除过期文件
		if shouldDelete {
			os.Remove(filepath.Join(dir, file.Name()))
		}
	}
}

// Write 写入日志记录
// 只有当日志级别与配置的级别匹配时才写入
func (w *fileWriter) Write(record *Record) error {
	// 级别过滤：只写入匹配级别的日志
	if record.Level != w.config.Level {
		return nil
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	// 检查并执行文件轮转
	if err := w.rotateFile(); err != nil {
		return err
	}

	// 写入日志内容
	_, err := w.file.WriteString(record.Message + "\n")
	return err
}

// Close 关闭文件写入器
// 关闭底层的文件句柄
func (w *fileWriter) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.file != nil {
		err := w.file.Close()
		w.file = nil
		return err
	}
	return nil
}
