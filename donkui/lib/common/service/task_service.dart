import 'dart:convert';
import 'package:donk/app/conf/config.dart' as app_config;
import '../client/http_client.dart';

/// 任务服务类
/// 封装定时任务调度器的所有HTTP API调用
class TaskService {
  static final String _baseUrl = app_config.apiBaseUrl;
  static final HttpClientSingleton _http = HttpClientSingleton.instance;

  /// 创建任务
  static Future<Map<String, dynamic>> createTask({
    required String name,
    required String taskType,
    required String schedule,
    required String executor,
    Map<String, dynamic>? config,
    int? maxRetries,
    String? createdBy,
  }) async {
    final body = await _http.post(
      '$_baseUrl/tasks',
      body: {
        'name': name,
        'task_type': taskType,
        'schedule': schedule,
        'executor': executor,
        if (config != null) 'config': config,
        if (maxRetries != null) 'max_retries': maxRetries,
        if (createdBy != null) 'created_by': createdBy,
      },
    );
    return jsonDecode(body) as Map<String, dynamic>;
  }

  /// 获取任务列表
  static Future<Map<String, dynamic>> getTasks({
    int page = 1,
    int size = 20,
    String? status,
    String? executor,
  }) async {
    final queryParams = <String, String>{
      'page': page.toString(),
      'size': size.toString(),
      if (status != null) 'status': status,
      if (executor != null) 'executor': executor,
    };

    final uri = Uri.parse(
      '$_baseUrl/tasks',
    ).replace(queryParameters: queryParams);
    final body = await _http.get(uri.toString());
    return jsonDecode(body) as Map<String, dynamic>;
  }

  /// 获取任务详情
  static Future<Map<String, dynamic>> getTaskDetail(String taskId) async {
    final body = await _http.get('$_baseUrl/tasks/$taskId');
    return jsonDecode(body) as Map<String, dynamic>;
  }

  /// 删除任务
  static Future<Map<String, dynamic>> deleteTask(String taskId) async {
    final body = await _http.delete('$_baseUrl/tasks/$taskId');
    return jsonDecode(body) as Map<String, dynamic>;
  }

  /// 取消任务
  static Future<Map<String, dynamic>> cancelTask(String taskId) async {
    final body = await _http.post('$_baseUrl/tasks/$taskId/cancel');
    return jsonDecode(body) as Map<String, dynamic>;
  }

  /// 手动触发任务
  static Future<Map<String, dynamic>> triggerTask(String taskId) async {
    final body = await _http.post('$_baseUrl/tasks/$taskId/run');
    return jsonDecode(body) as Map<String, dynamic>;
  }

  /// 获取任务执行结果
  static Future<Map<String, dynamic>> getTaskResult(String taskId) async {
    final body = await _http.get('$_baseUrl/tasks/$taskId/result');
    return jsonDecode(body) as Map<String, dynamic>;
  }

  /// 获取执行记录列表
  static Future<Map<String, dynamic>> getRunHistory({
    int page = 0,
    int size = 20,
    String? taskId,
    String? status,
  }) async {
    final queryParams = <String, String>{
      'page': page.toString(),
      'size': size.toString(),
      if (taskId != null) 'task_id': taskId,
      if (status != null) 'status': status,
    };

    final uri = Uri.parse(
      '$_baseUrl/runs',
    ).replace(queryParameters: queryParams);
    final body = await _http.get(uri.toString());
    return jsonDecode(body) as Map<String, dynamic>;
  }

  /// 获取任务的执行记录列表
  static Future<Map<String, dynamic>> getTaskRuns(
    String taskId, {
    int page = 1,
    int size = 20,
    String? status,
  }) async {
    final queryParams = <String, String>{
      'page': page.toString(),
      'size': size.toString(),
      if (status != null) 'status': status,
    };

    final uri = Uri.parse(
      '$_baseUrl/tasks/$taskId/runs',
    ).replace(queryParameters: queryParams);
    final body = await _http.get(uri.toString());
    return jsonDecode(body) as Map<String, dynamic>;
  }

  /// 获取执行记录详情
  static Future<Map<String, dynamic>> getRunDetail(String runId) async {
    final body = await _http.get('$_baseUrl/runs/$runId');
    return jsonDecode(body) as Map<String, dynamic>;
  }

  /// 删除执行记录
  static Future<Map<String, dynamic>> deleteRun(String runId) async {
    final body = await _http.delete('$_baseUrl/runs/$runId');
    return jsonDecode(body) as Map<String, dynamic>;
  }
}

/// 任务状态枚举
enum TaskStatus {
  pending,
  running,
  paused,
  completed,
  failed,
  cancelled;

  static TaskStatus fromString(String value) {
    return TaskStatus.values.firstWhere(
      (e) => e.name == value,
      orElse: () => TaskStatus.pending,
    );
  }
}

/// 执行记录状态枚举
enum RunStatus {
  running,
  completed,
  failed,
  cancelled;

  static RunStatus fromString(String value) {
    return RunStatus.values.firstWhere(
      (e) => e.name == value,
      orElse: () => RunStatus.completed,
    );
  }
}

/// 任务类型枚举
enum TaskType {
  cron,
  delay,
  once;

  static TaskType fromString(String value) {
    return TaskType.values.firstWhere(
      (e) => e.name == value,
      orElse: () => TaskType.cron,
    );
  }
}

/// 执行器类型枚举
enum ExecutorType {
  script,
  api,
  agent;

  static ExecutorType fromString(String value) {
    return ExecutorType.values.firstWhere(
      (e) => e.name == value,
      orElse: () => ExecutorType.agent,
    );
  }
}
