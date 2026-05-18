// knowledge 知识库模块
package knowledge

import (
	"fmt"
)

// Config 知识库配置
type Config struct {
	Enabled     bool     `yaml:"enabled"`       // 是否启用
	Interval    int      `yaml:"interval"`      // 扫描间隔（秒）
	BatchSize   int      `yaml:"batch_size"`    // 每批处理数量
	SleepMs     int      `yaml:"sleep_ms"`      // 处理间隔（毫秒）
	MaxDepth    int      `yaml:"max_depth"`     // 最大扫描深度
	MaxFileSize int64    `yaml:"max_file_size"` // 最大文件大小（字节）
	HotDays     int      `yaml:"hot_days"`      // 热数据天数
	WarmDays    int      `yaml:"warm_days"`     // 温数据天数
	Directories []string `yaml:"directories"`   // 扫描目录（空则使用默认）
}

// DefaultConfig 返回默认配置
func DefaultConfig() *Config {
	return &Config{
		Enabled:     true,
		Interval:    3600,             // 每小时扫描一次
		BatchSize:   50,               // 每批50个文件
		SleepMs:     100,              // 100ms间隔
		MaxDepth:    3,                // 3层深度
		MaxFileSize: 10 * 1024 * 1024, // 10MB
		HotDays:     7,                // 7天热数据
		WarmDays:    30,               // 30天温数据
		Directories: []string{},       // 使用默认目录
	}
}

// ToBuilderConfig 转换为BuilderConfig
func (c *Config) ToBuilderConfig() *BuilderConfig {
	return &BuilderConfig{
		Enabled:     c.Enabled,
		Interval:    c.Interval,
		BatchSize:   c.BatchSize,
		SleepMs:     c.SleepMs,
		MaxDepth:    c.MaxDepth,
		MaxFileSize: c.MaxFileSize,
		HotDays:     c.HotDays,
		WarmDays:    c.WarmDays,
		Directories: c.Directories,
	}
}

// Validate 验证配置
func (c *Config) Validate() error {
	if c.Interval < 60 {
		return fmt.Errorf("扫描间隔不能小于60秒")
	}
	if c.BatchSize < 1 {
		return fmt.Errorf("批处理大小不能小于1")
	}
	if c.MaxDepth < 1 {
		return fmt.Errorf("最大深度不能小于1")
	}
	if c.MaxFileSize < 1024 {
		return fmt.Errorf("最大文件大小不能小于1KB")
	}
	return nil
}
