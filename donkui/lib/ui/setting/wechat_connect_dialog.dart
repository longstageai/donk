import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter_smart_dialog/flutter_smart_dialog.dart';
import 'package:qr_flutter/qr_flutter.dart';
import '../../app/conf/colors.dart';
import '../../common/service/wechat_bot_service.dart';

/// 微信连接对话框
/// 显示二维码和连接状态
class WeChatConnectDialog extends StatefulWidget {
  const WeChatConnectDialog({super.key});

  @override
  State<WeChatConnectDialog> createState() => _WeChatConnectDialogState();

  /// 显示对话框
  static void show() {
    SmartDialog.show(
      clickMaskDismiss: true,
      maskColor: Colors.black.withAlpha(100),
      animationType: SmartAnimationType.fade,
      builder: (context) => const WeChatConnectDialog(),
    );
  }

  /// 关闭对话框
  static void dismiss() {
    SmartDialog.dismiss();
  }
}

class _WeChatConnectDialogState extends State<WeChatConnectDialog> {
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
    // 先设置 Stream 监听（必须在 initialize 之前，否则可能错过事件）
    _initStreams();

    // 初始化服务
    await _service.initialize();

    // 检查连接状态
    await _checkConnection();
  }

  void _initStreams() {
    // 监听状态变化
    _statusSubscription = _service.connectionStatus.listen((status) {
      if (!mounted) return;
      // 保存旧状态
      final oldStatus = _status;

      setState(() {
        _status = status;
      });

      // 如果断开连接且之前不是断开状态，自动重新获取二维码
      if (status == WeChatConnectionStatus.disconnected &&
          oldStatus != WeChatConnectionStatus.disconnected) {
        _handleDisconnectAndReconnect();
      }
    });

    // 监听二维码
    _qrCodeSubscription = _service.qrCodeStream.listen((url) {
      if (!mounted) return;
      setState(() {
        _qrCodeUrl = url;
        // 收到新二维码时清除错误信息
        if (url != null) {
          _errorMessage = null;
        }
      });
    });
  }

  /// 处理断开连接并重新获取二维码
  Future<void> _handleDisconnectAndReconnect() async {
    // 延迟一下确保状态更新完成
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
    // 检查是否已有连接
    if (_service.isConnected) {
      if (!mounted) return;
      setState(() {
        _status = WeChatConnectionStatus.connected;
      });
    } else {
      // 尝试自动连接
      await _connect();
    }
  }

  Future<void> _connect() async {
    try {
      await _service.connect();
    } catch (e) {
      if (!mounted) return;
      setState(() {
        _errorMessage = e.toString();
      });
    }
  }

  Future<void> _disconnect() async {
    await _service.disconnect();
  }

  Future<void> _reconnect() async {
    setState(() {
      _errorMessage = null;
      _qrCodeUrl = null;
    });
    try {
      await _service.forceReconnect();
    } catch (e) {
      if (!mounted) return;
      setState(() {
        _errorMessage = e.toString();
      });
    }
  }

  @override
  void dispose() {
    _service.cancelPendingConnection();
    _statusSubscription?.cancel();
    _qrCodeSubscription?.cancel();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return Container(
      width: 400,
      padding: const EdgeInsets.all(24),
      decoration: BoxDecoration(
        color: Colors.white,
        borderRadius: BorderRadius.circular(12),
      ),
      child: Column(
        mainAxisSize: MainAxisSize.min,
        children: [
          // 标题
          _buildHeader(),
          const SizedBox(height: 24),
          // 内容区域
          _buildContent(),
          const SizedBox(height: 24),
          // 底部按钮
          _buildFooter(),
        ],
      ),
    );
  }

  Widget _buildHeader() {
    return Row(
      mainAxisAlignment: MainAxisAlignment.spaceBetween,
      children: [
        Row(
          children: [
            Icon(Icons.phone_android, color: AppColors.c1, size: 24),
            const SizedBox(width: 12),
            const Text(
              '微信连接',
              style: TextStyle(fontSize: 18, fontWeight: FontWeight.w600),
            ),
          ],
        ),
        Row(
          mainAxisSize: MainAxisSize.min,
          children: [
            // 状态指示器
            Container(
              padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 4),
              decoration: BoxDecoration(
                color: _getStatusColor().withAlpha(20),
                borderRadius: BorderRadius.circular(12),
              ),
              child: Row(
                mainAxisSize: MainAxisSize.min,
                children: [
                  Container(
                    width: 8,
                    height: 8,
                    decoration: BoxDecoration(
                      color: _getStatusColor(),
                      shape: BoxShape.circle,
                    ),
                  ),
                  const SizedBox(width: 6),
                  Text(
                    _status.displayName,
                    style: TextStyle(fontSize: 12, color: _getStatusColor()),
                  ),
                ],
              ),
            ),
            const SizedBox(width: 8),
            // 关闭按钮
            IconButton(
              onPressed: () => WeChatConnectDialog.dismiss(),
              icon: const Icon(Icons.close, size: 20),
              padding: EdgeInsets.zero,
              constraints: const BoxConstraints(),
              tooltip: '关闭',
            ),
          ],
        ),
      ],
    );
  }

  Widget _buildContent() {
    // 已连接状态
    if (_status == WeChatConnectionStatus.connected) {
      return _buildConnectedView();
    }

    // 只要有二维码URL就显示（不限制状态）
    if (_qrCodeUrl != null) {
      return _buildQRCodeView();
    }

    // 已扫码等待确认
    if (_status == WeChatConnectionStatus.scanning) {
      return _buildScanningView();
    }

    // 错误状态
    if (_status == WeChatConnectionStatus.error) {
      return _buildErrorView();
    }

    // 连接中
    return _buildLoadingView();
  }

  Widget _buildQRCodeView() {
    return Column(
      children: [
        Container(
          width: 200,
          height: 200,
          padding: const EdgeInsets.all(12),
          decoration: BoxDecoration(
            color: Colors.white,
            borderRadius: BorderRadius.circular(8),
            border: Border.all(color: Colors.grey.shade300),
          ),
          child: QrImageView(
            data: _qrCodeUrl!,
            version: QrVersions.auto,
            size: 176,
            backgroundColor: Colors.white,
            embeddedImageStyle: const QrEmbeddedImageStyle(
              size: Size(176, 176),
            ),
            errorStateBuilder: (context, error) {
              return SizedBox(
                width: 176,
                height: 176,
                child: Column(
                  mainAxisAlignment: MainAxisAlignment.center,
                  children: [
                    Icon(Icons.qr_code, size: 80, color: Colors.grey.shade400),
                    const SizedBox(height: 8),
                    Text(
                      '二维码生成失败',
                      style: TextStyle(
                        color: Colors.grey.shade600,
                        fontSize: 12,
                      ),
                    ),
                  ],
                ),
              );
            },
          ),
        ),
        const SizedBox(height: 16),
        Text(
          '请使用微信扫描二维码登录',
          style: TextStyle(fontSize: 14, color: Colors.grey.shade700),
        ),
        const SizedBox(height: 8),
        Text(
          '二维码将在一段时间后过期，请尽快扫描',
          style: TextStyle(fontSize: 12, color: Colors.grey.shade500),
        ),
      ],
    );
  }

  Widget _buildScanningView() {
    return Column(
      children: [
        Container(
          width: 120,
          height: 120,
          decoration: BoxDecoration(
            color: Colors.green.withAlpha(20),
            shape: BoxShape.circle,
          ),
          child: Icon(
            Icons.check_circle_outline,
            size: 60,
            color: Colors.green.shade600,
          ),
        ),
        const SizedBox(height: 24),
        const Text(
          '已扫码',
          style: TextStyle(fontSize: 18, fontWeight: FontWeight.w600),
        ),
        const SizedBox(height: 8),
        Text(
          '请在手机上确认登录',
          style: TextStyle(fontSize: 14, color: Colors.grey.shade600),
        ),
      ],
    );
  }

  Widget _buildConnectedView() {
    return Column(
      children: [
        Container(
          width: 120,
          height: 120,
          decoration: BoxDecoration(
            color: Colors.green.withAlpha(20),
            shape: BoxShape.circle,
          ),
          child: Icon(Icons.wechat, size: 60, color: Colors.green.shade600),
        ),
        const SizedBox(height: 24),
        const Text(
          '微信已连接',
          style: TextStyle(fontSize: 18, fontWeight: FontWeight.w600),
        ),
        const SizedBox(height: 8),
        Text(
          '您可以接收和发送微信消息',
          style: TextStyle(fontSize: 14, color: Colors.grey.shade600),
        ),
      ],
    );
  }

  Widget _buildLoadingView() {
    return Column(
      children: [
        // 与二维码容器大小一致 (200x200)
        Container(
          width: 200,
          height: 200,
          decoration: BoxDecoration(
            color: Colors.grey.shade50,
            borderRadius: BorderRadius.circular(8),
            border: Border.all(color: Colors.grey.shade300),
          ),
          child: const Center(
            child: SizedBox(
              width: 50,
              height: 50,
              child: CircularProgressIndicator(strokeWidth: 3),
            ),
          ),
        ),
        const SizedBox(height: 24),
        Text(
          '正在连接微信...',
          style: TextStyle(fontSize: 14, color: Colors.grey.shade600),
        ),
        const SizedBox(height: 28),
      ],
    );
  }

  Widget _buildErrorView() {
    return Column(
      children: [
        Container(
          width: 120,
          height: 120,
          decoration: BoxDecoration(
            color: Colors.red.withAlpha(20),
            shape: BoxShape.circle,
          ),
          child: Icon(
            Icons.error_outline,
            size: 60,
            color: Colors.red.shade600,
          ),
        ),
        const SizedBox(height: 24),
        const Text(
          '连接失败',
          style: TextStyle(fontSize: 18, fontWeight: FontWeight.w600),
        ),
        const SizedBox(height: 8),
        Padding(
          padding: const EdgeInsets.symmetric(horizontal: 16),
          child: Text(
            _errorMessage ?? '未知错误',
            style: TextStyle(fontSize: 12, color: Colors.grey.shade600),
            textAlign: TextAlign.center,
            maxLines: 3,
            overflow: TextOverflow.ellipsis,
          ),
        ),
      ],
    );
  }

  Widget _buildFooter() {
    // 已连接状态显示断开按钮
    if (_status == WeChatConnectionStatus.connected) {
      return Row(
        mainAxisAlignment: MainAxisAlignment.center,
        children: [
          TextButton.icon(
            onPressed: _disconnect,
            icon: const Icon(Icons.logout, size: 18),
            label: const Text('断开连接'),
            style: TextButton.styleFrom(foregroundColor: Colors.red),
          ),
        ],
      );
    }

    // 错误状态显示重试按钮
    if (_status == WeChatConnectionStatus.error) {
      return Row(
        mainAxisAlignment: MainAxisAlignment.center,
        children: [
          ElevatedButton.icon(
            onPressed: _reconnect,
            icon: const Icon(Icons.refresh, size: 18),
            label: const Text('重新连接'),
            style: ElevatedButton.styleFrom(
              backgroundColor: AppColors.c1,
              foregroundColor: Colors.white,
            ),
          ),
        ],
      );
    }

    // 其他状态显示取消按钮
    return Row(
      mainAxisAlignment: MainAxisAlignment.center,
      children: [
        TextButton(
          onPressed: () => WeChatConnectDialog.dismiss(),
          child: const Text('取消'),
        ),
      ],
    );
  }

  Color _getStatusColor() {
    switch (_status) {
      case WeChatConnectionStatus.connected:
        return Colors.green;
      case WeChatConnectionStatus.connecting:
      case WeChatConnectionStatus.waitingForScan:
      case WeChatConnectionStatus.scanning:
        return Colors.orange;
      case WeChatConnectionStatus.error:
        return Colors.red;
      case WeChatConnectionStatus.disconnected:
        return Colors.grey;
    }
  }
}
