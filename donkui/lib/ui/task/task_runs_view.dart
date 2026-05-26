import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import '../../common/service/task_service.dart';
import '../../l10n/generated/app_localizations.dart';
import 'task_model.dart';

/// 任务运行记录页面
/// 展示指定任务的所有执行记录
class TaskRunsView extends StatefulWidget {
  /// 任务ID
  final String taskId;

  /// 任务名称
  final String taskName;

  const TaskRunsView({super.key, required this.taskId, required this.taskName});

  @override
  State<TaskRunsView> createState() => _TaskRunsViewState();
}

class _TaskRunsViewState extends State<TaskRunsView> {
  List<TaskHistory> _runs = [];
  bool _isLoading = true;
  bool _isLoadingMore = false;
  String? _errorMessage;

  // 分页相关
  int _currentPage = 1;
  final int _pageSize = 20;
  int _total = 0;
  bool _hasMore = true;

  String _selectedStatusFilter = '全部状态';
  final List<String> _statusFilters = ['全部状态', 'done', 'failed', 'running'];

  // ScrollController 用于监听滚动位置
  final ScrollController _scrollController = ScrollController();

  @override
  void initState() {
    super.initState();
    _loadRuns();
    _scrollController.addListener(_onScroll);
  }

  @override
  void dispose() {
    _scrollController.dispose();
    super.dispose();
  }

  /// 监听滚动，实现上拉加载更多
  void _onScroll() {
    if (_scrollController.position.pixels >=
        _scrollController.position.maxScrollExtent - 100) {
      if (!_isLoadingMore && _hasMore && !_isLoading) {
        _loadMoreRuns();
      }
    }
  }

  Future<void> _loadRuns() async {
    setState(() {
      _isLoading = true;
      _errorMessage = null;
      _currentPage = 1;
      _hasMore = true;
    });

    try {
      final status =
          _selectedStatusFilter == '全部状态' ? null : _selectedStatusFilter;
      final response = await TaskService.getTaskRuns(
        widget.taskId,
        page: _currentPage,
        size: _pageSize,
        status: status,
      );
      final items = response['items'] as List<dynamic>? ?? [];
      _total = response['total'] as int? ?? 0;
      if (!mounted) return;
      setState(() {
        _runs =
            items
                .map(
                  (item) => TaskHistory.fromJson(item as Map<String, dynamic>),
                )
                .toList();
        _hasMore = _runs.length < _total;
        _isLoading = false;
      });
    } catch (e) {
      final errorStr = e.toString();
      // 404 错误表示接口不存在，显示友好提示
      if (errorStr.contains('404')) {
        if (!mounted) return;
        setState(() {
          _errorMessage =
              'API 接口未找到 (404)\n\n可能原因：\n1. 后端服务未启动\n2. 接口路径错误\n3. 该功能尚未实现';
          _isLoading = false;
        });
      } else {
        if (!mounted) return;
        setState(() {
          _errorMessage = errorStr;
          _isLoading = false;
        });
      }
    }
  }

  /// 加载更多数据
  Future<void> _loadMoreRuns() async {
    if (_isLoadingMore || !_hasMore) return;

    setState(() {
      _isLoadingMore = true;
    });

    try {
      final status =
          _selectedStatusFilter == '全部状态' ? null : _selectedStatusFilter;
      final nextPage = _currentPage + 1;
      final response = await TaskService.getTaskRuns(
        widget.taskId,
        page: nextPage,
        size: _pageSize,
        status: status,
      );
      final items = response['items'] as List<dynamic>? ?? [];
      final newRuns =
          items
              .map((item) => TaskHistory.fromJson(item as Map<String, dynamic>))
              .toList();

      if (!mounted) return;
      setState(() {
        _runs.addAll(newRuns);
        _currentPage = nextPage;
        _hasMore = _runs.length < _total;
        _isLoadingMore = false;
      });
    } catch (e) {
      if (!mounted) return;
      setState(() {
        _isLoadingMore = false;
      });
    }
  }

