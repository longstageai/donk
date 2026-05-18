import 'package:flutter/material.dart';
import 'package:window_manager/window_manager.dart';

class WindowUtil {
  // 初始化窗口：无边框、透明背景、可居中、可最大化
  static Future<void> init({
    required Size size,
    required Size minimumSize,
    Size? maximumSize,
    String title = '示例',
  }) async {
    WidgetsFlutterBinding.ensureInitialized();
    await windowManager.ensureInitialized();

    final windowOptions = WindowOptions(
      size: size,
      minimumSize: minimumSize,
      maximumSize: maximumSize,
      center: true,
      backgroundColor: Colors.transparent,
      skipTaskbar: true,
      titleBarStyle: TitleBarStyle.hidden, // 隐藏系统标题栏
    );

    await windowManager.waitUntilReadyToShow(windowOptions, () async {
      await windowManager.show();
      await windowManager.focus();
      await windowManager.setAsFrameless(); // 设置为无边框
      await windowManager.setTitle(title);
      await windowManager.setResizable(true);
      await windowManager.setMaximizable(true);
    });
  }

  static void setResizable(bool resizable) {
    windowManager.setResizable(resizable);
  }

  static Future<Size> getSize() => windowManager.getSize();
  static void setSize(Size size) => windowManager.setSize(size);

  static Future<Offset> getPosition() => windowManager.getPosition();
  static void setPosition(Offset offset) => windowManager.setPosition(offset);

  static Future<bool> isMaximized() => windowManager.isMaximized();

  static Future<void> close() => windowManager.close();
  static Future<void> setMaximize() => windowManager.maximize();
  static Future<void> setMinimize() => windowManager.minimize();
  static Future<void> setUnMaximize() => windowManager.unmaximize();

  /// 指定边调整窗口大小
  static void startResizing(ResizeEdge resizeEdge) {
    windowManager.startResizing(resizeEdge);
  }

  /// 数字映射八方向缩放
  static void scaleWindow(int s) {
    switch (s) {
      case 1:
        startResizing(ResizeEdge.left);
        break;
      case 2:
        startResizing(ResizeEdge.right);
        break;
      case 3:
        startResizing(ResizeEdge.top);
        break;
      case 4:
        startResizing(ResizeEdge.bottom);
        break;
      case 5:
        startResizing(ResizeEdge.topLeft);
        break;
      case 6:
        startResizing(ResizeEdge.topRight);
        break;
      case 7:
        startResizing(ResizeEdge.bottomLeft);
        break;
      case 8:
        startResizing(ResizeEdge.bottomRight);
        break;
      default:
        break;
    }
  }
}
