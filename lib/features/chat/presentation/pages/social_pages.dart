import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:seyra/core/errors/result.dart';
import 'package:seyra/core/theme/app_colors.dart';
import 'package:seyra/features/chat/domain/entities/conversation.dart';
import 'package:seyra/features/chat/domain/entities/social_models.dart';
import 'package:seyra/features/chat/domain/repositories/chat_social_repository.dart';
import 'package:seyra/app/router/app_routes.dart';

class GlobalSearchPage extends StatefulWidget {
  const GlobalSearchPage({super.key, required this.social});

  final ChatSocialRepository social;

  @override
  State<GlobalSearchPage> createState() => _GlobalSearchPageState();
}

class _GlobalSearchPageState extends State<GlobalSearchPage> {
  var _query = '';
  var _loading = false;
  String? _error;
  GlobalSearchResult _result = const GlobalSearchResult();

  Future<void> _run(String query) async {
    setState(() {
      _loading = true;
      _error = null;
    });
    final result = await widget.social.searchGlobal(query);
    if (!mounted) {
      return;
    }
    switch (result) {
      case FailureResult(:final failure):
        setState(() {
          _loading = false;
          _error = failure.message;
        });
      case Success(:final value):
        setState(() {
          _loading = false;
          _result = value;
        });
    }
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text('Search')),
      backgroundColor: AppColors.surfaceMuted,
      body: Column(
        children: [
          Padding(
            padding: const EdgeInsets.all(16),
            child: TextField(
              onChanged: (value) {
                _query = value;
                if (value.trim().length >= 2) {
                  unawaited(_run(value.trim()));
                }
              },
              decoration: const InputDecoration(
                hintText: 'Users, chats, messages',
                prefixIcon: Icon(Icons.search),
                filled: true,
                fillColor: Colors.white,
              ),
            ),
          ),
          if (_loading) const LinearProgressIndicator(),
          if (_error != null)
            Padding(
              padding: const EdgeInsets.all(16),
              child: Text(_error!, style: const TextStyle(color: Colors.red)),
            ),
          Expanded(
            child: _query.trim().length < 2
                ? const Center(child: Text('Type at least 2 characters'))
                : ListView(
                    children: [
                      if (_result.users.isNotEmpty)
                        const ListTile(title: Text('People')),
                      for (final user in _result.users)
                        ListTile(
                          title: Text(user.username),
                          subtitle: Text('@${user.username}'),
                        ),
                      if (_result.conversations.isNotEmpty)
                        const ListTile(title: Text('Chats')),
                      for (final chat in _result.conversations)
                        ListTile(
                          title: Text(chat.title),
                          subtitle: Text(chat.lastMessagePreview),
                          onTap: () => Navigator.pushNamed(
                            context,
                            AppRoutes.conversation,
                            arguments: chat.id,
                          ),
                        ),
                      if (_result.messages.isNotEmpty)
                        const ListTile(title: Text('Messages')),
                      for (final message in _result.messages)
                        ListTile(
                          title: Text(message.body),
                          onTap: () => Navigator.pushNamed(
                            context,
                            AppRoutes.conversation,
                            arguments: message.conversationId,
                          ),
                        ),
                    ],
                  ),
          ),
        ],
      ),
    );
  }
}

class DiscoverChannelsPage extends StatefulWidget {
  const DiscoverChannelsPage({super.key, required this.social});

  final ChatSocialRepository social;

  @override
  State<DiscoverChannelsPage> createState() => _DiscoverChannelsPageState();
}

class _DiscoverChannelsPageState extends State<DiscoverChannelsPage> {
  var _loading = true;
  String? _error;
  var _items = <Conversation>[];

  @override
  void initState() {
    super.initState();
    unawaited(_load(''));
  }

