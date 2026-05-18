/// 任务数据模型
/// 用于表示定时任务的基本信息
class Task {
  /// 任务唯一标识符
  final String id;

  /// 任务名称
  final String name;

  /// 任务类型（cron/delay/once）
  final String taskType;

  /// 调度表达式（如 "0 8 * * *" 表示每天8点）
  final String schedule;

  /// 执行器名称
  final String executor;

  /// 任务配置参数
  final Map<String, dynamic>? config;

  /// 任务状态（pending/running/paused/completed/failed/cancelled）
  final String status;

  /// 当前重试次数
  final int? retries;

  /// 最大重试次数
  final int? maxRetries;

  /// 下次执行时间（Unix时间戳，秒）
  final int? nextRunAt;

  /// 上次执行时间（Unix时间戳，秒）
  final int? lastRunAt;

  /// 创建时间（Unix时间戳，秒）
  final int createdAt;

  /// 更新时间（Unix时间戳，秒）
  final int? updatedAt;

  /// 创建者
  final String? createdBy;

  /// 上次执行时间的格式化字符串
  final String? lastExecuteTime;

  /// 状态标签
  final String? statusLabel;

  Task({
    required this.id,
    required this.name,
    required this.taskType,
    required this.schedule,
    required this.executor,
    this.config,
    required this.status,
    this.retries,
    this.maxRetries,
    this.nextRunAt,
    this.lastRunAt,
    required this.createdAt,
    this.updatedAt,
    this.createdBy,
    this.lastExecuteTime,
    this.statusLabel,
  });

  /// 从 JSON 数据创建 Task 实例
  factory Task.fromJson(Map<String, dynamic> json) {
    return Task(
      id: json['id'] as String,
      name: json['name'] as String,
      taskType: json['task_type'] as String,
      schedule: json['schedule'] as String,
      executor: json['executor'] as String,
      config: json['config'] as Map<String, dynamic>?,
      status: json['status'] as String,
      retries: json['retries'] as int?,
      maxRetries: json['max_retries'] as int?,
      nextRunAt: json['next_run_at'] as int?,
      lastRunAt: json['last_run_at'] as int?,
      createdAt: json['created_at'] as int,
      updatedAt: json['updated_at'] as int?,
      createdBy: json['created_by'] as String?,
      lastExecuteTime: json['last_execute_time'] as String?,
      statusLabel: json['status_label'] as String?,
    );
  }

  /// 将 Task 实例转换为 JSON 数据
  Map<String, dynamic> toJson() {
    return {
      'id': id,
      'name': name,
      'task_type': taskType,
      'schedule': schedule,
      'executor': executor,
      if (config != null) 'config': config,
      'status': status,
      if (retries != null) 'retries': retries,
      if (maxRetries != null) 'max_retries': maxRetries,
      if (nextRunAt != null) 'next_run_at': nextRunAt,
      if (lastRunAt != null) 'last_run_at': lastRunAt,
      'created_at': createdAt,
      if (updatedAt != null) 'updated_at': updatedAt,
      if (createdBy != null) 'created_by': createdBy,
    };
  }

  /// 复制当前实例并修改指定字段
  Task copyWith({
    String? id,
    String? name,
    String? taskType,
    String? schedule,
    String? executor,
    Map<String, dynamic>? config,
    String? status,
    int? retries,
    int? maxRetries,
    int? nextRunAt,
    int? lastRunAt,
    int? createdAt,
    int? updatedAt,
    String? createdBy,
    String? lastExecuteTime,
    String? statusLabel,
  }) {
    return Task(
      id: id ?? this.id,
      name: name ?? this.name,
      taskType: taskType ?? this.taskType,
      schedule: schedule ?? this.schedule,
      executor: executor ?? this.executor,
      config: config ?? this.config,
      status: status ?? this.status,
      retries: retries ?? this.retries,
      maxRetries: maxRetries ?? this.maxRetries,
      nextRunAt: nextRunAt ?? this.nextRunAt,
      lastRunAt: lastRunAt ?? this.lastRunAt,
      createdAt: createdAt ?? this.createdAt,
      updatedAt: updatedAt ?? this.updatedAt,
      createdBy: createdBy ?? this.createdBy,
      lastExecuteTime: lastExecuteTime ?? this.lastExecuteTime,
      statusLabel: statusLabel ?? this.statusLabel,
    );
  }

  /// 根据任务名称获取对应的图标
  /// 包含星座、喝水、新闻、天气、提醒等关键词的匹配
  String get icon {
    if (name.contains('星座')) return '⏰';
    if (name.contains('喝水')) return '💧';
    if (name.contains('新闻')) return '📰';
    if (name.contains('天气')) return '🌤️';
    if (name.contains('提醒')) return '🔔';
    return '📋';
  }

  /// 获取格式化的执行时间描述
  /// 将 cron 表达式转换为中文描述
  String get displaySchedule {
    if (schedule.contains('8:00') || schedule.contains('08:00')) {
      return '每天上午8:00';
    }
    if (schedule.contains('10:00')) return '每日 10:00';
    if (schedule.contains('9:00') || schedule.contains('09:00')) {
      return '每天 9:00';
    }
    return schedule;
  }

  /// 判断任务是否处于启用状态
  /// running 或 pending 状态视为启用
  bool get isEnabled => status == 'running' || status == 'pending';

