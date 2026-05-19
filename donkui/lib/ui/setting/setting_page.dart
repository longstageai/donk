import 'package:donk/app/layout/app_dialog.dart';
import 'package:flutter/material.dart';
import '../../common/service/setting_service.dart';

/// 通用设置页面
/// 只保留应用级开关设置
class SettingPage extends StatefulWidget {
  const SettingPage({super.key});

  @override
  State<SettingPage> createState() => _SettingPageState();
}

class _SettingPageState extends State<SettingPage> {
  /// 开关状态
  bool _securityProtection = true;
  bool _sleepPrevention = true;
  bool _toolPermission = true;
  bool _knowledgeAutoBuild = true;

  /// 加载状态
  bool _isLoadingSleep = false;
  bool _isLoadingKnowledge = false;

  @override
  void initState() {
    super.initState();
    _loadSleepStatus();
    _loadKnowledgeConfig();
  }

  /// 加载睡眠管理状态
  Future<void> _loadSleepStatus() async {
    try {
      final data = await SettingService.getSleepStatus();
      if (!mounted) return;
      setState(() {
        _sleepPrevention = data['is_active'] ?? true;
      });
    } catch (e) {
      // 出错时默认开启阻止睡眠
      if (!mounted) return;
      setState(() => _sleepPrevention = true);
    }
  }

  /// 加载知识库配置
  Future<void> _loadKnowledgeConfig() async {
    try {
      final data = await SettingService.getKnowledgeConfig();
      if (!mounted) return;
      setState(() {
        _knowledgeAutoBuild = data['enabled'] ?? true;
      });
    } catch (e) {
      // 出错时默认开启知识库自动构建
      if (!mounted) return;
      setState(() => _knowledgeAutoBuild = true);
    }
  }

  /// 切换知识库自动构建状态
  Future<void> _toggleKnowledgeAutoBuild(bool value) async {
    setState(() => _isLoadingKnowledge = true);
    try {
      await SettingService.updateKnowledgeConfig(enabled: value);
      if (!mounted) return;
      setState(() => _knowledgeAutoBuild = value);
      _showToast(value ? '已开启知识库自动构建' : '已关闭知识库自动构建');
    } catch (e) {
      _showToast('操作失败: $e');
    } finally {
      if (mounted) {
        setState(() => _isLoadingKnowledge = false);
      }
    }
  }

  /// 切换睡眠阻止状态
  Future<void> _toggleSleepPrevention(bool value) async {
    setState(() => _isLoadingSleep = true);
    try {
      if (value) {
        await SettingService.preventSleep();
      } else {
        await SettingService.allowSleep();
      }
      if (!mounted) return;
      setState(() => _sleepPrevention = value);
      _showToast(value ? '已阻止系统睡眠' : '已恢复系统睡眠');
    } catch (e) {
      _showToast('操作失败: $e');
    } finally {
      if (mounted) {
        setState(() => _isLoadingSleep = false);
      }
    }
  }

  /// 显示提示
  void _showToast(String message) {
    final overlay = Overlay.of(context);
    final overlayEntry = OverlayEntry(
      builder:
          (context) => Positioned(
            bottom: 50,
            left: 0,
            right: 0,
            child: Center(
              child: Material(
                color: Colors.transparent,
                child: Container(
                  padding: const EdgeInsets.symmetric(
                    horizontal: 24,
                    vertical: 12,
                  ),
                  decoration: BoxDecoration(
                    color: Colors.black87,
                    borderRadius: BorderRadius.circular(8),
                  ),
                  child: Text(
                    message,
                    style: const TextStyle(color: Colors.white, fontSize: 14),
                  ),
                ),
              ),
            ),
          ),
    );

    overlay.insert(overlayEntry);
    Future.delayed(const Duration(seconds: 2), () {
      overlayEntry.remove();
    });
  }

  @override
  Widget build(BuildContext context) {
    return _buildRightContent();
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
              MouseRegion(
                cursor: SystemMouseCursors.click,
                child: GestureDetector(
                  onTap: () => AppDialog.dismiss(),
                  child: const Icon(Icons.close, size: 20, color: Colors.grey),
                ),
              ),
            ],
          ),
          const SizedBox(height: 24),

          /// 开关设置列表
          Expanded(
            child: SingleChildScrollView(
              child: Column(
                children: [
                  _buildSwitchItem(
                    title: '安全防护',
                    subtitle: '开启后可实时保护AI安全，防范漏洞攻击，拦截恶意指令、技能投毒等风险行为',
                    value: _securityProtection,
                    onChanged:
                        (value) => setState(() => _securityProtection = value),
                  ),
                  _buildSwitchItem(
                    title: '知识库自动构建',
                    subtitle:
                        '自动扫描桌面、下载、文档文件夹，支持 txt、md、pdf、docx 格式，构建可搜索的向量知识库',
                    value: _knowledgeAutoBuild,
                    isLoading: _isLoadingKnowledge,
                    onChanged:
                        _isLoadingKnowledge ? null : _toggleKnowledgeAutoBuild,
                  ),
                  _buildSwitchItem(
                    title: '休眠阻止',
                    subtitle: '开启后，电脑将不会进入休眠模式，donk 会保持活跃状态',
                    value: _sleepPrevention,
                    isLoading: _isLoadingSleep,
                    onChanged: _isLoadingSleep ? null : _toggleSleepPrevention,
                  ),
                  _buildSwitchItem(
                    title: '工具权限限制',
                    subtitle: '开启后，智能体调用工具时会限制按照低权限执行，防止误删关键文件',
                    value: _toolPermission,
                    onChanged:
                        (value) => setState(() => _toolPermission = value),
                  ),
                ],
              ),
            ),
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
    bool isLoading = false,
    ValueChanged<bool>? onChanged,
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
                Text(
                  title,
                  style: const TextStyle(
                    fontSize: 14,
                    fontWeight: FontWeight.w500,
                    color: Colors.black87,
                  ),
                ),
                const SizedBox(height: 4),
                Text(
                  subtitle,
                  style: const TextStyle(fontSize: 12, color: Colors.grey),
                ),
              ],
            ),
          ),
          isLoading
              ? const SizedBox(
                width: 40,
                height: 24,
                child: Center(
                  child: SizedBox(
                    width: 16,
                    height: 16,
                    child: CircularProgressIndicator(
                      strokeWidth: 2,
                      color: Color(0xFF07C160),
                    ),
                  ),
                ),
              )
              : Transform.scale(
                scale: 0.8,
                child: Switch(
                  value: value,
                  onChanged: onChanged,
                  activeColor: Colors.white,
                  activeTrackColor: const Color(0xFF07C160),
                  inactiveThumbColor: Colors.white,
                  inactiveTrackColor: Colors.grey.shade300,
                ),
              ),
        ],
      ),
    );
  }
}
