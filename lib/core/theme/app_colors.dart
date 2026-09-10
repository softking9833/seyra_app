import 'package:flutter/material.dart';

/// Light brand tokens stay Seyra blue. Dark chat: background `#0F111A`,
/// surface `#1A1D23`, accent `#5D5FEF`.
abstract final class AppColors {
  static const Color primary = Color(0xFF2F6FED);
  static const Color primaryDark = Color(0xFF1E4FC0);
  static const Color cyan = Color(0xFF3EC6F5);
  static const Color royal = Color(0xFF2F62F0);
  static const Color navy = Color(0xFF123A7A);
  static const Color blobCyan = Color(0xFF7EDDFF);
  static const Color blobBlue = Color(0xFF4C8CFF);
  static const Color surface = Color(0xFFFFFFFF);
  static const Color surfaceMuted = Color(0xFFF7F9FC);
  static const Color textPrimary = Color(0xFF111827);
  static const Color textSecondary = Color(0xFF6B7280);
  static const Color fieldBorder = Color(0xFFE5E7EB);
  static const Color wave = Color(0xFFDCE8FF);

  static const Color darkBg = Color(0xFF0F111A);
  static const Color darkSurface = Color(0xFF1A1D23);
  static const Color darkAccent = Color(0xFF5D5FEF);
  static const Color darkSecondaryText = Color(0xFFA0A4AB);
  static const Color darkReceivedBubble = Color(0xFF262933);
  static const Color darkDivider = Color(0xFF2A2D36);

  static bool isDark(BuildContext context) {
    return Theme.of(context).brightness == Brightness.dark;
  }

  static Color accentOf(BuildContext context) {
    return isDark(context) ? darkAccent : primary;
  }

  static Color scaffoldOf(BuildContext context) {
    return isDark(context) ? darkBg : surfaceMuted;
  }

  static Color cardOf(BuildContext context) {
    return isDark(context) ? darkSurface : surface;
  }

  static Color mutedOf(BuildContext context) {
    return isDark(context) ? darkSurface : surfaceMuted;
  }

  static Color textOf(BuildContext context) {
    return isDark(context) ? Colors.white : textPrimary;
  }

  static Color hintOf(BuildContext context) {
    return isDark(context) ? darkSecondaryText : textSecondary;
  }

  static Color borderOf(BuildContext context) {
    return isDark(context) ? darkDivider : fieldBorder;
  }

  static Color dangerOf(BuildContext context) {
    return isDark(context) ? const Color(0xFFFF8A80) : const Color(0xFFB91C1C);
  }

  static Color dangerFillOf(BuildContext context) {
    return isDark(context) ? const Color(0xFF2A1518) : const Color(0xFFFEF2F2);
  }

  static Color dangerBorderOf(BuildContext context) {
    return isDark(context) ? const Color(0xFF6B2C2C) : const Color(0xFFFECACA);
  }

  static Color dangerTitleOf(BuildContext context) {
    return isDark(context) ? const Color(0xFFFF8A80) : const Color(0xFF991B1B);
  }

  static Color dangerBodyOf(BuildContext context) {
    return isDark(context) ? const Color(0xFFE7B6B6) : const Color(0xFF7F1D1D);
  }

  static Color avatarFillOf(BuildContext context) {
    return isDark(context)
        ? darkAccent.withValues(alpha: 0.28)
        : wave;
  }

  static Color avatarFgOf(BuildContext context) {
    return isDark(context) ? Colors.white : primaryDark;
  }

  static InputDecoration searchField(
    BuildContext context, {
    required String hint,
    IconData icon = Icons.search,
  }) {
    final radius = BorderRadius.circular(16);
    return InputDecoration(
      hintText: hint,
      prefixIcon: Icon(icon),
      filled: true,
      fillColor: mutedOf(context),
      border: OutlineInputBorder(
        borderRadius: radius,
        borderSide: BorderSide(color: borderOf(context)),
      ),
      enabledBorder: OutlineInputBorder(
        borderRadius: radius,
        borderSide: BorderSide(color: borderOf(context)),
      ),
      focusedBorder: OutlineInputBorder(
        borderRadius: radius,
        borderSide: BorderSide(color: accentOf(context)),
      ),
    );
  }
}

abstract final class AppAssets {
  static const seyraIcon = 'assets/branding/Logo/Icon_Design_1.png';
  static const seyraLogo = 'assets/branding/seyra_logo.webp';
}
