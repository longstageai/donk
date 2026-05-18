import 'package:flutter/material.dart';
import '../../common/service/task_service.dart';
import 'task_model.dart';
import 'task_list_view.dart';
import 'task_runs_view.dart';

/// 任务页面
/// 展示任务列表，点击任务可查看运行记录
class TaskView extends StatefulWidget {
  const TaskView({super.key});

  @override
  State<TaskView> createState() => _TaskViewState();
}

class _TaskViewState extends State<TaskView> {
  List<Task> _tasks = [];
  bool _isLoading = true;
  String? _errorMessage;

  @override
  void initState() {
    super.initState();
    _loadTasks();
  }

  Future<void> _loadTasks() async {
    setState(() {
      _isLoading = true;
      _errorMessage = null;
    });

    try {
      final response = await TaskService.getTasks(page: 1, size: 100);
      final items = response['items'] as List<dynamic>? ?? [];
      setState(() {
        _tasks =
            items
                .map((item) => Task.fromJson(item as Map<String, dynamic>))
                .toList();
        _isLoading = false;
      });
    } catch (e) {
      setState(() {
        _tasks = _getMockTasks();
        _isLoading = false;
      });
    }
  }

  List<Task> _getMockTasks() {
    return [
      Task(
        id: '1',
        name: '双子座每日星座运势',
        taskType: 'cron',
        schedule: '0 8 * * *',
        executor: 'agent',
        status: 'running',
        createdAt: DateTime.now().millisecondsSinceEpoch ~/ 1000,
        lastRunAt: DateTime.now().millisecondsSinceEpoch ~/ 1000 - 3600,
        lastExecuteTime: '今天 15:27',
      ),
      Task(
        id: '2',
        name: '每日喝水提醒',
        taskType: 'cron',
        schedule: '0 10 * * *',
        executor: 'agent',
        status: 'running',
        createdAt: DateTime.now().millisecondsSinceEpoch ~/ 1000,
        lastRunAt: DateTime.now().millisecondsSinceEpoch ~/ 1000 - 18000,
        lastExecuteTime: '今天 10:00',
      ),
      Task(
        id: '3',
        name: '科技新闻推送',
        taskType: 'cron',
        schedule: '0 9 * * *',
        executor: 'agent',
        status: 'paused',
        createdAt: DateTime.now().millisecondsSinceEpoch ~/ 1000,
        lastRunAt: DateTime.now().millisecondsSinceEpoch ~/ 1000 - 86400,
        lastExecuteTime: '昨天 9:00',
      ),
    ];
  }

  /// 处理任务点击 - 跳转到运行记录页
  void _handleTaskTap(Task task) {
    Navigator.push(
      context,
      MaterialPageRoute(
        builder:
            (context) => TaskRunsView(taskId: task.id, taskName: task.name),
      ),
    );
  }

  Future<void> _handleTaskToggle(Task task) async {
    // 乐观更新：先更新本地状态
    final originalStatus = task.status;
    final newStatus = task.isEnabled ? 'paused' : 'running';

    setState(() {
      final index = _tasks.indexWhere((t) => t.id == task.id);
      if (index != -1) {
        _tasks[index] = task.copyWith(status: newStatus);
      }
    });

    try {
      if (task.isEnabled) {
        await TaskService.cancelTask(task.id);
      } else {
        await TaskService.triggerTask(task.id);
      }
    } catch (e) {
      // 失败时回滚状态
      setState(() {
        final index = _tasks.indexWhere((t) => t.id == task.id);
        if (index != -1) {
          _tasks[index] = task.copyWith(status: originalStatus);
        }
      });
      if (mounted) {
        ScaffoldMessenger.of(
          context,
        ).showSnackBar(SnackBar(content: Text('操作失败: $e')));
      }
    }
  }

