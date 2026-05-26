import 'dart:async';
import 'package:donk/app/conf/config.dart' as app_config;
import 'package:flutter/foundation.dart';
import 'package:get/get.dart';
import '../client/websocket_client.dart';
import '../model/notification_message.dart';
import 'notification_storage_service.dart';
import 'wechat_bot_service.dart';

/// 通知WebSocket服务
/// 负责连接WebSocket服务器，接收通知消息并存储到本地
/// 支持将消息推送到微信（当微信已登录时）
class NotificationWebSocketService extends GetxService {
  WebSocketClient? _client;
  final NotificationStorageService _storageService =
      NotificationStorageService();
  final WeChatBotService _wechatService = WeChatBotService();

  /// 普通通知消息流，用于监听新消息
  final StreamController<NotificationMessage> _messageController =
      StreamController<NotificationMessage>.broadcast();

  /// Stream类型消息流，用于监听Stream消息
  final StreamController<Map<String, dynamic>> _streamMessageController =
      StreamController<Map<String, dynamic>>.broadcast();

  /// 未读消息数量
  final RxInt unreadCount = 0.obs;

  /// 连接状态
  final RxBool isConnected = false.obs;

  /// 服务器地址（从配置获取）
  String? _serverUrl;

  /// 普通通知消息列表（按时间倒序排列）
  final RxList<NotificationMessage> messages = <NotificationMessage>[].obs;

  /// Stream类型消息列表（按时间倒序排列）
  final RxList<Map<String, dynamic>> streamMessages = <Map<String, dynamic>>[].obs;

  /// 普通消息流
  Stream<NotificationMessage> get messageStream => _messageController.stream;

  /// Stream消息流
  Stream<Map<String, dynamic>> get streamMessageStream => _streamMessageController.stream;

  /// 初始化服务
  Future<void> init() async {
    await _loadMessages();
    await _loadUnreadCount();
    await _connect();
  }

  /// 加载消息列表
  Future<void> _loadMessages() async {
    final loadedMessages = await _storageService.loadMessages();
    // 按时间倒序排列
    loadedMessages.sort((a, b) => b.timestamp.compareTo(a.timestamp));
    messages.assignAll(loadedMessages);
  }

  /// 加载未读消息数量
  Future<void> _loadUnreadCount() async {
    unreadCount.value = await _storageService.getUnreadCount();
  }

  /// 连接WebSocket服务器
  Future<void> _connect() async {
    try {
      // 从HTTP客户端获取服务器地址
      _serverUrl = await _getWebSocketUrl();

      if (_serverUrl == null || _serverUrl!.isEmpty) {
        if (kDebugMode) {
          print('WebSocket URL not configured');
        }
        return;
      }

      if (kDebugMode) {
        print('Connecting to WebSocket: $_serverUrl');
      }

      _client = WebSocketClient(
        url: _serverUrl!,
        reconnectInterval: const Duration(seconds: 5),
        maxReconnectAttempts: 100, // 大量重连次数，保持长期连接
        pingInterval: const Duration(seconds: 30),
      );

      // 监听消息
      _client!.messageStream.listen(
        _onMessage,
        onError: (error) {
          if (kDebugMode) {
            print('WebSocket error: $error');
          }
        },
        onDone: () {
          if (kDebugMode) {
            print('WebSocket message stream done');
          }
        },
      );

      if (kDebugMode) {
        print('WebSocket message stream listener registered');
      }

      // 监听连接状态
      _client!.statusStream.listen((status) {
        if (kDebugMode) {
          print('WebSocket status changed: $status');
        }
        isConnected.value = status == WebSocketStatus.connected;
      });

      await _client!.connect();

      if (kDebugMode) {
        print('WebSocket connect() called');
      }
    } catch (e) {
      if (kDebugMode) {
        print('Failed to connect WebSocket: $e');
      }
    }
  }

  /// 获取WebSocket URL
  /// 默认使用本地服务器，可通过修改此处配置服务器地址
  Future<String?> _getWebSocketUrl() async {
    // 从配置文件获取WebSocket地址
    return app_config.wsUrl;
  }

