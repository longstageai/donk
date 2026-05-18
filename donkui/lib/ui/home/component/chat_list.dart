import 'package:flutter/material.dart';
import 'package:get/get.dart';
import '../home_controller.dart';
import 'chat_message_item.dart';

/// 聊天列表组件
/// 显示用户和Agent的对话消息，支持自动滚动到底部
class ChatList extends StatefulWidget {
  const ChatList({super.key});

  @override
  State<ChatList> createState() => _ChatListState();
}

class _ChatListState extends State<ChatList> {
  final ScrollController _scrollController = ScrollController();

  @override
  void initState() {
    super.initState();
    // 组件初始化时延迟滚动到底部（等待列表数据渲染完成）
    WidgetsBinding.instance.addPostFrameCallback((_) async {
      await Future.delayed(const Duration(milliseconds: 500));
      _scrollToBottom();
    });
  }

  @override
  void dispose() {
    _scrollController.dispose();
    super.dispose();
  }

  /// 滚动到列表底部
  void _scrollToBottom() {
    if (_scrollController.hasClients) {
      _scrollController.animateTo(
        _scrollController.position.maxScrollExtent,
        duration: const Duration(milliseconds: 300),
        curve: Curves.easeOut,
      );
    }
  }

  @override
  Widget build(BuildContext context) {
    final controller = Get.find<HomeController>();

    return Obx(() {
      // 消息数量变化时，延迟滚动到底部（等待布局完成）
      WidgetsBinding.instance.addPostFrameCallback((_) {
        _scrollToBottom();
      });

      return ListView.builder(
        controller: _scrollController,
        padding: const EdgeInsets.symmetric(vertical: 16),
        itemCount: controller.chatMessages.length,
        itemBuilder: (context, index) {
          final message = controller.chatMessages[index];
          // 判断是否为最后一条消息
          final isLastMessage = index == controller.chatMessages.length - 1;
          return ChatMessageItem(
            message: message,
            isLastMessage: isLastMessage,
            onToggleReasoning:
                message.isAgent && message.reasoning != null
                    ? () => controller.toggleReasoning(index)
                    : null,
            onCopy: () {
              // 复制功能已在组件内实现
            },
            onEdit:
                message.isUser
                    ? () => controller.setEditText(message.content)
                    : null,
            onLike: () {
              // TODO: 实现点赞功能
            },
            onDislike: () {
              // TODO: 实现点踩功能
            },
            onRefresh: () {
              // TODO: 实现重新生成功能
            },
          );
        },
      );
    });
  }
}
