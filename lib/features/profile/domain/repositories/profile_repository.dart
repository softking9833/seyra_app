import 'package:seyra/core/errors/result.dart';
import 'package:seyra/features/profile/domain/entities/user_profile.dart';

abstract interface class ProfileRepository {
  Stream<UserProfile> watchProfile({
    required String userId,
    required String username,
  });

  Stream<UserPreferences> watchPreferences();

  Future<Result<UserProfile>> updateProfile({
    required String userId,
    required String displayName,
    required String bio,
  });

  Future<Result<UserPreferences>> updatePreferences(UserPreferences preferences);
}
