import 'package:flutter/material.dart';
import 'package:window_manager/window_manager.dart';

/// 自定义标题栏：左侧可拖拽，右侧窗口控制按钮（最小化/最大化或还原/关闭）
class WindowsControl extends StatefulWidget {
  const WindowsControl({super.key});

  @override
  State<WindowsControl> createState() => _WindowsControlState();
}

class _WindowsControlState extends State<WindowsControl> {
  bool _isMax = false;

  Future<void> _refreshMaxState() async {
    _isMax = await windowManager.isMaximized();
    if (mounted) setState(() {});
  }

  @override
  void initState() {
    super.initState();
    _refreshMaxState();
  }

  @override
  Widget build(BuildContext context) {
    final buttonStyle = IconButton.styleFrom(
      hoverColor: Colors.white.withValues(alpha: 0.08),
      highlightColor: Colors.white.withValues(alpha: 0.12),
      iconSize: 18,
    );

    return Material(
      color: const Color(0xFF2B2B2B),
      child: Row(
        children: [
          // 左侧：拖拽区 + 标题
          Expanded(
            child: GestureDetector(
              behavior: HitTestBehavior.translucent,
              onPanStart: (_) => windowManager.startDragging(),
              onDoubleTap: () async {
                if (await windowManager.isMaximized()) {
                  await windowManager.unmaximize();
                } else {
                  await windowManager.maximize();
                }
                await _refreshMaxState();
              },
              child: Padding(
                padding: const EdgeInsets.symmetric(horizontal: 12),
                child: Row(
                  children: const [
                    Icon(Icons.drag_indicator, color: Colors.white70, size: 16),
                    SizedBox(width: 6),
                    Text(
                      'Flutter',
                      style: TextStyle(color: Colors.white70, fontSize: 13),
                    ),
                  ],
                ),
              ),
            ),
          ),

          // 右侧：控制按钮
          Row(
            children: [
              IconButton(
                onPressed: windowManager.minimize,
                style: buttonStyle,
                icon: const Icon(Icons.remove, color: Colors.white),
                tooltip: '最小化',
              ),
              IconButton(
                onPressed: () async {
                  if (await windowManager.isMaximized()) {
                    await windowManager.unmaximize();
                  } else {
                    await windowManager.maximize();
                  }
                  await _refreshMaxState();
                },
                style: buttonStyle,
                icon: Icon(
                  _isMax ? Icons.filter_none : Icons.crop_square,
                  color: Colors.white,
                ),
                tooltip: _isMax ? '还原' : '最大化',
              ),
              IconButton(
                onPressed: windowManager.close,
                style: IconButton.styleFrom(
                  backgroundColor: Colors.red.withValues(alpha: 0.18),
                ),
                icon: const Icon(Icons.close, color: Colors.redAccent),
                tooltip: '关闭',
              ),
            ],
          ),
        ],
      ),
    );
  }
}
