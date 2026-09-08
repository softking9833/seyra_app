import 'package:flutter/material.dart';
import 'package:seyra/core/theme/app_colors.dart';

/// Soft corner blobs and bokeh used on auth screens.
class AuthLandingBackdrop extends StatelessWidget {
  const AuthLandingBackdrop({super.key});

  @override
  Widget build(BuildContext context) {
    return const IgnorePointer(
      child: CustomPaint(
        painter: AuthLandingBackdropPainter(),
        child: SizedBox.expand(),
      ),
    );
  }
}

class AuthLandingBackdropPainter extends CustomPainter {
  const AuthLandingBackdropPainter();

  @override
  void paint(Canvas canvas, Size size) {
    _paintBlob(
      canvas,
      rect: Rect.fromLTWH(
        -size.width * 0.28,
        -size.height * 0.16,
        size.width * 0.92,
        size.height * 0.38,
      ),
      colors: const [AppColors.blobCyan, AppColors.blobBlue],
    );
    _paintBlob(
      canvas,
      rect: Rect.fromLTWH(
        -size.width * 0.18,
        -size.height * 0.02,
        size.width * 0.62,
        size.height * 0.22,
      ),
      colors: const [Color(0xAA9BE8FF), Color(0xCC4F86F5)],
    );

    _paintBlob(
      canvas,
      rect: Rect.fromLTWH(
        size.width * 0.42,
        size.height * 0.72,
        size.width * 0.78,
        size.height * 0.38,
      ),
      colors: const [AppColors.blobBlue, AppColors.royal],
    );
    _paintBlob(
      canvas,
      rect: Rect.fromLTWH(
        size.width * 0.58,
        size.height * 0.78,
        size.width * 0.62,
        size.height * 0.28,
      ),
      colors: const [Color(0xCC5BA0FF), Color(0xFF2454D6)],
    );

    final bokeh = Paint()..color = const Color(0x3348B7FF);
    canvas.drawCircle(
      Offset(size.width * 0.86, size.height * 0.30),
      size.width * 0.16,
      bokeh,
    );
    canvas.drawCircle(
      Offset(size.width * 0.12, size.height * 0.78),
      size.width * 0.14,
      bokeh,
    );
  }

  void _paintBlob(
    Canvas canvas, {
    required Rect rect,
    required List<Color> colors,
  }) {
    final paint = Paint()
      ..shader = LinearGradient(
        begin: Alignment.topLeft,
        end: Alignment.bottomRight,
        colors: colors,
      ).createShader(rect);
    canvas.drawOval(rect, paint);
  }

  @override
  bool shouldRepaint(covariant CustomPainter oldDelegate) => false;
}
