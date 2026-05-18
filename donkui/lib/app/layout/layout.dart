import 'dart:io';

import 'package:donk/app/conf/colors.dart';
import 'package:donk/app/conf/config.dart';
import 'package:donk/app/layout/layout_controller.dart';
import 'package:donk/common/service/process_manager_service.dart';
import 'package:donk/common/util/img_util.dart';
import 'package:flutter/material.dart';
import 'package:get/get.dart';
import 'package:tray_manager/tray_manager.dart';
import 'package:window_manager/window_manager.dart';

import '../../common/service/single_instance_service.dart';
import '../../common/window/window_zoom.dart';
import 'donk_app_bar.dart';
import 'menu.dart';

class Layout extends StatefulWidget {
  Layout({super.key, required this.child});

  final LayoutController controller = Get.put(
    LayoutController(),
    permanent: true,
  );
  final Widget child;

  @override
  State<Layout> createState() => _LayoutState();
}

class _LayoutState extends State<Layout> with TrayListener, WindowListener {
  @override
  void initState() {
    super.initState();
    initTray();
    windowManager.addListener(this);
    trayManager.addListener(this);
  }

  @override
  void dispose() {
    trayManager.removeListener(this);
    windowManager.removeListener(this);
    super.dispose();
  }

  // 拦截关闭按钮，改为隐藏到托盘
  @override
  void onWindowClose() async {
    await windowManager.hide();
    await windowManager.setSkipTaskbar(true);
  }

  Future<void> initTray() async {
    // 准备图标路径（推荐绝对路径；若用 assets，写到临时目录）
    String iconPath = await ImgUtil.prepareTrayIconFromAssets(logo2);
    await trayManager.setIcon(iconPath);
    await trayManager.setToolTip(name);
    Menu menu = Menu(
      items: [
        MenuItem(key: 'show', label: '显示窗口'),
        MenuItem.separator(),
        MenuItem(key: 'exit', label: '退出'),
      ],
    );
    await trayManager.setContextMenu(menu);
  }

  // 托盘图标左键：显示并聚焦
  @override
  void onTrayIconMouseDown() async {
    await windowManager.setSkipTaskbar(false);
    await windowManager.show();
    await windowManager.focus();
  }

  // 右键弹出菜单（可选）
  @override
  void onTrayIconRightMouseDown() async {
    await trayManager.popUpContextMenu();
  }

  // 菜单点击分发
  @override
  void onTrayMenuItemClick(MenuItem menuItem) async {
    switch (menuItem.key) {
      case 'show':
        await windowManager.setSkipTaskbar(false);
        await windowManager.show();
        await windowManager.focus();
        break;
      case 'exit':
        await _exitApp();
        break;
    }
  }

  /// 退出应用
  /// 确保正确清理所有资源后再退出
  Future<void> _exitApp() async {
    // 移除监听器
    trayManager.removeListener(this);
    windowManager.removeListener(this);

    try {
      // 释放单实例服务
      await SingleInstanceService.dispose();
    } catch (_) {
      // 忽略错误
    }

    try {
      // 停止服务器进程（设置超时防止卡住）
      await ProcessManagerService.stopServer().timeout(
        const Duration(seconds: 5),
        onTimeout: () => false,
      );
    } catch (_) {
      // 忽略停止服务器的错误
    }

    try {
      // 销毁窗口
      await windowManager.destroy();
    } catch (_) {
      // 忽略销毁窗口的错误
    }

    // 强制退出
    exit(0);
  }

  @override
  Widget build(BuildContext context) {
    return Container(
      clipBehavior: Clip.hardEdge, // 边角圆角裁剪（需要较新 Flutter）
      decoration: BoxDecoration(
        borderRadius: BorderRadius.circular(circular),
        border: Border.all(color: Colors.white10, width: 1),
        color: AppColors.backgroundColor,
      ),
      child: Stack(
        children: [
          Scaffold(
            backgroundColor: Colors.transparent,
            appBar: DonkAppBar(windowManager: windowManager),
            body: Row(
              children: [
                Container(
                  color: AppColors.backgroundColor,
                  width: 60,
                  child: AppMenu(),
                ),
                Expanded(child: widget.child),
              ],
            ),
          ),
          const WindowZoom(thickness: 6),
        ],
      ),
    );
  }
}
