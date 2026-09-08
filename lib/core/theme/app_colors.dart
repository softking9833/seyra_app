import 'package:flutter/material.dart';

/// Brand color tokens. Dark counterparts can be added later.
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
}

abstract final class AppAssets {
  static const seyraIcon = 'assets/branding/seyra_icon.jpg';
  static const seyraLogo = 'assets/branding/seyra_logo.webp';
}
