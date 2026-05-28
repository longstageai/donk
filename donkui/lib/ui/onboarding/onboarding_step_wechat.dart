import 'dart:async';

import 'package:flutter/material.dart';
import 'package:qr_flutter/qr_flutter.dart';
import '../../app/conf/colors.dart';
import '../../common/service/wechat_bot_service.dart';
import '../../l10n/generated/app_localizations.dart';

class OnboardingStepWeChat extends StatefulWidget {
  final VoidCallback onCompleted;
  final VoidCallback onSkip;

  const OnboardingStepWeChat({
    super.key,
    required this.onCompleted,
    required this.onSkip,
  });

  @override
  State<OnboardingStepWeChat> createState() => _OnboardingStepWeChatState();
}

class _OnboardingStepWeChatState extends State<OnboardingStepWeChat> {
  final WeChatBotService _service = WeChatBotService();
  WeChatConnectionStatus _status = WeChatConnectionStatus.disconnected;
  String? _qrCodeUrl;
  String? _errorMessage;
  StreamSubscription<WeChatConnectionStatus>? _statusSubscription;
  StreamSubscription<String?>? _qrCodeSubscription;

  @override
  void initState() {
    super.initState();
    _initAndConnect();
  }

  Future<void> _initAndConnect() async {
    _initStreams();
    await _service.initialize();
    await _checkConnection();
  }

  void _initStreams() {
    _statusSubscription = _service.connectionStatus.listen((status) {
      if (!mounted) return;
      final oldStatus = _status;

      setState(() {
        _status = status;
      });

      if (status == WeChatConnectionStatus.connected) {
        Future.delayed(const Duration(seconds: 1), () {
          if (mounted) {
            widget.onCompleted();
          }
        });
      }

      if (status == WeChatConnectionStatus.disconnected &&
          oldStatus != WeChatConnectionStatus.disconnected) {
        _handleDisconnectAndReconnect();
      }
    });

    _qrCodeSubscription = _service.qrCodeStream.listen((url) {
      if (!mounted) return;
      setState(() {
        _qrCodeUrl = url;
        if (url != null) {
          _errorMessage = null;
        }
      });
    });
  }

  Future<void> _handleDisconnectAndReconnect() async {
    await Future.delayed(const Duration(milliseconds: 500));
    if (mounted && _status == WeChatConnectionStatus.disconnected) {
      setState(() {
        _qrCodeUrl = null;
        _errorMessage = null;
      });
      await _connect();
    }
  }

  Future<void> _checkConnection() async {
    final hasCredentials = await _service.hasValidCredentials();
    if (!mounted) return;
    if (hasCredentials) {
      await _service.connect();
    } else {
      await _connect();
    }
  }

  Future<void> _connect() async {
    try {
      setState(() {
        _errorMessage = null;
      });
      await _service.connect();
    } catch (e) {
      if (!mounted) return;
      setState(() {
        _errorMessage = AppLocalizations.of(
          context,
        )!.connectionFailedWithError(e);
      });
    }
  }

  @override
  void dispose() {
    _statusSubscription?.cancel();
    _qrCodeSubscription?.cancel();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context)!;
    final statusInfo = _getStatusInfo(l10n);
    final isConnecting = _status == WeChatConnectionStatus.connecting;

