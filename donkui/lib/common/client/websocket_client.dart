import 'dart:async';
import 'dart:convert';
import 'dart:io';

import 'package:flutter/foundation.dart';

/// WebSocket 连接状态枚举
enum WebSocketStatus {
  /// 未连接
  disconnected,

  /// 连接中
  connecting,

  /// 已连接
  connected,

  /// 重连中
  reconnecting,

  /// 错误
  error,
}

/// WebSocket 消息封装类
/// 支持协议格式: {"type": "stream", "event": "xxx", ...}
class WebSocketMessage {
  /// 消息类型 (如: stream, chat, ping)
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

  /// 错误信息
  final String? error;

  /// 原始数据
  final dynamic rawData;

  /// 消息时间戳
  final DateTime timestamp;

  WebSocketMessage({
    required this.type,
    this.event,
    this.content,
    this.reasoningContent,
    this.toolName,
    this.toolInput,
    this.toolResult,
    this.error,
    this.rawData,
    DateTime? timestamp,
  }) : timestamp = timestamp ?? DateTime.now();

  /// 从 JSON 构造消息对象
  /// 支持协议格式: {"type": "stream", "event": "xxx", ...}
  factory WebSocketMessage.fromJson(Map<String, dynamic> json) {
    return WebSocketMessage(
      type: json['type'] ?? 'unknown',
      event: json['event'],
      content: json['content'],
      reasoningContent: json['reasoning_content'],
      toolName: json['tool_name'],
      toolInput: json['tool_input'],
      toolResult: json['tool_result'],
      error: json['error'],
      rawData: json,
      timestamp: DateTime.now(),
    );
  }

  /// 转换为 JSON
  Map<String, dynamic> toJson() {
    return {
      'type': type,
      'event': event,
      'content': content,
      'reasoning_content': reasoningContent,
      'tool_name': toolName,
      'tool_input': toolInput,
      'tool_result': toolResult,
      'error': error,
      'timestamp': timestamp.toIso8601String(),
    };
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
}

/// WebSocket 客户端封装类
///
/// 提供自动重连、心跳检测、状态管理等功能
class WebSocketClient {
  /// WebSocket 实例
  WebSocket? _webSocket;

  /// 连接地址
  final String url;

  /// 重连间隔
  final Duration reconnectInterval;

  /// 最大重连次数
  final int maxReconnectAttempts;

  /// 心跳间隔
  final Duration pingInterval;

  /// 请求头
  final Map<String, String>? headers;

  /// 当前状态
  WebSocketStatus _status = WebSocketStatus.disconnected;

  /// 当前重连次数
  int _reconnectAttempts = 0;

  /// 重连定时器
  Timer? _reconnectTimer;

  /// 心跳定时器
  Timer? _pingTimer;

  /// 是否已释放
  bool _isDisposed = false;

  /// 消息流控制器
  final StreamController<WebSocketMessage> _messageController =
      StreamController<WebSocketMessage>.broadcast();

  /// 状态流控制器
  final StreamController<WebSocketStatus> _statusController =
      StreamController<WebSocketStatus>.broadcast();

  /// 错误流控制器
  final StreamController<dynamic> _errorController =
      StreamController<dynamic>.broadcast();

  /// 构造函数
  ///
  /// [url] - WebSocket 连接地址，如 ws://localhost:8080/ws
  /// [reconnectInterval] - 重连间隔，默认 5 秒
  /// [maxReconnectAttempts] - 最大重连次数，默认 10 次
  /// [pingInterval] - 心跳间隔，默认 30 秒
  /// [headers] - 自定义请求头
  WebSocketClient({
    required this.url,
    this.reconnectInterval = const Duration(seconds: 5),
    this.maxReconnectAttempts = 10,
    this.pingInterval = const Duration(seconds: 30),
    this.headers,
  });

  /// 获取当前状态
  WebSocketStatus get status => _status;

  /// 是否已连接
  bool get isConnected => _status == WebSocketStatus.connected;

  /// 是否连接中
  bool get isConnecting => _status == WebSocketStatus.connecting;

  /// 消息流，用于监听接收到的消息
  Stream<WebSocketMessage> get messageStream => _messageController.stream;

  /// 状态流，用于监听连接状态变化
  Stream<WebSocketStatus> get statusStream => _statusController.stream;

  /// 错误流，用于监听错误信息
  Stream<dynamic> get errorStream => _errorController.stream;

  /// 更新状态并通知监听者
  void _updateStatus(WebSocketStatus newStatus) {
    if (_status != newStatus) {
      _status = newStatus;
      _statusController.add(newStatus);
      if (kDebugMode) {
        print('WebSocket status changed: $newStatus');
      }
    }
  }