  /// 处理接收到的消息
  /// 支持三种格式:
  /// 1. 通知格式: {"type": "notification", "id": "xxx", "title": "xxx", "content": "xxx", "level": "info|success|warning|error"}
  /// 2. 标准格式: {"type": "message", "data": {"id": "xxx", "title": "xxx", "content": "xxx", ...}}
  /// 3. Stream格式: {"type": "stream", "event": "content_delta", "content": "xxx", "agent_id": "xxx", ...}
  void _onMessage(WebSocketMessage wsMessage) {
    if (kDebugMode) {
      print(
        'Received WebSocket message: type=${wsMessage.type}, rawData=${wsMessage.rawData}',
      );
    }

    try {
      // 忽略心跳消息
      if (wsMessage.type == 'pong') {
        if (kDebugMode) {
          print('Received pong, ignoring');
        }
        return;
      }

      // 根据消息类型分别处理
      switch (wsMessage.type) {
        case 'notification':
          // 格式1: 直接包含字段的notification类型
          _handleNotificationMessage(wsMessage.rawData);
          break;
        case 'message':
          // 格式2: 标准message类型，数据在data字段中
          final data = wsMessage.rawData?['data'] as Map<String, dynamic>?;
          if (data != null) {
            _handleNotificationMessage(data);
          } else {
            if (kDebugMode) {
              print('Message type "message" but no data field found');
            }
          }
          break;
        case 'stream':
          // 格式3: Stream类型消息（如content_delta事件）
          _handleStreamMessage(wsMessage.rawData);
          break;
        default:
          // 其他类型尝试作为通知处理（兼容旧格式）
          if (kDebugMode) {
            print('Unknown message type "${wsMessage.type}", trying to parse as notification');
          }
          _handleNotificationMessage(wsMessage.rawData);
      }
    } catch (e, stackTrace) {
      if (kDebugMode) {
        print('Error processing message: $e');
        print('Stack trace: $stackTrace');
      }
    }
  }

  /// 处理Stream类型消息
  /// 格式: {"type": "stream", "event": "content_delta", "content": "xxx", "agent_id": "xxx", ...}
  /// Stream消息单独存储，不存入本地数据库，也不推送到微信，不需要标记已读
  void _handleStreamMessage(Map<String, dynamic>? data) {
    if (data == null) return;

    final event = data['event']?.toString();
    final content = data['content']?.toString() ?? '';
    final agentId = data['agent_id']?.toString() ?? '未知Agent';
    final sessionId = data['session_id']?.toString() ?? '';
    final eventId = data['event_id']?.toString() ?? '';

    // 只处理content_delta事件（完整内容）
    if (event != 'content_delta' || content.isEmpty) {
      if (kDebugMode) {
        print('Stream message ignored: event=$event, hasContent=${content.isNotEmpty}');
      }
      return;
    }

    // 使用event_id作为消息唯一标识（每条消息都有唯一的event_id）
    final messageId = eventId.isNotEmpty ? eventId : DateTime.now().millisecondsSinceEpoch.toString();

    // 检查是否已存在相同event_id的消息（去重）
    final existingIndex = streamMessages.indexWhere((m) => m['event_id'] == messageId);
    if (existingIndex != -1) {
      // 已存在，更新内容
      streamMessages[existingIndex] = {
        ...streamMessages[existingIndex],
        'content': content,
        'timestamp': DateTime.now().millisecondsSinceEpoch,
      };
      // 通知Stream消息监听者
      _streamMessageController.add(streamMessages[existingIndex]);
      if (kDebugMode) {
        print('Updated existing stream message: event_id=$messageId');
      }
      return;
    }

    // 构建Stream消息数据（直接使用原始数据格式）
    final streamMessage = {
      'event_id': messageId,
      'session_id': sessionId,
      'agent_id': agentId,
      'content': content,
      'room_id': data['room_id'],
      'run_id': data['run_id'],
      'status': data['status'],
      'role': data['role'],
      'event': event,
      'original_timestamp': data['timestamp'],
      'received_at': DateTime.now().millisecondsSinceEpoch,
    };

    if (kDebugMode) {
      print('Creating stream message: event_id=$messageId, agent=$agentId');
    }

    // 添加到Stream消息列表头部（最新的在前面）
    streamMessages.insert(0, streamMessage);

    // 通知Stream消息监听者
    _streamMessageController.add(streamMessage);

    if (kDebugMode) {
      print('Stream message processed successfully');
    }
  }

  /// 处理通知消息数据
  void _handleNotificationMessage(Map<String, dynamic>? data) {
    if (data == null) return;

    final notification = NotificationMessage(
      id: data['id']?.toString() ?? DateTime.now().millisecondsSinceEpoch.toString(),
      title: data['title']?.toString() ?? '新消息',
      content: data['content']?.toString() ?? '',
      level: data['level']?.toString() ?? 'info',
      timestamp: DateTime.now(),
      isRead: false,
      extraData: data['extraData'] as Map<String, dynamic>?,
    );

    if (kDebugMode) {
      print(
        'Creating notification: id=${notification.id}, title=${notification.title}',
      );
    }

    // 保存到本地并更新列表
    _storageService
        .addMessage(notification)
        .then((_) {
          // 添加到消息列表头部（最新的在前面）
          messages.insert(0, notification);

          // 更新未读数量
          unreadCount.value++;

          // 通知监听者
          _messageController.add(notification);

          // 推送到微信（如果微信已登录）
          _pushToWeChat(notification);

          if (kDebugMode) {
            print('Notification processed successfully');
          }
        })
        .catchError((e) {
          if (kDebugMode) {
            print('Error saving notification: $e');
          }
        });
  }

