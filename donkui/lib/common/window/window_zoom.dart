import 'package:flutter/material.dart';
import 'package:window_manager/window_manager.dart';

import 'window_util.dart';

/// 在窗口四边与四个角添加“热区”，按下即可发起对应方向的缩放。
class WindowZoom extends StatelessWidget {
  const WindowZoom({super.key, this.thickness = 6});

  final double thickness;

  Widget _edge(
    Alignment alignment,
    VoidCallback onPointerDown, {
    double? width,
    double? height,
    MouseCursor? cursor,
  }) {
    return Align(
      alignment: alignment,
      child: Listener(
        onPointerDown: (_) => onPointerDown(),
        child: MouseRegion(
          cursor: cursor ?? SystemMouseCursors.basic,
          child: SizedBox(
            width: width,
            height: height,
            // 用透明容器占位成为“热区”
            child: Container(color: Colors.transparent),
          ),
        ),
      ),
    );
  }

  @override
  Widget build(BuildContext context) {
    final corner = thickness * 2;

    return IgnorePointer(
      ignoring: false, // 要响应鼠标
      child: Stack(
        children: [
          // 四边
          _edge(
            Alignment.topCenter,
            () => WindowUtil.startResizing(ResizeEdge.top),
            height: thickness,
            cursor: SystemMouseCursors.resizeUp,
          ),
          _edge(
            Alignment.bottomCenter,
            () => WindowUtil.startResizing(ResizeEdge.bottom),
            height: thickness,
            cursor: SystemMouseCursors.resizeDown,
          ),
          _edge(
            Alignment.centerLeft,
            () => WindowUtil.startResizing(ResizeEdge.left),
            width: thickness,
            cursor: SystemMouseCursors.resizeLeft,
          ),
          _edge(
            Alignment.centerRight,
            () => WindowUtil.startResizing(ResizeEdge.right),
            width: thickness,
            cursor: SystemMouseCursors.resizeRight,
          ),

          // 四角
          _edge(
            Alignment.topLeft,
            () => WindowUtil.startResizing(ResizeEdge.topLeft),
            width: corner,
            height: corner,
            cursor: SystemMouseCursors.resizeUpLeft,
          ),
          _edge(
            Alignment.topRight,
            () => WindowUtil.startResizing(ResizeEdge.topRight),
            width: corner,
            height: corner,
            cursor: SystemMouseCursors.resizeUpRight,
          ),
          _edge(
            Alignment.bottomLeft,
            () => WindowUtil.startResizing(ResizeEdge.bottomLeft),
            width: corner,
            height: corner,
            cursor: SystemMouseCursors.resizeDownLeft,
          ),
          _edge(
            Alignment.bottomRight,
            () => WindowUtil.startResizing(ResizeEdge.bottomRight),
            width: corner,
            height: corner,
            cursor: SystemMouseCursors.resizeDownRight,
          ),
        ],
      ),
    );
  }
}
