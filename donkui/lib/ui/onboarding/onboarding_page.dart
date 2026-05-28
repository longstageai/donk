import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import 'package:window_manager/window_manager.dart';
import '../../app/conf/colors.dart';
import '../../app/router/routes.dart';
import '../../common/service/onboarding_state_service.dart';
import '../../l10n/generated/app_localizations.dart';
import 'onboarding_step_llm.dart';
import 'onboarding_step_embedding.dart';
import 'onboarding_step_wechat.dart';

/// 引导页
/// 引导用户完成 LLM、Embedding 和微信登录配置
class OnboardingPage extends StatefulWidget {
  const OnboardingPage({super.key});

  @override
  State<OnboardingPage> createState() => _OnboardingPageState();
}

class _OnboardingPageState extends State<OnboardingPage> {
  /// 当前步骤索引
  int _currentStep = 0;
  bool _isMaximized = false;

  @override
  void initState() {
    super.initState();
    _refreshMaximizedState();
  }

  Future<void> _refreshMaximizedState() async {
    _isMaximized = await windowManager.isMaximized();
    if (mounted) {
      setState(() {});
    }
  }

  /// 进入下一步
  void _nextStep() {
    if (_currentStep < 2) {
      setState(() {
        _currentStep++;
      });
    } else {
      _finishOnboarding();
    }
  }

  Future<void> _finishOnboarding() async {
    await OnboardingStateService.setCompleted(true);
    if (mounted) {
      context.go(Routes.home);
    }
  }

  /// 跳过当前步骤（仅微信登录可跳过）
  void _skipStep() {
    if (_currentStep == 2) {
      _finishOnboarding();
    }
  }

  @override
  Widget build(BuildContext context) {
    return ClipRRect(
      borderRadius: BorderRadius.circular(8),
      child: Scaffold(
        backgroundColor: Colors.white,
        body: SafeArea(
          child: Column(
            children: [
              _buildWindowBar(),

              // 顶部进度指示器
              _buildProgressIndicator(),

              // 步骤内容
              Expanded(child: _buildStepContent()),

              // 底部按钮
              _buildBottomButtons(),
            ],
          ),
        ),
      ),
    );
  }

  Widget _buildWindowBar() {
    final l10n = AppLocalizations.of(context)!;

    return GestureDetector(
      behavior: HitTestBehavior.translucent,
      onPanStart: (_) => windowManager.startDragging(),
      child: Container(
        height: 42,
        padding: const EdgeInsets.only(left: 18),
        decoration: BoxDecoration(
          color: Colors.white,
          border: Border(bottom: BorderSide(color: Colors.grey.shade100)),
        ),
        child: Row(
          children: [
            Text(
              l10n.onboardingWindowTitle,
              style: TextStyle(
                fontSize: 13,
                fontWeight: FontWeight.w700,
                color: Colors.grey.shade700,
              ),
            ),
            const Spacer(),
            _buildWindowButton(
              icon: Icons.remove,
              tooltip: l10n.minimize,
              onPressed: windowManager.minimize,
            ),
            _buildWindowButton(
              icon: _isMaximized ? Icons.filter_none : Icons.crop_square,
              tooltip: _isMaximized ? l10n.restore : l10n.maximize,
              onPressed: () async {
                if (await windowManager.isMaximized()) {
                  await windowManager.unmaximize();
                } else {
                  await windowManager.maximize();
                }
                await _refreshMaximizedState();
              },
            ),
            _buildWindowButton(
              icon: Icons.close,
              tooltip: l10n.close,
              isClose: true,
              onPressed: windowManager.close,
            ),
          ],
        ),
      ),
    );
  }

  Widget _buildWindowButton({
    required IconData icon,
    required String tooltip,
    required VoidCallback onPressed,
    bool isClose = false,
  }) {
    return SizedBox(
      width: 44,
      height: 42,
      child: IconButton(
        onPressed: onPressed,
        tooltip: tooltip,
        style: IconButton.styleFrom(
          hoverColor: isClose ? Colors.redAccent : Colors.black12,
          highlightColor: isClose ? Colors.red.withAlpha(30) : Colors.black12,
          shape: const RoundedRectangleBorder(borderRadius: BorderRadius.zero),
        ),
        icon: Icon(icon, size: 16, color: Colors.black87),
      ),
    );
  }

  /// 构建进度指示器
  Widget _buildProgressIndicator() {
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 24, vertical: 16),
      child: Column(
        children: [
          Row(
            children: List.generate(5, (index) {
              if (index.isOdd) {
                // 连接线
                final stepIndex = index ~/ 2;
                return Expanded(
                  child: Container(
                    height: 2,
                    color:
                        stepIndex < _currentStep
                            ? AppColors.primary
                            : AppColors.divider,
                  ),
                );
              } else {
                // 步骤圆点
                final stepIndex = index ~/ 2;
                return Container(
                  width: 32,
                  height: 32,
                  decoration: BoxDecoration(
                    shape: BoxShape.circle,
                    color:
                        stepIndex <= _currentStep
                            ? AppColors.primary
                            : AppColors.divider,
                  ),
                  child: Center(
                    child: Text(
                      '${stepIndex + 1}',
                      style: TextStyle(
                        color:
                            stepIndex <= _currentStep
                                ? Colors.white
                                : AppColors.textSecondary,
                        fontSize: 14,
                        fontWeight: FontWeight.w600,
                      ),
                    ),
                  ),
                );
              }
            }),
          ),
        ],
      ),
    );
  }

  /// 构建步骤内容
  Widget _buildStepContent() {
    switch (_currentStep) {
      case 0:
        return OnboardingStepLLM(onCompleted: _nextStep);
      case 1:
        return OnboardingStepEmbedding(onCompleted: _nextStep);
      case 2:
        return OnboardingStepWeChat(onCompleted: _nextStep, onSkip: _skipStep);
      default:
        return const SizedBox.shrink();
    }
  }

  /// 构建底部按钮
  Widget _buildBottomButtons() {
    final l10n = AppLocalizations.of(context)!;

    return Container(
      padding: const EdgeInsets.all(24),
      child: Row(
        mainAxisAlignment: MainAxisAlignment.spaceBetween,
        children: [
          // 上一步按钮（第一步不显示）
          _currentStep > 0
              ? TextButton(
                onPressed: () {
                  setState(() {
                    _currentStep--;
                  });
                },
                child: Text(
                  l10n.previousStep,
                  style: TextStyle(
                    color: AppColors.textSecondary,
                    fontSize: 14,
                  ),
                ),
              )
              : const SizedBox(width: 80),
        ],
      ),
    );
  }
}
