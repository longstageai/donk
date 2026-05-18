import 'dart:io';
import 'package:window_manager/window_manager.dart';

/// 单实例服务
/// 确保程序只能运行一个实例
class SingleInstanceService {
  static const String _socketAddress = '127.0.0.1';
  static const int _socketPort = 65432; // 选择一个不常用的端口

  static ServerSocket? _serverSocket;
  static bool _isFirstInstance = false;

  /// 检查是否是第一个实例
  /// 如果是第一个实例，返回 true 并启动监听
  /// 如果不是第一个实例，返回 false
  static Future<bool> checkAndStart() async {
    try {
      // 尝试绑定端口，如果失败则说明已有实例在运行
      _serverSocket = await ServerSocket.bind(_socketAddress, _socketPort);
      _isFirstInstance = true;

      // 监听来自其他实例的连接
      _serverSocket!.listen((socket) {
        // 收到其他实例的连接请求，显示窗口
        _showWindow();
        socket.destroy();
      });

      return true;
    } on SocketException catch (_) {
      // 端口已被占用，说明已有实例在运行
      _isFirstInstance = false;
      // 尝试连接已有实例，通知它显示窗口
      await _notifyExistingInstance();
      return false;
    }
  }

  /// 通知已有实例显示窗口
  static Future<void> _notifyExistingInstance() async {
    try {
      final socket = await Socket.connect(_socketAddress, _socketPort);
      await socket.close();
    } catch (_) {
      // 连接失败，忽略错误
    }
  }

  /// 显示窗口
  static Future<void> _showWindow() async {
    try {
      await windowManager.setSkipTaskbar(false);
      await windowManager.show();
      await windowManager.focus();
    } catch (_) {
      // 窗口管理器可能尚未初始化，忽略错误
    }
  }

  /// 释放资源
  static Future<void> dispose() async {
    await _serverSocket?.close();
    _serverSocket = null;
  }

  /// 是否是第一个实例
  static bool get isFirstInstance => _isFirstInstance;
}
