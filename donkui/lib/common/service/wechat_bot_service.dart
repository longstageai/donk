import 'dart:async';
import 'dart:convert';
import 'dart:io';
import 'package:path_provider/path_provider.dart';
import '../wechatbot/wechatbot.dart';

/// 微信 Bot 服务
/// 管理微信 iLink Bot 的连接、消息处理和状态
class WeChatBotService {
  static WeChatBotService? _instance;
  WeChatBot? _bot;

  // 状态流
  final _connectionStatusController =
      StreamController<WeChatConnectionStatus>.broadcast();
  final _qrCodeController = StreamController<String?>.broadcast();
  final _messageController = StreamController<IncomingMessage>.broadcast();

  Stream<WeChatConnectionStatus> get connectionStatus =>
      _connectionStatusController.stream;
  Stream<String?> get qrCodeStream => _qrCodeController.stream;
  Stream<IncomingMessage> get messageStream => _messageController.stream;

  // 当前状态
  WeChatConnectionStatus _currentStatus = WeChatConnectionStatus.disconnected;
  String? _currentQRCode;
  String? _errorMessage;

  WeChatConnectionStatus get currentStatus => _currentStatus;
  String? get currentQRCode => _currentQRCode;
  String? get errorMessage => _errorMessage;
  bool get isConnected => _currentStatus == WeChatConnectionStatus.connected;
  bool get isConnecting => _currentStatus == WeChatConnectionStatus.connecting;

  WeChatBotService._internal();

  factory WeChatBotService() {
    _instance ??= WeChatBotService._internal();
    return _instance!;
  }

  /// 初始化 Bot
  Future<void> initialize() async {
    if (_bot != null) return;

    final credPath = await _getCredPath();

    _bot = WeChatBot(
      options: WeChatOptions(
        credPath: credPath,
        logLevel: LogLevel.info,
        onQrUrl: (url) {
          _currentQRCode = url;
          _qrCodeController.add(url);
          _updateStatus(WeChatConnectionStatus.waitingForScan);
        },
        onScanned: () {
          _updateStatus(WeChatConnectionStatus.scanning);
        },
        onExpired: () {
          _currentQRCode = null;
          _qrCodeController.add(null);
        },
        onError: (error) {
          _errorMessage = error.toString();
          _updateStatus(WeChatConnectionStatus.error);

          // 如果是会话过期错误，自动重新登录
          if (error.toString().contains('Session timeout') ||
              error.toString().contains('412')) {
            _handleSessionTimeout();
          }
        },
      ),
    );

    // 注册消息处理器
    _bot!.onMessage(_handleMessage);
  }

  /// 获取当前登录用户的微信ID
  /// 返回 null 如果未登录
  String? get currentUserId => _bot?.credentials?.userId;

  /// 检查是否有有效的微信凭证
  Future<bool> hasValidCredentials() async {
    final credPath = await _getCredPath();
    final file = File(credPath);
    if (!await file.exists()) {
      return false;
    }

    try {
      final content = await file.readAsString();
      final json = jsonDecode(content) as Map<String, dynamic>;

      // 检查必要的字段是否存在
      return json['token'] != null && json['token'].toString().isNotEmpty;
    } catch (e) {
      return false;
    }
  }

  /// 连接到微信
  Future<void> connect() async {
    if (_bot == null) {
      await initialize();
    }

    if (_currentStatus == WeChatConnectionStatus.connected) {
      return;
    }

    _updateStatus(WeChatConnectionStatus.connecting);
    _errorMessage = null;

    try {
      // 尝试登录（会自动加载已保存的凭证）
      await _bot!.login();

      // 登录成功，启动消息轮询
      _updateStatus(WeChatConnectionStatus.connected);
      _currentQRCode = null;
      _qrCodeController.add(null);

      // 启动轮询（非阻塞）
      _bot!.run().catchError((e) {
        _errorMessage = e.toString();
        _updateStatus(WeChatConnectionStatus.error);

        // 如果轮询出现 412 错误，可能是凭证过期，尝试强制重新登录
        if (e.toString().contains('412')) {
          _forceReLogin();
        }
      });
    } catch (e) {
      _errorMessage = e.toString();
      _updateStatus(WeChatConnectionStatus.error);
      rethrow;
    }
  }

