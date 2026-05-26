import 'dart:async';

import 'package:donk/app/conf/config.dart' as app_config;
import 'package:get/get.dart';

import '../../common/client/sse_client.dart';
import '../../common/model/chat_message.dart';
import '../../common/model/notification_message.dart';
import '../../common/service/chat_storage_service.dart';
import '../../common/service/notification_websocket_service.dart';
import '../../common/service/wechat_bot_service.dart';
import '../../common/wechatbot/wechatbot.dart';
import '../../core/constants/sse_events.dart';

/// 首页控制器
/// 用于管理首页的业务逻辑和状态
class HomeController extends GetxController {
  // ==================== 依赖注入 ====================
  final WeChatBotService _wechatService;
  final ChatStorageService _storageService;
  final NotificationWebSocketService _notificationService;

  HomeController({
    required WeChatBotService wechatService,
    required ChatStorageService storageService,
    NotificationWebSocketService? notificationService,
  }) : _wechatService = wechatService,
       _storageService = storageService,
       _notificationService = notificationService ?? Get.find<NotificationWebSocketService>();

  // ==================== 配置 ====================
  static final String _sseUrl = app_config.sseUrl;

  // ==================== 状态 ====================
  /// SSE 客户端
  SSEClient? _sseClient;

  /// SSE 连接状态
  final Rx<SSEStatus> sseStatus = SSEStatus.disconnected.obs;

  /// 微信连接状态
  final Rx<WeChatConnectionStatus> wechatStatus =
      WeChatConnectionStatus.disconnected.obs;

  /// 错误信息
  final RxString errorMessage = ''.obs;

  /// 聊天消息列表（用于UI展示）
  final RxList<ChatMessage> chatMessages = <ChatMessage>[].obs;

  // ==================== 计算属性 ====================
  /// 是否显示对话界面
  bool get hasChatMessages => chatMessages.isNotEmpty;

  /// SSE 是否正在处理请求
  bool get isProcessing =>
      sseStatus.value == SSEStatus.connecting ||
      sseStatus.value == SSEStatus.connected;

  /// 微信是否已连接
  bool get isWeChatConnected =>
      wechatStatus.value == WeChatConnectionStatus.connected;

  /// 是否禁用首页输入
  bool get isInputDisabled => isWeChatConnected;

  /// Token 刷新触发器流
  RxInt get tokenRefreshTrigger => _tokenRefreshTrigger;

  // ==================== 流式消息处理状态 ====================
  String? _currentAgentMessageId;
  final StringBuffer _currentAgentContent = StringBuffer();
  final StringBuffer _currentAgentReasoning = StringBuffer();
  bool _isProcessingAgentResponse = false;

  // ==================== 微信消息处理状态 ====================
  String? _currentProcessingWeChatMsgId;

  /// 正在输入状态定时器
  Timer? _typingTimer;

  /// Token 刷新触发器 - 当 Agent 回复完成时触发
  final RxInt _tokenRefreshTrigger = 0.obs;

  // ==================== WebSocket 通知消息 ====================
  /// WebSocket 通知消息列表
  RxList<NotificationMessage> get notificationMessages => _notificationService.messages;

  /// 未读消息数量
  RxInt get unreadNotificationCount => _notificationService.unreadCount;

  /// WebSocket 连接状态
  RxBool get isNotificationConnected => _notificationService.isConnected;

  // ==================== 生命周期 ====================
  @override
  void onInit() {
    super.onInit();
    _initSSE();
    _loadChatHistory();
    _initWeChatListener();

    // 监听消息列表变化，自动保存（带防抖）
    debounce(
      chatMessages,
      (_) => _saveChatHistory(),
      time: const Duration(milliseconds: 500),
    );
  }

  @override
  void onClose() {
    _stopTypingTimer();
    _disposeSSE();
    super.onClose();
  }

  // ==================== 初始化方法 ====================
  void _initSSE() {
    _sseClient = SSEClient(url: _sseUrl);

    _sseClient!.statusStream.listen((status) {
      sseStatus.value = status;
    });

    _sseClient!.messageStream.listen((message) {
      _handleServerMessage(message);
    });

    _sseClient!.errorStream.listen((error) {
      _handleSSEError(error.toString());
    });
  }

