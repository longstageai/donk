/// 聊天消息模型
class ChatMessage {
  /// 消息ID
  final String id;

  /// 发送者类型：user-用户, agent-Agent, system-系统
  final String sender;

  /// 消息内容
  final String content;

  /// 消息时间戳
  final DateTime timestamp;

  final String? filePath;
  final String? fileType;

  /// 思考过程（Agent消息可选）
  final String? reasoning;

  /// 是否已折叠思考过程
  final bool isReasoningCollapsed;

  /// 是否还在思考中
  final bool isReasoning;
  final bool isError;

  ChatMessage({
    required this.id,
    required this.sender,
    required this.content,
    required this.timestamp,
    this.filePath,
    this.fileType,
    this.reasoning,
    this.isReasoningCollapsed = true,
    this.isReasoning = false,
    this.isError = false,
  });

  /// 创建用户消息
  factory ChatMessage.user({
    required String id,
    required String content,
    DateTime? timestamp,
    String? filePath,
    String? fileType,
  }) {
    return ChatMessage(
      id: id,
      sender: 'user',
      content: content,
      timestamp: timestamp ?? DateTime.now(),
      filePath: filePath,
      fileType: fileType,
    );
  }

  /// 创建Agent消息
  factory ChatMessage.agent({
    required String id,
    required String content,
    String? reasoning,
    DateTime? timestamp,
    bool isReasoning = false,
    bool isSystem = false,
    bool isError = false,
  }) {
    return ChatMessage(
      id: id,
      sender: isSystem ? 'system' : 'agent',
      content: content,
      timestamp: timestamp ?? DateTime.now(),
      reasoning: reasoning,
      isReasoning: isReasoning,
      isError: isError,
      // 思考中时默认展开，完成后可折叠
      isReasoningCollapsed: !isReasoning,
    );
  }

  /// 切换思考过程折叠状态
  ChatMessage copyWithToggleReasoning() {
    return ChatMessage(
      id: id,
      sender: sender,
      content: content,
      timestamp: timestamp,
      filePath: filePath,
      fileType: fileType,
      reasoning: reasoning,
      isReasoningCollapsed: !isReasoningCollapsed,
      isReasoning: isReasoning,
      isError: isError,
    );
  }

  /// 标记思考完成
  ChatMessage copyWithReasoningComplete() {
    return ChatMessage(
      id: id,
      sender: sender,
      content: content,
      timestamp: timestamp,
      filePath: filePath,
      fileType: fileType,
      reasoning: reasoning,
      isReasoningCollapsed: true, // 完成后默认折叠
      isReasoning: false,
      isError: isError,
    );
  }

  /// 是否为Agent消息
  bool get isAgent => sender == 'agent';

  bool get hasFile => filePath != null && filePath!.isNotEmpty;

  /// 是否为用户消息
  bool get isUser => sender == 'user';
}
