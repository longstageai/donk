import 'package:donk/app/conf/config.dart';
import 'package:donk/app/layout/app_dialog.dart';
import 'package:flutter/material.dart';
import '../../l10n/generated/app_localizations.dart';

/// 关于我们页面
/// 展示应用版本信息、更新入口和官方网站链接
class AboutPage extends StatefulWidget {
  const AboutPage({super.key});

  @override
  State<AboutPage> createState() => _AboutPageState();
}

class _AboutPageState extends State<AboutPage> {
  /// 当前版本号
  final String _currentVersion = 'v1.0.0';

  /// 是否有新版本可更新
  final bool _hasUpdate = false;

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.all(24),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          /// 页面标题
          _buildHeader(),
          const SizedBox(height: 40),

          /// 应用Logo和名称
          _buildAppLogo(),
          const SizedBox(height: 40),

          /// 版本信息区域
          _buildVersionInfo(),
          const SizedBox(height: 16),

          /// 进入官网入口
          _buildOfficialWebsite(),
          const Spacer(),

          /// 底部协议链接
          _buildFooterLinks(),
        ],
      ),
    );
  }

  /// 构建页面标题
  Widget _buildHeader() {
    return Row(
      mainAxisAlignment: MainAxisAlignment.spaceBetween,
      children: [
        Text(
          AppLocalizations.of(context)!.about,
          style: const TextStyle(
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
    );
  }

  /// 构建应用Logo和名称
  Widget _buildAppLogo() {
    return Center(
      child: Column(
        children: [
          /// 应用图标
          Container(
            width: 80,
            height: 80,
            decoration: BoxDecoration(borderRadius: BorderRadius.circular(20)),
            child: Image.asset(logo, width: 35),
          ),
          const SizedBox(height: 16),

          /// 应用名称
          const Text(
            name,
            style: TextStyle(
              fontSize: 24,
              fontWeight: FontWeight.bold,
              color: Colors.black87,
            ),
          ),
        ],
      ),
    );
  }

  /// 构建版本信息区域
  Widget _buildVersionInfo() {
    final l10n = AppLocalizations.of(context)!;
    return Container(
      padding: const EdgeInsets.all(16),
      decoration: BoxDecoration(
        color: const Color(0xFFF8F8F8),
        borderRadius: BorderRadius.circular(8),
      ),
      child: Row(
        children: [
          /// 版本信息
          Row(
            children: [
              Text(
                l10n.currentVersion,
                style: const TextStyle(fontSize: 14, color: Colors.black87),
              ),
              const SizedBox(width: 8),
              Text(
                _currentVersion,
                style: const TextStyle(fontSize: 14, color: Colors.grey),
              ),
              if (_hasUpdate) ...[
                const SizedBox(width: 8),
                Container(
                  padding: const EdgeInsets.symmetric(
                    horizontal: 6,
                    vertical: 2,
                  ),
                  decoration: BoxDecoration(
                    color: const Color(0xFFE8F5E9),
                    borderRadius: BorderRadius.circular(4),
                  ),
                  child: Text(
                    l10n.updateAvailable,
                    style: const TextStyle(fontSize: 11, color: Color(0xFF07C160)),
                  ),
                ),
              ],
            ],
          ),
          const Spacer(),
          // /// 更新按钮
          // GestureDetector(
          //   onTap: () {
          //     /// 处理更新逻辑
          //   },
          //   child: Container(
          //     padding: const EdgeInsets.symmetric(horizontal: 24, vertical: 8),
          //     decoration: BoxDecoration(
          //       color: const Color(0xFF333333),
          //       borderRadius: BorderRadius.circular(16),
          //     ),
          //     child: const Text(
          //       '更新',
          //       style: TextStyle(
          //         fontSize: 13,
          //         color: Colors.white,
          //         fontWeight: FontWeight.w500,
          //       ),
          //     ),
          //   ),
          // ),
          // const SizedBox(width: 12),
          // /// 版本日志按钮
          // GestureDetector(
          //   onTap: () {
          //     /// 处理查看版本日志逻辑
          //   },
          //   child: Container(
          //     padding: const EdgeInsets.symmetric(horizontal: 20, vertical: 8),
          //     decoration: BoxDecoration(
          //       color: Colors.white,
          //       borderRadius: BorderRadius.circular(16),
          //       border: Border.all(color: const Color(0xFFE0E0E0)),
          //     ),
          //     child: const Text(
          //       '版本日志',
          //       style: TextStyle(
          //         fontSize: 13,
          //         color: Colors.black87,
          //       ),
          //     ),
          //   ),
          // ),
        ],
      ),
    );
  }

  /// 构建进入官网入口
  Widget _buildOfficialWebsite() {
    final l10n = AppLocalizations.of(context)!;
    return GestureDetector(
      onTap: () {
        /// 处理进入官网逻辑
      },
      child: Container(
        padding: const EdgeInsets.all(16),
        decoration: BoxDecoration(
          color: const Color(0xFFF8F8F8),
          borderRadius: BorderRadius.circular(8),
        ),
        child: Row(
          mainAxisAlignment: MainAxisAlignment.spaceBetween,
          children: [
            Text(l10n.officialWebsite, style: const TextStyle(fontSize: 14, color: Colors.black87)),
            const Icon(Icons.chevron_right, size: 20, color: Colors.grey),
          ],
        ),
      ),
    );
  }

  /// 构建底部协议链接
  Widget _buildFooterLinks() {
    final l10n = AppLocalizations.of(context)!;
    return Center(
      child: Row(
        mainAxisAlignment: MainAxisAlignment.center,
        children: [
          /// 服务协议
          GestureDetector(
            onTap: () {
              /// 处理查看服务协议逻辑
            },
            child: Text(
              l10n.serviceAgreement,
              style: const TextStyle(fontSize: 12, color: Colors.grey),
            ),
          ),

          /// 分隔符
          Container(
            margin: const EdgeInsets.symmetric(horizontal: 12),
            height: 12,
            width: 1,
            color: Colors.grey.shade300,
          ),

          /// 隐私保护协议
          GestureDetector(
            onTap: () {
              /// 处理查看隐私保护协议逻辑
            },
            child: Text(
              l10n.privacyPolicy,
              style: const TextStyle(fontSize: 12, color: Colors.grey),
            ),
          ),
        ],
      ),
    );
  }
}
