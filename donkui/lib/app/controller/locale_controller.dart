import 'package:flutter/material.dart';
import 'package:get/get.dart';
import '../../common/service/locale_service.dart';

/// 语言控制器
/// 使用 GetX 管理应用语言状态
class LocaleController extends GetxController {
  static LocaleController get to => Get.find();

  /// 当前语言代码
  final RxString _languageCode = 'zh'.obs;
  String get languageCode => _languageCode.value;

  /// 当前 Locale
  Locale get locale => Locale(_languageCode.value);

  /// 支持的语言列表
  final List<Map<String, String>> supportedLocales = [
    {'code': 'zh', 'name': '中文'},
    {'code': 'en', 'name': 'English'},
  ];

  @override
  void onInit() {
    super.onInit();
    _loadSavedLocale();
  }

  /// 加载保存的语言设置
  Future<void> _loadSavedLocale() async {
    final savedLocale = await LocaleService.getSavedLocale();
    _languageCode.value = savedLocale;
  }

  /// 切换语言
  Future<void> changeLocale(String languageCode) async {
    if (_languageCode.value == languageCode) return;

    // 保存到本地
    final success = await LocaleService.saveLocale(languageCode);
    if (success) {
      _languageCode.value = languageCode;
      // 更新 GetX 的 locale
      Get.updateLocale(Locale(languageCode));
      update();
    }
  }

  /// 获取当前语言的显示名称
  String getCurrentLanguageName() {
    final locale = supportedLocales.firstWhere(
      (l) => l['code'] == _languageCode.value,
      orElse: () => {'code': 'zh', 'name': '中文'},
    );
    return locale['name']!;
  }

  /// 获取语言名称
  String getLanguageName(String code) {
    final locale = supportedLocales.firstWhere(
      (l) => l['code'] == code,
      orElse: () => {'code': code, 'name': code},
    );
    return locale['name']!;
  }
}
