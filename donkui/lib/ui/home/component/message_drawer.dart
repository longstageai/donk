import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_markdown/flutter_markdown.dart';
import 'package:get/get.dart';
import 'package:intl/intl.dart';

import '../../../common/service/notification_websocket_service.dart';
import '../../../common/service/creative_service.dart';
import '../../../l10n/generated/app_localizations.dart';

/// WebSocket Stream消息抽屉组件
/// 展示多Agent实时讨论消息，支持Markdown渲染
class MessageDrawer extends StatefulWidget {
  const MessageDrawer({super.key});

  @override
  State<MessageDrawer> createState() => _MessageDrawerState();

  /// 全局状态刷新通知器
  static final _refreshNotifier = ValueNotifier<bool>(false);

  /// 刷新 Creative 运行状态
  /// 当抽屉打开时调用此方法刷新状态
  static void refreshStatus() {
    _refreshNotifier.value = !_refreshNotifier.value;
  }
}

class _MessageDrawerState extends State<MessageDrawer> {
  final notificationService = Get.find<NotificationWebSocketService>();
  final ScrollController _scrollController = ScrollController();
  final bool _isAutoScroll = true;
  bool _isSessionRunning = false;
  bool _isLoading = false;

  @override
  void initState() {
    super.initState();
    // 监听刷新通知
    MessageDrawer._refreshNotifier.addListener(_loadCreativeStatus);
    // 初始化时查询 Creative 运行状态
    _loadCreativeStatus();
    // 初始化时滚动到底部
    WidgetsBinding.instance.addPostFrameCallback((_) {
      _scrollToBottom();
    });
  }

  @override
  void dispose() {
    // 移除刷新监听
    MessageDrawer._refreshNotifier.removeListener(_loadCreativeStatus);
    _scrollController.dispose();
    super.dispose();
  }

  /// 加载 Creative 运行状态
  Future<void> _loadCreativeStatus() async {
    try {
      final status = await CreativeService.getStatus();
      debugPrint('Creative 状态查询结果: $status');
      if (mounted) {
        setState(() {
          _isSessionRunning = status['running'] ?? false;
        });
        debugPrint('Creative 状态已更新: _isSessionRunning = $_isSessionRunning');
      }
    } catch (e) {
      // 查询失败，保持默认状态
      debugPrint('查询 Creative 状态失败: $e');
    }
  }

  /// 滚动到底部（最新消息）
  void _scrollToBottom() {
    if (_scrollController.hasClients && _isAutoScroll) {
      _scrollController.animateTo(
        _scrollController.position.maxScrollExtent,
        duration: const Duration(milliseconds: 300),
        curve: Curves.easeOut,
      );
    }
  }

  /// 根据agent_id获取对应的颜色
  Color _getAgentColor(String agentId) {
    final colors = [
      Colors.blue[400]!,
      Colors.green[400]!,
      Colors.orange[400]!,
      Colors.purple[400]!,
      Colors.teal[400]!,
      Colors.pink[400]!,
      Colors.indigo[400]!,
      Colors.cyan[400]!,
    ];
    int hash = agentId.hashCode.abs();
    return colors[hash % colors.length];
  }

  @override
  Widget build(BuildContext context) {
    final screenWidth = MediaQuery.of(context).size.width;
    return Container(
      width: screenWidth * 0.6,
      decoration: const BoxDecoration(
        gradient: LinearGradient(
          colors: [Color(0xFFF7F7FF), Color(0xFFF8FBFF), Color(0xFFFFFFFF)],
          begin: Alignment.topCenter,
          end: Alignment.bottomCenter,
        ),
      ),
      child: Column(
        children: [
          _buildHeader(),
          // _buildConnectionStatus(),
          Expanded(
            child: Obx(() {
              final messages = notificationService.streamMessages;

              // 消息变化时自动滚动到底部
              WidgetsBinding.instance.addPostFrameCallback((_) {
                _scrollToBottom();
              });

              if (messages.isEmpty) {
                return _buildEmptyState();
              }

              return _buildMessageList(messages);
            }),
          ),
        ],
      ),
    );
  }

