import 'package:flutter/material.dart';
import '../../common/service/task_service.dart';
import '../../l10n/generated/app_localizations.dart';
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
      if (!mounted) return;
      setState(() {
        _tasks =
            items
                .map((item) => Task.fromJson(item as Map<String, dynamic>))
                .toList();
        _isLoading = false;
      });
    } catch (e) {
      if (!mounted) return;
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
      if (!mounted) return;
      setState(() {
        final index = _tasks.indexWhere((t) => t.id == task.id);
        if (index != -1) {
          _tasks[index] = task.copyWith(status: originalStatus);
        }
      });
      if (mounted) {
        final l10n = AppLocalizations.of(context)!;
        ScaffoldMessenger.of(
          context,
        ).showSnackBar(SnackBar(content: Text('${l10n.operationFailed}: $e')));
      }
    }
  }

  void _handleTaskExecute(Task task) {
    final l10n = AppLocalizations.of(context)!;
    showDialog(
      context: context,
      builder:
          (context) => AlertDialog(
            title: Text(l10n.executeConfirmTitle),
            content: Text(l10n.executeConfirmMessageTask(task.name)),
            actions: [
              TextButton(
                onPressed: () => Navigator.pop(context),
                child: Text(l10n.cancel),
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
                        SnackBar(content: Text(l10n.taskExecuted(taskName))),
                      );
                    }
                    await _loadTasks();
                  } catch (e) {
                    if (mounted) {
                      scaffoldMessenger.showSnackBar(
                        SnackBar(content: Text('${l10n.executeFailed}: $e')),
                      );
                    }
                  }
                },
                child: Text(l10n.confirm),
              ),
            ],
          ),
    );
  }

  void _handleTaskRemove(Task task) {
    final l10n = AppLocalizations.of(context)!;
    showDialog(
      context: context,
      builder:
          (context) => AlertDialog(
            title: Text(l10n.removeTask),
            content: Text(l10n.removeTaskConfirmMessage(task.name)),
            actions: [
              TextButton(
                onPressed: () => Navigator.pop(context),
                child: Text(l10n.cancel),
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
                        SnackBar(content: Text(l10n.taskRemoved)),
                      );
                    }
                  } catch (e) {
                    if (mounted) {
                      scaffoldMessenger.showSnackBar(
                        SnackBar(content: Text('${l10n.removeFailed}: $e')),
                      );
                    }
                  }
                },
                child: Text(l10n.confirm),
              ),
            ],
          ),
    );
  }

  /// 构建页面标题
  Widget _buildHeader() {
    final l10n = AppLocalizations.of(context)!;
    return Row(
      mainAxisAlignment: MainAxisAlignment.spaceBetween,
      children: [
        Text(
          l10n.taskManagement,
          style: const TextStyle(
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
              tooltip: l10n.refreshList,
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
              '${AppLocalizations.of(context)!.loadingFailed}: $_errorMessage',
              style: const TextStyle(color: Colors.red),
              textAlign: TextAlign.center,
            ),
            const SizedBox(height: 16),
            ElevatedButton(onPressed: _loadTasks, child: Text(AppLocalizations.of(context)!.retry)),
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
