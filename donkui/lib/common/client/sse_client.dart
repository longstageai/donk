import 'dart:async';
import 'dart:convert';
import 'dart:io';
import 'package:flutter/foundation.dart';

/// SSE 连接状态枚举
enum SSEStatus {
  /// 未连接
  disconnected,

  /// 连接中
  connecting,

  /// 已连接
  connected,

  /// 错误
  error,
}

/// SSE 消息封装类
/// 支持 agent_protocol.md 中定义的所有事件类型
class SSEMessage {
  /// 消息类型 (如: stream)
  final String type;

  /// 事件类型 (如: user_input, reasoning_delta, content_delta, assistant, stop, error)
  final String? event;

  /// 消息内容
  final String? content;

  /// 思考过程内容
  final String? reasoningContent;

  /// 工具名称
  final String? toolName;

  /// 工具输入
  final String? toolInput;

  /// 工具结果
  final String? toolResult;

  /// 警告信息
  final String? warning;

  /// 错误信息
  final String? error;

  /// 原始数据
  final dynamic rawData;

  /// 消息时间戳
  final DateTime timestamp;

  SSEMessage({
    required this.type,
    this.event,
    this.content,
    this.reasoningContent,
    this.toolName,
    this.toolInput,
    this.toolResult,
    this.warning,
    this.error,
    this.rawData,
    DateTime? timestamp,
  }) : timestamp = timestamp ?? DateTime.now();

  /// 从 JSON 构造消息对象
  factory SSEMessage.fromJson(Map<String, dynamic> json) {
    return SSEMessage(
      type: json['type'] ?? 'unknown',
      event: json['event'],
      content: json['content'],
      reasoningContent: json['reasoning_content'],
      toolName: json['tool_name'],
      toolInput: json['tool_input'],
      toolResult: json['tool_result'],
      warning: json['warning'],
      error: json['error'],
      rawData: json,
      timestamp: DateTime.now(),
    );
  }

  /// 是否为流式消息
  bool get isStream => type == 'stream';

  /// 是否为用户输入事件
  bool get isUserInput => isStream && event == 'user_input';

  /// 是否为思考过程增量
  bool get isReasoningDelta => isStream && event == 'reasoning_delta';

  /// 是否为内容增量
  bool get isContentDelta => isStream && event == 'content_delta';

  /// 是否为助手完整回复
  bool get isAssistant => isStream && event == 'assistant';

  /// 是否为工具调用
  bool get isToolCall => isStream && event == 'tool_call';

  /// 是否为工具结果
  bool get isToolResult => isStream && event == 'tool_result';

  /// 是否为停止事件
  bool get isStop => isStream && event == 'stop';

  /// 是否为错误事件
  bool get isError => isStream && event == 'error';

  /// 是否为取消事件
  bool get isCanceled => isStream && event == 'canceled';

  /// 是否为警告事件
  bool get isWarning => isStream && event == 'warning';
}

/// SSE 客户端封装类
///
/// 提供自动重连、状态管理等功能
/// 符合 agent_protocol.md 中定义的 HTTP SSE 协议
class SSEClient {
  /// SSE 服务器地址
  final String url;

  /// 自定义请求头
  final Map<String, String>? headers;

  /// 当前连接状态
  SSEStatus _status = SSEStatus.disconnected;

  /// 是否已释放
  bool _isDisposed = false;

  /// 是否正在手动断开连接
  bool _isDisconnecting = false;

  /// 数据缓冲区，用于处理分块传输
  String _buffer = '';

  /// HTTP 客户端实例
  HttpClient? _client;

  /// 消息流控制器
  final StreamController<SSEMessage> _messageController =
      StreamController<SSEMessage>.broadcast();

  /// 状态流控制器
  final StreamController<SSEStatus> _statusController =
      StreamController<SSEStatus>.broadcast();

  /// 错误流控制器
  final StreamController<dynamic> _errorController =
      StreamController<dynamic>.broadcast();

  /// 构造函数
  ///
  /// [url] - SSE 服务器地址，如 http://localhost:8080/api/v1/chat
  /// [headers] - 自定义请求头
  SSEClient({required this.url, this.headers});

  /// 获取当前状态
  SSEStatus get status => _status;

  /// 是否已连接
  bool get isConnected => _status == SSEStatus.connected;

  /// 是否正在连接
  bool get isConnecting => _status == SSEStatus.connecting;

  /// 消息流，用于监听接收到的消息
  Stream<SSEMessage> get messageStream => _messageController.stream;

  /// 状态流，用于监听连接状态变化
  Stream<SSEStatus> get statusStream => _statusController.stream;

  /// 错误流，用于监听错误信息
  Stream<dynamic> get errorStream => _errorController.stream;

  /// 更新状态并通知监听者
  void _updateStatus(SSEStatus newStatus) {
    if (_status != newStatus) {
      _status = newStatus;
      _statusController.add(newStatus);
      if (kDebugMode) {
        print('SSE status changed: $newStatus');
      }
    }
  }

