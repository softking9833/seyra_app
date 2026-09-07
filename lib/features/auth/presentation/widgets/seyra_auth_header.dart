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
        ClipRRect(
          borderRadius: BorderRadius.circular(20),
          child: Image.asset(
            AppAssets.seyraIcon,
            width: 76,
            height: 76,
            fit: BoxFit.cover,
          ),
        ),
        const SizedBox(height: 16),
        Text(AppConstants.appName, style: theme.textTheme.headlineMedium),
        const SizedBox(height: 8),
        Text(
          subtitle,
          textAlign: TextAlign.center,
          style: theme.textTheme.bodyMedium,
        ),
      ],
    );
  }
}