  void _initWeChatListener() {
    // 监听微信连接状态
    _wechatService.connectionStatus.listen((status) {
      wechatStatus.value = status;
    });

    // 监听微信消息
    _wechatService.messageStream.listen(_handleWeChatMessage);
  }

  Future<void> _loadChatHistory() async {
    final messages = await _storageService.loadMessages();
    chatMessages.assignAll(messages);
  }

  // ==================== 公共方法 ====================
  /// 添加用户消息并发送给 Agent
  Future<void> addUserMessage(
    String content, {
    String? filePath,
    String? fileType,
  }) async {
    final message = ChatMessage.user(
      id: DateTime.now().millisecondsSinceEpoch.toString(),
      content: content,
      filePath: filePath,
      fileType: fileType,
    );
    chatMessages.add(message);

    // 输入框发送消息时，保持当前用户ID不变（如果已设置）
    // 这样Agent回复会发送给最近交互的微信用户

    await sendMessage(content, filePath: filePath, fileType: fileType);
  }

  /// 发送消息到服务器（UI 调用，会等待完成）
  Future<void> sendMessage(
    String content, {
    String? filePath,
    String? fileType,
  }) async {
    if (_sseClient == null) {
      _handleSSEError('SSE 客户端未初始化');
      return;
    }

    if (isProcessing) {
      _handleSSEError('正在处理请求中，请稍候');
      return;
    }

    try {
      await _sseClient!.connect(
        content,
        filePath: filePath,
        fileType: fileType,
      );
    } catch (e) {
      _handleSSEError('发送失败: $e');
    }
  }

  /// 启动 Agent 请求（不等待完成，用于微信消息处理）
  /// 避免阻塞微信消息轮询，防止连接超时
  void startAgentRequest(String content) {
    if (_sseClient == null) {
      return;
    }

    if (isProcessing) {
      return;
    }

    // 启动异步操作，不等待完成（避免阻塞微信消息轮询）
    _sseClient!
        .connect(content)
        .then((_) {
          // 连接成功，不执行任何操作（通过 Stream 监听结果）
        })
        .catchError((e) {
          _handleSSEError('发送失败: $e');
        });
  }

  /// 取消当前请求
  void cancelRequest() {
    _disconnectSSE();
    // 停止正在输入状态
    final userId = _wechatService.currentUserId;
    if (userId != null) {
      _stopTypingTimer();
      _wechatService.stopTyping(userId);
    }
    _finishCurrentMessage();
    errorMessage.value = '请求已取消';
  }

  /// 清除所有聊天消息
  Future<void> clearAllMessages() async {
    chatMessages.clear();
    await _storageService.clearMessages();
  }

  /// 切换思考过程折叠状态
  void toggleReasoning(int index) {
    if (index < 0 || index >= chatMessages.length) return;

    final msg = chatMessages[index];
    if (msg.isAgent && msg.reasoning != null) {
      chatMessages[index] = msg.copyWithToggleReasoning();
    }
  }

  /// 设置编辑文本
  void setEditText(String text) {
    pendingEditText.value = text;
  }

  /// 清空编辑文本
  void clearEditText() {
    pendingEditText.value = '';
  }

  // ==================== 编辑功能 ====================
  final RxString pendingEditText = ''.obs;

  // ==================== 微信消息处理 ====================
  Future<void> _handleWeChatMessage(IncomingMessage message) async {
    // 只处理文本消息和语音消息（语音消息有转文字内容）
    final isTextMessage = message.contentType == ContentType.text;
    final isVoiceMessage = message.contentType == ContentType.voice;

    if ((!isTextMessage && !isVoiceMessage) || message.text == null) {
      return;
    }

    // 检查是否正在处理该消息（防止重复）
    if (_currentProcessingWeChatMsgId == message.id) {
      return;
    }

    // 检查 SSE 是否正在处理中
    if (isProcessing) {
      return;
    }

    // 记录当前处理的微信消息
    _currentProcessingWeChatMsgId = message.id;

    // 将微信消息添加到聊天栏
    final userMessage = ChatMessage.user(
      id: 'wechat_user_${message.id}_${DateTime.now().millisecondsSinceEpoch}',
      content: message.text!,
    );
    chatMessages.add(userMessage);

    // 异步处理 Agent 回复，不阻塞微信消息处理
    // 这样可以避免 Agent 回复时间过长导致微信连接超时
    _processWeChatMessageAsync(message.text!, message.fromUserId);
  }

