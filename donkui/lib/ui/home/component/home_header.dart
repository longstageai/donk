import 'package:donk/app/layout/app_dialog.dart';
import 'package:donk/common/service/setting_service.dart';
import 'package:donk/common/widget/app_button.dart';
import 'package:donk/ui/setting/setting_view.dart';
import 'package:flutter/material.dart';
import 'package:get/get.dart';

import '../home_controller.dart';

class HomeHeader extends StatefulWidget {
  final VoidCallback? onTap;

  const HomeHeader({super.key, this.onTap});

  @override
  State<HomeHeader> createState() => _HomeHeaderState();
}

class _HomeHeaderState extends State<HomeHeader> {
  final controller = Get.find<HomeController>();

  /// Token 使用量
  int _usedTokens = 0;

  /// 剩余百分比
  int _remainingPercent = 100;

  /// 是否加载中
  bool _isLoading = true;

  @override
  void initState() {
    super.initState();
    _loadTokenBudget();
  }

  /// 加载 Token 预算状态
  Future<void> _loadTokenBudget() async {
    try {
      final data = await SettingService.getTokenBudget();
      final used = data['used'] as int? ?? 0;
      final limit = data['limit'] as int? ?? -1;
      final usagePercent = (data['usage_percent'] as num?)?.toDouble() ?? 0;

      setState(() {
        _usedTokens = used;
        if (limit > 0) {
          _remainingPercent = (100 - usagePercent).round().clamp(0, 100);
        } else {
          _remainingPercent = -1; // -1 表示无限制，显示 100%
        }
        _isLoading = false;
      });
    } catch (e) {
      setState(() => _isLoading = false);
    }
  }

  /// 格式化数字显示
  String _formatNumber(int num) {
    if (num >= 100000000) {
      return '${(num / 100000000).toStringAsFixed(1)}亿';
    } else if (num >= 10000) {
      return '${(num / 10000).toStringAsFixed(1)}万';
    } else {
      return num.toString();
    }
  }

  @override
  Widget build(BuildContext context) {
    final displayText =
        _isLoading
            ? '加载中...'
            : '已使用${_formatNumber(_usedTokens)}，剩余${_remainingPercent == -1 ? 100 : _remainingPercent}%';

    return SizedBox(
      height: 50,
      child: Row(
        mainAxisAlignment: MainAxisAlignment.spaceBetween,
        children: [
          const SizedBox(),
          // 中间导航按钮组
          SizedBox(
            width: 300,
            child: Row(
              mainAxisAlignment: MainAxisAlignment.spaceEvenly,
              children: [
                // 已使用资源按钮
                Expanded(
                  child: AppButton(
                    icon: Icons.ac_unit_rounded,
                    label: displayText,
                    onTap:
                        _isLoading
                            ? null
                            : () {
                              // 跳转到设置页并选中用量统计（索引1）
                              AppDialog.show(
                                child: const SettingView(initialIndex: 1),
                              );
                            },
                  ),
                ),
                const SizedBox(width: 10),

                // 刷新按钮
                IconButton(
                  onPressed: _isLoading ? null : _loadTokenBudget,
                  icon:
                      _isLoading
                          ? const SizedBox(
                            width: 16,
                            height: 16,
                            child: CircularProgressIndicator(
                              strokeWidth: 2,
                              color: Colors.grey,
                            ),
                          )
                          : const Icon(Icons.refresh, size: 20),
                  tooltip: '刷新',
                ),

                // 清空消息按钮
                IconButton(
                  onPressed: () => _showClearConfirmDialog(context),
                  icon: const Icon(Icons.delete, size: 20),
                  tooltip: '清空消息',
                ),
              ],
            ),
          ),
        ],
      ),
    );
  }

  /// 显示清除消息确认对话框
  void _showClearConfirmDialog(BuildContext context) {
    controller.clearAllMessages();
  }
}
