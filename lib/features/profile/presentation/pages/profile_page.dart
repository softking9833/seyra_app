import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:seyra/core/theme/app_colors.dart';
import 'package:seyra/features/auth/domain/entities/user.dart';
import 'package:seyra/features/profile/domain/entities/user_profile.dart';
import 'package:seyra/features/profile/domain/usecases/watch_preferences_use_case.dart';
import 'package:seyra/features/profile/domain/usecases/watch_profile_use_case.dart';
import 'package:seyra/features/profile/presentation/widgets/profile_identity_card.dart';
import 'package:seyra/features/profile/presentation/widgets/settings_section.dart';

class ProfilePage extends StatefulWidget {
  const ProfilePage({
    super.key,
    required this.user,
    required this.watchProfile,
    required this.watchPreferences,
    required this.onOpenSettings,
    required this.onEditProfile,
  });

  final User user;
  final WatchProfileUseCase watchProfile;
  final WatchPreferencesUseCase watchPreferences;
  final VoidCallback onOpenSettings;
  final VoidCallback onEditProfile;

  @override
  State<ProfilePage> createState() => _ProfilePageState();
}

class _ProfilePageState extends State<ProfilePage> {
  late final Stream<UserProfile> _profile;
  late final Stream<UserPreferences> _preferences;

  @override
  void initState() {
    super.initState();
    _profile = widget.watchProfile(
      userId: widget.user.id,
      username: widget.user.username,
    );
    _preferences = widget.watchPreferences();
  }

  @override
  Widget build(BuildContext context) {
    return StreamBuilder<UserProfile>(
      stream: _profile,
      builder: (context, profileSnapshot) {
        return StreamBuilder<UserPreferences>(
          stream: _preferences,
          builder: (context, preferencesSnapshot) {
            final profile = profileSnapshot.data;
            final preferences = preferencesSnapshot.data;
            if (profile == null || preferences == null) {
              return const Center(child: CircularProgressIndicator());
            }

            return ColoredBox(
              color: AppColors.surfaceMuted,
              child: ListView(
              padding: const EdgeInsets.only(bottom: 32),
              children: [
                ProfileIdentityCard(
                  profile: profile,
                  onEdit: widget.onEditProfile,
                ),
                SettingsSection(
                  title: 'About',
                  children: [
                    SettingsTile(
                      icon: Icons.notes_outlined,
                      title: 'Bio',
                      subtitle: profile.bio.isEmpty
                          ? 'Add a bio'
                          : profile.bio,
                      onTap: widget.onEditProfile,
                    ),
                    SettingsTile(
                      icon: Icons.fingerprint,
                      title: 'User ID',
                      subtitle: profile.userId,
                      trailing: const Icon(
                        Icons.copy_outlined,
                        size: 18,
                        color: AppColors.textSecondary,
                      ),
                      onTap: () {
                        Clipboard.setData(ClipboardData(text: profile.userId));
                        ScaffoldMessenger.of(context).showSnackBar(
                          const SnackBar(
                            content: Text('User ID copied locally'),
                          ),
                        );
                      },
                    ),
                  ],
                ),
                SettingsSection(
                  title: 'Privacy & security',
                  children: [
                    SettingsTile(
                      icon: Icons.lock_outline,
                      title: 'End-to-end encryption',
                      subtitle: 'Prepared — not enabled yet',
                    ),
                    SettingsTile(
                      icon: Icons.visibility_outlined,
                      title: 'Last seen',
                      subtitle: visibilityLabel(preferences.lastSeenVisibility),
                      onTap: widget.onOpenSettings,
                    ),
                    SettingsTile(
                      icon: Icons.photo_outlined,
                      title: 'Profile photo',
                      subtitle: visibilityLabel(
                        preferences.profilePhotoVisibility,
                      ),
                      onTap: widget.onOpenSettings,
                    ),
                  ],
                ),
                SettingsSection(
                  title: 'Account',
                  children: [
                    SettingsTile(
                      icon: Icons.workspace_premium_outlined,
                      title: 'Seyra Premium',
                      subtitle: 'Coming soon',
                      onTap: () {
                        ScaffoldMessenger.of(context).showSnackBar(
                          const SnackBar(
                            content: Text('Premium is coming soon'),
                          ),
                        );
                      },
                    ),
                    SettingsTile(
                      icon: Icons.settings_outlined,
                      title: 'Settings',
                      subtitle: 'Privacy, notifications, and more',
                      onTap: widget.onOpenSettings,
                    ),
                  ],
                ),
              ],
            ),
            );
          },
        );
      },
    );
  }
}
