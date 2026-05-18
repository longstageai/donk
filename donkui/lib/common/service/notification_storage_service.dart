import 'dart:convert';
import 'dart:io';
import 'package:path/path.dart' as path;
import '../model/notification_message.dart';

/// 通知消息存储服务
/// 负责通知消息的本地持久化存储
class NotificationStorageService {
  static const String _fileName = 'notifications.json';
  static const String _dataDir = 'data/ui';
  static const int _maxStorageCount = 500; // 最多存储500条消息

  /// 获取存储文件路径
  /// 使用程序所在目录下的 data/ui 文件夹
  Future<String> get _filePath async {
    final executableDir = File(Platform.resolvedExecutable).parent.path;
    final dataDir = Directory(path.join(executableDir, _dataDir));
    if (!await dataDir.exists()) {
      await dataDir.create(recursive: true);
    }
    return path.join(dataDir.path, _fileName);
  }

  /// 保存消息列表到本地
  Future<void> saveMessages(List<NotificationMessage> messages) async {
    try {
      final path = await _filePath;
      final file = File(path);

      // 限制存储数量，只保留最新的消息
      final messagesToSave =
          messages.length > _maxStorageCount
              ? messages.sublist(messages.length - _maxStorageCount)
              : messages;

      final List<Map<String, dynamic>> jsonList =
          messagesToSave.map((msg) => msg.toJson()).toList();

      final jsonString = jsonEncode(jsonList);
      await file.writeAsString(jsonString, flush: true);
    } catch (e) {
      // 静默处理保存失败
    }
  }

  /// 从本地加载消息列表
  Future<List<NotificationMessage>> loadMessages() async {
    try {
      final path = await _filePath;
      final file = File(path);

      if (!await file.exists()) {
        return [];
      }

      final jsonString = await file.readAsString();
      final List<dynamic> jsonList = jsonDecode(jsonString);

      return jsonList
          .map(
            (json) =>
                NotificationMessage.fromJson(json as Map<String, dynamic>),
          )
          .toList();
    } catch (e) {
      return [];
    }
  }

  /// 添加单条消息
  Future<void> addMessage(NotificationMessage message) async {
    final messages = await loadMessages();
    messages.add(message);
    await saveMessages(messages);
  }

  /// 标记消息为已读
  Future<void> markAsRead(String messageId) async {
    final messages = await loadMessages();
    final index = messages.indexWhere((m) => m.id == messageId);
    if (index != -1) {
      messages[index] = messages[index].copyWith(isRead: true);
      await saveMessages(messages);
    }
  }

  /// 标记所有消息为已读
  Future<void> markAllAsRead() async {
    final messages = await loadMessages();
    for (var i = 0; i < messages.length; i++) {
      messages[i] = messages[i].copyWith(isRead: true);
    }
    await saveMessages(messages);
  }

  /// 删除单条消息
  Future<void> deleteMessage(String messageId) async {
    final messages = await loadMessages();
    messages.removeWhere((m) => m.id == messageId);
    await saveMessages(messages);
  }

  /// 清除所有消息
  Future<void> clearAllMessages() async {
    try {
      final path = await _filePath;
      final file = File(path);
      if (await file.exists()) {
        await file.delete();
      }
    } catch (e) {
      // 静默处理清除失败
    }
  }

  /// 获取未读消息数量
  Future<int> getUnreadCount() async {
    final messages = await loadMessages();
    return messages.where((m) => !m.isRead).length;
  }
}
