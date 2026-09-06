import 'package:flutter/material.dart';
import 'package:seyra/app/router/app_routes.dart';
import 'package:seyra/app/router/shell_home_page.dart';

/// Application routing. Session/auth redirects will be added later.
abstract final class AppRouter {
  static Route<dynamic> onGenerateRoute(RouteSettings settings) {
    switch (settings.name) {
      case AppRoutes.home:
        return MaterialPageRoute<void>(
          builder: (_) => const ShellHomePage(),
          settings: settings,
        );
      default:
        return MaterialPageRoute<void>(
          builder: (_) => const ShellHomePage(),
          settings: settings,
        );
    }
  }
}