  /// 构建顶部导航栏
  Widget _buildHeader() {
    final l10n = AppLocalizations.of(context)!;
    return Container(
      padding: EdgeInsets.only(
        left: 18,
        right: 14,
        top: MediaQuery.of(context).padding.top + 14,
        bottom: 14,
      ),
      decoration: BoxDecoration(
        gradient: LinearGradient(
          colors: [Colors.deepPurple[500]!, Colors.blue[500]!],
          begin: Alignment.topLeft,
          end: Alignment.bottomRight,
        ),
        borderRadius: const BorderRadius.only(
          bottomLeft: Radius.circular(24),
          bottomRight: Radius.circular(24),
        ),
        boxShadow: [
          BoxShadow(
            color: Colors.blue.withAlpha(45),
            blurRadius: 18,
            offset: const Offset(0, 8),
          ),
        ],
      ),
      child: Row(
        mainAxisAlignment: MainAxisAlignment.spaceBetween,
        children: [
          // 标题
          Flexible(
            child: Row(
              mainAxisSize: MainAxisSize.min,
              children: [
                Container(
                  width: 42,
                  height: 42,
                  decoration: BoxDecoration(
                    color: Colors.white.withAlpha(35),
                    borderRadius: BorderRadius.circular(14),
                    border: Border.all(color: Colors.white.withAlpha(70)),
                  ),
                  child: const Icon(
                    Icons.hub_outlined,
                    color: Colors.white,
                    size: 22,
                  ),
                ),
                const SizedBox(width: 12),
                Flexible(
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    mainAxisAlignment: MainAxisAlignment.center,
                    children: [
                      Text(
                        l10n.agentCollaboration,
                        style: TextStyle(
                          fontSize: 18,
                          fontWeight: FontWeight.w700,
                          color: Colors.white,
                          letterSpacing: 0.2,
                        ),
                      ),
                      Obx(() {
                        final count = notificationService.streamMessages.length;
                        return Text(
                          l10n.agentActivityStatus(count),
                          style: TextStyle(
                            fontSize: 12,
                            color: Colors.white.withAlpha(210),
                            fontWeight: FontWeight.w400,
                          ),
                          overflow: TextOverflow.ellipsis,
                        );
                      }),
                    ],
                  ),
                ),
              ],
            ),
          ),

          // 右侧操作按钮
          Row(
            mainAxisSize: MainAxisSize.min,
            children: [
              // 开始按钮
              _buildControlButton(
                Icons.play_arrow,
                l10n.start,
                _isSessionRunning,
                _isLoading,
                () => _startSession(),
                isStart: true,
              ),
              const SizedBox(width: 8),
              // 停止按钮
              _buildControlButton(
                Icons.stop,
                l10n.stop,
                !_isSessionRunning,
                _isLoading,
                () => _stopSession(),
                isStart: false,
              ),
              const SizedBox(width: 8),
              // 清空按钮
              _buildIconButton(
                Icons.delete_outline,
                () => _showClearConfirmDialog(),
                tooltip: l10n.clearMessages,
              ),
              const SizedBox(width: 8),
              // 关闭按钮
              _buildIconButton(
                Icons.close,
                () => Navigator.of(context).pop(),
                tooltip: l10n.close,
              ),
            ],
          ),
        ],
      ),
    );
  }

  /// 构建图标按钮
  Widget _buildIconButton(
    IconData icon,
    VoidCallback onTap, {
    String? tooltip,
    Color? color,
  }) {
    return Tooltip(
      message: tooltip ?? '',
      child: Material(
        color: Colors.transparent,
        child: InkWell(
          onTap: onTap,
          borderRadius: BorderRadius.circular(10),
          child: Container(
            padding: const EdgeInsets.all(9),
            decoration: BoxDecoration(
              color: Colors.white.withAlpha(35),
              borderRadius: BorderRadius.circular(10),
              border: Border.all(color: Colors.white.withAlpha(45)),
            ),
            child: Icon(icon, size: 18, color: color ?? Colors.white),
          ),
        ),
      ),
    );
  }

  /// 构建控制按钮（开始/停止）
  Widget _buildControlButton(
    IconData icon,
    String label,
    bool isDisabled,
    bool isLoading,
    VoidCallback onTap, {
    required bool isStart,
  }) {
    return Tooltip(
      message: label,
      child: Material(
        color: Colors.transparent,
        child: InkWell(
          onTap: isDisabled || isLoading ? null : onTap,
          borderRadius: BorderRadius.circular(8),
          child: Container(
            padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 6),
            decoration: BoxDecoration(
              color:
                  isDisabled
                      ? Colors.white.withAlpha(25)
                      : (isStart ? Colors.green[50] : Colors.red[50]),
              borderRadius: BorderRadius.circular(10),
              border: Border.all(
                color:
                    isDisabled
                        ? Colors.white.withAlpha(35)
                        : (isStart ? Colors.green[200]! : Colors.red[200]!),
              ),
            ),
            child: Row(
              mainAxisSize: MainAxisSize.min,
              children: [
                if (isLoading)
                  SizedBox(
                    width: 16,
                    height: 16,
                    child: CircularProgressIndicator(
                      strokeWidth: 2,
                      valueColor: AlwaysStoppedAnimation<Color>(
                        isStart ? Colors.green[600]! : Colors.red[600]!,
                      ),
                    ),
                  )
                else
                  Icon(
                    icon,
                    size: 16,
                    color:
                        isDisabled
                            ? Colors.white.withAlpha(150)
                            : (isStart ? Colors.green[600] : Colors.red[600]),
                  ),
                const SizedBox(width: 4),
                Text(
                  label,
                  style: TextStyle(
                    fontSize: 12,
                    fontWeight: FontWeight.w500,
                    color:
                        isDisabled
                            ? Colors.white.withAlpha(150)
                            : (isStart ? Colors.green[700] : Colors.red[700]),
                  ),
                ),
              ],
            ),
          ),
        ),
      ),
    );
  }

  /// 构建空状态显示
  Widget _buildEmptyState() {
    final l10n = AppLocalizations.of(context)!;
    return Center(
      child: Container(
        margin: const EdgeInsets.all(24),
        padding: const EdgeInsets.symmetric(horizontal: 28, vertical: 34),
        decoration: BoxDecoration(
          color: Colors.white.withAlpha(220),
          borderRadius: BorderRadius.circular(24),
          border: Border.all(color: Colors.white),
          boxShadow: [
            BoxShadow(
              color: Colors.black.withAlpha(10),
              blurRadius: 24,
              offset: const Offset(0, 10),
            ),
          ],
        ),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            Container(
              width: 86,
              height: 86,
              decoration: BoxDecoration(
                gradient: LinearGradient(
                  colors: [Colors.deepPurple[100]!, Colors.blue[100]!],
                  begin: Alignment.topLeft,
                  end: Alignment.bottomRight,
                ),
                borderRadius: BorderRadius.circular(28),
              ),
              child: Icon(
                Icons.auto_awesome,
                size: 42,
                color: Colors.deepPurple[400],
              ),
            ),
            const SizedBox(height: 22),
            Text(
              l10n.noAgentMessages,
              style: TextStyle(
                fontSize: 18,
                fontWeight: FontWeight.w700,
                color: Colors.grey[800],
              ),
            ),
            const SizedBox(height: 8),
            Text(
              l10n.agentActivityHint,
              textAlign: TextAlign.center,
              style: TextStyle(
                fontSize: 13,
                color: Colors.grey[500],
                height: 1.5,
              ),
            ),
          ],
        ),
      ),
    );
  }

  /// 构建消息列表
  /// 消息存储顺序：最新的在头部(index 0)
  /// 显示顺序：反转，最新的在底部（符合聊天习惯）
  Widget _buildMessageList(List<Map<String, dynamic>> messages) {
    // 反转消息列表，让最新的显示在底部
    final reversedMessages = messages.reversed.toList();
    return ListView.builder(
      controller: _scrollController,
      padding: const EdgeInsets.fromLTRB(16, 8, 16, 20),
      itemCount: reversedMessages.length,
      itemBuilder: (context, index) {
        final message = reversedMessages[index];
        final isLatest = index == reversedMessages.length - 1;
        return _buildAgentMessageItem(message: message, isLatest: isLatest);
      },
    );
  }

  /// 构建Agent消息项
  Widget _buildAgentMessageItem({
    required Map<String, dynamic> message,
    required bool isLatest,
  }) {
    final content = message['content']?.toString() ?? '';
    final agentId = message['agent_id']?.toString() ?? 'Agent';
    final timestamp = message['received_at'] as int?;
    final agentColor = _getAgentColor(agentId);
    final runId = message['run_id']?.toString() ?? '';

    return AnimatedContainer(
      duration: const Duration(milliseconds: 300),
      curve: Curves.easeOutCubic,
      margin: const EdgeInsets.only(bottom: 16),
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          // Agent头像（带颜色标识）
          _buildAgentAvatar(agentId, agentColor),
          const SizedBox(width: 10),
          // 消息内容
          Flexible(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Row(
                  children: [
                    Container(
                      padding: const EdgeInsets.symmetric(
                        horizontal: 9,
                        vertical: 4,
                      ),
                      decoration: BoxDecoration(
                        color: agentColor.withAlpha(22),
                        borderRadius: BorderRadius.circular(999),
                        border: Border.all(color: agentColor.withAlpha(45)),
                      ),
                      child: Text(
                        agentId,
                        style: TextStyle(
                          fontSize: 11,
                          fontWeight: FontWeight.w700,
                          color: agentColor,
                        ),
                      ),
                    ),
                    if (isLatest) ...[
                      const SizedBox(width: 8),
                      Container(
                        padding: const EdgeInsets.symmetric(
                          horizontal: 7,
                          vertical: 3,
                        ),
                        decoration: BoxDecoration(
                          color: const Color(0xFFE8F8EF),
                          borderRadius: BorderRadius.circular(999),
                          border: Border.all(color: const Color(0xFFC7EED7)),
                        ),
                        child: Row(
                          mainAxisSize: MainAxisSize.min,
                          children: [
                            Container(
                              width: 6,
                              height: 6,
                              decoration: BoxDecoration(
                                color: Colors.green[400],
                                shape: BoxShape.circle,
                              ),
                            ),
                            const SizedBox(width: 4),
                            Text(
                              AppLocalizations.of(context)!.latestMessage,
                              style: TextStyle(
                                fontSize: 10,
                                color: Colors.green[700],
                                fontWeight: FontWeight.w600,
                              ),
                            ),
                          ],
                        ),
                      ),
                    ],
                    const Spacer(),
                    if (timestamp != null)
                      Container(
                        padding: const EdgeInsets.symmetric(
                          horizontal: 8,
                          vertical: 3,
                        ),
                        decoration: BoxDecoration(
                          color: Colors.white.withAlpha(180),
                          borderRadius: BorderRadius.circular(999),
                          border: Border.all(color: const Color(0xFFE8ECF3)),
                        ),
                        child: Text(
                          _formatTime(timestamp),
                          style: TextStyle(
                            fontSize: 10,
                            color: Colors.grey[500],
                            fontWeight: FontWeight.w500,
                          ),
                        ),
                      ),
                  ],
                ),
                const SizedBox(height: 6),
                // 消息气泡
                Container(
                  padding: const EdgeInsets.all(14),
                  decoration: BoxDecoration(
                    color: Colors.white.withAlpha(245),
                    borderRadius: BorderRadius.circular(16),
                    border: Border.all(
                      color:
                          isLatest
                              ? agentColor.withAlpha(80)
                              : const Color(0xFFEFF2F7),
                      width: isLatest ? 1.5 : 1,
                    ),
                    boxShadow: [
                      BoxShadow(
                        color: agentColor.withAlpha(isLatest ? 18 : 8),
                        blurRadius: isLatest ? 18 : 12,
                        offset: const Offset(0, 6),
                      ),
                    ],
                  ),
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      // Markdown内容
                      MarkdownBody(
                        data: content,
                        selectable: true,
                        styleSheet: MarkdownStyleSheet(
                          p: const TextStyle(
                            fontSize: 13.5,
                            color: Color(0xFF2E3440),
                            height: 1.6,
                          ),
                          h1: const TextStyle(
                            fontSize: 16,
                            fontWeight: FontWeight.bold,
                            color: Colors.black87,
                          ),
                          h2: const TextStyle(
                            fontSize: 15,
                            fontWeight: FontWeight.bold,
                            color: Colors.black87,
                          ),
                          h3: const TextStyle(
                            fontSize: 14,
                            fontWeight: FontWeight.bold,
                            color: Colors.black87,
                          ),
                          code: TextStyle(
                            fontSize: 12,
                            color: Colors.pink[700],
                            backgroundColor: Colors.grey[100],
                            fontFamily: 'monospace',
                          ),
                          codeblockDecoration: BoxDecoration(
                            color: Colors.grey[50],
                            borderRadius: BorderRadius.circular(6),
                            border: Border.all(color: Colors.grey[200]!),
                          ),
                          blockquote: const TextStyle(
                            fontSize: 13,
                            color: Colors.black54,
                            fontStyle: FontStyle.italic,
                          ),
                          blockquoteDecoration: BoxDecoration(
                            border: Border(
                              left: BorderSide(
                                color: agentColor.withAlpha(100),
                                width: 3,
                              ),
                            ),
                          ),
                          listBullet: const TextStyle(
                            fontSize: 13,
                            color: Colors.black87,
                          ),
                        ),
                      ),
                      // 操作按钮
                      const SizedBox(height: 8),
                      Row(
                        mainAxisAlignment: MainAxisAlignment.end,
                        children: [
                          _buildActionButton(
                            Icons.copy_outlined,
                            AppLocalizations.of(context)!.copyContent,
                            () => _handleCopy(content),
                          ),
                          if (runId.isNotEmpty) ...[
                            const SizedBox(width: 10),
                            Flexible(
                              child: Container(
                                padding: const EdgeInsets.symmetric(
                                  horizontal: 8,
                                  vertical: 4,
                                ),
                                decoration: BoxDecoration(
                                  color: const Color(0xFFF8FAFD),
                                  borderRadius: BorderRadius.circular(999),
                                  border: Border.all(
                                    color: const Color(0xFFE9EDF5),
                                  ),
                                ),
                                child: Text(
                                  'Run: ${runId.substring(0, runId.length > 8 ? 8 : runId.length)}...',
                                  overflow: TextOverflow.ellipsis,
                                  style: TextStyle(
                                    fontSize: 10,
                                    color: Colors.grey[500],
                                    fontWeight: FontWeight.w500,
                                  ),
                                ),
                              ),
                            ),
                          ],
                        ],
                      ),
                    ],
                  ),
                ),
              ],
            ),
          ),
        ],
      ),
    );
  }

  /// 构建Agent头像（显示首字母）
  Widget _buildAgentAvatar(String agentId, Color color) {
    String initial = agentId.isNotEmpty ? agentId[0].toUpperCase() : 'A';
    return Container(
      width: 36,
      height: 36,
      decoration: BoxDecoration(
        gradient: LinearGradient(
          colors: [color.withAlpha(105), color.withAlpha(45)],
          begin: Alignment.topLeft,
          end: Alignment.bottomRight,
        ),
        borderRadius: BorderRadius.circular(12),
        border: Border.all(color: color.withAlpha(80), width: 1),
        boxShadow: [
          BoxShadow(
            color: color.withAlpha(20),
            blurRadius: 10,
            offset: const Offset(0, 4),
          ),
        ],
      ),
      child: Center(
        child: Text(
          initial,
          style: TextStyle(
            fontSize: 14,
            fontWeight: FontWeight.bold,
            color: color,
          ),
        ),
      ),
    );
  }

  /// 构建操作按钮
  Widget _buildActionButton(IconData icon, String label, VoidCallback onTap) {
    return Material(
      color: Colors.transparent,
      child: InkWell(
        onTap: onTap,
        borderRadius: BorderRadius.circular(999),
        child: Container(
          padding: const EdgeInsets.symmetric(horizontal: 9, vertical: 5),
          decoration: BoxDecoration(
            color: const Color(0xFFF6F8FB),
            borderRadius: BorderRadius.circular(999),
            border: Border.all(color: const Color(0xFFE9EDF5)),
          ),
          child: Row(
            mainAxisSize: MainAxisSize.min,
            children: [
              Icon(icon, size: 13, color: Colors.grey[600]),
              const SizedBox(width: 4),
              Text(
                label,
                style: TextStyle(
                  fontSize: 11,
                  fontWeight: FontWeight.w600,
                  color: Colors.grey[600],
                ),
              ),
            ],
          ),
        ),
      ),
    );
  }

  /// 处理复制操作
  void _handleCopy(String text) {
    Clipboard.setData(ClipboardData(text: text));
    _showToast(AppLocalizations.of(context)!.contentCopied);
  }

  /// 显示轻提示
  void _showToast(String message, {bool isError = false}) {
    final overlay = Overlay.of(context);
    final accentColor =
        isError ? const Color(0xFFE5484D) : const Color(0xFF2EBD85);
    final overlayEntry = OverlayEntry(
      builder:
          (context) => Positioned(
            top: MediaQuery.of(context).padding.top + 18,
            left: 20,
            right: 20,
            child: TweenAnimationBuilder<double>(
              tween: Tween(begin: 0, end: 1),
              duration: const Duration(milliseconds: 220),
              curve: Curves.easeOutCubic,
              builder: (context, value, child) {
                return Opacity(
                  opacity: value,
                  child: Transform.translate(
                    offset: Offset(0, -10 * (1 - value)),
                    child: child,
                  ),
                );
              },
              child: Center(
                child: Material(
                  color: Colors.transparent,
                  child: Container(
                    constraints: const BoxConstraints(maxWidth: 420),
                    padding: const EdgeInsets.symmetric(
                      horizontal: 14,
                      vertical: 12,
                    ),
                    decoration: BoxDecoration(
                      color: Colors.white,
                      borderRadius: BorderRadius.circular(16),
                      border: Border.all(color: const Color(0xFFE9EDF5)),
                      boxShadow: [
                        BoxShadow(
                          color: Colors.black.withAlpha(22),
                          blurRadius: 24,
                          offset: const Offset(0, 10),
                        ),
                      ],
                    ),
                    child: Row(
                      mainAxisSize: MainAxisSize.min,
                      children: [
                        Container(
                          width: 28,
                          height: 28,
                          decoration: BoxDecoration(
                            color: accentColor.withAlpha(25),
                            borderRadius: BorderRadius.circular(10),
                          ),
                          child: Icon(
                            isError
                                ? Icons.error_outline
                                : Icons.check_circle_outline,
                            size: 17,
                            color: accentColor,
                          ),
                        ),
                        const SizedBox(width: 10),
                        Flexible(
                          child: Text(
                            message,
                            style: const TextStyle(
                              color: Color(0xFF2E3440),
                              fontSize: 13,
                              fontWeight: FontWeight.w600,
                              height: 1.35,
                            ),
                          ),
                        ),
                      ],
                    ),
                  ),
                ),
              ),
            ),
          ),
    );

    overlay.insert(overlayEntry);
    Future.delayed(const Duration(milliseconds: 1600), () {
      overlayEntry.remove();
    });
  }

  /// 格式化时间显示
  String _formatTime(int timestamp) {
    final dateTime = DateTime.fromMillisecondsSinceEpoch(timestamp);
    final now = DateTime.now();
    final diff = now.difference(dateTime);

    if (diff.inSeconds < 10) {
      return AppLocalizations.of(context)!.justNow;
    } else if (diff.inSeconds < 60) {
      return AppLocalizations.of(context)!.secondsAgo(diff.inSeconds);
    } else if (diff.inMinutes < 60) {
      return AppLocalizations.of(context)!.minutesAgo(diff.inMinutes);
    } else if (diff.inHours < 24) {
      return AppLocalizations.of(context)!.hoursAgo(diff.inHours);
    } else {
      return DateFormat('MM-dd HH:mm').format(dateTime);
    }
  }

  /// 开始会话
  Future<void> _startSession() async {
    if (_isLoading) return;

    setState(() {
      _isLoading = true;
    });

    try {
      await CreativeService.startSession();
      if (!mounted) return;
      setState(() {
        _isSessionRunning = true;
      });
      _showToast(AppLocalizations.of(context)!.sessionStarted);
    } catch (e) {
      if (!mounted) return;
      _showToast(
        AppLocalizations.of(context)!.sessionStartFailed(e),
        isError: true,
      );
    } finally {
      if (mounted) {
        setState(() {
          _isLoading = false;
        });
      }
    }
  }

  /// 停止会话
  Future<void> _stopSession() async {
    if (_isLoading) return;

    setState(() {
      _isLoading = true;
    });

    try {
      await CreativeService.stopSession();
      if (!mounted) return;
      setState(() {
        _isSessionRunning = false;
      });
      _showToast(AppLocalizations.of(context)!.sessionStopped);
    } catch (e) {
      if (!mounted) return;
      _showToast(
        AppLocalizations.of(context)!.sessionStopFailed(e),
        isError: true,
      );
    } finally {
      if (mounted) {
        setState(() {
          _isLoading = false;
        });
      }
    }
  }

  /// 显示清空确认对话框
  void _showClearConfirmDialog() {
    final l10n = AppLocalizations.of(context)!;
    showDialog(
      context: context,
      builder:
          (context) => AlertDialog(
            shape: RoundedRectangleBorder(
              borderRadius: BorderRadius.circular(12),
            ),
            title: Text(l10n.clearMessages),
            content: Text(l10n.clearAgentMessagesConfirm),
            actions: [
              TextButton(
                onPressed: () => Navigator.of(context).pop(),
                child: Text(
                  l10n.cancel,
                  style: TextStyle(color: Colors.grey[600]),
                ),
              ),
              TextButton(
                onPressed: () {
                  notificationService.clearAllStreamMessages();
                  Navigator.of(context).pop();
                },
                style: TextButton.styleFrom(foregroundColor: Colors.red),
                child: Text(l10n.clearMessages),
              ),
            ],
          ),
    );
  }
}
