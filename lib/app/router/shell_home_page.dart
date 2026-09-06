import 'package:flutter/material.dart';
import 'package:seyra/core/constants/app_constants.dart';

/// Temporary shell screen until feature navigation exists.
class ShellHomePage extends StatelessWidget {
  const ShellHomePage({super.key});

  @override
  Widget build(BuildContext context) {
    return const Scaffold(
      body: Center(
        child: Text(AppConstants.appName),
      ),
    );
  }
}
