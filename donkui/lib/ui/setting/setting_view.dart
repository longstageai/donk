import 'package:donk/ui/setting/about_page.dart';
import 'package:donk/ui/setting/setting_page.dart';
import 'package:donk/ui/setting/token_page.dart';
import 'package:donk/ui/setting/llm_config_page.dart';
import 'package:donk/ui/setting/agent_config_page.dart';
import 'package:donk/ui/setting/embedding_config_page.dart';
import 'package:flutter/material.dart';

/// 设置视图组件
/// 包含左侧菜单导航和右侧设置页面内容
class SettingView extends StatefulWidget {
  /// 初始选中的菜单索引
  final int initialIndex;

  const SettingView({super.key, this.initialIndex = 0});

  @override
  State<SettingView> createState() => _SettingsDialogState();
}

/// 设置对话框状态类
class _SettingsDialogState extends State<SettingView> {
  /// 当前选中的菜单索引
  late int _selectedIndex;

  /// 菜单项数据模型
  final List<_MenuItemData> _menuItems = [
    _MenuItemData(index: 0, icon: Icons.settings_outlined, label: '通用设置'),
    _MenuItemData(index: 1, icon: Icons.pie_chart_outline, label: '用量统计'),
    _MenuItemData(index: 2, icon: Icons.smart_toy_outlined, label: 'LLM'),
    _MenuItemData(index: 3, icon: Icons.memory_outlined, label: 'Agent'),
    _MenuItemData(
      index: 4,
      icon: Icons.text_fields_outlined,
      label: 'Embedding',
    ),
    _MenuItemData(index: 5, icon: Icons.info_outline, label: '关于我们'),
  ];

  final List<Widget> _pageItems = [
    SettingPage(),
    TokenPage(),
    LLMConfigPage(),
    AgentConfigPage(),
    EmbeddingConfigPage(),
    AboutPage(),
  ];

  @override
  void initState() {
    super.initState();
    _selectedIndex = widget.initialIndex;
  }

  @override
  Widget build(BuildContext context) {
    return Container(
      /// 对话框尺寸
      width: 700,
      height: 500,

      /// 圆角装饰
      clipBehavior: Clip.hardEdge,
      decoration: BoxDecoration(
        color: Colors.white,
        borderRadius: BorderRadius.circular(12),
      ),
      child: Row(
        children: [
          /// 左侧菜单区域
          _buildLeftMenu(),

          /// 右侧内容区域
          Expanded(child: _pageItems[_selectedIndex]),
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
            final isSelected = _selectedIndex == item.index;
            return _buildMenuItem(
              icon: item.icon,
              label: item.label,
              isSelected: isSelected,
              onTap: () => setState(() => _selectedIndex = item.index),
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
    return MouseRegion(
      cursor: SystemMouseCursors.click,
      child: GestureDetector(
        onTap: onTap,
        child: Container(
          margin: const EdgeInsets.only(bottom: 4),
          padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 10),
          decoration: BoxDecoration(
            color: isSelected ? Colors.white : Colors.transparent,
            borderRadius: BorderRadius.circular(6),
          ),
          child: Row(
            children: [
              Icon(
                icon,
                size: 18,
                color: isSelected ? const Color(0xFFFF6B6B) : Colors.grey,
              ),
              const SizedBox(width: 10),
              Text(
                label,
                style: TextStyle(
                  fontSize: 13,
                  color: isSelected ? Colors.black87 : Colors.grey,
                  fontWeight: isSelected ? FontWeight.w500 : FontWeight.normal,
                ),
              ),
            ],
          ),
        ),
      ),
    );
  }
}

/// 菜单项数据模型
class _MenuItemData {
  final int index;
  final IconData icon;
  final String label;

  _MenuItemData({required this.index, required this.icon, required this.label});
}
