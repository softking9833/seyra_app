import 'package:flutter/material.dart';
import 'package:seyra/app/di/app_dependencies.dart';
import 'package:seyra/app/router/app_routes.dart';
import 'package:seyra/app/router/shell_home_page.dart';
import 'package:seyra/features/auth/domain/entities/user.dart';
import 'package:seyra/features/auth/presentation/pages/login_page.dart';
import 'package:seyra/features/auth/presentation/pages/register_page.dart';
import 'package:seyra/features/chat/presentation/pages/conversation_page.dart';
import 'package:seyra/features/profile/presentation/pages/edit_profile_page.dart';
import 'package:seyra/features/profile/presentation/pages/settings_page.dart';

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
            onAuthenticated: (_) => _openShell(context),
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
            onAuthenticated: (_) => _openShell(context),
          ),
          settings: settings,
        );
      case AppRoutes.conversation:
        final conversationId = settings.arguments as String? ?? '';
        return MaterialPageRoute<void>(
          builder: (_) => ConversationPage(
            conversationId: conversationId,
            watchConversations: AppDependencies.watchConversationsUseCase,
            watchMessages: AppDependencies.watchMessagesUseCase,
            watchPeerTyping: AppDependencies.watchPeerTypingUseCase,
            sendMessage: AppDependencies.sendMessageUseCase,
            deleteMessage: AppDependencies.deleteMessageUseCase,
            reactToMessage: AppDependencies.reactToMessageUseCase,
            markConversationRead: AppDependencies.markConversationReadUseCase,
            clearConversation: AppDependencies.clearConversationUseCase,
            setMuted: AppDependencies.setConversationMutedUseCase,
          ),
          settings: settings,
        );
      case AppRoutes.settings:
        final user = settings.arguments as User;
        return MaterialPageRoute<void>(
          builder: (context) => SettingsPage(
            user: user,
            watchPreferences: AppDependencies.watchPreferencesUseCase,
            updatePreferences: AppDependencies.updatePreferencesUseCase,
            onEditProfile: () {
              Navigator.of(context).pushNamed(
                AppRoutes.editProfile,
                arguments: user,
              );
            },
            onLogout: () async {
              await AppDependencies.logoutUseCase();
              if (context.mounted) {
                Navigator.of(context).pushNamedAndRemoveUntil(
                  AppRoutes.home,
                  (route) => false,
                );
              }
            },
          ),
          settings: settings,
        );
      case AppRoutes.editProfile:
        final user = settings.arguments as User;
        return MaterialPageRoute<void>(
          builder: (_) => EditProfilePage(
            user: user,
            watchProfile: AppDependencies.watchProfileUseCase,
            updateProfile: AppDependencies.updateProfileUseCase,
          ),
          settings: settings,
        );
      case AppRoutes.home:
      default:
        return MaterialPageRoute<void>(
          builder: (_) => ShellHomePage(
            restoreSessionUseCase: AppDependencies.restoreSessionUseCase,
            logoutUseCase: AppDependencies.logoutUseCase,
            watchConversationsUseCase: AppDependencies.watchConversationsUseCase,
            watchProfileUseCase: AppDependencies.watchProfileUseCase,
            watchPreferencesUseCase: AppDependencies.watchPreferencesUseCase,
          ),
          settings: settings,
        );
    }
  }

  static void _openShell(BuildContext context) {
    Navigator.of(context).pushNamedAndRemoveUntil(
      AppRoutes.home,
      (route) => false,
    );
  }
}