    return SingleChildScrollView(
      padding: const EdgeInsets.fromLTRB(24, 8, 24, 24),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Container(
            width: double.infinity,
            padding: const EdgeInsets.all(20),
            decoration: BoxDecoration(
              gradient: LinearGradient(
                colors: [
                  AppColors.primary.withAlpha(28),
                  AppColors.primary.withAlpha(8),
                ],
                begin: Alignment.topLeft,
                end: Alignment.bottomRight,
              ),
              borderRadius: BorderRadius.circular(20),
              border: Border.all(color: AppColors.primary.withAlpha(30)),
            ),
            child: Row(
              children: [
                Container(
                  width: 48,
                  height: 48,
                  decoration: BoxDecoration(
                    color: AppColors.primary,
                    borderRadius: BorderRadius.circular(16),
                  ),
                  child: const Icon(
                    Icons.wechat,
                    color: Colors.white,
                    size: 28,
                  ),
                ),
                const SizedBox(width: 16),
                Expanded(
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Text(
                        l10n.connectWeChat,
                        style: const TextStyle(
                          fontSize: 24,
                          fontWeight: FontWeight.w700,
                          color: Colors.black87,
                        ),
                      ),
                      const SizedBox(height: 6),
                      Text(
                        l10n.connectWeChatDesc,
                        style: TextStyle(
                          fontSize: 14,
                          height: 1.4,
                          color: Colors.grey.shade700,
                        ),
                      ),
                    ],
                  ),
                ),
              ],
            ),
          ),
          const SizedBox(height: 20),

          if (_errorMessage != null)
            Container(
              width: double.infinity,
              padding: const EdgeInsets.all(14),
              margin: const EdgeInsets.only(bottom: 16),
              decoration: BoxDecoration(
                color: Colors.red.shade50,
                borderRadius: BorderRadius.circular(14),
                border: Border.all(color: Colors.red.shade100),
              ),
              child: Row(
                children: [
                  Icon(
                    Icons.error_outline,
                    color: Colors.red.shade400,
                    size: 20,
                  ),
                  const SizedBox(width: 10),
                  Expanded(
                    child: Text(
                      _errorMessage!,
                      style: TextStyle(
                        color: Colors.red.shade700,
                        fontSize: 14,
                        height: 1.4,
                      ),
                    ),
                  ),
                ],
              ),
            ),

          Container(
            width: double.infinity,
            padding: const EdgeInsets.all(20),
            decoration: BoxDecoration(
              color: Colors.white,
              borderRadius: BorderRadius.circular(20),
              border: Border.all(color: Colors.grey.shade200),
              boxShadow: [
                BoxShadow(
                  color: Colors.black.withAlpha(10),
                  blurRadius: 18,
                  offset: const Offset(0, 8),
                ),
              ],
            ),
            child: Column(
              children: [
                Row(
                  children: [
                    Icon(statusInfo.icon, color: statusInfo.color, size: 20),
                    const SizedBox(width: 8),
                    Expanded(
                      child: Text(
                        statusInfo.title,
                        style: const TextStyle(
                          fontSize: 17,
                          fontWeight: FontWeight.w700,
                          color: Colors.black87,
                        ),
                      ),
                    ),
                    Container(
                      padding: const EdgeInsets.symmetric(
                        horizontal: 10,
                        vertical: 6,
                      ),
                      decoration: BoxDecoration(
                        color: statusInfo.color.withAlpha(18),
                        borderRadius: BorderRadius.circular(999),
                        border: Border.all(
                          color: statusInfo.color.withAlpha(45),
                        ),
                      ),
                      child: Text(
                        statusInfo.badge,
                        style: TextStyle(
                          color: statusInfo.color,
                          fontSize: 12,
                          fontWeight: FontWeight.w700,
                        ),
                      ),
                    ),
                  ],
                ),
                const SizedBox(height: 20),
                _buildQRCodeOrStatus(l10n),
                const SizedBox(height: 18),
                Text(
                  statusInfo.description,
                  textAlign: TextAlign.center,
                  style: TextStyle(
                    fontSize: 14,
                    height: 1.45,
                    color: Colors.grey.shade700,
                  ),
                ),
              ],
            ),
          ),
          const SizedBox(height: 18),

          _buildInstructions(l10n),
          const SizedBox(height: 18),

          _buildActionButtons(l10n, isConnecting),
          const SizedBox(height: 24),
        ],
      ),
    );
  }

  Widget _buildActionButtons(AppLocalizations l10n, bool isConnecting) {
    return Container(
      width: double.infinity,
      padding: const EdgeInsets.all(14),
      decoration: BoxDecoration(
        color: Colors.grey.shade50,
        borderRadius: BorderRadius.circular(18),
        border: Border.all(color: Colors.grey.shade200),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          ElevatedButton(
            onPressed: widget.onSkip,
            style: ElevatedButton.styleFrom(
              backgroundColor: AppColors.primary,
              foregroundColor: Colors.white,
              shape: RoundedRectangleBorder(
                borderRadius: BorderRadius.circular(16),
              ),
              padding: const EdgeInsets.symmetric(vertical: 16),
              elevation: 0,
            ),
            child: Row(
              mainAxisAlignment: MainAxisAlignment.center,
              children: [
                Text(
                  l10n.enterHome,
                  style: const TextStyle(
                    fontSize: 16,
                    fontWeight: FontWeight.w700,
                  ),
                ),
                const SizedBox(width: 8),
                const Icon(Icons.arrow_forward_rounded, size: 20),
              ],
            ),
          ),
          const SizedBox(height: 10),
          TextButton(
            onPressed: isConnecting ? null : _connect,
            style: TextButton.styleFrom(
              foregroundColor: AppColors.primary,
              disabledForegroundColor: Colors.grey.shade400,
              shape: RoundedRectangleBorder(
                borderRadius: BorderRadius.circular(14),
              ),
              padding: const EdgeInsets.symmetric(vertical: 13),
            ),
            child:
                isConnecting
                    ? Row(
                      mainAxisAlignment: MainAxisAlignment.center,
                      children: [
                        SizedBox(
                          width: 18,
                          height: 18,
                          child: CircularProgressIndicator(
                            strokeWidth: 2,
                            color: Colors.grey.shade400,
                          ),
                        ),
                        const SizedBox(width: 8),
                        Text(
                          l10n.fetchingQrCode,
                          style: const TextStyle(
                            fontSize: 14,
                            fontWeight: FontWeight.w600,
                          ),
                        ),
                      ],
                    )
                    : Row(
                      mainAxisAlignment: MainAxisAlignment.center,
                      children: [
                        const Icon(Icons.refresh_rounded, size: 18),
                        const SizedBox(width: 6),
                        Text(
                          l10n.refreshQrCode,
                          style: const TextStyle(
                            fontSize: 14,
                            fontWeight: FontWeight.w600,
                          ),
                        ),
                      ],
                    ),
          ),
          const SizedBox(height: 4),
          Text(
            l10n.wechatOptionalHint,
            textAlign: TextAlign.center,
            style: TextStyle(
              fontSize: 12,
              height: 1.35,
              color: Colors.grey.shade600,
            ),
          ),
        ],
      ),
    );
  }

  _WeChatStatusInfo _getStatusInfo(AppLocalizations l10n) {
    switch (_status) {
      case WeChatConnectionStatus.connected:
        return _WeChatStatusInfo(
          title: l10n.wechatConnected,
          badge: l10n.connected,
          description: l10n.wechatLoginSuccessDesc,
          color: Colors.green.shade600,
          icon: Icons.check_circle_outline,
        );
      case WeChatConnectionStatus.connecting:
        return _WeChatStatusInfo(
          title: l10n.wechatConnecting,
          badge: l10n.connecting,
          description: l10n.wechatFetchingQrDesc,
          color: AppColors.primary,
          icon: Icons.sync,
        );
      case WeChatConnectionStatus.waitingForScan:
        return _WeChatStatusInfo(
          title: l10n.wechatWaitScan,
          badge: l10n.waitingForScan,
          description: l10n.wechatScanConfirmDesc,
          color: Colors.orange.shade700,
          icon: Icons.qr_code_scanner,
        );
      case WeChatConnectionStatus.scanning:
        return _WeChatStatusInfo(
          title: l10n.wechatScanSuccess,
          badge: l10n.confirming,
          description: l10n.wechatScannedConfirmDesc,
          color: Colors.blue.shade600,
          icon: Icons.phonelink,
        );
      case WeChatConnectionStatus.error:
        return _WeChatStatusInfo(
          title: l10n.wechatConnectError,
          badge: l10n.error,
          description: l10n.wechatConnectErrorDesc,
          color: Colors.red.shade600,
          icon: Icons.error_outline,
        );
      case WeChatConnectionStatus.disconnected:
        return _WeChatStatusInfo(
          title: l10n.wechatDisconnected,
          badge: l10n.disconnected,
          description: l10n.wechatDisconnectedDesc,
          color: Colors.grey.shade600,
          icon: Icons.link_off,
        );
    }
  }

  Widget _buildQRCodeOrStatus(AppLocalizations l10n) {
    if (_status == WeChatConnectionStatus.connected) {
      return Container(
        width: 220,
        height: 220,
        decoration: BoxDecoration(
          color: Colors.green.shade50,
          borderRadius: BorderRadius.circular(24),
          border: Border.all(color: Colors.green.shade100),
        ),
        child: Column(
          mainAxisAlignment: MainAxisAlignment.center,
          children: [
            Icon(
              Icons.check_circle_outline,
              size: 72,
              color: Colors.green.shade500,
            ),
            const SizedBox(height: 14),
            Text(
              l10n.connectionSuccess,
              style: TextStyle(
                fontSize: 18,
                fontWeight: FontWeight.w700,
                color: Colors.green.shade700,
              ),
            ),
          ],
        ),
      );
    }

    if (_qrCodeUrl != null) {
      return Container(
        width: 220,
        height: 220,
        padding: const EdgeInsets.all(14),
        decoration: BoxDecoration(
          color: Colors.white,
          borderRadius: BorderRadius.circular(24),
          border: Border.all(color: Colors.grey.shade200),
          boxShadow: [
            BoxShadow(
              color: Colors.black.withAlpha(12),
              blurRadius: 18,
              offset: const Offset(0, 8),
            ),
          ],
        ),
        child: ClipRRect(
          borderRadius: BorderRadius.circular(16),
          child: QrImageView(
            data: _qrCodeUrl!,
            version: QrVersions.auto,
            size: 192,
            backgroundColor: Colors.white,
          ),
        ),
      );
    }

    final isConnecting = _status == WeChatConnectionStatus.connecting;

    return Container(
      width: 220,
      height: 220,
      decoration: BoxDecoration(
        color: Colors.grey.shade50,
        borderRadius: BorderRadius.circular(24),
        border: Border.all(color: Colors.grey.shade200),
      ),
      child: Column(
        mainAxisAlignment: MainAxisAlignment.center,
        children: [
          if (isConnecting)
            SizedBox(
              width: 52,
              height: 52,
              child: CircularProgressIndicator(
                strokeWidth: 3,
                color: AppColors.primary,
              ),
            )
          else
            Icon(Icons.qr_code_2_rounded, size: 58, color: AppColors.textHint),
          const SizedBox(height: 14),
          Text(
            isConnecting
                ? l10n.fetchingQrCodeEllipsis
                : l10n.clickRefreshQrCode,
            style: TextStyle(
              fontSize: 14,
              color: AppColors.textSecondary,
              fontWeight: FontWeight.w500,
            ),
          ),
        ],
      ),
    );
  }

  Widget _buildInstructions(AppLocalizations l10n) {
    return Container(
      width: double.infinity,
      padding: const EdgeInsets.all(16),
      decoration: BoxDecoration(
        color: Colors.blue.shade50,
        borderRadius: BorderRadius.circular(16),
        border: Border.all(color: Colors.blue.shade100),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            children: [
              Icon(Icons.info_outline, size: 18, color: Colors.blue.shade600),
              const SizedBox(width: 8),
              Text(
                l10n.scanInstructions,
                style: TextStyle(
                  fontSize: 14,
                  fontWeight: FontWeight.w700,
                  color: Colors.blue.shade700,
                ),
              ),
            ],
          ),
          const SizedBox(height: 12),
          _buildInstructionItem('1', l10n.scanInstructionOpenWeChat),
          const SizedBox(height: 8),
          _buildInstructionItem('2', l10n.scanInstructionTapScan),
          const SizedBox(height: 8),
          _buildInstructionItem('3', l10n.scanInstructionConfirm),
        ],
      ),
    );
  }

  Widget _buildInstructionItem(String index, String text) {
    return Row(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Container(
          width: 20,
          height: 20,
          alignment: Alignment.center,
          decoration: BoxDecoration(
            color: Colors.blue.shade100,
            shape: BoxShape.circle,
          ),
          child: Text(
            index,
            style: TextStyle(
              fontSize: 12,
              fontWeight: FontWeight.w700,
              color: Colors.blue.shade700,
            ),
          ),
        ),
        const SizedBox(width: 10),
        Expanded(
          child: Text(
            text,
            style: TextStyle(
              fontSize: 13,
              height: 1.45,
              color: Colors.blue.shade700,
            ),
          ),
        ),
      ],
    );
  }
}

class _WeChatStatusInfo {
  final String title;
  final String badge;
  final String description;
  final Color color;
  final IconData icon;

  const _WeChatStatusInfo({
    required this.title,
    required this.badge,
    required this.description,
    required this.color,
    required this.icon,
  });
}
