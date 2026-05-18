import 'dart:convert';
import 'dart:io';

import 'package:path/path.dart' as path;

class OnboardingStateService {
  static const String _completedKey = 'onboarding_completed';
  static const String _dataDir = 'data/ui';
  static const String _fileName = 'onboarding_state.json';

  static Future<bool> isCompleted() async {
    final data = await _readState();
    return data[_completedKey] == true;
  }

  static Future<void> setCompleted(bool completed) async {
    final data = await _readState();
    data[_completedKey] = completed;
    await _writeState(data);
  }

  /// 获取状态文件路径
  /// 使用程序所在目录下的 data/ui 文件夹
  static Future<File> _stateFile() async {
    final executableDir = File(Platform.resolvedExecutable).parent.path;
    final dataDir = Directory(path.join(executableDir, _dataDir));
    if (!await dataDir.exists()) {
      await dataDir.create(recursive: true);
    }
    return File(path.join(dataDir.path, _fileName));
  }

  static Future<Map<String, dynamic>> _readState() async {
    final file = await _stateFile();
    if (!await file.exists()) {
      return <String, dynamic>{};
    }
    final content = await file.readAsString();
    if (content.trim().isEmpty) {
      return <String, dynamic>{};
    }
    final decoded = jsonDecode(content);
    return decoded is Map<String, dynamic> ? decoded : <String, dynamic>{};
  }

  static Future<void> _writeState(Map<String, dynamic> data) async {
    final file = await _stateFile();
    await file.parent.create(recursive: true);
    await file.writeAsString(jsonEncode(data));
  }
}
