import 'dart:async';

import 'package:seyra/features/profile/data/datasources/profile_data_source.dart';
import 'package:seyra/features/profile/domain/entities/user_profile.dart';

final class MockProfileDataSource implements ProfileDataSource {
  MockProfileDataSource() : _preferences = const UserPreferences();

  final Map<String, UserProfile> _profiles = {};
  UserPreferences _preferences;

  final _profileController = StreamController<UserProfile>.broadcast();
  final _preferencesController =
      StreamController<UserPreferences>.broadcast();

  @override
  Stream<UserProfile> watchProfile({
    required String userId,
    required String username,
  }) async* {
    yield _ensureProfile(userId: userId, username: username);
    yield* _profileController.stream.where((profile) => profile.userId == userId);
  }

  @override
  Stream<UserPreferences> watchPreferences() async* {
    yield _preferences;
    yield* _preferencesController.stream;
  }

  @override
  UserProfile updateProfile({
    required String userId,
    required String displayName,
    required String bio,
  }) {
    final current = _profiles[userId];
    if (current == null) {
      final created = UserProfile(
        userId: userId,
        username: 'user',
        displayName: displayName,
        bio: bio,
      );
      _profiles[userId] = created;
      _profileController.add(created);
      return created;
    }
    final updated = current.copyWith(displayName: displayName, bio: bio);
    _profiles[userId] = updated;
    _profileController.add(updated);
    return updated;
  }

  @override
  UserPreferences updatePreferences(UserPreferences preferences) {
    _preferences = preferences;
    _preferencesController.add(_preferences);
    return _preferences;
  }

  UserProfile _ensureProfile({
    required String userId,
    required String username,
  }) {
    final existing = _profiles[userId];
    if (existing != null) {
      return existing;
    }
    final created = UserProfile(
      userId: userId,
      username: username,
      displayName: _titleCase(username),
      bio: 'Private messaging on Seyra.',
    );
    _profiles[userId] = created;
    return created;
  }

  static String _titleCase(String username) {
    final value = username.trim();
    if (value.isEmpty) {
      return 'Seyra user';
    }
    return '${value[0].toUpperCase()}${value.substring(1)}';
  }
}
