import 'package:donk/common/model/notification_message.dart';
import 'package:donk/common/service/notification_websocket_service.dart';
import 'package:flutter/material.dart';
import 'package:get/get.dart';
import '../../l10n/generated/app_localizations.dart';

/// 消息通知页面
class NotificationView extends StatefulWidget {
  const NotificationView({super.key});

  @override
  State<NotificationView> createState() => _NotificationViewState();
}

class _NotificationViewState extends State<NotificationView> {
  final NotificationWebSocketService _service =
      Get.find<NotificationWebSocketService>();

  @override
  void initState() {
    super.initState();
    // 页面进入时刷新消息列表
    _service.refreshMessages();
  }

  /// 格式化时间
  String _formatTime(DateTime time) {
    final l10n = AppLocalizations.of(context)!;
    final now = DateTime.now();
    final diff = now.difference(time);

    if (diff.inMinutes < 1) {
      return l10n.justNow;
    } else if (diff.inHours < 1) {
      return l10n.minutesAgo(diff.inMinutes.toString());
    } else if (diff.inDays < 1) {
      return l10n.hoursAgo(diff.inHours.toString());
    } else if (diff.inDays < 7) {
      return l10n.daysAgo(diff.inDays.toString());
    } else {
      return '${time.year}-${time.month.toString().padLeft(2, '0')}-${time.day.toString().padLeft(2, '0')} ${time.hour.toString().padLeft(2, '0')}:${time.minute.toString().padLeft(2, '0')}';
    }
  }

  /// 获取消息级别图标
  IconData _getLevelIcon(String level) {
    switch (level) {
      case 'success':
        return Icons.check_circle;
      case 'warning':
        return Icons.warning;
      case 'error':
        return Icons.error;
      case 'info':
      default:
        return Icons.info;
    }
  }

