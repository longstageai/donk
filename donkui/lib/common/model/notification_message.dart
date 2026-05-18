/// 通知消息数据模型
class NotificationMessage {
  /// 消息唯一ID
  final String id;

  /// 消息标题
  final String title;

  /// 消息内容
  final String content;

  /// 消息级别 (info|success|warning|error)
  final String level;

  /// 发送时间
  final DateTime timestamp;

  /// 是否已读
  final bool isRead;

  /// 附加数据
  final Map<String, dynamic>? extraData;

  NotificationMessage({
    required this.id,
    required this.title,
    required this.content,
    this.level = 'info',
    required this.timestamp,
    this.isRead = false,
    this.extraData,
  });

  /// 从JSON构造
  /// 支持格式: {"type": "notification", "id": "xxx", "title": "xxx", "content": "xxx", "level": "info"}
  factory NotificationMessage.fromJson(Map<String, dynamic> json) {
    return NotificationMessage(
      id: json['id'] as String,
      title: json['title'] as String,
      content: json['content'] as String,
      level: json['level'] as String? ?? 'info',
      timestamp: DateTime.parse(json['timestamp'] as String),
      isRead: json['isRead'] as bool? ?? false,
      extraData: json['extraData'] as Map<String, dynamic>?,
    );
  }

  /// 转换为JSON
  Map<String, dynamic> toJson() {
    return {
      'id': id,
      'title': title,
      'content': content,
      'level': level,
      'timestamp': timestamp.toIso8601String(),
      'isRead': isRead,
      'extraData': extraData,
    };
  }

  /// 复制并修改
  NotificationMessage copyWith({
    String? id,
    String? title,
    String? content,
    String? level,
    DateTime? timestamp,
    bool? isRead,
    Map<String, dynamic>? extraData,
  }) {
    return NotificationMessage(
      id: id ?? this.id,
      title: title ?? this.title,
      content: content ?? this.content,
      level: level ?? this.level,
      timestamp: timestamp ?? this.timestamp,
      isRead: isRead ?? this.isRead,
      extraData: extraData ?? this.extraData,
    );
  }
}
