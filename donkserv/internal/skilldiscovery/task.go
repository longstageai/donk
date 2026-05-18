// skilldiscovery 技能自动发现模块
// 任务定义
package skilldiscovery

import "time"

// DiscoveryTask 技能发现任务
// 用于定时执行技能发现
type DiscoveryTask struct {
	ID        string
	Name      string
	StartTime time.Time
}

// TaskResult 任务执行结果
type TaskResult struct {
	Output   string
	Error    string
	ExitCode int
	Duration int64
	DoneAt   int64
}
