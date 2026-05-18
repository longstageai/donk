import 'package:donk/app/layout/layout.dart';
import 'package:donk/common/service/onboarding_state_service.dart';
import 'package:donk/ui/home/home_view.dart';
import 'package:donk/ui/idea/idea_view.dart';
import 'package:donk/ui/notification/notification_view.dart';
import 'package:donk/ui/onboarding/onboarding_page.dart';
import 'package:donk/ui/task/task_view.dart';
import 'package:donk/ui/test/test_view.dart';
import 'package:flutter/cupertino.dart';
import 'package:flutter/material.dart';
import 'package:flutter_smart_dialog/flutter_smart_dialog.dart';
import 'package:go_router/go_router.dart';

/// 路由配置类
/// 定义应用的所有路由路径和页面跳转配置
class Routes {
  static const String test = "/test";
  static const String splash = "/splash";
  static const String login = "/login";
  static const String home = "/home";
  static const String idea = "/idea";
  static const String task = "/task";
  static const String recharge = "/recharge";
  static const String record = "/record";
  static const String pay = "/pay";
  static const String about = "/about";
  static const String off = "/off";
  static const String feedback = "/feedback";
  static const String notification = "/notification";
  static const String onboarding = "/onboarding";

  static final GlobalKey<NavigatorState> navigatorKey =
      GlobalKey<NavigatorState>();
  static final GlobalKey<NavigatorState> _shellKey =
      GlobalKey<NavigatorState>();

  /// 创建带淡入淡出动画的页面
  /// [child] 页面组件
  /// [state] 路由状态
  static CustomTransitionPage<void> _buildPageWithFade({
    required Widget child,
    required GoRouterState state,
  }) {
    return CustomTransitionPage<void>(
      key: state.pageKey,
      child: child,

      /// 动画持续时间
      transitionDuration: const Duration(milliseconds: 300),

      /// 反向动画持续时间
      reverseTransitionDuration: const Duration(milliseconds: 300),

      /// 动画构建器
      transitionsBuilder: (context, animation, secondaryAnimation, child) {
        return FadeTransition(
          opacity: CurveTween(curve: Curves.easeInOut).animate(animation),
          child: child,
        );
      },
    );
  }

  static String initialLocation = home;

  static Future<void> initInitialLocation() async {
    final isOnboardingCompleted = await OnboardingStateService.isCompleted();
    initialLocation = isOnboardingCompleted ? home : onboarding;
  }

  /// GoRouter 实例配置
  static final GoRouter router = GoRouter(
    navigatorKey: navigatorKey,
    initialLocation: initialLocation,
    observers: [FlutterSmartDialog.observer],
    routes: [
      /// 引导页（首次启动配置）
      GoRoute(
        path: onboarding,
        pageBuilder: (BuildContext context, GoRouterState state) {
          return _buildPageWithFade(
            state: state,
            child: const OnboardingPage(),
          );
        },
      ),

      /// 测试页面（带淡入淡出动画）
      GoRoute(
        path: test,
        pageBuilder: (BuildContext context, GoRouterState state) {
          return _buildPageWithFade(state: state, child: TestView());
        },
      ),

      /// ShellRoute 包裹的页面（带淡入淡出动画）
      ShellRoute(
        navigatorKey: _shellKey,
        builder: (context, state, child) {
          return Layout(child: child);
        },
        routes: [
          /// 首页
          GoRoute(
            path: home,
            pageBuilder: (BuildContext context, GoRouterState state) {
              return _buildPageWithFade(state: state, child: HomeView());
            },
          ),

          /// 灵感广场页面
          GoRoute(
            path: idea,
            pageBuilder: (BuildContext context, GoRouterState state) {
              return _buildPageWithFade(state: state, child: IdeaView());
            },
          ),

          /// 任务页面
          GoRoute(
            path: task,
            pageBuilder: (BuildContext context, GoRouterState state) {
              return _buildPageWithFade(state: state, child: TaskView());
            },
          ),

          /// 消息通知页面
          GoRoute(
            path: notification,
            pageBuilder: (BuildContext context, GoRouterState state) {
              return _buildPageWithFade(
                state: state,
                child: NotificationView(),
              );
            },
          ),
        ],
      ),
    ],
  );
}
