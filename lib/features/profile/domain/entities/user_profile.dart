enum VisibilityPreference { everyone, contacts, nobody }

enum AppearancePreference { system, light, dark }

final class UserProfile {
  const UserProfile({
    required this.userId,
    required this.username,
    required this.displayName,
    required this.bio,
    this.isPremium = false,
  });

  final String userId;
  final String username;
  final String displayName;
  final String bio;
  final bool isPremium;

  UserProfile copyWith({
    String? displayName,
    String? bio,
  }) {
    return UserProfile(
      userId: userId,
      username: username,
      displayName: displayName ?? this.displayName,
      bio: bio ?? this.bio,
      isPremium: isPremium,
    );
  }
}

final class UserPreferences {
  const UserPreferences({
    this.lastSeenVisibility = VisibilityPreference.everyone,
    this.profilePhotoVisibility = VisibilityPreference.everyone,
    this.messageNotifications = true,
    this.soundEnabled = true,
    this.callNotifications = true,
    this.appearance = AppearancePreference.light,
  });

  final VisibilityPreference lastSeenVisibility;
  final VisibilityPreference profilePhotoVisibility;
  final bool messageNotifications;
  final bool soundEnabled;
  final bool callNotifications;
  final AppearancePreference appearance;

  UserPreferences copyWith({
    VisibilityPreference? lastSeenVisibility,
    VisibilityPreference? profilePhotoVisibility,
    bool? messageNotifications,
    bool? soundEnabled,
    bool? callNotifications,
    AppearancePreference? appearance,
  }) {
    return UserPreferences(
      lastSeenVisibility: lastSeenVisibility ?? this.lastSeenVisibility,
      profilePhotoVisibility:
          profilePhotoVisibility ?? this.profilePhotoVisibility,
      messageNotifications: messageNotifications ?? this.messageNotifications,
      soundEnabled: soundEnabled ?? this.soundEnabled,
      callNotifications: callNotifications ?? this.callNotifications,
      appearance: appearance ?? this.appearance,
    );
  }
}
