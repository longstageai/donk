import 'dart:async';
import 'dart:convert';
import 'dart:developer' as developer;
import 'dart:io';
import 'dart:math';
import 'dart:typed_data';
import 'package:path_provider/path_provider.dart';
import 'types.dart';
import 'protocol/api.dart';
import 'auth/login.dart';
import 'crypto/aes.dart';

/// 微信 Bot 客户端主类
class WeChatBot {
  final WeChatOptions options;
  final ProtocolClient _protocol = ProtocolClient();
  final List<MessageHandler> _handlers = [];
  final Map<String, String> _typingTickets = {};

  Credentials? _credentials;
  String? _cursor;
  bool _stopped = true;
  bool _isRunning = false;
  LoginManager? _loginManager;
  String? _contextTokensFilePath;

  WeChatBot({this.options = const WeChatOptions()});

  /// 是否已登录
  bool get isLoggedIn => _credentials != null;

  /// 是否正在运行
  bool get isRunning => _isRunning;

  /// 获取凭证
  Credentials? get credentials => _credentials;

  /// 登录
  /// [force] 是否强制重新登录（忽略已保存的凭证）
  Future<Credentials> login({bool force = false}) async {
    _loginManager ??= LoginManager(client: _protocol, options: options);

    _credentials = await _loginManager!.login(force: force);
    _log(
      LogLevel.info,
      '登录成功，token=${_credentials!.token.substring(0, 20)}..., baseUrl=${_credentials!.baseUrl}',
    );

    // 更新协议客户端的 baseUrl
    if (_credentials!.baseUrl.isNotEmpty) {
      _protocol.updateBaseUrl(_credentials!.baseUrl);
      _log(LogLevel.info, '已更新 baseUrl: ${_credentials!.baseUrl}');
    }

    return _credentials!;
  }

  /// 登出
  Future<void> logout() async {
    stop();
    _credentials = null;
    _cursor = null;
    await _clearContextTokens();
    _typingTickets.clear();
    await _loginManager?.clearCredentials();
  }

  /// 注册消息处理器
  void onMessage(MessageHandler handler) {
    _handlers.add(handler);
  }

  /// 移除消息处理器
  void offMessage(MessageHandler handler) {
    _handlers.remove(handler);
  }

  /// 启动消息轮询
  Future<void> run() async {
    if (_isRunning) {
      throw StateError('Bot is already running');
    }

    if (_credentials == null) {
      throw StateError('Not logged in. Call login() first.');
    }

    _stopped = false;
    _isRunning = true;
    _log(LogLevel.info, 'Bot started');

    // 启动轮询循环
    _pollingLoop();
  }

  /// 停止消息轮询
  void stop() {
    _stopped = true;
    _isRunning = false;
    _log(LogLevel.info, 'Bot stopped');
  }

  /// 消息轮询循环
  Future<void> _pollingLoop() async {
    _log(
      LogLevel.info,
      '开始轮询，token=${_credentials!.token.substring(0, 20)}..., cursor=$_cursor',
    );
    while (!_stopped) {
      try {
        final response = await _protocol.getUpdates(
          token: _credentials!.token,
          cursor: _cursor ?? '',
        );

        // 检查会话过期
        if (response.isSessionTimeout) {
          _log(LogLevel.error, '会话已过期，需要重新登录');
          _handleSessionTimeout();
          break;
        }

        // 处理消息
        if (response.msgs.isNotEmpty) {
          for (final wireMsg in response.msgs) {
            await _handleIncomingMessage(wireMsg);
          }
        }

        // 更新游标
        if (response.nextCursor != null) {
          _cursor = response.nextCursor;
        }
      } catch (e) {
        _log(LogLevel.error, 'Polling error: $e');
        options.onError?.call(e);

        // 出错后等待一段时间再重试
        if (!_stopped) {
          await Future.delayed(const Duration(seconds: 5));
        }
      }
    }

    _isRunning = false;
  }

  /// 处理收到的消息
  Future<void> _handleIncomingMessage(WireMessage wireMsg) async {
    try {
      // 保存 context_token 到磁盘
      await _setContextToken(wireMsg.fromUserId, wireMsg.contextToken);
      // 转换为 IncomingMessage
      final incomingMsg = _parseWireMessage(wireMsg);

      _log(
        LogLevel.debug,
        'Received message from ${incomingMsg.fromUserId}: ${incomingMsg.contentType.value}',
      );

      // 调用所有处理器
      for (final handler in _handlers) {
        try {
          await handler(incomingMsg);
        } catch (e) {
          _log(LogLevel.error, 'Message handler error: $e');
        }
      }
    } catch (e) {
      _log(LogLevel.error, 'Error handling message: $e');
    }
  }