  /// 推送消息到微信
  /// 当微信已登录时，将通知消息推送给登录用户自己
  Future<void> _pushToWeChat(NotificationMessage notification) async {
    try {
      // 检查微信是否已连接
      if (!_wechatService.isConnected) {
        if (kDebugMode) {
          print('WeChat not connected, skipping push');
        }
        return;
      }

      // 获取当前登录用户的微信ID
      final targetUserId = _wechatService.currentUserId;

      if (targetUserId == null || targetUserId.isEmpty) {
        if (kDebugMode) {
          print('WeChat user ID not available, skipping push');
        }
        return;
      }

      // 构建微信消息内容
      final StringBuffer messageContent = StringBuffer();
      messageContent.writeln('[通知]');
      messageContent.writeln(notification.content);
      if (notification.level != 'info') {
        // messageContent.writeln('级别: ${notification.level}');
      }

      // 发送消息到微信
      await _wechatService.sendMessage(targetUserId, messageContent.toString());

      if (kDebugMode) {
        print('Notification pushed to WeChat user: $targetUserId');
      }
    } catch (e) {
      if (kDebugMode) {
        print('Error pushing notification to WeChat: $e');
      }
      // 推送失败不影响本地消息存储
    }
  }

  /// 手动重新连接
  Future<void> reconnect() async {
    await disconnect();
    await _connect();
  }

  /// 断开连接
  Future<void> disconnect() async {
    _client?.dispose();
    _client = null;
    isConnected.value = false;
  }

  /// 获取所有消息
  Future<List<NotificationMessage>> getAllMessages() async {
    return await _storageService.loadMessages();
  }

  /// 标记消息为已读
  Future<void> markAsRead(String messageId) async {
    await _storageService.markAsRead(messageId);
    // 更新本地列表中的消息状态
    final index = messages.indexWhere((m) => m.id == messageId);
    if (index != -1) {
      messages[index] = messages[index].copyWith(isRead: true);
    }
    await _loadUnreadCount();
  }

  /// 标记所有消息为已读
  Future<void> markAllAsRead() async {
    await _storageService.markAllAsRead();
    // 更新本地列表中的所有消息状态
    for (var i = 0; i < messages.length; i++) {
      messages[i] = messages[i].copyWith(isRead: true);
    }
    unreadCount.value = 0;
  }

  /// 删除消息
  Future<void> deleteMessage(String messageId) async {
    await _storageService.deleteMessage(messageId);
    messages.removeWhere((m) => m.id == messageId);
    await _loadUnreadCount();
  }

  /// 清除所有消息
  Future<void> clearAllMessages() async {
    await _storageService.clearAllMessages();
    messages.clear();
    unreadCount.value = 0;
  }

  /// 刷新消息列表
  Future<void> refreshMessages() async {
    await _loadMessages();
  }

  // ==================== Stream消息管理方法 ====================

  /// 删除Stream消息
  void deleteStreamMessage(String sessionId) {
    streamMessages.removeWhere((m) => m['session_id'] == sessionId);
  }

  /// 清除所有Stream消息
  void clearAllStreamMessages() {
    streamMessages.clear();
  }

  // ==================== 测试方法 ====================

  /// 测试方法：模拟接收一条普通消息（用于调试）
  void testReceiveMessage() {
    if (kDebugMode) {
      print('Test: Simulating message reception');
    }

    final notification = NotificationMessage(
      id: DateTime.now().millisecondsSinceEpoch.toString(),
      title: '测试消息',
      content: '这是一条测试消息，时间: ${DateTime.now()}',
      level: 'info',
      timestamp: DateTime.now(),
      isRead: false,
    );

    _storageService.addMessage(notification).then((_) {
      messages.insert(0, notification);
      unreadCount.value++;
      _messageController.add(notification);

      if (kDebugMode) {
        print('Test: Message added successfully');
      }
    });
  }

  /// 测试方法：模拟接收一条Stream消息（用于调试）
  void testReceiveStreamMessage() {
    if (kDebugMode) {
      print('Test: Simulating stream message reception');
    }

    _handleStreamMessage({
      'type': 'stream',
      'event': 'content_delta',
      'session_id': 'test_session_${DateTime.now().millisecondsSinceEpoch}',
      'room_id': 'test_room',
      'agent_id': 'test_agent',
      'run_id': 'test_run',
      'status': 'succeeded',
      'role': 'agent',
      'content': '这是一条测试Stream消息，时间: ${DateTime.now()}',
      'timestamp': DateTime.now().millisecondsSinceEpoch ~/ 1000,
    });

    if (kDebugMode) {
      print('Test: Stream message processed');
    }
  }

  @override
  void onClose() {
    _messageController.close();
    _streamMessageController.close();
    _client?.dispose();
    super.onClose();
  }
}
