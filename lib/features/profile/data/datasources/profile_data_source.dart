import 'package:seyra/features/profile/domain/entities/user_profile.dart';

abstract interface class ProfileDataSource {
  Stream<UserProfile> watchProfile({
    required String userId,
    required String username,
  });

  Stream<UserPreferences> watchPreferences();

  UserProfile updateProfile({
    required String userId,
    required String displayName,
    required String bio,
  });

  UserPreferences updatePreferences(UserPreferences preferences);
}
