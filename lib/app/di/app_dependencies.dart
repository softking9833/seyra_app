import 'package:seyra/core/config/app_config.dart';
import 'package:seyra/features/auth/data/datasources/auth_remote_data_source.dart';
import 'package:seyra/features/auth/data/datasources/deferred_auth_remote_data_source.dart';
import 'package:seyra/features/auth/data/repositories/auth_repository_impl.dart';
import 'package:seyra/features/auth/domain/repositories/auth_repository.dart';
import 'package:seyra/features/auth/domain/usecases/delete_account_use_case.dart';
import 'package:seyra/features/auth/domain/usecases/login_use_case.dart';
import 'package:seyra/features/auth/domain/usecases/logout_use_case.dart';
import 'package:seyra/features/auth/domain/usecases/refresh_session_use_case.dart';
import 'package:seyra/features/auth/domain/usecases/register_use_case.dart';
import 'package:seyra/features/auth/domain/usecases/restore_session_use_case.dart';

/// Composition root.
///
/// Register implementations here when they exist. No DI package is used yet.
/// Core and features must not look up dependencies from widgets directly.
/// Swap [authRemoteDataSource] to change backend provider without rewriting domain.
abstract final class AppDependencies {
  static late AppConfig appConfig;
  static late AuthRemoteDataSource authRemoteDataSource;
  static late AuthRepository authRepository;
  static late LoginUseCase loginUseCase;
  static late RegisterUseCase registerUseCase;
  static late LogoutUseCase logoutUseCase;
  static late RestoreSessionUseCase restoreSessionUseCase;
  static late RefreshSessionUseCase refreshSessionUseCase;
  static late DeleteAccountUseCase deleteAccountUseCase;

  static Future<void> initialize({AppConfig? config}) async {
    appConfig = config ?? AppConfig.fromEnvironment();
    appConfig.validate();

    authRemoteDataSource = const DeferredAuthRemoteDataSource();
    authRepository = AuthRepositoryImpl(
      remoteDataSource: authRemoteDataSource,
    );
    loginUseCase = LoginUseCase(authRepository);
    registerUseCase = RegisterUseCase(authRepository);
    logoutUseCase = LogoutUseCase(authRepository);
    restoreSessionUseCase = RestoreSessionUseCase(authRepository);
    refreshSessionUseCase = RefreshSessionUseCase(authRepository);
    deleteAccountUseCase = DeleteAccountUseCase(authRepository);
  }
}
