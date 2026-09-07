import 'package:seyra/core/config/app_config.dart';
import 'package:seyra/core/network/http_api_client.dart';
import 'package:seyra/core/storage/flutter_secure_storage_adapter.dart';
import 'package:seyra/features/auth/data/datasources/auth_remote_data_source.dart';
import 'package:seyra/features/auth/data/datasources/http_auth_remote_data_source.dart';
import 'package:seyra/features/auth/data/datasources/mock_auth_remote_data_source.dart';
import 'package:seyra/features/auth/data/repositories/auth_repository_impl.dart';
import 'package:seyra/features/auth/domain/repositories/auth_repository.dart';
import 'package:seyra/features/auth/domain/usecases/delete_account_use_case.dart';
import 'package:seyra/features/auth/domain/usecases/login_use_case.dart';
import 'package:seyra/features/auth/domain/usecases/logout_use_case.dart';
import 'package:seyra/features/auth/domain/usecases/refresh_session_use_case.dart';
import 'package:seyra/features/auth/domain/usecases/register_use_case.dart';
import 'package:seyra/features/auth/domain/usecases/restore_session_use_case.dart';
import 'package:seyra/features/chat/data/datasources/chat_data_source.dart';
import 'package:seyra/features/chat/data/datasources/mock_chat_data_source.dart';
import 'package:seyra/features/chat/data/repositories/chat_repository_impl.dart';
import 'package:seyra/features/chat/domain/repositories/chat_repository.dart';
import 'package:seyra/features/chat/domain/usecases/clear_conversation_use_case.dart';
import 'package:seyra/features/chat/domain/usecases/delete_message_use_case.dart';
import 'package:seyra/features/chat/domain/usecases/get_conversation_use_case.dart';
import 'package:seyra/features/chat/domain/usecases/mark_conversation_read_use_case.dart';
import 'package:seyra/features/chat/domain/usecases/react_to_message_use_case.dart';
import 'package:seyra/features/chat/domain/usecases/send_message_use_case.dart';
import 'package:seyra/features/chat/domain/usecases/set_conversation_muted_use_case.dart';
import 'package:seyra/features/chat/domain/usecases/watch_conversations_use_case.dart';
import 'package:seyra/features/chat/domain/usecases/watch_messages_use_case.dart';
import 'package:seyra/features/chat/domain/usecases/watch_peer_typing_use_case.dart';
import 'package:seyra/features/profile/data/datasources/mock_profile_data_source.dart';
import 'package:seyra/features/profile/data/datasources/profile_data_source.dart';
import 'package:seyra/features/profile/data/repositories/profile_repository_impl.dart';
import 'package:seyra/features/profile/domain/repositories/profile_repository.dart';
import 'package:seyra/features/profile/domain/usecases/update_preferences_use_case.dart';
import 'package:seyra/features/profile/domain/usecases/update_profile_use_case.dart';
import 'package:seyra/features/profile/domain/usecases/watch_preferences_use_case.dart';
import 'package:seyra/features/profile/domain/usecases/watch_profile_use_case.dart';

/// Composition root.
///
/// Swap [authRemoteDataSource] to a real API adapter later. Domain and UI stay.
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
  static late ChatDataSource chatDataSource;
  static late ChatRepository chatRepository;
  static late WatchConversationsUseCase watchConversationsUseCase;
  static late WatchMessagesUseCase watchMessagesUseCase;
  static late WatchPeerTypingUseCase watchPeerTypingUseCase;
  static late GetConversationUseCase getConversationUseCase;
  static late SendMessageUseCase sendMessageUseCase;
  static late DeleteMessageUseCase deleteMessageUseCase;
  static late ReactToMessageUseCase reactToMessageUseCase;
  static late MarkConversationReadUseCase markConversationReadUseCase;
  static late ClearConversationUseCase clearConversationUseCase;
  static late SetConversationMutedUseCase setConversationMutedUseCase;
  static late ProfileDataSource profileDataSource;
  static late ProfileRepository profileRepository;
  static late WatchProfileUseCase watchProfileUseCase;
  static late UpdateProfileUseCase updateProfileUseCase;
  static late WatchPreferencesUseCase watchPreferencesUseCase;
  static late UpdatePreferencesUseCase updatePreferencesUseCase;

  static Future<void> initialize({
    AppConfig? config,
    AuthRemoteDataSource? remoteDataSource,
    ChatDataSource? chatSource,
    ProfileDataSource? profileSource,
  }) async {
    appConfig = config ?? AppConfig.fromEnvironment();
    appConfig.validate();

    authRemoteDataSource =
        remoteDataSource ??
        _createAuthRemoteDataSource();
    authRepository = AuthRepositoryImpl(
      remoteDataSource: authRemoteDataSource,
    );
    loginUseCase = LoginUseCase(authRepository);
    registerUseCase = RegisterUseCase(authRepository);
    logoutUseCase = LogoutUseCase(authRepository);
    restoreSessionUseCase = RestoreSessionUseCase(authRepository);
    refreshSessionUseCase = RefreshSessionUseCase(authRepository);
    deleteAccountUseCase = DeleteAccountUseCase(authRepository);

    chatDataSource =
        chatSource ??
        MockChatDataSource(deliveryDelay: const Duration(milliseconds: 400));
    chatRepository = ChatRepositoryImpl(dataSource: chatDataSource);
    watchConversationsUseCase = WatchConversationsUseCase(chatRepository);
    watchMessagesUseCase = WatchMessagesUseCase(chatRepository);
    watchPeerTypingUseCase = WatchPeerTypingUseCase(chatRepository);
    getConversationUseCase = GetConversationUseCase(chatRepository);
    sendMessageUseCase = SendMessageUseCase(chatRepository);
    deleteMessageUseCase = DeleteMessageUseCase(chatRepository);
    reactToMessageUseCase = ReactToMessageUseCase(chatRepository);
    markConversationReadUseCase = MarkConversationReadUseCase(chatRepository);
    clearConversationUseCase = ClearConversationUseCase(chatRepository);
    setConversationMutedUseCase = SetConversationMutedUseCase(chatRepository);

    profileDataSource = profileSource ?? MockProfileDataSource();
    profileRepository = ProfileRepositoryImpl(dataSource: profileDataSource);
    watchProfileUseCase = WatchProfileUseCase(profileRepository);
    updateProfileUseCase = UpdateProfileUseCase(profileRepository);
    watchPreferencesUseCase = WatchPreferencesUseCase(profileRepository);
    updatePreferencesUseCase = UpdatePreferencesUseCase(profileRepository);
  }

  static AuthRemoteDataSource _createAuthRemoteDataSource() {
    if (appConfig.authBackendMode == AuthBackendMode.mock) {
      return MockAuthRemoteDataSource(
        latency: const Duration(milliseconds: 400),
      );
    }
    return HttpAuthRemoteDataSource(
      apiClient: HttpApiClient(),
      secureStorage: FlutterSecureStorageAdapter(),
      baseUrl: Uri.parse(appConfig.apiBaseUrl),
    );
  }
}
