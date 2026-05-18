import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'task_model.dart';

/// 任务列表页面
/// 展示所有定时任务，支持下拉刷新
class TaskListView extends StatefulWidget {
  /// 任务列表数据
  final List<Task> tasks;

  /// 任务开关状态切换回调
  final Function(Task) onTaskToggle;

  /// 立即执行任务回调
  final Function(Task) onTaskExecute;

  /// 移除任务回调
  final Function(Task) onTaskRemove;

  /// 任务点击回调 - 跳转到运行记录页
  final Function(Task)? onTaskTap;

  /// 下拉刷新回调
  final Future<void> Function() onRefresh;

  const TaskListView({
    super.key,
    required this.tasks,
    required this.onTaskToggle,
    required this.onTaskExecute,
    required this.onTaskRemove,
    this.onTaskTap,
    required this.onRefresh,
  });

  @override
  State<TaskListView> createState() => _TaskListViewState();
}

class _TaskListViewState extends State<TaskListView> {
  @override
  Widget build(BuildContext context) {
    // 当任务列表为空时显示空状态视图
    if (widget.tasks.isEmpty) {
      return _buildEmptyView();
    }
    // 任务列表不为空时显示可刷新的列表
    return RefreshIndicator(
      onRefresh: widget.onRefresh,
      child: ListView.builder(
        padding: const EdgeInsets.all(16),
        itemCount: widget.tasks.length,
        itemBuilder: (context, index) {
          return _buildTaskCard(widget.tasks[index]);
        },
      ),
    );
  }

  /// 构建空状态视图
  Widget _buildEmptyView() {
    return Center(
      child: Column(
        mainAxisAlignment: MainAxisAlignment.center,
        children: [
          // 空状态图标
          Container(
            width: 80,
            height: 80,
            decoration: BoxDecoration(
              color: const Color(0xFFF5F5F5),
              borderRadius: BorderRadius.circular(40),
            ),
            child: const Icon(Icons.task_alt, size: 40, color: Colors.grey),
          ),
          const SizedBox(height: 16),
          const Text(
            '暂无任务',
            style: TextStyle(fontSize: 16, color: Colors.grey),
          ),
          const SizedBox(height: 8),
          const Text(
            '点击下方按钮创建新任务',
            style: TextStyle(fontSize: 14, color: Colors.grey),
          ),
        ],
      ),
    );
  }

