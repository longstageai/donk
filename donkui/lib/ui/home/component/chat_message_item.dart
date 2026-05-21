import 'dart:io';

import 'package:donk/common/util/color_util.dart';
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_markdown/flutter_markdown.dart';
import 'package:open_filex/open_filex.dart';
import 'package:path/path.dart' as p;

import '../../../common/model/chat_message.dart';

/// 聊天消息项组件
/// 根据发送者类型显示不同样式的消息气泡
class ChatMessageItem extends StatefulWidget {
  final ChatMessage message;
  final bool isLastMessage;
  final VoidCallback? onToggleReasoning;
  final VoidCallback? onCopy;
  final VoidCallback? onEdit;
  final VoidCallback? onLike;
  final VoidCallback? onDislike;
  final VoidCallback? onRefresh;

  const ChatMessageItem({
    super.key,
    required this.message,
    this.isLastMessage = false,
    this.onToggleReasoning,
    this.onCopy,
    this.onEdit,
    this.onLike,
    this.onDislike,
    this.onRefresh,
  });

  @override
  State<ChatMessageItem> createState() => _ChatMessageItemState();
}

class _ChatMessageItemState extends State<ChatMessageItem> {
  bool _isHovering = false;

  @override
  Widget build(BuildContext context) {
    if (widget.message.isUser) {
      return _buildUserMessage();
    } else {
      return _buildAgentMessage();
    }
  }

