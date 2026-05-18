import 'dart:async';
import 'dart:io';

import 'package:path/path.dart' as path;

/// 进程管理服务
/// 用于管理外部服务器进程的生命周期
class ProcessManagerService {
  static Process? _serverProcess;
  static final StreamController<String> _logController =
      StreamController<String>.broadcast();

  /// 日志流，用于监听服务器输出
  static Stream<String> get logStream => _logController.stream;

  /// 获取服务器可执行文件路径
  /// 服务器程序位于 Flutter 主程序所在目录下的 server/donk.exe
  static String _getServerPath() {
    // 获取当前可执行文件的路径
    final executablePath = Platform.resolvedExecutable;
    // 获取可执行文件所在的目录
    final executableDir = path.dirname(executablePath);
    // 服务器程序路径
    final serverPath = path.join(executableDir, 'server', 'donk.exe');
    return serverPath;
  }

  /// 启动服务器进程
  static Future<bool> startServer() async {
    try {
      // 如果进程已经在运行，先停止
      if (_serverProcess != null) {
        await stopServer();
      }

      final serverPath = _getServerPath();
      final serverFile = File(serverPath);

      // 检查服务器文件是否存在
      if (!await serverFile.exists()) {
        _log('服务器程序不存在: $serverPath');
        return false;
      }

      _log('正在启动服务器: $serverPath');

      // 启动进程
      _serverProcess = await Process.start(
        serverPath,
        [],
        workingDirectory: File(serverPath).parent.path,
        mode: ProcessStartMode.normal,
      );

      // 监听标准输出
      _serverProcess!.stdout.transform(const SystemEncoding().decoder).listen((
        data,
      ) {
        _log('[SERVER OUT] $data');
      });

      // 监听错误输出
      _serverProcess!.stderr.transform(const SystemEncoding().decoder).listen((
        data,
      ) {
        _log('[SERVER ERR] $data');
      });

      // 监听进程退出
      _serverProcess!.exitCode.then((code) {
        _log('服务器进程已退出，退出码: $code');
        _serverProcess = null;
      });

      _log('服务器进程已启动，PID: ${_serverProcess!.pid}');
      return true;
    } catch (e, stackTrace) {
      _log('启动服务器失败: $e');
      _log('堆栈: $stackTrace');
      return false;
    }
  }

  /// 停止服务器进程
  static Future<bool> stopServer() async {
    try {
      if (_serverProcess == null) {
        return true;
      }

      final pid = _serverProcess!.pid;

      // Windows 上使用 taskkill 来终止进程树
      if (Platform.isWindows) {
        try {
          // 先尝试优雅终止（不带 /F 参数）
          final result = await Process.run('taskkill', [
            '/T',
            '/PID',
            '$pid',
          ], runInShell: true);

          if (result.exitCode != 0) {
            // 优雅终止失败，强制终止
            await Process.run('taskkill', [
              '/F',
              '/T',
              '/PID',
              '$pid',
            ], runInShell: true);
          }
        } catch (e) {
          // taskkill 失败，尝试直接 kill
          _serverProcess!.kill(ProcessSignal.sigterm);
        }
      } else {
        // Linux/Mac 上使用 kill
        _serverProcess!.kill(ProcessSignal.sigterm);
      }

      // 等待进程退出，最多等待 3 秒
      try {
        await _serverProcess!.exitCode.timeout(
          const Duration(seconds: 3),
          onTimeout: () {
            // 超时后强制杀死
            _serverProcess?.kill(ProcessSignal.sigkill);
            return -1;
          },
        );
      } catch (_) {
        // 忽略等待过程中的错误
      }

      _serverProcess = null;
      return true;
    } catch (e) {
      // 强制清理
      _serverProcess = null;
      return false;
    }
  }

  /// 检查服务器是否正在运行
  static bool get isRunning => _serverProcess != null;

  /// 获取服务器进程 ID
  static int? get pid => _serverProcess?.pid;

  /// 发送日志
  static void _log(String message) {
    final timestamp = DateTime.now().toIso8601String();
    final logMessage = '[$timestamp] $message';
    // ignore: avoid_print
    print(logMessage);
    _logController.add(logMessage);
  }

  /// 释放资源
  static Future<void> dispose() async {
    await stopServer();
    await _logController.close();
  }
}