  /// 连接 WebSocket 服务器
  ///
  /// 如果已经连接或正在连接，则不会重复执行
  Future<void> connect() async {
    if (_isDisposed) return;
    if (_status == WebSocketStatus.connected ||
        _status == WebSocketStatus.connecting) {
      return;
    }

    _updateStatus(WebSocketStatus.connecting);

    try {
      _webSocket = await WebSocket.connect(url, headers: headers);

      _webSocket!.pingInterval = pingInterval;
      _reconnectAttempts = 0;
      _updateStatus(WebSocketStatus.connected);

      _webSocket!.listen(
        _onMessage,
        onError: _onError,
        onDone: _onDone,
        cancelOnError: false,
      );

      _startPingTimer();
    } catch (e) {
      _updateStatus(WebSocketStatus.error);
      _errorController.add(e);
      _scheduleReconnect();
    }
  }

  /// 处理接收到的消息
  void _onMessage(dynamic message) {
    if (_isDisposed) return;

    if (kDebugMode) {
      print('WebSocket raw message received: $message');
    }

    try {
      final decoded = jsonDecode(message);
      final wsMessage = WebSocketMessage.fromJson(decoded);
      if (kDebugMode) {
        print(
          'WebSocket message parsed: type=${wsMessage.type}, event=${wsMessage.event}',
        );
      }
      _messageController.add(wsMessage);
    } catch (e) {
      if (kDebugMode) {
        print('WebSocket message parse error: $e, treating as raw');
      }
      // 如果不是 JSON 格式，作为原始消息处理
      final wsMessage = WebSocketMessage(
        type: 'raw',
        content: message.toString(),
      );
      _messageController.add(wsMessage);
    }
  }

  /// 处理错误
  void _onError(dynamic error) {
    if (_isDisposed) return;
    _updateStatus(WebSocketStatus.error);
    _errorController.add(error);
  }

  /// 连接关闭时的处理
  void _onDone() {
    if (_isDisposed) return;
    _updateStatus(WebSocketStatus.disconnected);
    _scheduleReconnect();
  }

  /// 安排重连
  void _scheduleReconnect() {
    if (_isDisposed) return;
    if (_reconnectAttempts >= maxReconnectAttempts) {
      if (kDebugMode) {
        print('Max reconnect attempts reached');
      }
      return;
    }

    _reconnectAttempts++;
    _updateStatus(WebSocketStatus.reconnecting);

    _reconnectTimer?.cancel();
    _reconnectTimer = Timer(reconnectInterval, () {
      if (!_isDisposed) {
        connect();
      }
    });
  }

  /// 启动心跳定时器
  void _startPingTimer() {
    _pingTimer?.cancel();
    _pingTimer = Timer.periodic(pingInterval, (timer) {
      if (isConnected) {
        send(type: 'ping', data: {});
      }
    });
  }

  /// 发送结构化消息
  ///
  /// [type] - 消息类型
  /// [data] - 消息数据
  ///
  /// 如果未连接会抛出 StateError
  /// 发送格式: {"type": "chat", "content": "消息内容"}
  void send({required String type, required dynamic data}) {
    if (!isConnected) {
      throw StateError('WebSocket is not connected');
    }

    // 构建符合服务器要求的格式
    final Map<String, dynamic> messageJson;
    if (data is Map<String, dynamic> && data.containsKey('content')) {
      // 如果 data 包含 content 字段，直接使用该格式
      messageJson = {'type': type, 'content': data['content']};
    } else if (data is String) {
      // 如果 data 是字符串，作为 content 发送
      messageJson = {'type': type, 'content': data};
    } else {
      // 其他情况，将 data 作为 content
      messageJson = {'type': type, 'content': data.toString()};
    }

    _webSocket!.add(jsonEncode(messageJson));
  }

  /// 发送原始字符串消息
  ///
  /// [message] - 原始消息字符串
  ///
  /// 如果未连接会抛出 StateError
  void sendRaw(String message) {
    if (!isConnected) {
      throw StateError('WebSocket is not connected');
    }
    _webSocket!.add(message);
  }

  /// 断开连接
  ///
  /// 关闭 WebSocket 连接，停止重连和心跳
  Future<void> disconnect() async {
    _reconnectTimer?.cancel();
    _pingTimer?.cancel();

    if (_webSocket != null) {
      await _webSocket!.close();
      _webSocket = null;
    }

    _reconnectAttempts = 0;
    _updateStatus(WebSocketStatus.disconnected);
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
