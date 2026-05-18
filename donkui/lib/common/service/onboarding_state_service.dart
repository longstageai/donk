import 'dart:convert';
import 'dart:io';

import 'package:path_provider/path_provider.dart';

class OnboardingStateService {
  static const String _completedKey = 'onboarding_completed';

  static Future<bool> isCompleted() async {
    final data = await _readState();
    return data[_completedKey] == true;
  }

  static Future<void> setCompleted(bool completed) async {
    final data = await _readState();
    data[_completedKey] = completed;
    await _writeState(data);
  }

  static Future<File> _stateFile() async {
    final dir = await getApplicationSupportDirectory();
    return File('${dir.path}${Platform.pathSeparator}onboarding_state.json');
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