  /// 构建用户消息（右侧灰色气泡）
  Widget _buildUserMessage() {
    return MouseRegion(
      onEnter: (_) => setState(() => _isHovering = true),
      onExit: (_) => setState(() => _isHovering = false),
      child: Container(
        margin: const EdgeInsets.only(left: 60, right: 16, top: 8, bottom: 8),
        child: Row(
          mainAxisAlignment: MainAxisAlignment.end,
          crossAxisAlignment: CrossAxisAlignment.center,
          children: [
            // 操作按钮（气泡左侧，hover时显示）
            AnimatedOpacity(
              opacity: _isHovering ? 1.0 : 0.0,
              duration: const Duration(milliseconds: 200),
              child: Row(
                mainAxisSize: MainAxisSize.min,
                children: [
                  _buildIconButton(
                    Icons.copy,
                    () => _handleCopy(widget.message.content),
                  ),
                  const SizedBox(width: 4),
                  _buildIconButton(Icons.edit, widget.onEdit),
                ],
              ),
            ),
            const SizedBox(width: 8),
            // 消息气泡 - 使用 Flexible 防止溢出
            Flexible(
              child: Container(
                padding: const EdgeInsets.symmetric(
                  horizontal: 16,
                  vertical: 12,
                ),
                decoration: BoxDecoration(
                  color: ColorUtil.fromHex('#e6e8eb'),
                  borderRadius: BorderRadius.circular(12),
                ),
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  mainAxisSize: MainAxisSize.min,
                  children: [
                    if (widget.message.hasFile) ...[
                      _buildUserFileCard(),
                      const SizedBox(height: 8),
                    ],
                    Text(
                      widget.message.content,
                      style: const TextStyle(
                        fontSize: 14,
                        color: Colors.black87,
                        height: 1.5,
                      ),
                    ),
                  ],
                ),
              ),
            ),
          ],
        ),
      ),
    );
  }

  Widget _buildUserFileCard() {
    final filePath = widget.message.filePath!;
    final fileType = widget.message.fileType?.toUpperCase() ?? 'FILE';
    final fileName = p.basename(filePath);

    return Material(
      color: Colors.transparent,
      child: InkWell(
        onTap: () => _openFile(filePath),
        borderRadius: BorderRadius.circular(8),
        child: Container(
          constraints: const BoxConstraints(maxWidth: 320),
          padding: const EdgeInsets.all(10),
          decoration: BoxDecoration(
            color: Colors.white.withValues(alpha: 0.7),
            borderRadius: BorderRadius.circular(8),
            border: Border.all(color: ColorUtil.fromHex('#d4d8dd')),
          ),
          child: Row(
            mainAxisSize: MainAxisSize.min,
            children: [
              Container(
                width: 34,
                height: 34,
                decoration: BoxDecoration(
                  color: ColorUtil.fromHex('#f3f4f5'),
                  borderRadius: BorderRadius.circular(8),
                ),
                child: Icon(
                  Icons.insert_drive_file_outlined,
                  size: 18,
                  color: ColorUtil.fromHex('#5e6267'),
                ),
              ),
              const SizedBox(width: 8),
              Flexible(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  mainAxisSize: MainAxisSize.min,
                  children: [
                    Text(
                      fileName,
                      overflow: TextOverflow.ellipsis,
                      maxLines: 1,
                      style: const TextStyle(
                        fontSize: 13,
                        color: Colors.black87,
                        fontWeight: FontWeight.w500,
                      ),
                    ),
                    const SizedBox(height: 2),
                    Tooltip(
                      message: filePath,
                      child: Text(
                        '$fileType · $filePath',
                        overflow: TextOverflow.ellipsis,
                        maxLines: 1,
                        style: TextStyle(
                          fontSize: 11,
                          color: ColorUtil.fromHex('#8a8f94'),
                        ),
                      ),
                    ),
                  ],
                ),
              ),
              const SizedBox(width: 8),
              Icon(
                Icons.open_in_new,
                size: 14,
                color: ColorUtil.fromHex('#8a8f94'),
              ),
            ],
          ),
        ),
      ),
    );
  }

  Future<void> _openFile(String filePath) async {
    if (!await File(filePath).exists()) {
      _showToast('文件不存在');
      return;
    }

    final result = await OpenFilex.open(filePath);
    if (result.type != ResultType.done) {
      _showToast(result.message.isEmpty ? '打开文件失败' : result.message);
    }
  }

  /// 处理复制操作
  void _handleCopy(String text) {
    Clipboard.setData(ClipboardData(text: text));
    _showToast('已复制到剪切板');
    widget.onCopy?.call();
  }

  /// 显示轻提示
  void _showToast(String message) {
    final overlay = Overlay.of(context);
    final overlayEntry = OverlayEntry(
      builder:
          (context) => Positioned(
            top: MediaQuery.of(context).padding.top + 50,
            left: 0,
            right: 0,
            child: Center(
              child: Container(
                padding: const EdgeInsets.symmetric(
                  horizontal: 16,
                  vertical: 8,
                ),
                decoration: BoxDecoration(
                  color: Colors.black87,
                  borderRadius: BorderRadius.circular(20),
                ),
                child: Text(
                  message,
                  style: const TextStyle(color: Colors.white, fontSize: 14),
                ),
              ),
            ),
          ),
    );

    overlay.insert(overlayEntry);
    Future.delayed(const Duration(seconds: 1), () {
      overlayEntry.remove();
    });
  }

  /// 构建Agent消息（左侧，包含思考过程和回复内容）
  Widget _buildAgentMessage() {
    return SelectionArea(
      child: Container(
        margin: const EdgeInsets.only(left: 16, right: 60, top: 8, bottom: 8),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            // 思考过程折叠区域
            if (widget.message.reasoning != null) _buildReasoningSection(),
            const SizedBox(height: 8),
            // Agent回复内容 - 使用 Markdown 渲染
            Container(
              padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 12),
              decoration: BoxDecoration(
                color: Colors.transparent,
                borderRadius: BorderRadius.circular(12),
              ),
              child:
                  widget.message.isError
                      ? _buildErrorContent()
                      : MarkdownBody(
                        data: widget.message.content,
                        selectable: false,
                        styleSheet: MarkdownStyleSheet(
                          p: const TextStyle(
                            fontSize: 14,
                            color: Colors.black87,
                            height: 1.6,
                          ),
                          h1: const TextStyle(
                            fontSize: 20,
                            fontWeight: FontWeight.bold,
                            color: Colors.black87,
                          ),
                          h2: const TextStyle(
                            fontSize: 18,
                            fontWeight: FontWeight.bold,
                            color: Colors.black87,
                          ),
                          h3: const TextStyle(
                            fontSize: 16,
                            fontWeight: FontWeight.bold,
                            color: Colors.black87,
                          ),
                          code: TextStyle(
                            fontSize: 13,
                            color: Colors.pink[700],
                            backgroundColor: Colors.grey[200],
                            fontFamily: 'monospace',
                          ),
                          codeblockDecoration: BoxDecoration(
                            color: Colors.grey[100],
                            borderRadius: BorderRadius.circular(8),
                          ),
                          blockquote: const TextStyle(
                            fontSize: 14,
                            color: Colors.black54,
                            fontStyle: FontStyle.italic,
                          ),
                          blockquoteDecoration: BoxDecoration(
                            border: Border(
                              left: BorderSide(
                                color: Colors.grey[400]!,
                                width: 4,
                              ),
                            ),
                          ),
                          listBullet: const TextStyle(
                            fontSize: 14,
                            color: Colors.black87,
                          ),
                        ),
                      ),
            ),
            // 操作按钮栏 - 只有最后一条Agent消息且回复已完成才显示
            if (widget.message.isAgent &&
                widget.isLastMessage &&
                !widget.message.isReasoning)
              _buildActionBar(),
          ],
        ),
      ),
    );
  }

  Widget _buildErrorContent() {
    return Row(
      mainAxisSize: MainAxisSize.min,
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        const Icon(Icons.error_outline, size: 18, color: Color(0xFFE53935)),
        const SizedBox(width: 6),
        Flexible(
          child: Text(
            widget.message.content,
            style: const TextStyle(
              fontSize: 14,
              color: Color(0xFFE53935),
              height: 1.6,
            ),
          ),
        ),
      ],
    );
  }

  /// 构建思考过程区域
  /// 思考中：显示"思考中..."标签 + 思考内容
  /// 思考完成：只显示"已思考"标签，隐藏思考内容
  Widget _buildReasoningSection() {
    final bool isReasoning = widget.message.isReasoning;
    // 只在思考中显示内容，思考完成后隐藏
    final bool showContent = isReasoning;

    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        // 标签（思考中/已完成）-  pill 样式
        Container(
          padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 5),
          decoration: BoxDecoration(
            color:
                isReasoning
                    ? const Color(0xFFEBF5FF) // 思考中：淡蓝色背景
                    : const Color(0xFFF5F5F5), // 已完成：浅灰色背景
            borderRadius: BorderRadius.circular(20),
            border: Border.all(
              color:
                  isReasoning
                      ? const Color(0xFF1890FF).withValues(
                        alpha: 0.3,
                      ) // 思考中：蓝色边框
                      : const Color(0xFFD9D9D9), // 已完成：灰色边框
              width: 1,
            ),
          ),
          child: Row(
            mainAxisSize: MainAxisSize.min,
            children: [
              if (isReasoning) ...[
                // 思考中：显示脉冲动画点
                _buildPulsingDot(),
                const SizedBox(width: 6),
                Text(
                  '思考中',
                  style: TextStyle(
                    fontSize: 12,
                    color: Colors.blue[600],
                    fontWeight: FontWeight.w500,
                  ),
                ),
              ] else ...[
                // 思考完成：显示勾选图标
                Container(
                  width: 16,
                  height: 16,
                  decoration: BoxDecoration(
                    color: const Color(0xFF10B981).withValues(alpha: 0.1),
                    shape: BoxShape.circle,
                  ),
                  child: Icon(
                    Icons.check,
                    size: 10,
                    color: const Color(0xFF10B981),
                  ),
                ),
                const SizedBox(width: 6),
                Text(
                  '已完成思考',
                  style: TextStyle(
                    fontSize: 12,
                    color: Colors.grey[600],
                    fontWeight: FontWeight.w500,
                  ),
                ),
              ],
            ],
          ),
        ),
        // 思考过程内容
        if (showContent &&
            widget.message.reasoning != null &&
            widget.message.reasoning!.isNotEmpty)
          AnimatedContainer(
            duration: const Duration(milliseconds: 300),
            margin: const EdgeInsets.only(top: 8),
            height: isReasoning ? 60 : null,
            // 思考中固定高度，完成后自适应
            constraints:
                isReasoning ? null : const BoxConstraints(maxHeight: 200),
            // 完成后最大高度200
            padding: const EdgeInsets.all(12),
            decoration: BoxDecoration(
              color:
                  isReasoning
                      ? const Color(0xFFFAFBFC) // 思考中：更淡的背景
                      : const Color(0xFFF5F5F5), // 已完成：灰色背景
              borderRadius: BorderRadius.circular(8),
              border: Border.all(
                color:
                    isReasoning
                        ? const Color(0xFFE8E8E8)
                        : const Color(0xFFE0E0E0),
              ),
            ),
            child:
                isReasoning
                    ? _ReasoningDisplay(
                      text: widget.message.reasoning!,
                      style: TextStyle(
                        fontSize: 12,
                        color: Colors.grey[600],
                        height: 1.5,
                      ),
                    )
                    : SingleChildScrollView(
                      child: Text(
                        widget.message.reasoning!,
                        style: TextStyle(
                          fontSize: 12,
                          color: Colors.grey[600],
                          height: 1.6,
                        ),
                      ),
                    ),
          ),
      ],
    );
  }

  /// 构建脉冲动画点（思考中指示器）
  Widget _buildPulsingDot() {
    return SizedBox(
      width: 12,
      height: 12,
      child: Center(
        child: Container(
          width: 8,
          height: 8,
          decoration: BoxDecoration(
            color: Colors.blue[500],
            shape: BoxShape.circle,
          ),
        ),
      ),
    );
  }

  /// 构建操作按钮栏
  Widget _buildActionBar() {
    return Container(
      margin: const EdgeInsets.only(top: 8),
      child: Row(
        children: [
          _buildActionButton(
            Icons.copy,
            () => _handleCopy(widget.message.content),
          ),
          _buildActionButton(Icons.thumb_up_outlined, widget.onLike),
          _buildActionButton(Icons.thumb_down_outlined, widget.onDislike),
          _buildActionButton(Icons.refresh, widget.onRefresh),
          // _buildActionButton(Icons.volume_up_outlined, null),
        ],
      ),
    );
  }

  /// 构建操作按钮
  Widget _buildActionButton(IconData icon, VoidCallback? onTap) {
    return Material(
      color: Colors.transparent,
      child: InkWell(
        onTap: onTap,
        borderRadius: BorderRadius.circular(16),
        child: Container(
          padding: const EdgeInsets.all(8),
          child: Icon(icon, size: 16, color: Colors.grey[500]),
        ),
      ),
    );
  }

  /// 构建小图标按钮
  Widget _buildIconButton(IconData icon, VoidCallback? onTap) {
    return Material(
      color: Colors.transparent,
      child: InkWell(
        onTap: onTap,
        borderRadius: BorderRadius.circular(12),
        child: Container(
          padding: const EdgeInsets.all(4),
          child: Icon(icon, size: 14, color: Colors.grey[400]),
        ),
      ),
    );
  }
}

/// 思考过程显示组件
/// 固定高度，自动滚动到底部显示最新内容
class _ReasoningDisplay extends StatefulWidget {
  final String text;
  final TextStyle style;

  const _ReasoningDisplay({required this.text, required this.style});

  @override
  State<_ReasoningDisplay> createState() => _ReasoningDisplayState();
}

class _ReasoningDisplayState extends State<_ReasoningDisplay> {
  final ScrollController _scrollController = ScrollController();

  @override
  void didUpdateWidget(_ReasoningDisplay oldWidget) {
    super.didUpdateWidget(oldWidget);
    // 文字更新时自动滚动到底部
    if (oldWidget.text != widget.text &&
        widget.text.length > oldWidget.text.length) {
      WidgetsBinding.instance.addPostFrameCallback((_) {
        if (_scrollController.hasClients) {
          _scrollController.animateTo(
            _scrollController.position.maxScrollExtent,
            duration: const Duration(milliseconds: 200),
            curve: Curves.easeOut,
          );
        }
      });
    }
  }

  @override
  void dispose() {
    _scrollController.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return SingleChildScrollView(
      controller: _scrollController,
      physics: const ClampingScrollPhysics(),
      child: Text(widget.text, style: widget.style),
    );
  }
}
