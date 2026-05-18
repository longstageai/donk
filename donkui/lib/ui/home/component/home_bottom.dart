import 'package:donk/common/util/color_util.dart';
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:get/get.dart';

import '../home_controller.dart';

class HomeBottom extends StatefulWidget {
  const HomeBottom({super.key});

  @override
  State<HomeBottom> createState() => _HomeBottomState();
}

class _HomeBottomState extends State<HomeBottom> {
  final FocusNode _focusNode = FocusNode();
  final controller = Get.find<HomeController>();
  final TextEditingController _textController = TextEditingController();
  Worker? _editTextWorker;

  @override
  void initState() {
    super.initState();

    // 监听编辑文本变化
    _editTextWorker = ever(controller.pendingEditText, (String text) {
      if (text.isNotEmpty) {
        _textController.text = text;
        _textController.selection = TextSelection.fromPosition(
          TextPosition(offset: text.length),
        );
        _focusNode.requestFocus();
        controller.clearEditText();
      }
    });

    // 检查初始值（如果监听设置前已经有值，立即处理）
    if (controller.pendingEditText.isNotEmpty) {
      final text = controller.pendingEditText.value;
      _textController.text = text;
      _textController.selection = TextSelection.fromPosition(
        TextPosition(offset: text.length),
      );
      _focusNode.requestFocus();
      controller.clearEditText();
    }
  }

  @override
  void dispose() {
    _editTextWorker?.dispose();
    _focusNode.dispose();
    _textController.dispose();
    super.dispose();
  }

  /// 发送消息
  Future<void> _sendMessage() async {
    // 检查微信是否已连接，如果已连接则禁止发送
    if (controller.isWeChatConnected) {
      return;
    }

    final text = _textController.text.trim();
    if (text.isEmpty) return;

    // 检查是否正在处理中，如果是则禁止发送
    if (controller.isProcessing) {
      return;
    }

    // 先清空输入框，避免重复发送
    _textController.clear();

    await controller.addUserMessage(text);
  }

  Widget header(BuildContext context) {
    return Container(
      height: 40,
      alignment: Alignment.centerLeft,
      child: SizedBox(
        width: 100,
        child: Row(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.center,
          mainAxisAlignment: MainAxisAlignment.spaceAround,
          children: [
            IconButton(
              onPressed: () {},
              icon: Icon(Icons.receipt_long, size: 16),
              padding: EdgeInsets.zero,
              constraints: BoxConstraints(),
            ),
            TextButton(
              onPressed: () {},
              style: TextButton.styleFrom(
                padding: EdgeInsets.zero,
                minimumSize: Size(0, 0),
              ),
              child: Text(
                "@Builder",
                style: TextStyle(
                  fontSize: 12,
                  color: ColorUtil.fromHex("#5e6267"),
                ),
              ),
            ),
            IconButton(
              onPressed: () {},
              icon: Icon(Icons.settings_suggest_outlined, size: 16),
              padding: EdgeInsets.zero,
              constraints: BoxConstraints(),
            ),
          ],
        ),
      ),
    );
  }

