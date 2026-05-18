import 'package:donk/app/conf/config.dart';
import 'package:flutter/material.dart';
import 'package:get/get.dart';
import 'package:window_manager/window_manager.dart';

import '../conf/colors.dart';
import 'layout_controller.dart';

class RunningDonkeyPainter extends CustomPainter {
  final double progress;

  RunningDonkeyPainter(this.progress) : super(repaint: null);

  static final Paint _fillPaint =
      Paint()
        ..color = Colors.brown
        ..style = PaintingStyle.fill;

  static final Paint _outlinePaint =
      Paint()
        ..color = Colors.black87
        ..style = PaintingStyle.stroke
        ..strokeWidth = 1.5;

  static final Paint _eyePaint = Paint()..color = Colors.black;

  static final Paint _manePaint =
      Paint()
        ..color = Colors.black54
        ..style = PaintingStyle.fill;

  @override
  void paint(Canvas canvas, Size size) {
    double x = progress * size.width;
    double y = size.height * 0.5;
    _drawDonkey(canvas, x, y, progress);
  }

  void _drawDonkey(Canvas canvas, double x, double y, double animProgress) {
    const double scale = 0.9;
    double legOffset = (animProgress * 64) % 4;
    int legPhase = legOffset.floor();
    double legAngle = (legOffset - legPhase) * 0.3;

    double baseLegY = y + 7.2;
    double legMove = legAngle * 6;

    double frontLeg1Y =
        baseLegY + (legPhase == 0 || legPhase == 2 ? legMove : -legMove * 0.5);
    double frontLeg2Y =
        baseLegY + (legPhase == 1 || legPhase == 3 ? legMove : -legMove * 0.5);
    double backLeg1Y =
        baseLegY + (legPhase == 1 || legPhase == 3 ? -legMove * 0.5 : legMove);
    double backLeg2Y =
        baseLegY + (legPhase == 0 || legPhase == 2 ? -legMove * 0.5 : legMove);

    final bodyPath =
        Path()
          ..moveTo(x + 3, y)
          ..lineTo(x + 21, y)
          ..lineTo(x + 22.8, y - 3)
          ..lineTo(x + 21, y + 4.8)
          ..lineTo(x + 3, y + 4.8)
          ..close();

    canvas.drawPath(bodyPath, _fillPaint);
    canvas.drawPath(bodyPath, _outlinePaint);

    final neckPath =
        Path()
          ..moveTo(x + 18, y)
          ..lineTo(x + 22.8, y - 7.2)
          ..lineTo(x + 25.2, y - 6)
          ..lineTo(x + 21, y + 1.2)
          ..close();

    canvas.drawPath(neckPath, _fillPaint);
    canvas.drawPath(neckPath, _outlinePaint);

    final headPath =
        Path()
          ..moveTo(x + 22.8, y - 7.2)
          ..lineTo(x + 28.8, y - 4.8)
          ..lineTo(x + 27.6, y - 1.2)
          ..lineTo(x + 25.2, y - 2.4)
          ..close();

    canvas.drawPath(headPath, _fillPaint);
    canvas.drawPath(headPath, _outlinePaint);

    final earPath =
        Path()
          ..moveTo(x + 24, y - 7.2)
          ..lineTo(x + 25.2, y - 10.8)
          ..lineTo(x + 26.4, y - 6.6)
          ..close();

    canvas.drawPath(earPath, _fillPaint);
    canvas.drawPath(earPath, _outlinePaint);

    canvas.drawCircle(Offset(x + 27, y - 5.4), 0.9, _eyePaint);

    final tailPath =
        Path()
          ..moveTo(x + 3, y + 1.2)
          ..quadraticBezierTo(x - 1.8, y - 1.2, x - 3, y + 3);

    canvas.drawPath(tailPath, _outlinePaint);

    _drawLeg(canvas, x + 4.8, y + 4.8, backLeg1Y, scale);
    _drawLeg(canvas, x + 7.2, y + 4.8, backLeg2Y, scale);
    _drawLeg(canvas, x + 16.8, y + 4.8, frontLeg1Y, scale);
    _drawLeg(canvas, x + 19.2, y + 4.8, frontLeg2Y, scale);

    final manePath =
        Path()
          ..moveTo(x + 21.6, y - 4.8)
          ..lineTo(x + 20.4, y - 8.4)
          ..lineTo(x + 22.8, y - 6)
          ..lineTo(x + 24, y - 9)
          ..lineTo(x + 24, y - 4.8);

    canvas.drawPath(manePath, _manePaint);
  }