  void _handleTaskExecute(Task task) {
    showDialog(
      context: context,
      builder:
          (context) => AlertDialog(
            title: const Text('立即执行'),
            content: Text('确定要立即执行任务"${task.name}"吗？'),
            actions: [
              TextButton(
                onPressed: () => Navigator.pop(context),
                child: const Text('取消'),
              ),
              TextButton(
                onPressed: () async {
                  final taskName = task.name;
                  final taskId = task.id;
                  final scaffoldMessenger = ScaffoldMessenger.of(context);
                  Navigator.pop(context);
                  try {
                    await TaskService.triggerTask(taskId);
                    if (mounted) {
                      scaffoldMessenger.showSnackBar(
                        SnackBar(content: Text('任务"$taskName"已开始执行')),
                      );
                    }
                    await _loadTasks();
                  } catch (e) {
                    if (mounted) {
                      scaffoldMessenger.showSnackBar(
                        SnackBar(content: Text('执行失败: $e')),
                      );
                    }
                  }
                },
                child: const Text('确定'),
              ),
            ],
          ),
    );
  }

  void _handleTaskRemove(Task task) {
    showDialog(
      context: context,
      builder:
          (context) => AlertDialog(
            title: const Text('移除任务'),
            content: Text('确定要移除任务"${task.name}"吗？此操作不可恢复。'),
            actions: [
              TextButton(
                onPressed: () => Navigator.pop(context),
                child: const Text('取消'),
              ),
              TextButton(
                onPressed: () async {
                  final taskId = task.id;
                  final scaffoldMessenger = ScaffoldMessenger.of(context);
                  Navigator.pop(context);
                  try {
                    await TaskService.deleteTask(taskId);
                    await _loadTasks();
                    if (mounted) {
                      scaffoldMessenger.showSnackBar(
                        const SnackBar(content: Text('任务已移除')),
                      );
                    }
                  } catch (e) {
                    if (mounted) {
                      scaffoldMessenger.showSnackBar(
                        SnackBar(content: Text('移除失败: $e')),
                      );
                    }
                  }
                },
                child: const Text('确定'),
              ),
            ],
          ),
    );
  }

  /// 构建页面标题
  Widget _buildHeader() {
    return Row(
      mainAxisAlignment: MainAxisAlignment.spaceBetween,
      children: [
        const Text(
          '任务管理',
          style: TextStyle(
            fontSize: 20,
            fontWeight: FontWeight.bold,
            color: Colors.black87,
          ),
        ),
        Row(
          children: [
            /// 刷新按钮
            IconButton(
              onPressed: _isLoading ? null : _loadTasks,
              icon: const Icon(Icons.refresh, size: 20),
              tooltip: '刷新列表',
            ),
          ],
        ),
      ],
    );
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      backgroundColor: Colors.white,
      body: Container(
        padding: const EdgeInsets.all(24),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            /// 页面标题
            _buildHeader(),
            const SizedBox(height: 20),

            /// 任务列表
            Expanded(child: _buildTaskList()),
          ],
        ),
      ),
    );
  }

  /// 构建任务列表
  Widget _buildTaskList() {
    if (_isLoading) {
      return const Center(child: CircularProgressIndicator());
    }

    if (_errorMessage != null) {
      return Center(
        child: Column(
          mainAxisAlignment: MainAxisAlignment.center,
          children: [
            const Icon(Icons.error_outline, size: 48, color: Colors.red),
            const SizedBox(height: 16),
            Text(
              '加载失败: $_errorMessage',
              style: const TextStyle(color: Colors.red),
              textAlign: TextAlign.center,
            ),
            const SizedBox(height: 16),
            ElevatedButton(onPressed: _loadTasks, child: const Text('重试')),
          ],
        ),
      );
    }

    return TaskListView(
      tasks: _tasks,
      onTaskToggle: _handleTaskToggle,
      onTaskExecute: _handleTaskExecute,
      onTaskRemove: _handleTaskRemove,
      onTaskTap: _handleTaskTap,
      onRefresh: _loadTasks,
    );
  }
}
