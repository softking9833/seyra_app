import 'package:flutter/material.dart';
import 'package:seyra/app/router/app_routes.dart';
import 'package:seyra/core/constants/app_constants.dart';

/// Temporary shell screen until feature navigation exists.
class ShellHomePage extends StatelessWidget {
  const ShellHomePage({super.key});

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);

    return Scaffold(
      body: Center(
        child: Padding(
          padding: const EdgeInsets.all(24),
          child: Column(
            mainAxisAlignment: MainAxisAlignment.center,
            children: [
              Text(
                AppConstants.appName,
                style: theme.textTheme.headlineMedium,
              ),
              const SizedBox(height: 32),
              FilledButton(
                onPressed: () {
                  Navigator.of(context).pushNamed(AppRoutes.login);
                },
                child: const Text('Sign in'),
              ),
              TextButton(
                onPressed: () {
                  Navigator.of(context).pushNamed(AppRoutes.register);
                },
                child: const Text('Create an account'),
              ),
            ],
          ),
        ),
      ),
    );
  }
}