  /// 删除运行记录
  Future<void> _deleteRun(TaskHistory run) async {
    final l10n = AppLocalizations.of(context)!;
    // 显示确认对话框
    final confirmed = await showDialog<bool>(
      context: context,
      builder:
          (context) => AlertDialog(
            title: Text(l10n.confirmDelete),
            content: Text(
              l10n.deleteRunConfirmMessage(_truncateId(run.id), _formatDateTime(run.executeTime)),
            ),
            actions: [
              TextButton(
                onPressed: () => Navigator.of(context).pop(false),
                child: Text(l10n.cancel),
              ),
              TextButton(
                onPressed: () => Navigator.of(context).pop(true),
                style: TextButton.styleFrom(foregroundColor: Colors.red),
                child: Text(l10n.delete),
              ),
            ],
          ),
    );

    if (confirmed != true) return;

    try {
      await TaskService.deleteRun(run.id);
      if (mounted) {
        ScaffoldMessenger.of(
          context,
        ).showSnackBar(SnackBar(content: Text(l10n.runRecordDeleted)));
      }
      // 从列表中移除
      setState(() {
        _runs.removeWhere((r) => r.id == run.id);
        _total--;
      });
    } catch (e) {
      if (mounted) {
        ScaffoldMessenger.of(
          context,
        ).showSnackBar(SnackBar(content: Text('${l10n.deleteFailed}: $e')));
      }
    }
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      backgroundColor: const Color(0xFFF5F5F5),
      appBar: AppBar(
        backgroundColor: Colors.white,
        elevation: 0,
        leading: IconButton(
          icon: const Icon(Icons.arrow_back, color: Colors.black87),
          onPressed: () => Navigator.pop(context),
        ),
        title: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text(
              widget.taskName,
              style: const TextStyle(
                fontSize: 16,
                fontWeight: FontWeight.w600,
                color: Colors.black87,
              ),
            ),
            Text(
              AppLocalizations.of(context)!.runRecords,
              style: TextStyle(fontSize: 12, color: Colors.grey[600]),
            ),
          ],
        ),
        actions: [_buildStatusFilter(), const SizedBox(width: 16)],
      ),
      body: _buildBody(),
    );
  }

  Widget _buildStatusFilter() {
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 6),
      decoration: BoxDecoration(
        color: const Color(0xFFF5F5F5),
        borderRadius: BorderRadius.circular(16),
      ),
      child: DropdownButtonHideUnderline(
        child: DropdownButton<String>(
          value: _selectedStatusFilter,
          isDense: true,
          icon: const Icon(Icons.keyboard_arrow_down, size: 18),
          style: const TextStyle(fontSize: 13, color: Colors.black87),
          onChanged: (value) {
            setState(() {
              _selectedStatusFilter = value!;
            });
            _loadRuns();
          },
          items:
              _statusFilters.map((String item) {
                return DropdownMenuItem<String>(
                  value: item,
                  child: Text(_getStatusDisplayName(item)),
                );
              }).toList(),
        ),
      ),
    );
  }

  String _getStatusDisplayName(String status) {
    final l10n = AppLocalizations.of(context)!;
    switch (status) {
      case 'done':
        return l10n.statusDone;
      case 'failed':
        return l10n.statusFailed;
      case 'running':
        return l10n.statusRunning;
      default:
        return status;
    }
  }

  Widget _buildBody() {
    if (_isLoading) {
      return const Center(child: CircularProgressIndicator());
    }

    if (_errorMessage != null) {
      return Center(
        child: Padding(
          padding: const EdgeInsets.all(32),
          child: Column(
            mainAxisAlignment: MainAxisAlignment.center,
            children: [
              const Icon(Icons.error_outline, size: 48, color: Colors.grey),
              const SizedBox(height: 16),
              Text(
                _errorMessage!,
                textAlign: TextAlign.center,
                style: const TextStyle(
                  fontSize: 14,
                  color: Colors.grey,
                  height: 1.5,
                ),
              ),
              const SizedBox(height: 24),
              ElevatedButton(onPressed: _loadRuns, child: Text(AppLocalizations.of(context)!.retry)),
            ],
          ),
        ),
      );
    }

    if (_runs.isEmpty) {
      return Center(
        child: Column(
          mainAxisAlignment: MainAxisAlignment.center,
          children: [
            const Icon(Icons.play_circle_outline, size: 48, color: Colors.grey),
            const SizedBox(height: 16),
            Text(AppLocalizations.of(context)!.noRunRecords, style: const TextStyle(fontSize: 14, color: Colors.grey)),
          ],
        ),
      );
    }

    return RefreshIndicator(
      onRefresh: _loadRuns,
      child: ListView.builder(
        controller: _scrollController,
        padding: const EdgeInsets.all(16),
        itemCount: _runs.length + (_hasMore ? 1 : 0),
        itemBuilder: (context, index) {
          if (index == _runs.length) {
            // 底部加载更多指示器
            return Container(
              padding: const EdgeInsets.symmetric(vertical: 16),
              alignment: Alignment.center,
              child:
                  _isLoadingMore
                      ? const SizedBox(
                        width: 24,
                        height: 24,
                        child: CircularProgressIndicator(strokeWidth: 2),
                      )
                      : Text(
                        '上拉加载更多',
                        style: TextStyle(fontSize: 12, color: Colors.grey[400]),
                      ),
            );
          }
          return _buildRunCard(_runs[index]);
        },
      ),
    );
  }

  Widget _buildRunCard(TaskHistory run) {
    return Container(
      margin: const EdgeInsets.only(bottom: 12),
      decoration: BoxDecoration(
        color: Colors.white,
        borderRadius: BorderRadius.circular(12),
        boxShadow: [
          BoxShadow(
            color: Colors.black.withValues(alpha: 0.05),
            blurRadius: 4,
            offset: const Offset(0, 2),
          ),
        ],
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          // 头部：执行ID、状态
          Padding(
            padding: const EdgeInsets.all(16),
            child: Row(
              children: [
                Expanded(
                  child: MouseRegion(
                    cursor: SystemMouseCursors.click,
                    child: GestureDetector(
                      onTap:
                          () => _copyToClipboard(run.output ?? '', '执行输出已复制'),
                      child: Row(
                        mainAxisSize: MainAxisSize.min,
                        children: [
                          const Icon(Icons.tag, size: 14, color: Colors.grey),
                          const SizedBox(width: 4),
                          Text(
                            'ID: ${_truncateId(run.id)}',
                            style: const TextStyle(
                              fontSize: 13,
                              color: Colors.grey,
                            ),
                          ),
                          const SizedBox(width: 4),
                          const Icon(Icons.copy, size: 12, color: Colors.grey),
                        ],
                      ),
                    ),
                  ),
                ),
                _buildStatusBadge(run.status),
              ],
            ),
          ),
          // 输入参数（如有）
          if (run.input != null && run.input!.isNotEmpty)
            Padding(
              padding: const EdgeInsets.fromLTRB(16, 0, 16, 12),
              child: Container(
                width: double.infinity,
                padding: const EdgeInsets.all(12),
                decoration: BoxDecoration(
                  color: const Color(0xFFE3F2FD),
                  borderRadius: BorderRadius.circular(8),
                ),
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    const Text(
                      '输入参数:',
                      style: TextStyle(
                        fontSize: 11,
                        color: Colors.grey,
                        fontWeight: FontWeight.w500,
                      ),
                    ),
                    const SizedBox(height: 4),
                    SelectionArea(
                      child: Text(
                        run.input!,
                        style: const TextStyle(
                          fontSize: 13,
                          color: Colors.black87,
                          height: 1.5,
                        ),
                      ),
                    ),
                  ],
                ),
              ),
            ),
          // 执行输出（如有）
          if (run.output != null && run.output!.isNotEmpty)
            Padding(
              padding: const EdgeInsets.fromLTRB(16, 0, 16, 12),
              child: Container(
                width: double.infinity,
                padding: const EdgeInsets.all(12),
                decoration: BoxDecoration(
                  color: const Color(0xFFF5F5F5),
                  borderRadius: BorderRadius.circular(8),
                ),
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    const Text(
                      '执行输出:',
                      style: TextStyle(
                        fontSize: 11,
                        color: Colors.grey,
                        fontWeight: FontWeight.w500,
                      ),
                    ),
                    const SizedBox(height: 4),
                    SelectionArea(
                      child: Text(
                        run.output!,
                        style: const TextStyle(
                          fontSize: 13,
                          color: Colors.black54,
                          height: 1.5,
                        ),
                      ),
                    ),
                  ],
                ),
              ),
            ),
          // 错误信息（如有）
          if (run.error != null && run.error!.isNotEmpty)
            Padding(
              padding: const EdgeInsets.fromLTRB(16, 0, 16, 12),
              child: Container(
                width: double.infinity,
                padding: const EdgeInsets.all(12),
                decoration: BoxDecoration(
                  color: const Color(0xFFFFEBEE),
                  borderRadius: BorderRadius.circular(8),
                ),
                child: Row(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    const Icon(
                      Icons.error_outline,
                      size: 16,
                      color: Color(0xFFE53935),
                    ),
                    const SizedBox(width: 8),
                    Expanded(
                      child: Text(
                        run.error!,
                        style: const TextStyle(
                          fontSize: 13,
                          color: Color(0xFFE53935),
                          height: 1.5,
                        ),
                      ),
                    ),
                  ],
                ),
              ),
            ),
          // 底部：执行器、时长、时间、删除按钮
          Container(
            padding: const EdgeInsets.fromLTRB(16, 0, 16, 16),
            child: Row(
              children: [
                Icon(
                  _getExecutorIcon(run.executor),
                  size: 14,
                  color: Colors.grey,
                ),
                const SizedBox(width: 4),
                Text(
                  _getExecutorDisplayName(run.executor),
                  style: const TextStyle(fontSize: 12, color: Colors.grey),
                ),
                if (run.duration != null) ...[
                  const SizedBox(width: 12),
                  const Icon(Icons.timer, size: 14, color: Colors.grey),
                  const SizedBox(width: 4),
                  Text(
                    _formatDuration(run.duration!),
                    style: const TextStyle(fontSize: 12, color: Colors.grey),
                  ),
                ],
                if (run.retryCount != null && run.retryCount! > 0) ...[
                  const SizedBox(width: 12),
                  const Icon(Icons.refresh, size: 14, color: Colors.grey),
                  const SizedBox(width: 4),
                  Text(
                    '重试${run.retryCount}次',
                    style: const TextStyle(fontSize: 12, color: Colors.grey),
                  ),
                ],
                const Spacer(),
                Text(
                  _formatDateTime(run.executeTime),
                  style: const TextStyle(fontSize: 12, color: Colors.grey),
                ),
                const SizedBox(width: 8),
                // 删除按钮
                MouseRegion(
                  cursor: SystemMouseCursors.click,
                  child: GestureDetector(
                    onTap: () => _deleteRun(run),
                    child: Container(
                      padding: const EdgeInsets.all(4),
                      decoration: BoxDecoration(
                        color: Colors.red.withValues(alpha: 0.1),
                        borderRadius: BorderRadius.circular(4),
                      ),
                      child: const Icon(
                        Icons.delete_outline,
                        size: 16,
                        color: Colors.red,
                      ),
                    ),
                  ),
                ),
              ],
            ),
          ),
        ],
      ),
    );
  }

  Widget _buildStatusBadge(String status) {
    final color = _getStatusColor(status);
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 4),
      decoration: BoxDecoration(
        color: color.withValues(alpha: 0.1),
        borderRadius: BorderRadius.circular(4),
      ),
      child: Text(
        _getStatusDisplayName(status),
        style: TextStyle(
          fontSize: 12,
          color: color,
          fontWeight: FontWeight.w500,
        ),
      ),
    );
  }

  Color _getStatusColor(String status) {
    switch (status) {
      case 'done':
        return const Color(0xFF4CAF50);
      case 'failed':
        return const Color(0xFFE53935);
      case 'running':
        return const Color(0xFF2196F3);
      default:
        return Colors.grey;
    }
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

  String _getExecutorDisplayName(String executor) {
    switch (executor) {
      case 'script':
        return '脚本';
      case 'api':
        return 'API';
      case 'agent':
        return 'Agent';
      default:
        return executor;
    }
  }

  String _truncateId(String id) {
    if (id.length <= 8) return id;
    return '${id.substring(0, 8)}...';
  }

  String _formatDuration(int seconds) {
    if (seconds < 60) {
      return '${seconds}s';
    }
    final minutes = seconds ~/ 60;
    final remainingSeconds = seconds % 60;
    return '${minutes}m ${remainingSeconds}s';
  }

  String _formatDateTime(DateTime dateTime) {
    return '${dateTime.month.toString().padLeft(2, '0')}/${dateTime.day.toString().padLeft(2, '0')} ${dateTime.hour.toString().padLeft(2, '0')}:${dateTime.minute.toString().padLeft(2, '0')}:${dateTime.second.toString().padLeft(2, '0')}';
  }

  void _copyToClipboard(String text, String message) {
    Clipboard.setData(ClipboardData(text: text));
    ScaffoldMessenger.of(context).showSnackBar(
      SnackBar(content: Text(message), duration: const Duration(seconds: 2)),
    );
  }
}
