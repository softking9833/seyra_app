import 'package:flutter/material.dart';
import 'package:seyra/app/router/app_routes.dart';
import 'package:seyra/core/constants/app_constants.dart';
import 'package:seyra/core/errors/result.dart';
import 'package:seyra/core/theme/app_colors.dart';
import 'package:seyra/features/auth/domain/entities/auth_session.dart';
import 'package:seyra/features/auth/domain/usecases/logout_use_case.dart';
import 'package:seyra/features/auth/domain/usecases/restore_session_use_case.dart';
import 'package:seyra/features/chat/domain/usecases/watch_conversations_use_case.dart';
import 'package:seyra/features/chat/presentation/pages/chats_home_view.dart';
import 'package:seyra/features/profile/domain/usecases/watch_preferences_use_case.dart';
import 'package:seyra/features/profile/domain/usecases/watch_profile_use_case.dart';
import 'package:seyra/features/profile/presentation/pages/profile_page.dart';

class ShellHomePage extends StatefulWidget {
  const ShellHomePage({
    super.key,
    required this.restoreSessionUseCase,
    required this.logoutUseCase,
    required this.watchConversationsUseCase,
    required this.watchProfileUseCase,
    required this.watchPreferencesUseCase,
  });

  final RestoreSessionUseCase restoreSessionUseCase;
  final LogoutUseCase logoutUseCase;
  final WatchConversationsUseCase watchConversationsUseCase;
  final WatchProfileUseCase watchProfileUseCase;
  final WatchPreferencesUseCase watchPreferencesUseCase;

  @override
  State<ShellHomePage> createState() => _ShellHomePageState();
}

class _ShellHomePageState extends State<ShellHomePage> {
  bool _loading = true;
  AuthSession? _session;
  int _tabIndex = 0;
  bool _searching = false;
  String _query = '';
  ChatFilter _filter = ChatFilter.chats;

  @override
  void initState() {
    super.initState();
    _restore();
  }

  Future<void> _restore() async {
    final result = await widget.restoreSessionUseCase();
    if (!mounted) {
      return;
    }

    setState(() {
      _loading = false;
      _session = switch (result) {
        Success(:final value) => value,
        FailureResult() => null,
      };
    });
  }

  void _openSettings() {
    final user = _session?.user;
    if (user == null) {
      return;
    }
    Navigator.of(context).pushNamed(AppRoutes.settings, arguments: user);
  }

  void _openEditProfile() {
    final user = _session?.user;
    if (user == null) {
      return;
    }
    Navigator.of(context).pushNamed(AppRoutes.editProfile, arguments: user);
  }

  Future<void> _signOut() async {
    setState(() {
      _loading = true;
      _tabIndex = 0;
      _searching = false;
      _query = '';
    });
    await widget.logoutUseCase();
    if (!mounted) {
      return;
    }
    setState(() {
      _loading = false;
      _session = null;
    });
  }

