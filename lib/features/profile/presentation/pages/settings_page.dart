import 'dart:async';

import 'package:flutter/material.dart';
import 'package:seyra/core/errors/result.dart';
import 'package:seyra/core/theme/app_colors.dart';
import 'package:seyra/features/auth/domain/entities/user.dart';
import 'package:seyra/features/notifications/domain/entities/notification_models.dart';
import 'package:seyra/features/notifications/domain/usecases/notification_use_cases.dart';
import 'package:seyra/features/profile/domain/entities/user_profile.dart';
import 'package:seyra/features/profile/domain/usecases/update_preferences_use_case.dart';
import 'package:seyra/features/profile/domain/usecases/watch_preferences_use_case.dart';
import 'package:seyra/features/profile/presentation/widgets/profile_identity_card.dart';
import 'package:seyra/features/profile/presentation/widgets/settings_section.dart';

class SettingsPage extends StatefulWidget {
  const SettingsPage({
    super.key,
    required this.user,
    required this.watchPreferences,
    required this.updatePreferences,
    required this.getNotificationPreferences,
    required this.updateNotificationPreferences,
    required this.onEditProfile,
    required this.onLogout,
    required this.onDeleteAccount,
    this.onPrivacy,
    this.onSessions,
    this.onBots,
    this.onCalls,
  });

  final User user;
  final WatchPreferencesUseCase watchPreferences;
  final UpdatePreferencesUseCase updatePreferences;
  final GetNotificationPreferencesUseCase getNotificationPreferences;
  final UpdateNotificationPreferencesUseCase updateNotificationPreferences;
  final VoidCallback onEditProfile;
  final Future<void> Function() onLogout;
  final VoidCallback onDeleteAccount;
  final VoidCallback? onPrivacy;
  final VoidCallback? onSessions;
  final VoidCallback? onBots;
  final VoidCallback? onCalls;

  @override
  State<SettingsPage> createState() => _SettingsPageState();
}

class _SettingsPageState extends State<SettingsPage> {
  late final Stream<UserPreferences> _preferences;
  NotificationPreferences _alerts = const NotificationPreferences();

  @override
  void initState() {
    super.initState();
    _preferences = widget.watchPreferences();
    unawaited(_loadAlerts());
  }

  Future<void> _loadAlerts() async {
    final result = await widget.getNotificationPreferences();
    if (!mounted) {
      return;
    }
    if (result is Success<NotificationPreferences>) {
      setState(() => _alerts = result.value);
    }
  }