  /// 解析 WireMessage 为 IncomingMessage
  IncomingMessage _parseWireMessage(WireMessage wireMsg) {
    final item = wireMsg.itemList.firstOrNull;
    if (item == null) {
      throw Exception('Message has no items');
    }

    ContentType contentType;
    String? text;
    CDNMedia? media;

    switch (item.type) {
      case MessageItemType.text:
        contentType = ContentType.text;
        text = item.textItem?.text;
        break;
      case MessageItemType.image:
        contentType = ContentType.image;
        media = item.imageItem?.media;
        break;
      case MessageItemType.voice:
        contentType = ContentType.voice;
        text = item.voiceItem?.recognitionResult;
        media = item.voiceItem?.media;
        break;
      case MessageItemType.file:
        contentType = ContentType.file;
        media = item.fileItem?.media;
        break;
      case MessageItemType.video:
        contentType = ContentType.video;
        media = item.videoItem?.media;
        break;
    }

    return IncomingMessage(
      id: wireMsg.clientId,
      fromUserId: wireMsg.fromUserId,
      toUserId: wireMsg.toUserId,
      timestamp: DateTime.fromMillisecondsSinceEpoch(wireMsg.createTimeMs),
      contentType: contentType,
      text: text,
      media: media,
      contextToken: wireMsg.contextToken,
      rawMessage: wireMsg,
    );
  }

  /// 处理会话过期
  Future<void> _handleSessionTimeout() async {
    _credentials = null;
    _cursor = null;
    await _clearContextTokens();
    _typingTickets.clear();
    options.onError?.call(Exception('Session timeout, please login again'));
  }

  /// 回复消息
  Future<void> reply(IncomingMessage msg, String text) async {
    await sendMessage(
      userId: msg.fromUserId,
      text: text,
      contextToken: msg.contextToken,
    );
  }

  /// 发送文本消息
  Future<void> sendMessage({
    required String userId,
    required String text,
    String? contextToken,
  }) async {
    _ensureLoggedIn();
    final token = contextToken ?? await _getContextToken(userId);
    if (token == null) {
      throw Exception('No context token for user $userId');
    }

    final message = WireMessage(
      fromUserId: '',
      toUserId: userId,
      clientId: _generateClientId(),
      createTimeMs: DateTime.now().millisecondsSinceEpoch,
      messageType: MessageType.bot,
      messageState: MessageState.finish,
      contextToken: token,
      itemList: [
        MessageItem(type: MessageItemType.text, textItem: TextItem(text: text)),
      ],
    );

    await _protocol.sendMessage(token: _credentials!.token, message: message);

    _log(LogLevel.debug, 'Sent message to $userId: $text');
  }

  /// 发送输入状态
  Future<void> sendTyping(String userId) async {
    await _sendTypingStatus(userId, 1);
  }

  /// 停止输入状态
  Future<void> stopTyping(String userId) async {
    await _sendTypingStatus(userId, 2);
  }

  /// 发送输入状态（内部方法）
  Future<void> _sendTypingStatus(String userId, int status) async {
    _ensureLoggedIn();

    // 获取或缓存 typing_ticket
    var typingTicket = _typingTickets[userId];
    final contextToken = await _getContextToken(userId);

    if (typingTicket == null && contextToken != null) {
      try {
        final config = await _protocol.getConfig(
          token: _credentials!.token,
          ilinkUserId: userId,
          contextToken: contextToken,
        );
        typingTicket = config.typingTicket;
        _typingTickets[userId] = typingTicket;
      } catch (e) {
        _log(LogLevel.warn, 'Failed to get typing ticket: $e');
        return;
      }
    }

    if (typingTicket == null) {
      _log(LogLevel.warn, 'No typing ticket available for $userId');
      return;
    }

    await _protocol.sendTyping(
      token: _credentials!.token,
      ilinkUserId: userId,
      typingTicket: typingTicket,
      status: status,
    );
  }