  Future<void> _load(String query) async {
    setState(() {
      _loading = true;
      _error = null;
    });
    final result = await widget.social.discoverChannels(query);
    if (!mounted) {
      return;
    }
    switch (result) {
      case FailureResult(:final failure):
        setState(() {
          _loading = false;
          _error = failure.message;
        });
      case Success(:final value):
        setState(() {
          _loading = false;
          _items = value;
        });
    }
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text('Discover channels')),
      backgroundColor: AppColors.surfaceMuted,
      body: Column(
        children: [
          Padding(
            padding: const EdgeInsets.all(16),
            child: TextField(
              onChanged: (value) => unawaited(_load(value.trim())),
              decoration: const InputDecoration(
                hintText: 'Search public channels',
                prefixIcon: Icon(Icons.search),
                filled: true,
                fillColor: Colors.white,
              ),
            ),
          ),
          if (_loading) const LinearProgressIndicator(),
          if (_error != null)
            TextButton(onPressed: () => _load(''), child: Text('Retry: $_error')),
          Expanded(
            child: _items.isEmpty && !_loading
                ? const Center(child: Text('No public channels'))
                : ListView.builder(
                    itemCount: _items.length,
                    itemBuilder: (context, index) {
                      final chat = _items[index];
                      return ListTile(
                        title: Text(chat.title),
                        subtitle: Text(chat.statusText),
                        trailing: TextButton(
                          onPressed: () async {
                            final joined = await widget.social.joinChannel(chat.id);
                            if (joined is Success && context.mounted) {
                              Navigator.pushNamed(
                                context,
                                AppRoutes.conversation,
                                arguments: chat.id,
                              );
                            }
                          },
                          child: const Text('Join'),
                        ),
                      );
                    },
                  ),
          ),
        ],
      ),
    );
  }
}

class PrivacyControlsPage extends StatefulWidget {
  const PrivacyControlsPage({super.key, required this.social});

  final ChatSocialRepository social;

  @override
  State<PrivacyControlsPage> createState() => _PrivacyControlsPageState();
}

class _PrivacyControlsPageState extends State<PrivacyControlsPage> {
  PrivacySettings? _settings;
  String? _error;

  @override
  void initState() {
    super.initState();
    unawaited(_load());
  }

  Future<void> _load() async {
    final result = await widget.social.getPrivacy();
    if (!mounted) {
      return;
    }
    switch (result) {
      case FailureResult(:final failure):
        setState(() => _error = failure.message);
      case Success(:final value):
        setState(() => _settings = value);
    }
  }

  Future<void> _save(PrivacySettings next) async {
    setState(() => _settings = next);
    await widget.social.putPrivacy(next);
  }

  @override
  Widget build(BuildContext context) {
    final settings = _settings;
    return Scaffold(
      appBar: AppBar(title: const Text('Privacy')),
      body: settings == null
          ? Center(
              child: _error == null
                  ? const CircularProgressIndicator()
                  : TextButton(onPressed: _load, child: Text(_error!)),
            )
          : ListView(
              children: [
                SwitchListTile(
                  title: const Text('Last seen visible'),
                  subtitle: const Text(
                    'When off, people in a 1-to-1 chat will not see your last-seen time. Not a live online indicator.',
                  ),
                  value: settings.lastSeenVisible,
                  onChanged: (value) =>
                      _save(settings.copyWith(lastSeenVisible: value)),
                ),
                SwitchListTile(
                  title: const Text('Read receipts'),
                  subtitle: const Text(
                    'When off, others will not see when you last read a 1-to-1 chat. Your own unread counts still update.',
                  ),
                  value: settings.readReceipts,
                  onChanged: (value) =>
                      _save(settings.copyWith(readReceipts: value)),
                ),
                SwitchListTile(
                  title: const Text('Typing indicators'),
                  subtitle: const Text(
                    'When off, the server will not broadcast that you are typing.',
                  ),
                  value: settings.typingVisible,
                  onChanged: (value) =>
                      _save(settings.copyWith(typingVisible: value)),
                ),
                SwitchListTile(
                  title: const Text('Profile visible in search'),
                  value: settings.profileVisible,
                  onChanged: (value) =>
                      _save(settings.copyWith(profileVisible: value)),
                ),
                SwitchListTile(
                  title: const Text('Notification previews'),
                  value: settings.notificationPreview,
                  onChanged: (value) =>
                      _save(settings.copyWith(notificationPreview: value)),
                ),
              ],
            ),
    );
  }
}

class SessionsPage extends StatefulWidget {
  const SessionsPage({super.key, required this.social});

  final ChatSocialRepository social;

  @override
  State<SessionsPage> createState() => _SessionsPageState();
}

class _SessionsPageState extends State<SessionsPage> {
  var _items = const <AuthDeviceSession>[];
  var _loading = true;

  @override
  void initState() {
    super.initState();
    unawaited(_load());
  }

