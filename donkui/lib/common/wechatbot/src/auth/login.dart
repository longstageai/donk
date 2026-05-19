import 'dart:async';
import 'dart:developer' as developer;
import 'dart:io';
import '../types.dart';
import '../protocol/api.dart';

/// 登录管理器
/// 处理二维码登录流程
class LoginManager {
  final ProtocolClient _client;
  final WeChatOptions _options;

  LoginManager({required ProtocolClient client, required WeChatOptions options})
    : _client = client,
      _options = options;

  /// 执行登录流程
  /// 返回凭证信息
  Future<Credentials> login({bool force = false}) async {
    // 如果非强制登录，尝试加载已有凭证
    if (!force) {
      final savedCreds = await _loadCredentials();
      if (savedCreds != null) {
        _log(LogLevel.info, '使用已保存的凭证');
        return savedCreds;
      }
    }

    // 开始新的登录流程
    return _performLogin();
  }

  /// 执行完整的登录流程
  Future<Credentials> _performLogin() async {
    try {
      // 1. 获取二维码
      final qrResponse = await _client.getBotQRCode();
      _log(LogLevel.info, '获取二维码成功: ${qrResponse.qrcodeImgContent}');

      // 通知二维码URL
      if (qrResponse.qrcodeImgContent.isNotEmpty) {
        _options.onQrUrl?.call(qrResponse.qrcodeImgContent);
      } else {
        _log(LogLevel.error, '二维码URL为空');
      }

      // 2. 轮询扫码状态
      final credentials = await _pollQRCodeStatus(qrResponse.qrcode);

      if (credentials != null) {
        // 保存凭证
        await _saveCredentials(credentials);
        _log(LogLevel.info, '登录成功');
        return credentials;
      }
    } on QRCodeExpiredException {
      _log(LogLevel.warn, '二维码已过期');
      _options.onExpired?.call();
      throw Exception('登录失败：二维码已过期，请手动重新获取二维码');
    }

    throw Exception('登录失败：未获取到登录凭证');
  }

  /// 轮询二维码状态
  Future<Credentials?> _pollQRCodeStatus(String qrcode) async {
    String currentQrcode = qrcode;
    String? redirectHost;

    while (true) {
      try {
        final statusResponse = await _client.getQRCodeStatus(currentQrcode);

        switch (statusResponse.status) {
          case QRCodeStatus.wait:
            // 继续等待扫码
            await Future.delayed(const Duration(seconds: 2));
            break;

          case QRCodeStatus.scaned:
            // 已扫码，等待确认
            _log(LogLevel.info, '已扫码，等待确认');
            _options.onScanned?.call();
            await Future.delayed(const Duration(seconds: 2));
            break;

          case QRCodeStatus.confirmed:
            // 登录确认成功
            if (statusResponse.botToken != null &&
                statusResponse.ilinkBotId != null &&
                statusResponse.ilinkUserId != null) {
              return Credentials(
                token: statusResponse.botToken!,
                baseUrl:
                    statusResponse.baseurl ?? WeChatConstants.defaultBaseURL,
                accountId: statusResponse.ilinkBotId!,
                userId: statusResponse.ilinkUserId!,
                savedAt: DateTime.now(),
              );
            }
            throw Exception('登录确认成功但缺少必要信息');

          case QRCodeStatus.expired:
            // 二维码过期
            throw QRCodeExpiredException();

          case QRCodeStatus.scanedButRedirect:
            // 需要重定向到新的IDC
            _log(LogLevel.info, '需要重定向到新的服务器: ${statusResponse.redirectHost}');
            if (statusResponse.redirectHost != null) {
              redirectHost = statusResponse.redirectHost;
              _client.updateBaseUrl('https://$redirectHost');
              // 使用新的 baseUrl 重新获取二维码状态
              await Future.delayed(const Duration(seconds: 1));
            }
            break;
        }
      } catch (e) {
        if (e is QRCodeExpiredException) {
          rethrow;
        }
        _log(LogLevel.error, '轮询二维码状态出错: $e');
        _options.onError?.call(e);
        await Future.delayed(const Duration(seconds: 2));
      }
    }
  }

  /// 从文件加载凭证
  Future<Credentials?> _loadCredentials() async {
    try {
      final credPath = _options.credPath;
      if (credPath == null) return null;

      final file = File(credPath);
      if (!await file.exists()) return null;

      final jsonString = await file.readAsString();
      return Credentials.fromJsonString(jsonString);
    } catch (e) {
      _log(LogLevel.warn, '加载凭证失败: $e');
      return null;
    }
  }

  /// 保存凭证到文件
  Future<void> _saveCredentials(Credentials credentials) async {
    try {
      final credPath = _options.credPath;
      if (credPath == null) return;

      final file = File(credPath);
      final dir = file.parent;
      if (!await dir.exists()) {
        await dir.create(recursive: true);
      }

      await file.writeAsString(credentials.toJsonString());
      _log(LogLevel.info, '凭证已保存');
    } catch (e) {
      _log(LogLevel.warn, '保存凭证失败: $e');
    }
  }

  /// 清除保存的凭证
  Future<void> clearCredentials() async {
    try {
      final credPath = _options.credPath;
      if (credPath == null) return;

      final file = File(credPath);
      if (await file.exists()) {
        await file.delete();
        _log(LogLevel.info, '凭证已清除');
      }
    } catch (e) {
      _log(LogLevel.warn, '清除凭证失败: $e');
    }
  }

  /// 日志输出
  void _log(LogLevel level, String message) {
    if (level.value >= _options.logLevel.value) {
      final prefix = '[WeChatBot][${level.name.toUpperCase()}]';
      developer.log('$prefix $message');
    }
  }
}

/// 二维码过期异常
class QRCodeExpiredException implements Exception {
  @override
  String toString() => 'QRCodeExpiredException: 二维码已过期';
}