  /// 回复媒体内容
  Future<void> replyContent(IncomingMessage msg, SendContent content) async {
    if (content is TextContent) {
      await reply(msg, content.text);
      return;
    }

    if (content is MediaContent) {
      await _sendMediaMessage(
        userId: msg.fromUserId,
        contextToken: msg.contextToken,
        content: content,
      );
      return;
    }

    throw ArgumentError('Unsupported content type: ${content.runtimeType}');
  }

  /// 发送媒体消息（内部方法）
  Future<void> _sendMediaMessage({
    required String userId,
    required String contextToken,
    required MediaContent content,
  }) async {
    _ensureLoggedIn();

    MessageItem item;
    switch (content.contentType) {
      case ContentType.image:
        item = MessageItem(
          type: MessageItemType.image,
          imageItem: ImageItem(media: content.media, midSize: content.size),
        );
        break;
      case ContentType.voice:
        item = MessageItem(
          type: MessageItemType.voice,
          voiceItem: VoiceItem(media: content.media),
        );
        break;
      case ContentType.file:
        item = MessageItem(
          type: MessageItemType.file,
          fileItem: FileItem(
            media: content.media,
            fileName: '', // 需要从content中获取
          ),
        );
        break;
      case ContentType.video:
        item = MessageItem(
          type: MessageItemType.video,
          videoItem: VideoItem(media: content.media),
        );
        break;
      default:
        throw ArgumentError('Unsupported media type: ${content.contentType}');
    }

    final message = WireMessage(
      fromUserId: '',
      toUserId: userId,
      clientId: _generateClientId(),
      createTimeMs: DateTime.now().millisecondsSinceEpoch,
      messageType: MessageType.bot,
      messageState: MessageState.finish,
      contextToken: contextToken,
      itemList: [item],
    );

    await _protocol.sendMessage(token: _credentials!.token, message: message);
  }

  /// 下载媒体文件
  Future<DownloadedMedia?> download(IncomingMessage msg) async {
    final media = msg.media;
    if (media == null) {
      _log(LogLevel.warn, 'Message has no media to download');
      return null;
    }

    try {
      // 构建下载URL
      final downloadUrl =
          media.fullUrl ??
          '${WeChatConstants.cdnBaseURL}/download?encrypted_query_param=${Uri.encodeComponent(media.encryptQueryParam)}';

      // 下载加密数据
      final encryptedData = await _protocol.downloadFromCDN(downloadUrl);

      // 解密数据
      final aesKey = AESKeyCodec.decode(media.aesKey);
      final decryptedData = AESCrypto.decrypt(encryptedData, aesKey);

      // 提取额外信息
      String? fileName;
      int? duration;
      String? recognitionResult;

      final item = msg.rawMessage.itemList.firstOrNull;
      if (item != null) {
        if (item.fileItem != null) {
          fileName = item.fileItem!.fileName;
        }
        if (item.voiceItem != null) {
          duration = item.voiceItem!.duration;
          recognitionResult = item.voiceItem!.recognitionResult;
        }
        if (item.videoItem != null) {
          duration = item.videoItem!.duration;
        }
      }

      return DownloadedMedia(
        contentType: msg.contentType,
        data: decryptedData,
        fileName: fileName,
        duration: duration,
        recognitionResult: recognitionResult,
      );
    } catch (e) {
      _log(LogLevel.error, 'Failed to download media: $e');
      return null;
    }
  }

