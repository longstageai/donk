import 'dart:convert';
import 'dart:io';
import 'package:path_provider/path_provider.dart';
import '../model/chat_message.dart';

/// 聊天消息存储服务
/// 负责消息的本地持久化存储
class ChatStorageService {
  static const String _fileName = 'chat_history.json';

  /// 获取存储文件路径
  /// 使用应用支持目录，适合存储应用数据文件
  Future<String> get _filePath async {
    final directory = await getApplicationSupportDirectory();
    return '${directory.path}/$_fileName';
  }

  /// 保存消息列表到本地
  /// 使用 flush: true 确保数据写入磁盘并关闭文件句柄
  Future<void> saveMessages(List<ChatMessage> messages) async {
    try {
      final path = await _filePath;
      final file = File(path);

      // 将消息列表转换为 JSON 列表
      final List<Map<String, dynamic>> jsonList =
          messages
              .map(
                (msg) => {
                  'id': msg.id,
                  'sender': msg.sender,
                  'content': msg.content,
                  'timestamp': msg.timestamp.toIso8601String(),
                  'reasoning': msg.reasoning,
                  'isReasoningCollapsed': msg.isReasoningCollapsed,
                  'isReasoning': msg.isReasoning,
                  'isError': msg.isError,
                },
              )
              .toList();

      final jsonString = jsonEncode(jsonList);
      // 使用 flush: true 确保数据完全写入磁盘
      await file.writeAsString(jsonString, flush: true);
    } catch (e) {
      // 静默处理保存失败
    }
  }

  /// 从本地加载消息列表
  Future<List<ChatMessage>> loadMessages() async {
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
            (json) => ChatMessage(
              id: json['id'] as String,
              sender: json['sender'] as String,
              content: json['content'] as String,
              timestamp: DateTime.parse(json['timestamp'] as String),
              reasoning: json['reasoning'] as String?,
              isReasoningCollapsed:
                  json['isReasoningCollapsed'] as bool? ?? true,
              isReasoning: json['isReasoning'] as bool? ?? false,
              isError: json['isError'] as bool? ?? false,
            ),
          )
          .toList();
    } catch (e) {
      // 加载失败返回空列表
      return [];
    }
  }

  /// 清除所有消息
  /// 使用重试机制处理文件被占用的情况
  Future<void> clearMessages() async {
    try {
      final path = await _filePath;
      final file = File(path);
      if (await file.exists()) {
        // 重试机制：最多重试3次
        for (int i = 0; i < 3; i++) {
          try {
            await file.delete();
            return;
          } catch (e) {
            if (i == 2) rethrow;
            // 等待一段时间后重试
            await Future.delayed(Duration(milliseconds: 100 * (i + 1)));
          }
        }
      }
    } catch (e) {
      // 静默处理清除失败
    }
  }
}
