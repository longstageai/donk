import 'package:donk/app/conf/colors.dart';
import 'package:donk/app/conf/config.dart';
import 'package:donk/app/router/routes.dart';
import 'package:donk/common/service/notification_websocket_service.dart';
import 'package:flutter/material.dart';
import 'package:get/get.dart';
import 'package:go_router/go_router.dart';

import 'app_dialog.dart';
import '../../ui/setting/wechat_connect_dialog.dart';

class AppMenu extends StatefulWidget {
  const AppMenu({super.key});

  @override
  State<AppMenu> createState() => _AppMenuState();
}

class _AppMenuState extends State<AppMenu> with SingleTickerProviderStateMixin {
  int selectedIndex = 0;
  late AnimationController _rotationController;
  late Animation<double> _rotationAnimation;

  @override
  void initState() {
    super.initState();
    _rotationController = AnimationController(
      duration: const Duration(seconds: 300),
      vsync: this,
    );
    _rotationAnimation = Tween<double>(
      begin: 0,
      end: 2 * 3.141592653589793,
    ).animate(_rotationController);
    _rotationController.repeat();
  }

  @override
  void dispose() {
    _rotationController.dispose();
    super.dispose();
  }

  @override
  void didChangeDependencies() {
    super.didChangeDependencies();
    // 根据当前路由更新选中状态
    // 在 didChangeDependencies 中访问 GoRouterState 是安全的
    _updateSelectedIndexFromRoute();
  }

  /// 根据当前路由路径更新选中索引
  void _updateSelectedIndexFromRoute() {
    final location = GoRouterState.of(context).uri.toString();
    final newIndex = _getIndexFromRoute(location);
    // 只有在索引变化时才更新状态，避免不必要的重建
    if (newIndex != selectedIndex) {
      setState(() {
        selectedIndex = newIndex;
      });
    }
  }

  /// 根据路由路径获取对应的菜单索引
  int _getIndexFromRoute(String location) {
    if (location.startsWith(Routes.home)) {
      return 0;
    } else if (location.startsWith(Routes.idea)) {
      return 1;
    } else if (location.startsWith(Routes.task)) {
      return 2;
    }
    return 0;
  }

  Widget _buildAvatar() {
    return MouseRegion(
      cursor: SystemMouseCursors.click,
      child: GestureDetector(
        onTap: () {},
        child: Stack(
          alignment: Alignment.center,
          children: [
            // 旋转的边框动画
            AnimatedBuilder(
              animation: _rotationAnimation,
              builder: (context, child) {
                return Transform.rotate(
                  angle: _rotationAnimation.value,
                  child: Container(
                    width: 50,
                    height: 50,
                    decoration: BoxDecoration(
                      shape: BoxShape.circle,
                      gradient: const SweepGradient(
                        colors: [
                          Colors.blue,
                          Colors.purple,
                          Colors.pink,
                          Colors.orange,
                          Colors.yellow,
                          Colors.green,
                          Colors.blue,
                        ],
                        stops: [0.0, 0.15, 0.3, 0.45, 0.6, 0.75, 1.0],
                      ),
                    ),
                  ),
                );
              },
            ),
            // 头像主体
            Container(
              width: 48,
              height: 48,
              decoration: BoxDecoration(
                shape: BoxShape.circle,
                color: const Color(0xFFE0E0E0),
                border: Border.all(color: Colors.white, width: 2),
              ),
              child: ClipOval(
                child: Container(
                  width: 48,
                  height: 48,
                  decoration: BoxDecoration(
                    borderRadius: BorderRadius.circular(24),
                  ),
                  child: Center(
                    child: Image.asset(logo, width: 35),
                  ),
                ),
              ),
            ),
          ],
        ),
      ),
    );
  }

  onSelect(index) {
    setState(() {
      selectedIndex = index;
    });

    switch (index) {
      case 0:
        context.go(Routes.home);
        break;
      case 1:
        context.go(Routes.idea);
        break;
      case 2:
        context.go(Routes.task);
        break;
    }
  }

  Widget top(BuildContext context) {
    return Column(
      mainAxisAlignment: MainAxisAlignment.spaceAround,
      children: [
        _buildAvatar(),
        _NavItem(
          index: 0,
          icon: Icons.mark_unread_chat_alt_sharp,
          label: '对话',
          selectedIndex: selectedIndex,
          onSelect: onSelect,
        ),
        _NavItem(
          index: 1,
          icon: Icons.auto_awesome,
          label: '灵感',
          selectedIndex: selectedIndex,
          onSelect: onSelect,
        ),
        _NavItem(
          index: 2,
          icon: Icons.task_sharp,
          label: '任务',
          selectedIndex: selectedIndex,
          onSelect: onSelect,
        ),
      ],
    );
  }

