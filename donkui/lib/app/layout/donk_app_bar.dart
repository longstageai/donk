import 'package:donk/app/conf/config.dart';
import 'package:flutter/material.dart';
import 'package:get/get.dart';
import 'package:window_manager/window_manager.dart';

import '../conf/colors.dart';
import 'layout_controller.dart';

class DonkAppBar extends StatefulWidget implements PreferredSizeWidget {
  final double height;
  final WindowManager windowManager;

  const DonkAppBar({super.key, this.height = 40, required this.windowManager});

  @override
  State<DonkAppBar> createState() => _DonkAppBarState();

  @override
  Size get preferredSize => Size.fromHeight(height);
}

class _DonkAppBarState extends State<DonkAppBar> {
  bool _isMax = false;
  final LayoutController layoutController = Get.find<LayoutController>();

  @override
  void initState() {
    super.initState();
    _refreshMaxState();
  }

  Future<void> _refreshMaxState() async {
    _isMax = await windowManager.isMaximized();
    if (mounted) setState(() {});
  }

  Widget menu() {
    return Container(
      padding: EdgeInsets.only(left: 10),
      child: Row(
        children: [
          // Image.asset(logo, width: 20),
          Text(
            name2,
            style: TextStyle(
              fontSize: 14,
              fontWeight: FontWeight.bold,
              color: Colors.black87,
            ),
          ),
        ],
      ),
    );
  }

  Widget right() {
    return Row(
      crossAxisAlignment: CrossAxisAlignment.center,
      mainAxisAlignment: MainAxisAlignment.end,
      children: [control()],
    );
  }

  Widget control() {
    final buttonStyle = IconButton.styleFrom(
      hoverColor: Colors.black26,
      highlightColor: Colors.black26,
      iconSize: 16,
      shape: const RoundedRectangleBorder(borderRadius: BorderRadius.zero),
    );
    final buttonStyle2 = IconButton.styleFrom(
      hoverColor: Colors.redAccent,
      highlightColor: Colors.white10,
      iconSize: 16,
      shape: const RoundedRectangleBorder(borderRadius: BorderRadius.zero),
    );
    return Row(
      crossAxisAlignment: CrossAxisAlignment.center,
      mainAxisAlignment: MainAxisAlignment.end,
      children: [
        IconButton(
          onPressed: windowManager.minimize,
          style: buttonStyle,
          icon: const Icon(Icons.remove, color: Colors.black87, size: 16),
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
            color: Colors.black87,
            size: 16,
          ),
          tooltip: _isMax ? '还原' : '最大化',
        ),
        IconButton(
          onPressed: windowManager.close,
          style: buttonStyle2,
          icon: const Icon(Icons.close, color: Colors.black87, size: 16),
          tooltip: '关闭',
        ),
      ],
    );
  }

  @override
  Widget build(BuildContext context) {
    return SafeArea(
      bottom: false,
      child: GestureDetector(
        behavior: HitTestBehavior.translucent,
        onPanStart: (details) {
          windowManager.startDragging();
        },
        child: RepaintBoundary(
          child: Container(
            decoration: BoxDecoration(
              borderRadius: BorderRadius.only(
                topLeft: Radius.circular(circular),
                topRight: Radius.circular(circular),
              ),
              color: AppColors.backgroundColor,
            ),
            child: Row(children: [menu(), Expanded(child: right())]),
          ),
        ),
      ),
    );
  }
}
