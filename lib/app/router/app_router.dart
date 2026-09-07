import 'package:flutter/material.dart';
import 'package:seyra/app/di/app_dependencies.dart';
import 'package:seyra/app/router/app_routes.dart';
import 'package:seyra/app/router/shell_home_page.dart';
import 'package:seyra/features/auth/presentation/pages/login_page.dart';
import 'package:seyra/features/auth/presentation/pages/register_page.dart';

/// Application routing. Session/auth redirects will be added later.
abstract final class AppRouter {
  static Route<dynamic> onGenerateRoute(RouteSettings settings) {
    switch (settings.name) {
      case AppRoutes.login:
        return MaterialPageRoute<void>(
          builder: (context) => LoginPage(
            loginUseCase: AppDependencies.loginUseCase,
            onCreateAccount: () {
              Navigator.of(context).pushNamed(AppRoutes.register);
            },
          ),
          settings: settings,
        );
      case AppRoutes.register:
        return MaterialPageRoute<void>(
          builder: (context) => RegisterPage(
            registerUseCase: AppDependencies.registerUseCase,
            onSignIn: () {
              Navigator.of(context).pushNamed(AppRoutes.login);
            },
          ),
          settings: settings,
        );
      case AppRoutes.home:
      default:
        return MaterialPageRoute<void>(
          builder: (_) => const ShellHomePage(),
          settings: settings,
        );
    }
  }
}
