import 'package:donk/ui/setting/setting_view.dart';
import 'package:flutter/material.dart';
import 'package:flutter_smart_dialog/flutter_smart_dialog.dart';

/// 应用对话框工具类
/// 封装了基于 flutter_smart_dialog 的各种对话框显示方法
class AppDialog {
  /// 显示设置对话框
  /// 包含左侧菜单和右侧内容区域的设置页面
  static void showSettingsDialog() {
    SmartDialog.show(
      /// 点击遮罩层关闭对话框
      clickMaskDismiss: false,

      /// 遮罩层颜色（设置为透明）
      maskColor: Colors.transparent,

      /// 对话框动画
      animationType: SmartAnimationType.fade,

      /// 遮罩层圆角
      maskWidget: Container(
        decoration: BoxDecoration(
          color: Colors.black.withAlpha(100),
          borderRadius: BorderRadius.circular(10),
        ),
      ),

      /// 构建对话框内容
      builder: (context) => const SettingView(),
    );
  }

  /// 显示自定义对话框
  /// [child] 对话框内容组件
  static void show({required Widget child}) {
    SmartDialog.show(
      /// 点击遮罩层关闭对话框
      clickMaskDismiss: false,

      /// 遮罩层颜色（设置为透明）
      maskColor: Colors.transparent,

      /// 对话框动画
      animationType: SmartAnimationType.fade,

      /// 遮罩层圆角
      maskWidget: Container(
        decoration: BoxDecoration(
          color: Colors.black.withAlpha(100),
          borderRadius: BorderRadius.circular(10),
        ),
      ),

      /// 构建对话框内容
      builder: (context) => child,
    );
  }

  /// 关闭当前显示的对话框
  static void dismiss() {
    SmartDialog.dismiss();
  }
}

/// 设置对话框组件
/// 包含左侧菜单导航和右侧设置内容区域
class SettingsDialog extends StatefulWidget {
  const SettingsDialog({super.key});

  @override
  State<SettingsDialog> createState() => _SettingsDialogState();
}

class _SettingsDialogState extends State<SettingsDialog> {
  /// 当前选中的菜单索引
  int _selectedIndex = 0;

  /// 菜单项列表
  final List<Map<String, dynamic>> _menuItems = [
    {'icon': Icons.settings_outlined, 'label': '通用设置'},
    {'icon': Icons.pie_chart_outline, 'label': '用量统计'},
    {'icon': Icons.extension_outlined, 'label': '技能管理'},
    {'icon': Icons.link_outlined, 'label': '连接应用'},
    {'icon': Icons.computer_outlined, 'label': '远控通道'},
    {'icon': Icons.info_outline, 'label': '关于我们'},
  ];

  /// 开关状态
  bool _securityProtection = true;
  bool _sleepPrevention = true;
  bool _toolPermission = true;

  @override
  Widget build(BuildContext context) {
    return Container(
      /// 对话框尺寸
      width: 700,
      height: 500,

      /// 圆角装饰
      decoration: BoxDecoration(
        color: Colors.white,
        borderRadius: BorderRadius.circular(12),
      ),
      child: Row(
        children: [
          /// 左侧菜单区域
          _buildLeftMenu(),

          /// 右侧内容区域
          Expanded(child: _buildRightContent()),
        ],
      ),
    );
  }