  /// 异步处理微信消息
  /// 在后台处理 Agent 请求，避免阻塞微信消息处理流程
  void _processWeChatMessageAsync(String text, String userId) {
    // 启动正在输入状态定时器（每3秒发送一次）
    _startTypingTimer(userId);

    // 启动 Agent 请求（不等待完成，避免阻塞微信消息轮询）
    startAgentRequest(text);
  }

  /// 启动正在输入状态定时器
  void _startTypingTimer(String userId) {
    // 先停止之前的定时器
    _stopTypingTimer();

    // 立即发送一次
    _wechatService.sendTyping(userId);

    // 每3秒发送一次，保持正在输入状态
    _typingTimer = Timer.periodic(const Duration(seconds: 3), (_) {
      // 检查当前用户是否仍然是同一个
      if (_wechatService.currentUserId != null) {
        _wechatService.sendTyping(userId);
      }
    });
  }

  /// 停止正在输入状态定时器
  void _stopTypingTimer() {
    _typingTimer?.cancel();
    _typingTimer = null;
  }

  // ==================== SSE 消息处理 ====================
  void _handleServerMessage(SSEMessage message) {
    if (!message.isStream) return;

    switch (message.event) {
      case SSEEvents.userInput:
        break;

      case SSEEvents.reasoningDelta:
        _handleReasoningDelta(message.reasoningContent ?? '');
        break;

      case SSEEvents.contentDelta:
        _handleContentDelta(message.content ?? '');
        break;

      case SSEEvents.toolCall:
        _handleToolCall(message.toolName ?? '', message.toolInput ?? '');
        break;

      case SSEEvents.toolResult:
        _handleToolResult(message.toolName ?? '', message.toolResult ?? '');
        break;

      case SSEEvents.warning:
        _handleWarning(message.warning ?? '');
        break;

      case SSEEvents.stop:
        _finishCurrentMessage();
        _disconnectSSE();
        break;

      case SSEEvents.error:
        _handleSSEError(message.error ?? '未知错误');
        _finishCurrentMessage();
        _disconnectSSE();
        break;

      case SSEEvents.canceled:
        _finishCurrentMessage();
        _disconnectSSE();
        break;
    }
  }

  void _handleReasoningDelta(String delta) {
    _isProcessingAgentResponse = true;
    if (_currentAgentMessageId == null) {
      _currentAgentMessageId = DateTime.now().millisecondsSinceEpoch.toString();
      _currentAgentReasoning.write(delta);
      chatMessages.add(
        ChatMessage.agent(
          id: _currentAgentMessageId!,
          content: '',
          reasoning: _currentAgentReasoning.toString(),
          isReasoning: true,
        ),
      );
    } else {
      _currentAgentReasoning.write(delta);
      _updateCurrentAgentMessage(isReasoning: true);
    }
  }

  void _handleContentDelta(String delta) {
    if (_currentAgentMessageId == null) {
      _currentAgentMessageId = DateTime.now().millisecondsSinceEpoch.toString();
      _currentAgentContent.write(delta);
      chatMessages.add(
        ChatMessage.agent(
          id: _currentAgentMessageId!,
          content: _currentAgentContent.toString(),
          reasoning:
              _currentAgentReasoning.isNotEmpty
                  ? _currentAgentReasoning.toString()
                  : null,
          isReasoning: false,
        ),
      );
    } else {
      _currentAgentContent.write(delta);
      _updateCurrentAgentMessage(isReasoning: false);
    }
  }

  void _handleToolCall(String toolName, String toolInput) {
    if (!_isProcessingAgentResponse) return;

    _currentAgentReasoning.writeln('\n[工具调用] $toolName');
    if (toolInput.isNotEmpty) {
      _currentAgentReasoning.writeln('输入: $toolInput');
    }

    if (_currentAgentMessageId != null) {
      _updateCurrentAgentMessage(isReasoning: true);
    }
  }

