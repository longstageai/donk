import 'package:donk/app/conf/colors.dart';
import 'package:flutter/material.dart';

class AppButton extends StatefulWidget {
  final IconData icon;
  final String label;
  final Function()? onTap;

  const AppButton({
    super.key,
    required this.icon,
    required this.label,
    this.onTap,
  });

  @override
  State<AppButton> createState() => _AppButtonState();
}

class _AppButtonState extends State<AppButton> {
  bool _isHovered = false;

  @override
  Widget build(BuildContext context) {
    return MouseRegion(
      cursor: SystemMouseCursors.click,
      onEnter: (_) => setState(() => _isHovered = true),
      onExit: (_) => setState(() => _isHovered = false),
      child: InkWell(
        onTap: widget.onTap,
        borderRadius: BorderRadius.circular(10),
        hoverColor: const Color(0xFFE0E0E0),
        highlightColor: const Color(0xFFE0E0E0),
        splashColor: Colors.transparent,
        child: Container(
          padding: EdgeInsets.all(8),
          decoration: BoxDecoration(
            color: _isHovered ? const Color(0xFFE0E0E0) : Colors.transparent,
            borderRadius: BorderRadius.circular(10),
          ),
          child: Row(
            mainAxisAlignment: MainAxisAlignment.center,
            children: [
              Icon(widget.icon, size: 20, color: AppColors.c1),
              if (widget.label.isNotEmpty) ...[
                const SizedBox(width: 2),
                Text(
                  widget.label,
                  style: TextStyle(
                    fontSize: 14,
                    color: AppColors.c1,
                    fontWeight: FontWeight.w600,
                  ),
                ),
              ],
            ],
          ),
        ),
      ),
    );
  }
}
