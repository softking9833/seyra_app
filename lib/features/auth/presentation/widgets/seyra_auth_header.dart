import 'package:flutter/material.dart';
import 'package:seyra/core/constants/app_constants.dart';
import 'package:seyra/core/theme/app_colors.dart';

class SeyraAuthHeader extends StatelessWidget {
  const SeyraAuthHeader({
    super.key,
    this.subtitle = 'Private. Secure. Yours.',
    this.compact = false,
  });

  final String subtitle;
  final bool compact;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final logoSize = compact ? 72.0 : 112.0;

    return Column(
      children: [
        Image.asset(
          AppAssets.seyraIcon,
          height: logoSize,
          width: logoSize,
          fit: BoxFit.cover,
          filterQuality: FilterQuality.high,
          semanticLabel: AppConstants.appName,
        ),
        SizedBox(height: compact ? 12 : 20),
        ExcludeSemantics(
          child: Text(
            AppConstants.appName,
            textAlign: TextAlign.center,
            style: theme.textTheme.headlineMedium?.copyWith(
              color: AppColors.isDark(context) ? Colors.white : AppColors.navy,
              fontWeight: FontWeight.w800,
              letterSpacing: -0.6,
            ),
          ),
        ),
        const SizedBox(height: 8),
        Text(
          subtitle,
          textAlign: TextAlign.center,
          style: theme.textTheme.bodyMedium?.copyWith(
            color: AppColors.hintOf(context),
            fontSize: 15,
          ),
        ),
      ],
    );
  }
}