  Future<void> _load() async {
    final result = await widget.social.listSessions();
    if (!mounted) {
      return;
    }
    if (result is Success<List<AuthDeviceSession>>) {
      setState(() {
        _items = result.value;
        _loading = false;
      });
    } else {
      setState(() => _loading = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text('Devices & sessions')),
      body: _loading
          ? const Center(child: CircularProgressIndicator())
          : _items.isEmpty
          ? const Center(child: Text('No sessions'))
          : ListView(
              children: [
                for (final session in _items)
                  ListTile(
                    title: Text(session.id),
                    subtitle: Text(session.revoked ? 'Revoked' : 'Active'),
                    trailing: IconButton(
                      icon: const Icon(Icons.logout),
                      onPressed: () async {
                        await widget.social.revokeSession(session.id);
                        await _load();
                      },
                    ),
                  ),
              ],
            ),
    );
  }
}

class BotsPage extends StatefulWidget {
  const BotsPage({super.key, required this.social});

  final ChatSocialRepository social;

  @override
  State<BotsPage> createState() => _BotsPageState();
}

class _BotsPageState extends State<BotsPage> {
  var _bots = const <BotAccount>[];
  final _username = TextEditingController(text: 'echo_bot');
  final _grantChat = TextEditingController();

  @override
  void initState() {
    super.initState();
    unawaited(_load());
  }

  @override
  void dispose() {
    _username.dispose();
    _grantChat.dispose();
    super.dispose();
  }

  Future<void> _load() async {
    final result = await widget.social.listBots();
    if (result is Success<List<BotAccount>> && mounted) {
      setState(() => _bots = result.value);
    }
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text('Bots')),
      body: ListView(
        padding: const EdgeInsets.all(16),
        children: [
          TextField(
            controller: _username,
            decoration: const InputDecoration(labelText: 'Bot username'),
          ),
          const SizedBox(height: 8),
          const Text(
            'Commands in a granted room: /ping, /help, /whoami. Token is shown once.',
            style: TextStyle(color: AppColors.textSecondary),
          ),
          FilledButton(
            onPressed: () async {
              final created = await widget.social.createBot(_username.text.trim());
              if (created is Success<BotAccount> && context.mounted) {
                final token = created.value.token ?? '';
                await Clipboard.setData(ClipboardData(text: token));
                if (!context.mounted) {
                  return;
                }
                ScaffoldMessenger.of(context).showSnackBar(
                  SnackBar(
                    content: Text(
                      'Token copied once: $token — grant it into a group then send /ping',
                    ),
                  ),
                );
                await _load();
              }
            },
            child: const Text('Create bot'),
          ),
          const SizedBox(height: 16),
          TextField(
            controller: _grantChat,
            decoration: const InputDecoration(labelText: 'Conversation ID to grant'),
          ),
          for (final bot in _bots)
            ListTile(
              title: Text(bot.username),
              subtitle: Text(bot.id),
              trailing: Wrap(
                children: [
                  TextButton(
                    onPressed: () async {
                      await widget.social.grantBot(
                        botId: bot.id,
                        conversationId: _grantChat.text.trim(),
                      );
                    },
                    child: const Text('Grant'),
                  ),
                  IconButton(
                    onPressed: () async {
                      await widget.social.deleteBot(bot.id);
                      await _load();
                    },
                    icon: const Icon(Icons.delete_outline),
                  ),
                ],
              ),
            ),
        ],
      ),
    );
  }
}

class CallHistoryPage extends StatefulWidget {
  const CallHistoryPage({super.key, required this.social});

  final ChatSocialRepository social;

  @override
  State<CallHistoryPage> createState() => _CallHistoryPageState();
}

class _CallHistoryPageState extends State<CallHistoryPage> {
  var _items = const <CallRecord>[];

  @override
  void initState() {
    super.initState();
    unawaited(() async {
      final result = await widget.social.listCalls();
      if (result is Success<List<CallRecord>> && mounted) {
        setState(() => _items = result.value);
      }
    }());
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text('Calls')),
      body: _items.isEmpty
          ? const Center(child: Text('No calls yet'))
          : ListView(
              children: [
                for (final call in _items)
                  ListTile(
                    leading: Icon(
                      call.kind == 'video' ? Icons.videocam : Icons.call,
                    ),
                    title: Text('${call.kind} · ${call.state}'),
                    subtitle: Text(
                      '${call.createdAt.toLocal()} · ${call.durationSeconds ?? 0}s',
                    ),
                  ),
              ],
            ),
    );
  }
}
