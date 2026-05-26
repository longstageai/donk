import 'package:flutter/material.dart';
import 'package:flutter_localizations/flutter_localizations.dart';
import 'package:flutter_smart_dialog/flutter_smart_dialog.dart';
import 'package:get/get.dart';
import 'package:window_manager/window_manager.dart';

import '../../common/service/chat_storage_service.dart';
import '../../common/service/notification_websocket_service.dart';
import '../../common/service/single_instance_service.dart';
import '../../common/service/wechat_bot_service.dart';
import '../../l10n/generated/app_localizations.dart';
import '../../ui/home/home_controller.dart';
import '../conf/config.dart';
import '../controller/locale_controller.dart';
import '../router/routes.dart';

class App extends StatefulWidget {
  const App({super.key});

  static Future<void> wm() async {
    await windowManager.ensureInitialized();
    final windowOptions = WindowOptions(
      size: const Size(960, 720),
      minimumSize: const Size(960, 720),
      maximumSize: const Size(960, 720),
      center: true,
      backgroundColor: Colors.transparent,
      skipTaskbar: false,
      titleBarStyle: TitleBarStyle.hidden,
    );

    await windowManager.waitUntilReadyToShow(windowOptions, () async {
      await windowManager.show();
      await windowManager.focus();
      await windowManager.setAsFrameless();
      await windowManager.setTitle(name);
      await windowManager.setResizable(true);
      await windowManager.setMaximizable(true);
    });
    await windowManager.setPreventClose(true);
  }

  static Future<bool> init() async {
    WidgetsFlutterBinding.ensureInitialized();

    // 检查单实例
    final isFirstInstance = await SingleInstanceService.checkAndStart();
    if (!isFirstInstance) {
      // 已有实例在运行，退出当前程序
      return false;
    }

    await Routes.initInitialLocation();
    await wm();
    // 启动外部服务器程序
    // await ProcessManagerService.startServer();
    // 初始化依赖注入
    _initDependencies();
    // 检查微信登录状态，如有凭证则自动连接
    _checkWeChatLoginStatus();
    // 初始化通知WebSocket服务
    await _initNotificationService();
    return true;
  }

  /// 初始化通知WebSocket服务
  static Future<void> _initNotificationService() async {
    try {
      final notificationService = Get.find<NotificationWebSocketService>();
      await notificationService.init();
    } catch (e) {
      // 静默处理，不影响启动
    }
  }

  /// 检查微信登录状态，如果有有效凭证则自动连接
  static Future<void> _checkWeChatLoginStatus() async {
    try {
      final wechatService = Get.find<WeChatBotService>();
      final hasCredentials = await wechatService.hasValidCredentials();
      if (hasCredentials) {
        await wechatService.connect(allowInteractiveLogin: false);
      }
    } catch (e) {
      // 静默处理，不影响启动
    }
  }

  /// 初始化依赖注入
  static void _initDependencies() {
    // 注册服务
    Get.lazyPut(() => WeChatBotService(), fenix: true);
    Get.lazyPut(() => ChatStorageService(), fenix: true);
    // 注册通知WebSocket服务（全局存在，程序启动时连接）
    Get.put(NotificationWebSocketService(), permanent: true);

    // 注册 HomeController（全局存在，使用 permanent: true）
    Get.put(
      HomeController(wechatService: Get.find(), storageService: Get.find()),
      permanent: true,
    );

    // 注册 LocaleController（全局存在，使用 permanent: true）
    Get.put(LocaleController(), permanent: true);
  }

  @override
  State<App> createState() => _AppState();
}

class _AppState extends State<App> with WindowListener {
  @override
  void initState() {
    super.initState();
    windowManager.addListener(this);
  }

  @override
  void dispose() {
    windowManager.removeListener(this);
    super.dispose();
  }

  @override
  void onWindowClose() async {
    // 注意：窗口关闭时只是隐藏到托盘，不停止服务器
    // 真正的退出在 Layout 的托盘菜单 "exit" 中处理
    await windowManager.hide();
    await windowManager.setSkipTaskbar(true);
  }

  @override
  Widget build(BuildContext context) {
    return GetBuilder<LocaleController>(
      builder: (controller) {
        return MaterialApp.router(
          title: name,
          theme: ThemeData(
            colorScheme: ColorScheme.fromSeed(seedColor: Colors.blue),
            useMaterial3: true,
            fontFamily: 'SourceHanSansSC',
          ),
          debugShowCheckedModeBanner: false,
          routerConfig: Routes.router,
          builder: FlutterSmartDialog.init(),
          locale: controller.locale,
          supportedLocales: const [
            Locale('zh'),
            Locale('en'),
          ],
          localizationsDelegates: const [
            AppLocalizations.delegate,
            GlobalMaterialLocalizations.delegate,
            GlobalWidgetsLocalizations.delegate,
            GlobalCupertinoLocalizations.delegate,
          ],
        );
      },
    );
  }
}
