import 'package:seyra/core/errors/result.dart';
import 'package:seyra/features/profile/data/datasources/profile_data_source.dart';
import 'package:seyra/features/profile/domain/entities/user_profile.dart';
import 'package:seyra/features/profile/domain/repositories/profile_repository.dart';

final class ProfileRepositoryImpl implements ProfileRepository {
  const ProfileRepositoryImpl({required this.dataSource});

  final ProfileDataSource dataSource;

  @override
  Stream<UserProfile> watchProfile({
    required String userId,
    required String username,
  }) {
    return dataSource.watchProfile(userId: userId, username: username);
  }

  @override
  Stream<UserPreferences> watchPreferences() {
    return dataSource.watchPreferences();
  }

  @override
  Future<Result<UserProfile>> updateProfile({
    required String userId,
    required String displayName,
    required String bio,
  }) async {
    return Success(
      dataSource.updateProfile(
        userId: userId,
        displayName: displayName,
        bio: bio,
      ),
    );
  }

  @override
  Future<Result<UserPreferences>> updatePreferences(
    UserPreferences preferences,
  ) async {
    return Success(dataSource.updatePreferences(preferences));
  }
}
