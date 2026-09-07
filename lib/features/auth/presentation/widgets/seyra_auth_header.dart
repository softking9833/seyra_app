import 'package:flutter/material.dart';
import 'package:seyra/core/constants/app_constants.dart';
import 'package:seyra/core/theme/app_colors.dart';

class SeyraAuthHeader extends StatelessWidget {
  const SeyraAuthHeader({
    super.key,
    required this.subtitle,
  });

  final String subtitle;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);

    return Column(
      children: [
        Image.asset(
          AppAssets.seyraLogo,
          height: 56,
          fit: BoxFit.contain,
          filterQuality: FilterQuality.high,
          semanticLabel: AppConstants.appName,
        ),
        const SizedBox(height: 16),
        Text(
          subtitle,
          textAlign: TextAlign.center,
          style: theme.textTheme.bodyMedium,
        ),
      ],
    );
  }
}
