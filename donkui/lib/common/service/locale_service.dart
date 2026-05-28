import 'package:shared_preferences/shared_preferences.dart';

/// 语言设置服务类
/// 用于持久化存储和读取用户的语言偏好设置
class LocaleService {
  static const String _localeKey = 'app_locale';
  static const String _defaultLocale = 'en';
  // static const String _defaultLocale = 'zh';
  /// 获取保存的语言代码
  static Future<String> getSavedLocale() async {
    try {
      final prefs = await SharedPreferences.getInstance();
      return prefs.getString(_localeKey) ?? _defaultLocale;
    } catch (e) {
      return _defaultLocale;
    }
  }

  /// 保存语言代码
  static Future<bool> saveLocale(String languageCode) async {
    try {
      final prefs = await SharedPreferences.getInstance();
      return await prefs.setString(_localeKey, languageCode);
    } catch (e) {
      return false;
    }
  }

  /// 获取支持的语言列表
  static List<Map<String, String>> getSupportedLocales() {
    return [
      {'code': 'zh', 'name': '中文'},
      {'code': 'en', 'name': 'English'},
    ];
  }
}