  /// 构建任务卡片
  Widget _buildTaskCard(Task task) {
    return MouseRegion(
      cursor: SystemMouseCursors.click,
      child: GestureDetector(
        onTap: () => widget.onTaskTap?.call(task),
        child: Container(
          margin: const EdgeInsets.only(bottom: 12),
          decoration: BoxDecoration(
            color: const Color(0xFFF8F8F8),
            borderRadius: BorderRadius.circular(8),
            border: Border.all(color: const Color(0xFFEEEEEE)),
            boxShadow: [
              BoxShadow(
                color: Colors.black.withValues(alpha: 0.05),
                blurRadius: 4,
                offset: const Offset(0, 2),
              ),
            ],
          ),
          child: Padding(
            padding: const EdgeInsets.all(16),
            child: Row(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                // 左侧：图标
                Container(
                  width: 44,
                  height: 44,
                  decoration: BoxDecoration(
                    color: const Color(0xFFF5F5F5),
                    borderRadius: BorderRadius.circular(10),
                  ),
                  child: Center(
                    child: Text(
                      task.icon,
                      style: const TextStyle(fontSize: 22),
                    ),
                  ),
                ),
                const SizedBox(width: 12),
                // 中间：任务信息
                Expanded(
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      // 任务名称
                      Text(
                        task.name,
                        style: const TextStyle(
                          fontSize: 15,
                          fontWeight: FontWeight.w600,
                          color: Colors.black87,
                        ),
                      ),
                      const SizedBox(height: 6),
                      // 任务ID（可点击复制）
                      GestureDetector(
                        onTap: () => _copyToClipboard(task.id, '任务ID已复制'),
                        child: Row(
                          mainAxisSize: MainAxisSize.min,
                          children: [
                            const Text(
                              '# ',
                              style: TextStyle(
                                fontSize: 12,
                                color: Colors.grey,
                              ),
                            ),
                            Text(
                              _truncateId(task.id),
                              style: const TextStyle(
                                fontSize: 12,
                                color: Colors.grey,
                                fontFamily: 'monospace',
                              ),
                            ),
                            const SizedBox(width: 4),
                            const Icon(
                              Icons.copy,
                              size: 12,
                              color: Colors.grey,
                            ),
                          ],
                        ),
                      ),
                      const SizedBox(height: 10),
                      // 执行器和类型标签
                      Row(
                        children: [
                          // 执行器图标和名称
                          Row(
                            mainAxisSize: MainAxisSize.min,
                            children: [
                              Icon(
                                _getExecutorIcon(task.executor),
                                size: 14,
                                color: Colors.grey[600],
                              ),
                              const SizedBox(width: 4),
                              Text(
                                task.executor,
                                style: TextStyle(
                                  fontSize: 12,
                                  color: Colors.grey[600],
                                ),
                              ),
                            ],
                          ),
                          const SizedBox(width: 8),
                          // 任务类型标签
                          Container(
                            padding: const EdgeInsets.symmetric(
                              horizontal: 8,
                              vertical: 2,
                            ),
                            decoration: BoxDecoration(
                              color: const Color(0xFFF0F0F0),
                              borderRadius: BorderRadius.circular(4),
                            ),
                            child: Text(
                              task.displayTaskType,
                              style: const TextStyle(
                                fontSize: 11,
                                color: Colors.grey,
                              ),
                            ),
                          ),
                          const SizedBox(width: 8),
                          // 状态标签
                          Container(
                            padding: const EdgeInsets.symmetric(
                              horizontal: 8,
                              vertical: 2,
                            ),
                            decoration: BoxDecoration(
                              color: _getStatusBgColor(task.status),
                              borderRadius: BorderRadius.circular(4),
                            ),
                            child: Text(
                              task.displayStatus,
                              style: TextStyle(
                                fontSize: 11,
                                color: _getStatusColor(task.status),
                              ),
                            ),
                          ),
                          const SizedBox(width: 8),
                          // 时间信息
                          _buildTimeInfoRow(
                            Icons.schedule,
                            task.displaySchedule,
                          ),
                          const SizedBox(width: 8),
                          _buildTimeInfoRow(
                            Icons.play_circle_outline,
                            '下次: ${task.displayNextRunTime}',
                          ),
                          if (task.lastRunAt != null) ...[
                            const SizedBox(width: 8),
                            _buildTimeInfoRow(
                              Icons.history,
                              '上次: ${task.displayLastExecuteTime}',
                            ),
                          ],
                          const SizedBox(width: 8),
                          _buildTimeInfoRow(
                            Icons.calendar_today,
                            _formatDateShort(
                              DateTime.fromMillisecondsSinceEpoch(
                                task.createdAt * 1000,
                              ),
                            ),
                          ),
                          const SizedBox(width: 8),
                        ],
                      ),
                    ],
                  ),
                ),
                const SizedBox(width: 12),
                // 右侧：开关和时间信息
                Column(
                  crossAxisAlignment: CrossAxisAlignment.end,
                  children: [
                    // 开关
                    Transform.scale(
                      scale: 0.75,
                      child: Switch(
                        value: task.isEnabled,
                        onChanged: (value) => widget.onTaskToggle(task),
                        activeColor: const Color(0xFF07C160),
                        inactiveThumbColor: Colors.white,
                        inactiveTrackColor: Colors.grey.shade300,
                      ),
                    ),
                    const SizedBox(height: 8),
                    // 更多菜单按钮
                    _buildMoreMenu(task),
                  ],
                ),
              ],
            ),
          ),
        ),
      ),
    );
  }

  Widget _buildTimeInfoRow(IconData icon, String text) {
    return Row(
      mainAxisSize: MainAxisSize.min,
      children: [
        Icon(icon, size: 12, color: Colors.grey[400]),
        const SizedBox(width: 4),
        Text(text, style: TextStyle(fontSize: 11, color: Colors.grey[500])),
      ],
    );
  }

  IconData _getExecutorIcon(String executor) {
    switch (executor) {
      case 'script':
        return Icons.code;
      case 'api':
        return Icons.api;
      case 'agent':
        return Icons.smart_toy;
      default:
        return Icons.device_unknown;
    }
  }

  Color _getStatusColor(String status) {
    switch (status) {
      case 'running':
        return const Color(0xFF2196F3);
      case 'pending':
        return const Color(0xFFFF9800);
      case 'paused':
        return const Color(0xFF9E9E9E);
      case 'completed':
        return const Color(0xFF4CAF50);
      case 'failed':
        return const Color(0xFFE53935);
      default:
        return Colors.grey;
    }
  }

  Color _getStatusBgColor(String status) {
    switch (status) {
      case 'running':
        return const Color(0xFFE3F2FD);
      case 'pending':
        return const Color(0xFFFFF3E0);
      case 'paused':
        return const Color(0xFFF5F5F5);
      case 'completed':
        return const Color(0xFFE8F5E9);
      case 'failed':
        return const Color(0xFFFFEBEE);
      default:
        return const Color(0xFFF5F5F5);
    }
  }

  String _truncateId(String id) {
    if (id.length <= 10) return id;
    return '${id.substring(0, 10)}...';
  }

  String _formatDateShort(DateTime dateTime) {
    return '${dateTime.year}/${dateTime.month.toString().padLeft(2, '0')}/${dateTime.day.toString().padLeft(2, '0')}';
  }

  /// 构建更多操作菜单
  Widget _buildMoreMenu(Task task) {
    return PopupMenuButton<String>(
      padding: EdgeInsets.zero,
      icon: Icon(Icons.more_vert, size: 18, color: Colors.grey[400]),
      onSelected: (value) {
        switch (value) {
          case 'execute':
            widget.onTaskExecute(task);
            break;
          case 'remove':
            widget.onTaskRemove(task);
            break;
        }
      },
      itemBuilder:
          (context) => [
            const PopupMenuItem(
              value: 'execute',
              child: Row(
                children: [
                  Icon(Icons.play_arrow, size: 18),
                  SizedBox(width: 8),
                  Text('立即执行'),
                ],
              ),
            ),
            const PopupMenuItem(
              value: 'remove',
              child: Row(
                children: [
                  Icon(Icons.delete_outline, size: 18, color: Colors.red),
                  SizedBox(width: 8),
                  Text('移除任务', style: TextStyle(color: Colors.red)),
                ],
              ),
            ),
          ],
    );
  }

  /// 复制文本到剪贴板并显示提示
  void _copyToClipboard(String text, String message) {
    Clipboard.setData(ClipboardData(text: text));
    ScaffoldMessenger.of(context).showSnackBar(
      SnackBar(content: Text(message), duration: const Duration(seconds: 2)),
    );
  }
}