  /// 上传并发送媒体文件
  Future<void> sendMedia({
    required String userId,
    required Uint8List data,
    required ContentType contentType,
    String? fileName,
    int? duration,
  }) async {
    _ensureLoggedIn();

    final contextToken = await _getContextToken(userId);
    if (contextToken == null) {
      throw Exception('No context token for user $userId');
    }

    // 1. 生成AES密钥并加密文件
    final aesKey = AESCrypto.generateKey();
    final encryptedData = AESCrypto.encrypt(data, aesKey);

    // 2. 计算文件信息
    final rawSize = data.length;
    final fileSize = encryptedData.length;
    final rawFileMd5 = AESCrypto.md5(data);
    final fileKey = rawFileMd5.substring(0, 16); // 取MD5前16位

    // 3. 确定媒体类型
    final mediaType = _contentTypeToMediaType(contentType);

    // 4. 获取上传URL
    final uploadUrlResponse = await _protocol.getUploadURL(
      token: _credentials!.token,
      fileKey: fileKey,
      mediaType: mediaType,
      toUserId: userId,
      rawSize: rawSize,
      rawFileMd5: rawFileMd5,
      fileSize: fileSize,
      aesKey: AESKeyCodec.encodeHex(aesKey),
    );

    // 5. 上传到CDN
    final encryptedParam = await _protocol.uploadToCDN(
      uploadUrl:
          uploadUrlResponse.uploadFullUrl ??
          '${WeChatConstants.cdnBaseURL}/upload?encrypted_query_param=${Uri.encodeComponent(uploadUrlResponse.uploadParam)}&filekey=$fileKey',
      encryptedData: encryptedData,
      headers: {},
    );

    // 6. 构建CDNMedia
    final cdnMedia = CDNMedia(
      encryptQueryParam: encryptedParam,
      aesKey: AESKeyCodec.encodeBase64(aesKey),
      encryptType: 1,
    );

    // 7. 发送媒体消息
    final mediaContent = MediaContent(
      contentType: contentType,
      media: cdnMedia,
      size: rawSize,
    );

    await _sendMediaMessage(
      userId: userId,
      contextToken: contextToken,
      content: mediaContent,
    );

    _log(LogLevel.debug, 'Sent media to $userId: ${contentType.value}');
  }

  /// 内容类型转媒体类型
  MediaType _contentTypeToMediaType(ContentType type) {
    switch (type) {
      case ContentType.image:
        return MediaType.image;
      case ContentType.video:
        return MediaType.video;
      case ContentType.voice:
        return MediaType.voice;
      case ContentType.file:
        return MediaType.file;
      default:
        return MediaType.file;
    }
  }

  /// 确保已登录
  void _ensureLoggedIn() {
    if (_credentials == null) {
      throw StateError('Not logged in. Call login() first.');
    }
  }

  /// 生成客户端ID
  String _generateClientId() {
    final random = Random.secure();
    final bytes = List<int>.generate(16, (_) => random.nextInt(256));
    return base64Encode(bytes);
  }

  /// 日志输出
  void _log(LogLevel level, String message) {
    if (level.value >= options.logLevel.value) {
      final prefix = '[WeChatBot][${level.name.toUpperCase()}]';
      developer.log('$prefix $message');
    }
  }

  /// 获取 context tokens 存储文件路径
  Future<String> _getContextTokensFilePath() async {
    if (_contextTokensFilePath != null) {
      return _contextTokensFilePath!;
    }
    final directory = await getApplicationDocumentsDirectory();
    final botDir = Directory('${directory.path}/wechatbot');
    if (!await botDir.exists()) {
      await botDir.create(recursive: true);
    }
    _contextTokensFilePath = '${botDir.path}/context_tokens.json';
    return _contextTokensFilePath!;
  }

  /// 从磁盘加载 context tokens
  Future<Map<String, String>> _loadContextTokens() async {
    try {
      final filePath = await _getContextTokensFilePath();
      final file = File(filePath);
      if (await file.exists()) {
        final content = await file.readAsString();
        final Map<String, dynamic> json = jsonDecode(content);
        return json.map((key, value) => MapEntry(key, value.toString()));
      }
    } catch (e) {
      _log(LogLevel.error, 'Failed to load context tokens: $e');
    }
    return {};
  }

  /// 保存 context tokens 到磁盘
  Future<void> _saveContextTokens(Map<String, String> tokens) async {
    try {
      final filePath = await _getContextTokensFilePath();
      final file = File(filePath);
      await file.writeAsString(jsonEncode(tokens));
    } catch (e) {
      _log(LogLevel.error, 'Failed to save context tokens: $e');
    }
  }

  /// 获取指定用户的 context token
  Future<String?> _getContextToken(String userId) async {
    final tokens = await _loadContextTokens();
    return tokens[userId];
  }

  /// 设置指定用户的 context token
  Future<void> _setContextToken(String userId, String token) async {
    final tokens = await _loadContextTokens();
    tokens[userId] = token;
    await _saveContextTokens(tokens);
  }

  /// 清除所有 context tokens
  Future<void> _clearContextTokens() async {
    try {
      final filePath = await _getContextTokensFilePath();
      final file = File(filePath);
      if (await file.exists()) {
        await file.delete();
      }
    } catch (e) {
      _log(LogLevel.error, 'Failed to clear context tokens: $e');
    }
  }
}

// 扩展 List 的 firstOrNull
extension ListExtension<T> on List<T> {
  T? get firstOrNull => isEmpty ? null : first;
}