  Future<void> _saveAlerts(NotificationPreferences next) async {
    setState(() => _alerts = next);
    await widget.updateNotificationPreferences(next);
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      backgroundColor: AppColors.surfaceMuted,
      appBar: AppBar(
        title: const Text('Settings'),
      ),
      body: StreamBuilder<UserPreferences>(
        stream: _preferences,
        builder: (context, snapshot) {
          final preferences = snapshot.data;
          if (preferences == null) {
            return const Center(child: CircularProgressIndicator());
          }

          return ListView(
            key: const Key('settings_scroll'),
            padding: const EdgeInsets.only(bottom: 40),
            children: [
              SettingsSection(
                title: 'Account',
                children: [
                  SettingsTile(
                    key: const Key('settings_edit_profile_tile'),
                    icon: Icons.person_outline,
                    title: 'Edit profile',
                    subtitle: 'Local placeholder — not saved to the server',
                    onTap: widget.onEditProfile,
                  ),
                  SettingsTile(
                    icon: Icons.alternate_email,
                    title: 'Username',
                    subtitle: '@${widget.user.username}',
                    onTap: () => _comingSoon(context, 'Username changes'),
                  ),
                  SettingsTile(
                    icon: Icons.manage_accounts_outlined,
                    title: 'Devices & sessions',
                    subtitle: 'Revoke signed-in sessions',
                    onTap: widget.onSessions ?? () => _comingSoon(context, 'Sessions'),
                  ),
                  SettingsTile(
                    key: const Key('settings_delete_account_tile'),
                    icon: Icons.delete_outline,
                    iconColor: const Color(0xFFB91C1C),
                    titleColor: const Color(0xFFB91C1C),
                    title: 'Delete account',
                    subtitle: 'Permanently delete your Seyra account',
                    onTap: widget.onDeleteAccount,
                  ),
                ],
              ),
              SettingsSection(
                title: 'Privacy & Security',
                children: [
                  SettingsTile(
                    icon: Icons.lock_outline,
                    title: 'Privacy controls',
                    subtitle: 'Presence, receipts, profile, previews',
                    onTap: widget.onPrivacy ?? () => _comingSoon(context, 'Privacy'),
                  ),
                  SettingsTile(
                    icon: Icons.schedule,
                    title: 'Last seen',
                    subtitle: visibilityLabel(preferences.lastSeenVisibility),
                    onTap: () => _pickVisibility(
                      context,
                      title: 'Last seen',
                      current: preferences.lastSeenVisibility,
                      onSelected: (value) {
                        widget.updatePreferences(
                          preferences.copyWith(lastSeenVisibility: value),
                        );
                      },
                    ),
                  ),
                  SettingsTile(
                    icon: Icons.photo_outlined,
                    title: 'Profile photo',
                    subtitle: visibilityLabel(
                      preferences.profilePhotoVisibility,
                    ),
                    onTap: () => _pickVisibility(
                      context,
                      title: 'Profile photo',
                      current: preferences.profilePhotoVisibility,
                      onSelected: (value) {
                        widget.updatePreferences(
                          preferences.copyWith(profilePhotoVisibility: value),
                        );
                      },
                    ),
                  ),
                  SettingsTile(
                    icon: Icons.smart_toy_outlined,
                    title: 'Bots',
                    onTap: widget.onBots ?? () => _comingSoon(context, 'Bots'),
                  ),
                  SettingsTile(
                    icon: Icons.call_outlined,
                    title: 'Call history',
                    onTap: widget.onCalls ?? () => _comingSoon(context, 'Calls'),
                  ),
                  SettingsTile(
                    icon: Icons.lock_outline,
                    title: 'Encryption',
                    subtitle: 'Signal Protocol for new 1:1 messages when keys exist. Older messages stay plaintext.',
                    onTap: () => _info(
                      context,
                      title: 'Encryption',
                      body:
                          'Seyra will support end-to-end encryption for messages. No encryption protocol is active in this build.',
                    ),
                  ),
                  SettingsTile(
                    icon: Icons.devices_outlined,
                    title: 'Active sessions',
                    subtitle: 'Placeholder — this device only',
                    onTap: () => _info(
                      context,
                      title: 'Active sessions',
                      body: 'Session management for this signed-in device.',
                    ),
                  ),
                  SettingsTile(
                    icon: Icons.security_outlined,
                    title: 'Security settings',
                    subtitle: 'Passkeys and 2FA coming soon',
                    onTap: () => _comingSoon(context, 'Security settings'),
                  ),
                ],
              ),
              SettingsSection(
                title: 'Notifications',
                children: [
                  SwitchListTile(
                    secondary: _iconWrap(Icons.notifications_outlined),
                    title: const Text('Message notifications'),
                    subtitle: const Text('Synced to the Seyra server'),
                    value: _alerts.messagesEnabled,
                    onChanged: (value) {
                      widget.updatePreferences(
                        preferences.copyWith(messageNotifications: value),
                      );
                      unawaited(
                        _saveAlerts(_alerts.copyWith(messagesEnabled: value)),
                      );
                    },
                  ),
                  SwitchListTile(
                    secondary: _iconWrap(Icons.visibility_outlined),
                    title: const Text('Message preview'),
                    subtitle: const Text('Off = generic “New message” text'),
                    value: _alerts.showPreview,
                    onChanged: (value) {
                      unawaited(
                        _saveAlerts(_alerts.copyWith(showPreview: value)),
                      );
                    },
                  ),
                  SwitchListTile(
                    secondary: _iconWrap(Icons.volume_up_outlined),
                    title: const Text('Sounds'),
                    value: preferences.soundEnabled,
                    onChanged: (value) {
                      widget.updatePreferences(
                        preferences.copyWith(soundEnabled: value),
                      );
                    },
                  ),
                  SwitchListTile(
                    secondary: _iconWrap(Icons.call_outlined),
                    title: const Text('Call notifications'),
                    value: _alerts.callsEnabled,
                    onChanged: (value) {
                      widget.updatePreferences(
                        preferences.copyWith(callNotifications: value),
                      );
                      unawaited(
                        _saveAlerts(_alerts.copyWith(callsEnabled: value)),
                      );
                    },
                  ),
                  const SettingsTile(
                    icon: Icons.smartphone_outlined,
                    title: 'This device',
                    subtitle:
                        'Registered for Seyra alerts after you sign in. Closed-app delivery needs SEYRA_PUSH_WEBHOOK_URL on the server.',
                  ),
                ],
              ),
              SettingsSection(
                title: 'Appearance',
                children: [
                  SettingsTile(
                    icon: Icons.brightness_6_outlined,
                    title: 'Theme',
                    subtitle: appearanceLabel(preferences.appearance),
                    onTap: () => _pickAppearance(context, preferences),
                  ),
                  SettingsTile(
                    icon: Icons.dark_mode_outlined,
                    title: 'Dark mode',
                    subtitle: 'Prepared — app stays on light theme',
                    onTap: () => _comingSoon(context, 'Dark mode'),
                  ),
                ],
              ),
              SettingsSection(
                title: 'About Seyra',
                children: [
                  SettingsTile(
                    icon: Icons.info_outline,
                    title: 'Seyra',
                    subtitle: 'Version 1.0.0',
                    onTap: () => _info(
                      context,
                      title: 'About Seyra',
                      body:
                          'Seyra is a private messenger. Encryption, calls, and cloud sync are not enabled in this build.',
                    ),
                  ),
                  SettingsTile(
                    icon: Icons.sd_storage_outlined,
                    title: 'Storage usage',
                    subtitle: '128 MB (placeholder)',
                    onTap: () => _info(
                      context,
                      title: 'Storage',
                      body:
                          'Media and cache usage will appear here once a real store is connected.',
                    ),
                  ),
                  SettingsTile(
                    icon: Icons.cleaning_services_outlined,
                    title: 'Clear cache',
                    subtitle: 'Local placeholder only',
                    onTap: () {
                      ScaffoldMessenger.of(context).showSnackBar(
                        const SnackBar(
                          content: Text('Cache cleared on this device'),
                        ),
                      );
                    },
                  ),
                ],
              ),
              SettingsSection(
                title: 'Danger zone',
                children: [
                  SettingsTile(
                    key: const Key('settings_logout_tile'),
                    icon: Icons.logout,
                    iconColor: const Color(0xFFB91C1C),
                    titleColor: const Color(0xFFB91C1C),
                    title: 'Log out',
                    onTap: () => widget.onLogout(),
                  ),
                ],
              ),
            ],
          );
        },
      ),
    );
  }

  Widget _iconWrap(IconData icon) {
    return Container(
      width: 36,
      height: 36,
      decoration: BoxDecoration(
        color: AppColors.primary.withValues(alpha: 0.12),
        borderRadius: BorderRadius.circular(10),
      ),
      child: Icon(icon, color: AppColors.primary, size: 20),
    );
  }

  void _comingSoon(BuildContext context, String feature) {
    ScaffoldMessenger.of(context).showSnackBar(
      SnackBar(content: Text('$feature coming soon')),
    );
  }

  void _info(BuildContext context, {required String title, required String body}) {
    showDialog<void>(
      context: context,
      builder: (context) {
        return AlertDialog(
          title: Text(title),
          content: Text(body),
          actions: [
            TextButton(
              onPressed: () => Navigator.pop(context),
              child: const Text('OK'),
            ),
          ],
        );
      },
    );
  }

  Future<void> _pickVisibility(
    BuildContext context, {
    required String title,
    required VisibilityPreference current,
    required ValueChanged<VisibilityPreference> onSelected,
  }) async {
    await showModalBottomSheet<void>(
      context: context,
      showDragHandle: true,
      builder: (context) {
        return SafeArea(
          child: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              Padding(
                padding: const EdgeInsets.all(16),
                child: Text(title, style: Theme.of(context).textTheme.titleMedium),
              ),
              for (final value in VisibilityPreference.values)
                ListTile(
                  title: Text(visibilityLabel(value)),
                  trailing: value == current
                      ? const Icon(Icons.check, color: AppColors.primary)
                      : null,
                  onTap: () {
                    onSelected(value);
                    Navigator.pop(context);
                  },
                ),
            ],
          ),
        );
      },
    );
  }

  Future<void> _pickAppearance(
    BuildContext context,
    UserPreferences preferences,
  ) async {
    await showModalBottomSheet<void>(
      context: context,
      showDragHandle: true,
      builder: (context) {
        return SafeArea(
          child: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              Padding(
                padding: const EdgeInsets.all(16),
                child: Text(
                  'Theme',
                  style: Theme.of(context).textTheme.titleMedium,
                ),
              ),
              for (final value in AppearancePreference.values)
                ListTile(
                  title: Text(appearanceLabel(value)),
                  subtitle: value == AppearancePreference.dark
                      ? const Text('Saved locally. Light theme stays active.')
                      : null,
                  trailing: value == preferences.appearance
                      ? const Icon(Icons.check, color: AppColors.primary)
                      : null,
                  onTap: () {
                    widget.updatePreferences(preferences.copyWith(appearance: value));
                    Navigator.pop(context);
                  },
                ),
            ],
          ),
        );
      },
    );
  }
}