  Widget bottom(BuildContext context) {
    final buttonStyle = IconButton.styleFrom(
      hoverColor: Color(0xFFE0E0E0),
      highlightColor: Color(0xFFE0E0E0),
      iconSize: 20,
      shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(10)),
    );
    return Column(
      mainAxisAlignment: MainAxisAlignment.spaceAround,
      children: [
        // 消息通知按钮
        _buildNotificationButton(buttonStyle),
        IconButton(
          onPressed: () {
            WeChatConnectDialog.show();
          },
          style: buttonStyle,
          icon: Icon(Icons.phone_android, color: AppColors.c1, size: 20),
        ),
        IconButton(
          onPressed: () {
            AppDialog.showSettingsDialog();
          },
          style: buttonStyle,
          icon: Icon(Icons.settings, color: AppColors.c1, size: 20),
        ),
        SizedBox(height: 10),
      ],
    );
  }

  /// 构建消息通知按钮（带未读红点提示）
  Widget _buildNotificationButton(ButtonStyle buttonStyle) {
    return Stack(
      children: [
        IconButton(
          onPressed: () {
            context.go(Routes.notification);
          },
          style: buttonStyle,
          icon: Icon(
            Icons.notifications_outlined,
            color: AppColors.c1,
            size: 20,
          ),
        ),
        // 未读红点提示
        Positioned(
          right: 6,
          top: 6,
          child: Obx(() {
            final notificationService =
                Get.find<NotificationWebSocketService>();
            final hasUnread = notificationService.unreadCount.value > 0;
            if (!hasUnread) return const SizedBox.shrink();
            return Container(
              width: 8,
              height: 8,
              decoration: const BoxDecoration(
                color: Colors.red,
                shape: BoxShape.circle,
              ),
            );
          }),
        ),
      ],
    );
  }

  @override
  Widget build(BuildContext context) {
    return Column(
      children: [
        SizedBox(height: 280, width: double.infinity, child: top(context)),
        Expanded(child: Container()),
        SizedBox(height: 200, width: double.infinity, child: bottom(context)),
      ],
    );
  }
}

class _NavItem extends StatefulWidget {
  final int index;
  final IconData icon;
  final String label;
  final int selectedIndex;
  final Function(int) onSelect;

  const _NavItem({
    required this.index,
    required this.icon,
    required this.label,
    required this.selectedIndex,
    required this.onSelect,
  });

  @override
  State<_NavItem> createState() => _NavItemState();
}

class _NavItemState extends State<_NavItem> {
  bool _isHovered = false;

  @override
  Widget build(BuildContext context) {
    final isSelected = widget.selectedIndex == widget.index;

    return MouseRegion(
      cursor: SystemMouseCursors.click,
      onEnter: (_) => setState(() => _isHovered = true),
      onExit: (_) => setState(() => _isHovered = false),
      child: InkWell(
        onTap: () => widget.onSelect(widget.index),
        borderRadius: BorderRadius.circular(10),
        hoverColor: const Color(0xFFE0E0E0),
        highlightColor: const Color(0xFFE0E0E0),
        splashColor: Colors.transparent,
        child: Container(
          width: 45,
          height: 45,
          decoration: BoxDecoration(
            color: _isHovered ? const Color(0xFFE0E0E0) : Colors.transparent,
            borderRadius: BorderRadius.circular(10),
          ),
          child: Column(
            mainAxisAlignment: MainAxisAlignment.center,
            children: [
              Icon(
                widget.icon,
                size: 20,
                color:
                    isSelected
                        ? Colors.redAccent
                        : _isHovered
                        ? const Color(0xFF333333)
                        : AppColors.c1,
              ),
              if (widget.label.isNotEmpty) ...[
                const SizedBox(height: 1),
                Text(
                  widget.label,
                  style: TextStyle(
                    fontSize: 12,
                    color: isSelected ? Colors.redAccent : AppColors.c1,
                    fontWeight:
                        isSelected ? FontWeight.bold : FontWeight.normal,
                  ),
                ),
              ],
            ],
          ),
        ),
      ),
    );
  }
}