  @override
  Widget build(BuildContext context) {
    if (_loading) {
      return const Scaffold(
        body: Center(child: CircularProgressIndicator()),
      );
    }

    if (_session == null) {
      return const _WelcomeView();
    }

    return Scaffold(
      appBar: AppBar(
        titleSpacing: 16,
        title: _tabIndex == 3
            ? const Text('Profile')
            : _searching
            ? TextField(
                autofocus: true,
                decoration: const InputDecoration(
                  hintText: 'Search',
                  border: InputBorder.none,
                  enabledBorder: InputBorder.none,
                  focusedBorder: InputBorder.none,
                  filled: false,
                ),
                onChanged: (value) => setState(() => _query = value),
              )
            : Row(
                children: [
                  ClipRRect(
                    borderRadius: BorderRadius.circular(8),
                    child: Image.asset(
                      AppAssets.seyraIcon,
                      width: 28,
                      height: 28,
                      fit: BoxFit.cover,
                    ),
                  ),
                  const SizedBox(width: 10),
                  Text(
                    AppConstants.appName,
                    style: Theme.of(context).textTheme.titleMedium?.copyWith(
                      fontSize: 20,
                      fontWeight: FontWeight.w700,
                    ),
                  ),
                ],
              ),
        actions: [
          if (_tabIndex == 0)
            IconButton(
              tooltip: 'Search',
              onPressed: () {
                setState(() {
                  _searching = !_searching;
                  if (!_searching) {
                    _query = '';
                  }
                });
              },
              icon: Icon(_searching ? Icons.close : Icons.search),
            ),
          if (_tabIndex == 3)
            IconButton(
              key: const Key('profile_open_settings_button'),
              tooltip: 'Settings',
              onPressed: _openSettings,
              icon: const Icon(Icons.settings_outlined),
            ),
          PopupMenuButton<String>(
            key: const Key('shell_overflow_button'),
            icon: const Icon(Icons.more_vert),
            onSelected: (value) {
              if (value == 'logout') {
                _signOut();
              } else if (value == 'settings') {
                _openSettings();
              }
            },
            itemBuilder: (context) => [
              const PopupMenuItem(value: 'settings', child: Text('Settings')),
              const PopupMenuItem(
                key: Key('shell_sign_out_button'),
                value: 'logout',
                child: Text('Log out'),
              ),
            ],
          ),
        ],
      ),
      body: IndexedStack(
        index: _tabIndex,
        children: [
          ChatsHomeView(
            query: _query,
            filter: _filter,
            onFilterChanged: (value) => setState(() => _filter = value),
            watchConversations: widget.watchConversationsUseCase,
            onOpenConversation: (id) {
              Navigator.of(context).pushNamed(
                AppRoutes.conversation,
                arguments: id,
              );
            },
          ),
          const _PlaceholderTab(
            icon: Icons.call_outlined,
            title: 'Calls',
            subtitle: 'Voice and video calls will live here.',
          ),
          const           _PlaceholderTab(
            icon: Icons.people_outline,
            title: 'Contacts',
            subtitle: 'Your Seyra contacts will live here.',
          ),
          ProfilePage(
            user: _session!.user,
            watchProfile: widget.watchProfileUseCase,
            watchPreferences: widget.watchPreferencesUseCase,
            onOpenSettings: _openSettings,
            onEditProfile: _openEditProfile,
          ),
        ],
      ),
      floatingActionButton: _tabIndex == 0
          ? FloatingActionButton(
              onPressed: () {
                ScaffoldMessenger.of(context).showSnackBar(
                  const SnackBar(content: Text('New messages coming soon')),
                );
              },
              backgroundColor: AppColors.primary,
              foregroundColor: Colors.white,
              child: const Icon(Icons.edit_outlined),
            )
          : null,
      bottomNavigationBar: NavigationBar(
        selectedIndex: _tabIndex,
        onDestinationSelected: (index) => setState(() {
          _tabIndex = index;
          if (index != 0) {
            _searching = false;
            _query = '';
          }
        }),
        destinations: const [
          NavigationDestination(
            icon: Icon(Icons.chat_bubble_outline),
            selectedIcon: Icon(Icons.chat_bubble),
            label: 'Chats',
          ),
          NavigationDestination(
            icon: Icon(Icons.call_outlined),
            selectedIcon: Icon(Icons.call),
            label: 'Calls',
          ),
          NavigationDestination(
            icon: Icon(Icons.people_outline),
            selectedIcon: Icon(Icons.people),
            label: 'Contacts',
          ),
          NavigationDestination(
            icon: Icon(Icons.person_outline),
            selectedIcon: Icon(Icons.person),
            label: 'Profile',
          ),
        ],
      ),
    );
  }
}

class _WelcomeView extends StatelessWidget {
  const _WelcomeView();

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
              ClipRRect(
                borderRadius: BorderRadius.circular(16),
                child: Image.asset(
                  AppAssets.seyraIcon,
                  width: 72,
                  height: 72,
                  fit: BoxFit.cover,
                ),
              ),
              const SizedBox(height: 16),
              Text(AppConstants.appName, style: theme.textTheme.headlineMedium),
              const SizedBox(height: 8),
              Text(
                'Private. Secure. Yours.',
                style: theme.textTheme.bodyMedium,
              ),
              const SizedBox(height: 32),
              FilledButton(
                onPressed: () {
                  Navigator.of(context).pushNamed(AppRoutes.login);
                },
                child: const Text('Sign in'),
              ),
              const SizedBox(height: 8),
              OutlinedButton(
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

class _PlaceholderTab extends StatelessWidget {
  const _PlaceholderTab({
    required this.icon,
    required this.title,
    required this.subtitle,
  });

  final IconData icon;
  final String title;
  final String subtitle;

  @override
  Widget build(BuildContext context) {
    return Center(
      child: Padding(
        padding: const EdgeInsets.all(32),
        child: Column(
          mainAxisAlignment: MainAxisAlignment.center,
          children: [
            Icon(icon, size: 48, color: AppColors.textSecondary),
            const SizedBox(height: 12),
            Text(title, style: Theme.of(context).textTheme.titleMedium),
            const SizedBox(height: 8),
            Text(
              subtitle,
              textAlign: TextAlign.center,
              style: Theme.of(context).textTheme.bodyMedium,
            ),
          ],
        ),
      ),
    );
  }
}