  /// 获取格式化的上次执行时间
  /// 显示为 "今天 HH:mm"、"昨天 HH:mm" 或具体日期
  String get displayLastExecuteTime {
    if (lastExecuteTime != null) return lastExecuteTime!;
    if (lastRunAt == null) return '';
    final dt = DateTime.fromMillisecondsSinceEpoch(lastRunAt! * 1000);
    final now = DateTime.now();
    if (dt.year == now.year && dt.month == now.month && dt.day == now.day) {
      return '今天 ${dt.hour.toString().padLeft(2, '0')}:${dt.minute.toString().padLeft(2, '0')}';
    }
    final yesterday = now.subtract(const Duration(days: 1));
    if (dt.year == yesterday.year &&
        dt.month == yesterday.month &&
        dt.day == yesterday.day) {
      return '昨天 ${dt.hour.toString().padLeft(2, '0')}:${dt.minute.toString().padLeft(2, '0')}';
    }
    return '${dt.month}/${dt.day} ${dt.hour.toString().padLeft(2, '0')}:${dt.minute.toString().padLeft(2, '0')}';
  }

  /// 获取中文状态描述
  /// 将英文状态码转换为中文显示
  String get displayStatus {
    if (statusLabel != null) return statusLabel!;
    switch (status) {
      case 'pending':
        return '待执行';
      case 'running':
        return '执行中';
      case 'paused':
        return '已暂停';
      case 'completed':
        return '已完成';
      case 'failed':
        return '失败';
      case 'cancelled':
        return '已取消';
      default:
        return status;
    }
  }

  /// 获取中文任务类型描述
  /// cron: 定时任务, delay: 延迟任务, once: 一次性任务
  String get displayTaskType {
    switch (taskType) {
      case 'cron':
        return '定时任务';
      case 'delay':
        return '延迟任务';
      case 'once':
        return '一次性任务';
      default:
        return taskType;
    }
  }

  /// 获取格式化的下次执行时间
  /// 显示为 "今天 HH:mm"、"明天 HH:mm" 或具体日期
  String get displayNextRunTime {
    if (nextRunAt == null) return '未设置';
    final dt = DateTime.fromMillisecondsSinceEpoch(nextRunAt! * 1000);
    final now = DateTime.now();
    if (dt.year == now.year && dt.month == now.month && dt.day == now.day) {
      return '今天 ${dt.hour.toString().padLeft(2, '0')}:${dt.minute.toString().padLeft(2, '0')}';
    }
    final tomorrow = now.add(const Duration(days: 1));
    if (dt.year == tomorrow.year &&
        dt.month == tomorrow.month &&
        dt.day == tomorrow.day) {
      return '明天 ${dt.hour.toString().padLeft(2, '0')}:${dt.minute.toString().padLeft(2, '0')}';
    }
    return '${dt.month}/${dt.day} ${dt.hour.toString().padLeft(2, '0')}:${dt.minute.toString().padLeft(2, '0')}';
  }

  /// 获取重试信息描述
  /// 格式为 "重试 当前次数/最大次数"
  String get displayRetryInfo {
    if (maxRetries == null || maxRetries == 0) return '';
    final retryStr = retries != null ? '$retries' : '0';
    return '重试 $retryStr/$maxRetries';
  }
}

/// 历史任务记录数据模型
/// 用于表示任务的执行历史记录
class TaskHistory {
  /// 执行记录唯一标识符
  final String id;

  /// 关联的任务ID
  final String taskId;

  /// 任务名称
  final String taskName;

  /// 执行器名称
  final String executor;

  /// 执行状态（running/completed/failed/cancelled）
  final String status;

  /// 开始时间（Unix时间戳，秒）
  final int startTime;

  /// 结束时间（Unix时间戳，秒）
  final int? endTime;

  /// 执行时长（毫秒）
  final int? duration;

  /// 退出码
  final int? exitCode;

  /// 重试次数
  final int? retryCount;

  /// 执行输出内容
  final String? output;

  /// 错误信息
  final String? error;

  /// 输入参数
  final String? input;

  /// 创建时间（Unix时间戳，秒）
  final int createdAt;

  TaskHistory({
    required this.id,
    required this.taskId,
    required this.taskName,
    required this.executor,
    required this.status,
    required this.startTime,
    this.endTime,
    this.duration,
    this.exitCode,
    this.retryCount,
    this.output,
    this.error,
    this.input,
    required this.createdAt,
  });

  /// 从 JSON 数据创建 TaskHistory 实例
  factory TaskHistory.fromJson(Map<String, dynamic> json) {
    return TaskHistory(
      id: json['id'] as String,
      taskId: json['task_id'] as String,
      taskName: json['task_name'] as String? ?? json['name'] as String? ?? '',
      executor: json['executor'] as String,
      status: json['status'] as String,
      startTime: json['start_time'] as int,
      endTime: json['end_time'] as int?,
      duration: json['duration'] as int?,
      exitCode: json['exit_code'] as int?,
      retryCount: json['retry_count'] as int?,
      output: json['output'] as String?,
      error: json['error'] as String?,
      input: json['input']?.toString(),
      createdAt: json['created_at'] as int,
    );
  }

  /// 获取执行时间的 DateTime 对象
  DateTime get executeTime =>
      DateTime.fromMillisecondsSinceEpoch(startTime * 1000);

  /// 获取中文执行状态描述
  String get displayStatus {
    switch (status) {
      case 'running':
        return '执行中';
      case 'completed':
        return '成功';
      case 'failed':
        return '失败';
      case 'cancelled':
        return '已取消';
      default:
        return status;
    }
  }

  /// 根据任务名称获取对应的图标
  String get icon {
    if (taskName.contains('星座')) return '⏰';
    if (taskName.contains('喝水')) return '💧';
    if (taskName.contains('新闻')) return '📰';
    if (taskName.contains('天气')) return '🌤️';
    if (taskName.contains('提醒')) return '🔔';
    return '📋';
  }
}