  /// 获取消息级别颜色
  Color _getLevelColor(String level) {
    switch (level) {
      case 'success':
        return Colors.green;
      case 'warning':
        return Colors.orange;
      case 'error':
        return Colors.red;
      case 'info':
      default:
        return Colors.blue;
    }
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      backgroundColor: Colors.white,
      body: Column(
        children: [
          // 页面头部
          _buildHeader(),
          // 连接状态指示器
          _buildConnectionStatus(),
          // 消息列表
          Expanded(
            child: Obx(() {
              final messages = _service.messages;
              if (messages.isEmpty) {
                return _buildEmptyView();
              }
              return _buildMessageList();
            }),
          ),
        ],
      ),
    );
  }

  /// 构建页面头部
  Widget _buildHeader() {
    final l10n = AppLocalizations.of(context)!;
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 20, vertical: 16),
      decoration: BoxDecoration(
        color: Colors.white,
        border: Border(bottom: BorderSide(color: Colors.grey.shade200)),
      ),
      child: Row(
        children: [
          Text(
            l10n.notifications,
            style: const TextStyle(fontSize: 18, fontWeight: FontWeight.bold),
          ),
          const Spacer(),
          // 未读数量徽章
          Obx(() {
            final count = _service.unreadCount.value;
            if (count == 0) return const SizedBox.shrink();
            return Container(
              padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 2),
              decoration: BoxDecoration(
                color: Colors.red,
                borderRadius: BorderRadius.circular(10),
              ),
              child: Text(
                '$count',
                style: const TextStyle(color: Colors.white, fontSize: 12),
              ),
            );
          }),
          const SizedBox(width: 12),
          // 标记全部已读按钮
          TextButton.icon(
            onPressed: () async {
              await _service.markAllAsRead();
            },
            icon: const Icon(Icons.done_all, size: 16),
            label: Text(l10n.markAllRead),
          ),
          // 清空按钮
          IconButton(
            onPressed: () async {
              final confirm = await showDialog<bool>(
                context: context,
                builder:
                    (context) => AlertDialog(
                      title: Text(l10n.clearConfirmTitle),
                      content: Text(l10n.clearConfirmMessage),
                      actions: [
                        TextButton(
                          onPressed: () => Navigator.pop(context, false),
                          child: Text(l10n.cancel),
                        ),
                        TextButton(
                          onPressed: () => Navigator.pop(context, true),
                          child: Text(l10n.confirm),
                        ),
                      ],
                    ),
              );
              if (confirm == true) {
                await _service.clearAllMessages();
              }
            },
            icon: const Icon(Icons.delete_outline),
            tooltip: l10n.clearAll,
          ),
        ],
      ),
    );
  }

  /// 构建连接状态指示器（仅在连接异常时显示）
  Widget _buildConnectionStatus() {
    return Obx(() {
      final connected = _service.isConnected.value;
      // 连接正常时不显示
      if (connected) return const SizedBox.shrink();

      final l10n = AppLocalizations.of(context)!;
      return Container(
        padding: const EdgeInsets.symmetric(horizontal: 20, vertical: 8),
        color: Colors.orange.shade50,
        child: Row(
          children: [
            Icon(Icons.cloud_off, size: 14, color: Colors.orange),
            const SizedBox(width: 8),
            Text(
              l10n.websocketDisconnected,
              style: TextStyle(fontSize: 12, color: Colors.orange.shade700),
            ),
            const Spacer(),
            TextButton(
              onPressed: () => _service.reconnect(),
              child: Text(l10n.reconnect),
            ),
          ],
        ),
      );
    });
  }

  /// 构建空状态视图
  Widget _buildEmptyView() {
    final l10n = AppLocalizations.of(context)!;
    return Center(
      child: Column(
        mainAxisAlignment: MainAxisAlignment.center,
        children: [
          Icon(Icons.notifications_none, size: 64, color: Colors.grey.shade300),
          const SizedBox(height: 16),
          Text(
            l10n.noMessages,
            style: TextStyle(fontSize: 16, color: Colors.grey.shade500),
          ),
        ],
      ),
    );
  }

  /// 构建消息列表
  Widget _buildMessageList() {
    return RefreshIndicator(
      onRefresh: () => _service.refreshMessages(),
      child: Obx(() {
        return ListView.builder(
          padding: const EdgeInsets.all(12),
          itemCount: _service.messages.length,
          itemBuilder: (context, index) {
            final message = _service.messages[index];
            return _buildMessageItem(message);
          },
        );
      }),
    );
  }

  /// 构建消息项
  Widget _buildMessageItem(NotificationMessage message) {
    return Dismissible(
      key: Key(message.id),
      direction: DismissDirection.endToStart,
      background: Container(
        alignment: Alignment.centerRight,
        padding: const EdgeInsets.only(right: 20),
        color: Colors.red,
        child: const Icon(Icons.delete, color: Colors.white),
      ),
      onDismissed: (_) async {
        await _service.deleteMessage(message.id);
      },
      child: Card(
        elevation: 0,
        color: message.isRead ? Colors.grey.shade50 : Colors.blue.shade50,
        shape: RoundedRectangleBorder(
          borderRadius: BorderRadius.circular(8),
          side: BorderSide(
            color: message.isRead ? Colors.grey.shade200 : Colors.blue.shade100,
          ),
        ),
        child: InkWell(
          onTap: () async {
            if (!message.isRead) {
              await _service.markAsRead(message.id);
            }
          },
          borderRadius: BorderRadius.circular(8),
          child: Padding(
            padding: const EdgeInsets.all(16),
            child: Row(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                // 级别图标
                Container(
                  width: 40,
                  height: 40,
                  decoration: BoxDecoration(
                    color: _getLevelColor(message.level).withValues(alpha: 0.1),
                    borderRadius: BorderRadius.circular(20),
                  ),
                  child: Icon(
                    _getLevelIcon(message.level),
                    color: _getLevelColor(message.level),
                    size: 20,
                  ),
                ),
                const SizedBox(width: 12),
                // 消息内容
                Expanded(
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Row(
                        children: [
                          Expanded(
                            child: Text(
                              message.title,
                              style: TextStyle(
                                fontSize: 14,
                                fontWeight:
                                    message.isRead
                                        ? FontWeight.normal
                                        : FontWeight.bold,
                              ),
                            ),
                          ),
                          if (!message.isRead)
                            Container(
                              width: 8,
                              height: 8,
                              decoration: const BoxDecoration(
                                color: Colors.red,
                                shape: BoxShape.circle,
                              ),
                            ),
                        ],
                      ),
                      const SizedBox(height: 4),
                      Text(
                        message.content,
                        style: TextStyle(
                          fontSize: 13,
                          color: Colors.grey.shade600,
                        ),
                      ),
                      const SizedBox(height: 8),
                      Text(
                        _formatTime(message.timestamp),
                        style: TextStyle(
                          fontSize: 11,
                          color: Colors.grey.shade400,
                        ),
                      ),
                    ],
                  ),
                ),
              ],
            ),
          ),
        ),
      ),
    );
  }
}
