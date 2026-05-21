import 'package:flutter/material.dart';

class AnimatedAvatarBorder extends StatefulWidget {
  const AnimatedAvatarBorder({super.key});

  @override
  State<AnimatedAvatarBorder> createState() => _AnimatedAvatarBorderState();
}

class _AnimatedAvatarBorderState extends State<AnimatedAvatarBorder>
    with SingleTickerProviderStateMixin {
  late final AnimationController _rotationController;
  late final Animation<double> _rotationAnimation;

  @override
  void initState() {
    super.initState();
    _rotationController = AnimationController(
      duration: const Duration(seconds: 3),
      vsync: this,
    );
    _rotationAnimation = Tween<double>(
      begin: 0,
      end: 2 * 3.141592653589793,
    ).animate(_rotationController);
    _rotationController.repeat();
  }

  @override
  void dispose() {
    _rotationController.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return AnimatedBuilder(
      animation: _rotationAnimation,
      builder: (context, child) {
        return Transform.rotate(
          angle: _rotationAnimation.value,
          child: const CustomPaint(
            size: Size.square(16),
            painter: _AnimatedAvatarBorderPainter(),
          ),
        );
      },
    );
  }
}

class _AnimatedAvatarBorderPainter extends CustomPainter {
  const _AnimatedAvatarBorderPainter();

  static const _gradient = SweepGradient(
    colors: [
      Color(0xFFC8E6C9),
      Color(0xFFA5D6A7),
      Color(0xFF81C784),
      Color(0xFF66BB6A),
      Color(0xFF4CAF50),
      Color(0xFF388E3C),
      Color(0xFFC8E6C9),
    ],
    stops: [0.0, 0.15, 0.3, 0.45, 0.6, 0.75, 1.0],
  );

  @override
  void paint(Canvas canvas, Size size) {
    const strokeWidth = 2.0;
    final rect = Offset.zero & size;
    final paint =
        Paint()
          ..shader = _gradient.createShader(rect)
          ..style = PaintingStyle.stroke
          ..strokeWidth = strokeWidth;
    canvas.drawCircle(
      size.center(Offset.zero),
      (size.width - strokeWidth) / 2,
      paint,
    );
  }

  @override
  bool shouldRepaint(covariant CustomPainter oldDelegate) => false;
}
