import 'dart:async';

import 'package:flutter/material.dart';
import 'package:seyra/app/di/app_dependencies.dart';
import 'package:seyra/app/router/app_routes.dart';
import 'package:seyra/core/constants/app_constants.dart';
import 'package:seyra/core/errors/result.dart';
import 'package:seyra/core/theme/app_colors.dart';
import 'package:seyra/features/auth/domain/entities/auth_session.dart';
import 'package:seyra/features/auth/domain/usecases/logout_use_case.dart';
import 'package:seyra/features/auth/domain/usecases/restore_session_use_case.dart';
import 'package:seyra/features/auth/presentation/widgets/auth_pill_button.dart';
import 'package:seyra/features/auth/presentation/widgets/auth_screen_scaffold.dart';
import 'package:seyra/features/auth/presentation/widgets/seyra_auth_header.dart';
import 'package:seyra/features/chat/domain/usecases/refresh_conversations_use_case.dart';
import 'package:seyra/features/chat/domain/usecases/watch_conversations_use_case.dart';
import 'package:seyra/features/chat/presentation/pages/chats_home_view.dart';
import 'package:seyra/features/chat/presentation/pages/social_pages.dart';
import 'package:seyra/features/profile/domain/usecases/get_account_use_case.dart';
import 'package:seyra/features/profile/domain/usecases/watch_preferences_use_case.dart';
import 'package:seyra/features/profile/domain/usecases/watch_profile_use_case.dart';
import 'package:seyra/features/profile/presentation/pages/profile_page.dart';

class ShellHomePage extends StatefulWidget {
  const ShellHomePage({
    super.key,
    required this.restoreSessionUseCase,
    required this.logoutUseCase,
    required this.watchConversationsUseCase,
    required this.refreshConversationsUseCase,
    required this.watchProfileUseCase,
    required this.watchPreferencesUseCase,
    required this.getAccountUseCase,
  });

  final RestoreSessionUseCase restoreSessionUseCase;
  final LogoutUseCase logoutUseCase;
  final WatchConversationsUseCase watchConversationsUseCase;
  final RefreshConversationsUseCase refreshConversationsUseCase;
  final WatchProfileUseCase watchProfileUseCase;
  final WatchPreferencesUseCase watchPreferencesUseCase;
  final GetAccountUseCase getAccountUseCase;

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
  StreamSubscription<Map<String, dynamic>>? _callSub;
  String? _ringingCallId;

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
    if (_session != null) {
      unawaited(AppDependencies.pushCoordinator.start());
      _callSub?.cancel();
      _callSub = AppDependencies.chatSocial.watchCallSignals().listen((payload) {
        if (!mounted) {
          return;
        }
        final action = payload['action'] as String?;
        final callId = payload['call_id'] as String?;
        if (action == 'hangup' || action == 'reject' || action == 'missed') {
          if (callId != null && callId == _ringingCallId) {
            _ringingCallId = null;
            Navigator.of(context, rootNavigator: true).pop();
          }
          return;
        }
        if (action == 'offer') {
          unawaited(_promptIncomingCall(payload));
        }
      });
    }
  }

  @override
  void dispose() {
    unawaited(_callSub?.cancel());
    super.dispose();
  }

  Future<void> _promptIncomingCall(Map<String, dynamic> payload) async {
    final callId = payload['call_id'] as String? ?? '';
    if (callId.isEmpty || _ringingCallId == callId) {
      return;
    }
    _ringingCallId = callId;
    final accepted = await showDialog<bool>(
      context: context,
      barrierDismissible: false,
      builder: (dialogContext) {
        final video = payload['kind'] == 'video';
        return AlertDialog(
          title: Text(video ? 'Incoming video call' : 'Incoming voice call'),
          content: const Text('Accept this call? Camera and microphone start only after you accept.'),
          actions: [
            TextButton(
              onPressed: () => Navigator.pop(dialogContext, false),
              child: const Text('Decline'),
            ),
            TextButton(
              onPressed: () => Navigator.pop(dialogContext, true),
              child: const Text('Accept'),
            ),
          ],
        );
      },
    );
    if (!mounted || _ringingCallId != callId) {
      return;
    }
    _ringingCallId = null;
    if (accepted != true) {
      unawaited(
        AppDependencies.chatSocial.signalCall(callId: callId, action: 'reject'),
      );
      return;
    }
    Navigator.of(context).pushNamed(
      AppRoutes.call,
      arguments: {
        'conversationId': payload['conversation_id'] as String? ?? '',
        'video': payload['kind'] == 'video',
        'outgoing': false,
        'payload': payload,
        'peerId': payload['caller_id'] as String? ?? '',
        'title': payload['username'] as String? ?? '',
      },
    );
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
              } else if (value == 'search') {
                Navigator.of(context).pushNamed(AppRoutes.globalSearch);
              } else if (value == 'discover') {
                Navigator.of(context).pushNamed(AppRoutes.discoverChannels);
              }
            },
            itemBuilder: (context) => [
              const PopupMenuItem(value: 'search', child: Text('Search')),
              const PopupMenuItem(value: 'discover', child: Text('Discover channels')),
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
            refreshConversations: widget.refreshConversationsUseCase,
            onOpenConversation: (id) {
              Navigator.of(context).pushNamed(
                AppRoutes.conversation,
                arguments: id,
              );
            },
          ),
          CallHistoryPage(
            social: AppDependencies.chatSocial,
            currentUserId: _session!.user.id,
            active: _tabIndex == 1,
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
            getAccount: widget.getAccountUseCase,
            onOpenSettings: _openSettings,
            onEditProfile: _openEditProfile,
            onLogout: _signOut,
          ),
        ],
      ),
      floatingActionButton: _tabIndex == 0 &&
              (ModalRoute.of(context)?.isCurrent ?? true)
          ? FloatingActionButton(
              key: const Key('new_chat_fab'),
              onPressed: () {
                final route = switch (_filter) {
                  ChatFilter.chats => AppRoutes.newChat,
                  ChatFilter.groups => AppRoutes.newGroup,
                  ChatFilter.channels => AppRoutes.newChannel,
                };
                Navigator.of(context).pushNamed(route);
              },
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
    return AuthScreenScaffold(
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          const Spacer(flex: 2),
          const SeyraAuthHeader(),
          const Spacer(flex: 3),
          AuthGradientButton(
            label: 'Sign in',
            icon: Icons.person_outline,
            onPressed: () {
              Navigator.of(context).pushNamed(AppRoutes.login);
            },
          ),
          const SizedBox(height: 12),
          AuthOutlinedPillButton(
            label: 'Create an account',
            icon: Icons.add,
            onPressed: () {
              Navigator.of(context).pushNamed(AppRoutes.register);
            },
          ),
          const SizedBox(height: 24),
        ],
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
            Icon(icon, size: 48, color: AppColors.hintOf(context)),
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