  /// 连接 SSE 服务器并发送消息
  ///
  /// [message] - 要发送的用户消息内容
  ///
  /// 发送格式符合 agent_protocol.md:
  /// POST /api/v1/chat
  /// Content-Type: application/json
  /// Accept: text/event-stream
  /// Body: {"content": "用户消息"}
  Future<void> connect(String message) async {
    if (kDebugMode) {
      print('SSE connect called, status: $_status, isDisposed: $_isDisposed');
    }
    if (_isDisposed) return;
    if (_status == SSEStatus.connected || _status == SSEStatus.connecting) {
      if (kDebugMode) {
        print('SSE connect skipped: already connected or connecting');
      }
      return;
    }

    // 重置断开标志，确保可以正常连接
    _isDisconnecting = false;
    _updateStatus(SSEStatus.connecting);

    try {
      _client = HttpClient();
      _client!.connectionTimeout = const Duration(seconds: 30);

      final request = await _client!.postUrl(Uri.parse(url));

      // 设置请求头，符合 SSE 协议要求
      request.headers.set('Content-Type', 'application/json');
      request.headers.set('Accept', 'text/event-stream');
      request.headers.set('Cache-Control', 'no-cache');
      request.headers.set('Connection', 'keep-alive');

      // 添加自定义请求头
      if (headers != null) {
        headers!.forEach((key, value) {
          request.headers.set(key, value);
        });
      }

      // 发送请求体，使用 UTF-8 编码支持中文
      final body = jsonEncode({'content': message});
      request.add(utf8.encode(body));

      final response = await request.close();

      if (response.statusCode != 200) {
        throw Exception('Server returned ${response.statusCode}');
      }

      _updateStatus(SSEStatus.connected);

      // 流式读取响应数据，使用 listen 以便正确处理错误
      final subscription = response
          .transform(utf8.decoder)
          .listen(
            (chunk) {
              if (_isDisposed || _isDisconnecting) return;
              _processChunk(chunk);
            },
            onError: (e) {
              // 流读取过程中的错误，如果是手动断开则忽略
              if (!_isDisconnecting && !_isDisposed) {
                if (kDebugMode) {
                  print('SSE stream error: $e');
                }
                _updateStatus(SSEStatus.error);
                _errorController.add(e);
              }
            },
            onDone: () {
              if (!_isDisposed && !_isDisconnecting) {
                _updateStatus(SSEStatus.disconnected);
              }
            },
            cancelOnError: false,
          );

      // 等待流完成或断开
      await subscription.asFuture();
    } catch (e, stackTrace) {
      if (!_isDisposed && !_isDisconnecting) {
        if (kDebugMode) {
          print('SSE connect error: $e');
          print('Stack trace: $stackTrace');
        }
        _updateStatus(SSEStatus.error);
        _errorController.add(e);
        // SSE 对话模式下不重连，由调用方决定是否重试
      }
    } finally {
      _isDisconnecting = false;
    }
  }

  /// 处理接收到的数据块
  ///
  /// SSE 格式：`event: event_type\ndata: json\n\n`
  void _processChunk(String chunk) {
    _buffer += chunk;

    // 按双换行符分割，处理完整的事件块
    while (_buffer.contains('\n\n')) {
      final endIndex = _buffer.indexOf('\n\n');
      final eventBlock = _buffer.substring(0, endIndex);
      _buffer = _buffer.substring(endIndex + 2);

      final message = _parseEventBlock(eventBlock);
      if (message != null) {
        _messageController.add(message);
      }
    }
  }

  /// 解析 SSE 事件块
  ///
  /// 解析格式：
  /// `event: event_type`
  /// `data: json_data`
  SSEMessage? _parseEventBlock(String block) {
    String? data;

    final lines = block.split('\n');
    for (final line in lines) {
      if (line.startsWith('event:')) {
        // 事件类型已在 data 中包含，这里不需要额外处理
      } else if (line.startsWith('data:')) {
        data = line.substring(5).trim();
      }
    }

    if (data == null) return null;

    try {
      final json = jsonDecode(data);
      return SSEMessage.fromJson(json);
    } catch (e) {
      return null;
    }
  }

  /// 断开连接
  ///
  /// 关闭 HTTP 连接
  Future<void> disconnect() async {
    if (kDebugMode) {
      print('SSE disconnect called, current status: $_status');
    }
    _isDisconnecting = true;
    _client?.close(force: true);
    _client = null;
    _updateStatus(SSEStatus.disconnected);
    if (kDebugMode) {
      print('SSE disconnect completed, status set to disconnected');
    }
  }

  /// 释放资源
  ///
  /// 断开连接并关闭所有流控制器，释放后不能再使用
  Future<void> dispose() async {
    _isDisposed = true;
    await disconnect();
    await _messageController.close();
    await _statusController.close();
    await _errorController.close();
  }
}