  void _handleToolResult(String toolName, String toolResult) {
    if (!_isProcessingAgentResponse) return;

    _currentAgentReasoning.writeln('[工具结果] $toolName: $toolResult\n');

    if (_currentAgentMessageId != null) {
      _updateCurrentAgentMessage(isReasoning: true);
    }
  }

  void _handleWarning(String warning) {
    chatMessages.add(
      ChatMessage.agent(
        id: DateTime.now().millisecondsSinceEpoch.toString(),
        content: '⚠️ $warning',
        isSystem: true,
      ),
    );
  }

  void _handleSSEError(String error) {
    errorMessage.value = error;
    chatMessages.add(
      ChatMessage.agent(
        id: DateTime.now().microsecondsSinceEpoch.toString(),
        content: '出错：$error',
        isSystem: true,
        isError: true,
      ),
    );

    // 发送错误信息给微信
    _sendErrorToWeChat(error);

    final userId = _wechatService.currentUserId;
    if (userId != null) {
      _stopTypingTimer();
      _wechatService.stopTyping(userId);
      _currentProcessingWeChatMsgId = null;
    }
  }

  /// 发送错误信息到微信
  Future<void> _sendErrorToWeChat(String error) async {
    final userId = _wechatService.currentUserId;
    if (userId == null || !_wechatService.isConnected) {
      return;
    }

    try {
      await _wechatService.sendMessage(userId, '抱歉，处理过程中出现错误：$error');
    } catch (e) {
      // 发送失败静默处理
    }
  }

  void _updateCurrentAgentMessage({bool? isReasoning}) {
    if (_currentAgentMessageId == null) return;

    final index = chatMessages.indexWhere(
      (m) => m.id == _currentAgentMessageId,
    );
    if (index >= 0) {
      final currentMsg = chatMessages[index];
      chatMessages[index] = ChatMessage.agent(
        id: _currentAgentMessageId!,
        content: _currentAgentContent.toString(),
        reasoning:
            _currentAgentReasoning.isNotEmpty
                ? _currentAgentReasoning.toString()
                : null,
        timestamp: currentMsg.timestamp,
        isReasoning: isReasoning ?? currentMsg.isReasoning,
      );
    }
  }

  // ==================== 消息完成处理 ====================
  Future<void> _finishCurrentMessage() async {
    // 停止正在输入状态定时器
    _stopTypingTimer();

    // 发送回复到微信
    await _sendReplyToWeChat();

    // 停止输入状态
    final userId = _wechatService.currentUserId;
    if (userId != null) {
      await _wechatService.stopTyping(userId);
    }

    // 清理状态
    _currentAgentMessageId = null;
    _currentAgentContent.clear();
    _currentAgentReasoning.clear();
    _isProcessingAgentResponse = false;
    _currentProcessingWeChatMsgId = null;

    // 保存历史
    await _saveChatHistoryImmediately();

    // 触发 Token 刷新
    _tokenRefreshTrigger.value++;
  }

  Future<void> _sendReplyToWeChat() async {
    // 从 WeChatBotService 获取当前用户ID（bot自己的ID，即聊天对象）
    final userId = _wechatService.currentUserId;
    if (userId == null ||
        _currentAgentContent.isEmpty ||
        !_wechatService.isConnected) {
      return;
    }

    final replyContent = _currentAgentContent.toString();

    try {
      await _wechatService.sendMessage(userId, replyContent);
    } catch (e) {
      // 发送失败静默处理
    }
  }

  // ==================== 存储 ====================
  Future<void> _saveChatHistory() async {
    await _storageService.saveMessages(chatMessages.toList());
  }

  Future<void> _saveChatHistoryImmediately() async {
    await _storageService.saveMessages(chatMessages.toList());
  }

  // ==================== SSE 连接管理 ====================
  void _disconnectSSE() {
    _sseClient?.disconnect();
  }

  void _disposeSSE() {
    _sseClient?.dispose();
    _sseClient = null;
  }
}