  /// 构建左侧菜单区域
  Widget _buildLeftMenu() {
    return Container(
      /// 菜单宽度
      width: 160,

      /// 背景色
      color: const Color(0xFFF5F5F5),
      padding: const EdgeInsets.symmetric(vertical: 20, horizontal: 12),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          /// 菜单标题
          const Padding(
            padding: EdgeInsets.only(left: 12, bottom: 20),
            child: Text(
              '设置',
              style: TextStyle(
                fontSize: 18,
                fontWeight: FontWeight.bold,
                color: Colors.black87,
              ),
            ),
          ),

          /// 菜单项列表
          ...List.generate(_menuItems.length, (index) {
            final item = _menuItems[index];
            final isSelected = _selectedIndex == index;
            return _buildMenuItem(
              icon: item['icon'] as IconData,
              label: item['label'] as String,
              isSelected: isSelected,
              onTap: () => setState(() => _selectedIndex = index),
            );
          }),
        ],
      ),
    );
  }

  /// 构建单个菜单项
  Widget _buildMenuItem({
    required IconData icon,
    required String label,
    required bool isSelected,
    required VoidCallback onTap,
  }) {
    return GestureDetector(
      onTap: onTap,
      child: Container(
        margin: const EdgeInsets.only(bottom: 4),
        padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 10),
        decoration: BoxDecoration(
          /// 选中时显示白色背景
          color: isSelected ? Colors.white : Colors.transparent,
          borderRadius: BorderRadius.circular(8),
        ),
        child: Row(
          children: [
            Icon(
              icon,
              size: 18,
              color: isSelected ? Colors.black87 : Colors.grey,
            ),
            const SizedBox(width: 10),
            Text(
              label,
              style: TextStyle(
                fontSize: 14,
                color: isSelected ? Colors.black87 : Colors.grey,
              ),
            ),
          ],
        ),
      ),
    );
  }

  /// 构建右侧内容区域
  Widget _buildRightContent() {
    return Container(
      padding: const EdgeInsets.all(24),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          /// 顶部标题栏
          Row(
            mainAxisAlignment: MainAxisAlignment.spaceBetween,
            children: [
              const Text(
                '通用设置',
                style: TextStyle(
                  fontSize: 18,
                  fontWeight: FontWeight.bold,
                  color: Colors.black87,
                ),
              ),

              /// 关闭按钮
              GestureDetector(
                onTap: () => AppDialog.dismiss(),
                child: const Icon(Icons.close, size: 20, color: Colors.grey),
              ),
            ],
          ),
          const SizedBox(height: 24),

          /// 头像设置项
          _buildAvatarItem(),
          const Divider(height: 1, color: Color(0xFFEEEEEE)),

          /// 用户名设置项
          _buildUsernameItem(),
          const SizedBox(height: 16),

          /// 安全防护开关
          _buildSwitchItem(
            title: '安全防护',
            subtitle: '开启后可实时保护AI安全，防范漏洞攻击，拦截恶意指令、技能投毒等风险行为',
            value: _securityProtection,
            onChanged: (value) => setState(() => _securityProtection = value),
          ),

          /// 休眠阻止开关
          _buildSwitchItem(
            title: '休眠阻止',
            subtitle: '开启后，电脑将不会进入休眠模式，QClaw 会保持活跃状态',
            value: _sleepPrevention,
            onChanged: (value) => setState(() => _sleepPrevention = value),
          ),

          /// 工具权限限制开关
          _buildSwitchItem(
            title: '工具权限限制',
            subtitle: '开启后，智能体调用工具时会限制按照低权限执行，防止误删关键文件',
            value: _toolPermission,
            onChanged: (value) => setState(() => _toolPermission = value),
          ),
          const Spacer(),

          /// 退出登录按钮
          _buildLogoutButton(),
        ],
      ),
    );
  }

  /// 构建头像设置项
  Widget _buildAvatarItem() {
    return Container(
      padding: const EdgeInsets.symmetric(vertical: 16),
      child: Row(
        mainAxisAlignment: MainAxisAlignment.spaceBetween,
        children: [
          const Text(
            '头像',
            style: TextStyle(fontSize: 14, color: Colors.black87),
          ),

          /// 头像图片
          Container(
            width: 48,
            height: 48,
            decoration: BoxDecoration(
              color: const Color(0xFFFFE4E1),
              borderRadius: BorderRadius.circular(24),
            ),
            child: const Icon(Icons.person, size: 28, color: Colors.white),
          ),
        ],
      ),
    );
  }

  /// 构建用户名设置项
  Widget _buildUsernameItem() {
    return Container(
      padding: const EdgeInsets.symmetric(vertical: 16),
      child: Row(
        mainAxisAlignment: MainAxisAlignment.spaceBetween,
        children: [
          const Text(
            '用户名',
            style: TextStyle(fontSize: 14, color: Colors.black87),
          ),

          /// 用户名和微信图标
          Row(
            children: [
              /// 微信图标
              Container(
                width: 20,
                height: 20,
                decoration: const BoxDecoration(
                  color: Color(0xFF07C160),
                  shape: BoxShape.circle,
                ),
                child: const Icon(Icons.wechat, size: 12, color: Colors.white),
              ),
              const SizedBox(width: 8),
              const Text(
                'x',
                style: TextStyle(fontSize: 14, color: Colors.black87),
              ),
            ],
          ),
        ],
      ),
    );
  }

  /// 构建开关设置项
  Widget _buildSwitchItem({
    required String title,
    required String subtitle,
    required bool value,
    required ValueChanged<bool> onChanged,
  }) {
    return Container(
      margin: const EdgeInsets.only(bottom: 12),
      padding: const EdgeInsets.all(16),
      decoration: BoxDecoration(
        color: const Color(0xFFF8F8F8),
        borderRadius: BorderRadius.circular(8),
      ),
      child: Row(
        children: [
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                /// 标题
                Text(
                  title,
                  style: const TextStyle(
                    fontSize: 14,
                    fontWeight: FontWeight.w500,
                    color: Colors.black87,
                  ),
                ),
                const SizedBox(height: 4),

                /// 副标题
                Text(
                  subtitle,
                  style: const TextStyle(fontSize: 12, color: Colors.grey),
                ),
              ],
            ),
          ),

          /// 开关
          Switch(
            value: value,
            onChanged: onChanged,

            /// 激活颜色
            activeColor: Colors.white,
            activeTrackColor: const Color(0xFF07C160),

            /// 非激活颜色
            inactiveThumbColor: Colors.white,
            inactiveTrackColor: Colors.grey.shade300,
          ),
        ],
      ),
    );
  }

  /// 构建退出登录按钮
  Widget _buildLogoutButton() {
    return GestureDetector(
      onTap: () {
        /// 处理退出登录逻辑
        AppDialog.dismiss();
      },
      child: Container(
        width: double.infinity,
        padding: const EdgeInsets.symmetric(vertical: 12),
        decoration: BoxDecoration(
          color: const Color(0xFFF8F8F8),
          borderRadius: BorderRadius.circular(8),
        ),
        child: const Center(
          child: Text(
            '退出登录',
            style: TextStyle(
              fontSize: 14,
              color: Colors.red,
              fontWeight: FontWeight.w500,
            ),
          ),
        ),
      ),
    );
  }
}