  void _drawLeg(
    Canvas canvas,
    double x,
    double yTop,
    double yBottom,
    double scale,
  ) {
    final legPath =
        Path()
          ..moveTo(x - 1.2, yTop)
          ..lineTo(x + 1.2, yTop)
          ..lineTo(x + 0.9, yBottom)
          ..lineTo(x - 0.9, yBottom)
          ..close();

    canvas.drawPath(legPath, _fillPaint);
    canvas.drawPath(legPath, _outlinePaint);
  }

  @override
  bool shouldRepaint(covariant RunningDonkeyPainter oldDelegate) {
    return oldDelegate.progress != progress;
  }
}

class DonkAppBar extends StatefulWidget implements PreferredSizeWidget {
  final double height;
  final WindowManager windowManager;

  const DonkAppBar({super.key, this.height = 40, required this.windowManager});

  @override
  State<DonkAppBar> createState() => _DonkAppBarState();

  @override
  Size get preferredSize => Size.fromHeight(height);
}

class _DonkAppBarState extends State<DonkAppBar>
    with SingleTickerProviderStateMixin {
  bool _isMax = false;
  final LayoutController layoutController = Get.find<LayoutController>();
  late AnimationController _animationController;
  late Animation<double> _animation;

  @override
  void initState() {
    super.initState();
    _refreshMaxState();
    _animationController = AnimationController(
      vsync: this,
      duration: const Duration(seconds: 60),
    );
    _animation = Tween<double>(begin: -0.15, end: 1.15).animate(
      CurvedAnimation(parent: _animationController, curve: Curves.linear),
    );
    _animationController.repeat();
  }

  @override
  void dispose() {
    _animationController.dispose();
    super.dispose();
  }

  Future<void> _refreshMaxState() async {
    _isMax = await windowManager.isMaximized();
    if (mounted) setState(() {});
  }

  Widget menu() {
    return Container(
      padding: EdgeInsets.only(left: 10),
      child: Row(
        children: [
          // Image.asset(logo, width: 20),
          Text(
            name2,
            style: TextStyle(
              fontSize: 14,
              fontWeight: FontWeight.bold,
              color: Colors.black87,
            ),
          ),
        ],
      ),
    );
  }

  Widget right() {
    return Row(
      crossAxisAlignment: CrossAxisAlignment.center,
      mainAxisAlignment: MainAxisAlignment.end,
      children: [control()],
    );
  }

  Widget control() {
    final buttonStyle = IconButton.styleFrom(
      hoverColor: Colors.black26,
      highlightColor: Colors.black26,
      iconSize: 16,
      shape: const RoundedRectangleBorder(borderRadius: BorderRadius.zero),
    );
    final buttonStyle2 = IconButton.styleFrom(
      hoverColor: Colors.redAccent,
      highlightColor: Colors.white10,
      iconSize: 16,
      shape: const RoundedRectangleBorder(borderRadius: BorderRadius.zero),
    );
    return Row(
      crossAxisAlignment: CrossAxisAlignment.center,
      mainAxisAlignment: MainAxisAlignment.end,
      children: [
        IconButton(
          onPressed: windowManager.minimize,
          style: buttonStyle,
          icon: const Icon(Icons.remove, color: Colors.black87, size: 16),
          tooltip: '最小化',
        ),
        IconButton(
          onPressed: () async {
            if (await windowManager.isMaximized()) {
              await windowManager.unmaximize();
            } else {
              await windowManager.maximize();
            }
            await _refreshMaxState();
          },
          style: buttonStyle,
          icon: Icon(
            _isMax ? Icons.filter_none : Icons.crop_square,
            color: Colors.black87,
            size: 16,
          ),
          tooltip: _isMax ? '还原' : '最大化',
        ),
        IconButton(
          onPressed: windowManager.close,
          style: buttonStyle2,
          icon: const Icon(Icons.close, color: Colors.black87, size: 16),
          tooltip: '关闭',
        ),
      ],
    );
  }

  @override
  Widget build(BuildContext context) {
    return SafeArea(
      bottom: false,
      child: GestureDetector(
        behavior: HitTestBehavior.translucent,
        onPanStart: (details) {
          windowManager.startDragging();
        },
        child: RepaintBoundary(
          child: AnimatedBuilder(
            animation: _animation,
            builder: (context, child) {
              return Container(
                decoration: BoxDecoration(
                  borderRadius: BorderRadius.only(
                    topLeft: Radius.circular(circular),
                    topRight: Radius.circular(circular),
                  ),
                  color: AppColors.backgroundColor,
                ),
                child: Stack(
                  children: [
                    CustomPaint(
                      size: Size.infinite,
                      painter: RunningDonkeyPainter(_animation.value),
                    ),
                    Row(children: [menu(), Expanded(child: right())]),
                  ],
                ),
              );
            },
          ),
        ),
      ),
    );
  }
}