  Widget bottomLeft() {
    return SizedBox(
      width: 50,
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.center,
        mainAxisAlignment: MainAxisAlignment.spaceAround,
        children: [
          TextButton(
            onPressed: () {},
            style: TextButton.styleFrom(
              padding: EdgeInsets.zero,
              minimumSize: Size(0, 0),
            ),
            child: Text(
              "@",
              style: TextStyle(
                color: ColorUtil.fromHex("#252729"),
                fontSize: 16,
              ),
            ),
          ),
          TextButton(
            onPressed: () {},
            style: TextButton.styleFrom(
              padding: EdgeInsets.zero,
              minimumSize: Size(0, 0),
            ),
            child: Text(
              "#",
              style: TextStyle(
                color: ColorUtil.fromHex("#252729"),
                fontSize: 16,
              ),
            ),
          ),
        ],
      ),
    );
  }

  Widget bottomRight() {
    return Obx(() {
      final isProcessing = controller.isProcessing;
      return SizedBox(
        width: 280,
        child: Row(
          crossAxisAlignment: CrossAxisAlignment.center,
          mainAxisAlignment: MainAxisAlignment.end,
          children: [
            // 发送/取消按钮
            Container(
              width: 28,
              height: 28,
              decoration: BoxDecoration(
                color: ColorUtil.fromHex("#0fdc78"),
                borderRadius: BorderRadius.circular(6),
              ),
              child:
                  isProcessing
                      ? IconButton(
                        onPressed: () => _cancelRequest(),
                        icon: const Icon(
                          Icons.stop_circle_outlined,
                          size: 18,
                          color: Colors.black,
                        ),
                        padding: EdgeInsets.zero,
                        constraints: const BoxConstraints(),
                      )
                      : Transform.rotate(
                        angle: 1.5708, // -90度（弧度制）
                        child: IconButton(
                          onPressed: () => _sendMessage(),
                          icon: const Icon(
                            Icons.arrow_back_outlined,
                            size: 16,
                            color: Colors.black,
                          ),
                          padding: EdgeInsets.zero,
                          constraints: const BoxConstraints(),
                        ),
                      ),
            ),
            SizedBox(width: 10),
          ],
        ),
      );
    });
  }

  /// 取消请求
  void _cancelRequest() {
    controller.cancelRequest();
  }

  Widget bottom(BuildContext context) {
    return Obx(() {
      final isWeChatConnected = controller.isWeChatConnected;

      return Container(
        width: double.infinity,
        height: double.infinity,
        decoration: BoxDecoration(
          color: ColorUtil.fromHex("#f3f4f5"),
          borderRadius: BorderRadius.only(
            topLeft: Radius.circular(5),
            topRight: Radius.circular(5),
          ),
          border: Border.all(
            color:
                _focusNode.hasFocus
                    ? ColorUtil.fromHex("#b7c2d3")
                    : Colors.transparent,
            width: 1,
          ),
        ),
        child: Column(
          children: [
            // 微信已连接提示
            if (isWeChatConnected)
              Container(
                width: double.infinity,
                padding: const EdgeInsets.symmetric(
                  horizontal: 12,
                  vertical: 8,
                ),
                color: ColorUtil.fromHex("#0fdc78").withValues(alpha: 0.1),
                child: Row(
                  children: [
                    Icon(
                      Icons.check_circle,
                      color: ColorUtil.fromHex("#0fdc78"),
                      size: 16,
                    ),
                    const SizedBox(width: 8),
                    Text(
                      '微信已连接，正在处理微信消息',
                      style: TextStyle(
                        color: ColorUtil.fromHex("#0fdc78"),
                        fontSize: 12,
                      ),
                    ),
                  ],
                ),
              ),
            Expanded(
              child: _CustomTextField(
                focusNode: _focusNode,
                controller: _textController,
                onSubmitted: () => _sendMessage(),
                enabled: !isWeChatConnected,
                hintText: isWeChatConnected ? '微信已连接，请在微信中发送消息' : '输入消息...',
              ),
            ),
            SizedBox(
              height: 40,
              child: Row(
                mainAxisAlignment: MainAxisAlignment.spaceBetween,
                children: [bottomLeft(), bottomRight()],
              ),
            ),
          ],
        ),
      );
    });
  }

  @override
  Widget build(BuildContext context) {
    return Container(
      width: double.infinity,
      height: 150,
      clipBehavior: Clip.hardEdge,
      margin: EdgeInsets.only(top: 2),
      decoration: BoxDecoration(
        color: ColorUtil.fromHex("#dfe2e5"),
        borderRadius: BorderRadius.all(Radius.circular(5)),
        border: Border.all(color: ColorUtil.fromHex("#dfe2e5"), width: 1),
        boxShadow: [
          BoxShadow(
            color: Colors.black.withAlpha(100),
            spreadRadius: 0,
            blurRadius: 4,
            offset: Offset(0, 2),
          ),
        ],
      ),
      child: Column(
        children: [header(context), Expanded(child: bottom(context))],
      ),
    );
  }
}

class _CustomTextField extends StatelessWidget {
  final FocusNode focusNode;
  final TextEditingController controller;
  final VoidCallback onSubmitted;
  final bool enabled;
  final String hintText;

  const _CustomTextField({
    required this.focusNode,
    required this.controller,
    required this.onSubmitted,
    this.enabled = true,
    this.hintText = '输入消息...',
  });

  @override
  Widget build(BuildContext context) {
    return Focus(
      focusNode: focusNode,
      onKeyEvent: (FocusNode node, KeyEvent event) {
        // 如果禁用，不处理键盘事件
        if (!enabled) return KeyEventResult.ignored;

        // 监听回车键按下事件
        if (event is KeyDownEvent &&
            event.logicalKey == LogicalKeyboardKey.enter) {
          // 如果按下了 Shift 键，允许换行（不拦截）
          if (HardwareKeyboard.instance.isShiftPressed) {
            return KeyEventResult.ignored;
          }
          // 普通回车：发送消息
          onSubmitted();
          // 返回 KeyEventResult.handled 阻止事件继续传递（阻止换行）
          return KeyEventResult.handled;
        }
        // 其他按键正常处理
        return KeyEventResult.ignored;
      },
      child: TextField(
        controller: controller,
        enabled: enabled,
        maxLines: null,
        expands: true,
        textAlignVertical: TextAlignVertical.top,
        cursorColor: ColorUtil.fromHex("#00D9A5"),
        cursorWidth: 2,
        cursorHeight: 12,
        cursorRadius: Radius.circular(5),
        decoration: InputDecoration(
          hintText: hintText,
          hintStyle: TextStyle(
            fontSize: 12,
            color: ColorUtil.fromHex("#b6babd"),
          ),
          border: InputBorder.none,
          enabledBorder: InputBorder.none,
          focusedBorder: InputBorder.none,
          contentPadding: const EdgeInsets.only(
            left: 12,
            right: 12,
            top: 8,
            bottom: 8,
          ),
        ),
      ),
    );
  }
}