  /// 断开连接
  Future<void> disconnect() async {
    if (_bot == null) return;

    _bot!.stop();
    await _bot!.logout();
    _updateStatus(WeChatConnectionStatus.disconnected);
  }

  /// 重新连接
  Future<void> reconnect() async {
    await disconnect();
    await connect();
  }

  /// 强制重新登录（清除凭证）
  Future<void> forceReconnect() async {
    if (_bot != null) {
      await _bot!.logout();
    }
    _currentQRCode = null;
    await connect();
  }

  /// 强制重新登录（删除凭证文件并显示二维码）
  Future<void> _forceReLogin() async {
    // 停止当前连接
    _bot?.stop();

    // 删除凭证文件
    try {
      final credPath = await _getCredPath();
      final file = File(credPath);
      if (await file.exists()) {
        await file.delete();
      }
    } catch (e) {
      // 删除失败静默处理
    }

    // 重置状态
    _currentStatus = WeChatConnectionStatus.disconnected;
    _currentQRCode = null;
    _errorMessage = null;

    // 延迟后重新连接（会显示二维码）
    await Future.delayed(const Duration(seconds: 1));

    // 重新初始化 bot（因为凭证已删除）
    _bot = null;
    await initialize();

    // 重新连接（会显示二维码）
    await connect();
  }

  /// 发送消息
  Future<void> sendMessage(String userId, String text) async {
    if (_bot == null || !isConnected) {
      throw StateError('Bot not connected');
    }
    await _bot!.sendMessage(userId: userId, text: text);
  }

  /// 回复消息
  Future<void> replyMessage(IncomingMessage msg, String text) async {
    if (_bot == null || !isConnected) {
      throw StateError('Bot not connected');
    }
    await _bot!.reply(msg, text);
  }

  /// 发送正在输入状态
  Future<void> sendTyping(String userId) async {
    if (_bot == null || !isConnected) {
      return;
    }
    try {
      await _bot!.sendTyping(userId);
    } catch (e) {
      // 发送失败静默处理
    }
  }

  /// 停止输入状态
  Future<void> stopTyping(String userId) async {
    if (_bot == null || !isConnected) {
      return;
    }
    try {
      await _bot!.stopTyping(userId);
    } catch (e) {
      // 停止失败静默处理
    }
  }

  /// 处理收到的消息
  Future<void> _handleMessage(IncomingMessage message) async {
    _messageController.add(message);
  }

  /// 更新状态
  void _updateStatus(WeChatConnectionStatus status) {
    _currentStatus = status;
    _connectionStatusController.add(status);
  }

  /// 处理会话过期
  Future<void> _handleSessionTimeout() async {
    // 停止当前连接
    _bot?.stop();

    // 清除凭证文件
    try {
      final credPath = await _getCredPath();
      final file = File(credPath);
      if (await file.exists()) {
        await file.delete();
      }
    } catch (e) {
      // 删除失败静默处理
    }

    // 重置状态
    _currentQRCode = null;
    _qrCodeController.add(null);

    // 延迟后重新连接（显示二维码）
    await Future.delayed(const Duration(seconds: 1));
    await connect();
  }

  /// 获取凭证保存路径
  Future<String> _getCredPath() async {
    final appDir = await getApplicationSupportDirectory();
    return '${appDir.path}/wechat_credentials.json';
  }

  /// 释放资源
  void dispose() {
    _bot?.stop();
    _connectionStatusController.close();
    _qrCodeController.close();
    _messageController.close();
    _instance = null;
  }
}

/// 微信连接状态
enum WeChatConnectionStatus {
  disconnected, // 未连接
  connecting, // 连接中
  waitingForScan, // 等待扫码
  scanning, // 已扫码，等待确认
  connected, // 已连接
  error, // 错误
}

/// 微信连接状态扩展
extension WeChatConnectionStatusExtension on WeChatConnectionStatus {
  String get displayName {
    switch (this) {
      case WeChatConnectionStatus.disconnected:
        return '未连接';
      case WeChatConnectionStatus.connecting:
        return '连接中...';
      case WeChatConnectionStatus.waitingForScan:
        return '等待扫码';
      case WeChatConnectionStatus.scanning:
        return '已扫码，等待确认';
      case WeChatConnectionStatus.connected:
        return '已连接';
      case WeChatConnectionStatus.error:
        return '连接错误';
    }
  }

  bool get isError => this == WeChatConnectionStatus.error;
}
