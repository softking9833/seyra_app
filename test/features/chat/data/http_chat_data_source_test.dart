import 'dart:async';

import 'package:flutter_test/flutter_test.dart';
import 'package:seyra/core/errors/result.dart';
import 'package:seyra/core/network/api_client.dart';
import 'package:seyra/core/storage/memory_secure_storage.dart';
import 'package:seyra/features/auth/data/datasources/auth_remote_data_source.dart';
import 'package:seyra/features/auth/data/exceptions/auth_remote_exceptions.dart';
import 'package:seyra/features/auth/data/models/auth_session_model.dart';
import 'package:seyra/features/auth/data/models/current_account_model.dart';
import 'package:seyra/features/auth/data/storage/auth_secure_storage_keys.dart';
import 'package:seyra/features/chat/data/datasources/http_chat_data_source.dart';
import 'package:seyra/features/chat/data/models/conversation_summary_model.dart';
import 'package:seyra/features/chat/data/realtime/chat_realtime_port.dart';
import 'package:seyra/features/chat/data/repositories/chat_repository_impl.dart';
import 'package:seyra/features/chat/domain/entities/chat_message.dart';
import 'package:seyra/features/chat/domain/entities/conversation.dart';
import 'package:seyra/features/chat/domain/entities/user_preview.dart';
import 'package:seyra/features/chat/domain/failures/chat_failures.dart';

void main() {
  test('maps chat list JSON to conversation tiles', () {
    final conversation = ConversationSummaryModel.fromJson({
      'id': 'cht_1',
      'peer': {'id': 'usr_lin', 'username': 'lin'},
      'last_message_preview': 'Hello from Seyra',
      'last_message_at': '2026-09-08T00:00:00.000Z',
      'unread_count': 2,
    }).toEntity();

    expect(conversation.title, 'lin');
    expect(conversation.initials, 'LI');
    expect(conversation.kind, ConversationKind.direct);
    expect(conversation.lastMessagePreview, 'Hello from Seyra');
    expect(conversation.unreadCount, 2);
  });

  test('maps group and channel JSON to conversation tiles', () {
    final group = ConversationSummaryModel.fromJson({
      'id': 'cht_g',
      'kind': 'group',
      'title': 'Design Team',
      'member_count': 3,
      'peer': {'id': '', 'username': ''},
      'last_message_preview': 'Ship the header',
      'last_message_at': '2026-09-08T00:00:00.000Z',
      'unread_count': 0,
    }).toEntity();
    expect(group.kind, ConversationKind.group);
    expect(group.title, 'Design Team');
    expect(group.statusText, '3 members');

    final channel = ConversationSummaryModel.fromJson({
      'id': 'cht_c',
      'kind': 'channel',
      'title': 'Seyra News',
      'member_count': 2,
      'last_message_preview': 'Privacy update',
      'last_message_at': '2026-09-08T00:00:00.000Z',
      'unread_count': 1,
    }).toEntity();
    expect(channel.kind, ConversationKind.channel);
    expect(channel.statusText, 'Channel');
  });

  test('loads messages and sends without exposing tokens', () async {
    final api = _FakeApiClient()
      ..responses.add(
        const ApiResponse(
          statusCode: 200,
          body:
              '{"chats":[{"id":"cht_1","peer":{"id":"usr_lin","username":"lin"},"last_message_preview":"Hi","last_message_at":"2026-09-08T00:00:00.000Z","unread_count":0}]}',
        ),
      )
      ..responses.add(
        const ApiResponse(
          statusCode: 200,
          body:
              '{"messages":[{"id":"msg_1","conversation_id":"cht_1","sender_id":"usr_lin","body":"Hi","created_at":"2026-09-08T00:00:00.000Z"}]}',
        ),
      )
      ..responses.add(
        const ApiResponse(
          statusCode: 201,
          body:
              '{"id":"msg_2","conversation_id":"cht_1","sender_id":"usr_ada","body":"Hello from Seyra","created_at":"2026-09-08T00:01:00.000Z"}',
        ),
      );
    final storage = MemorySecureStorage();
    await storage.write(key: AuthSecureStorageKeys.accessToken, value: 'secret-token');
    final source = HttpChatDataSource(
      apiClient: api,
      secureStorage: storage,
      authRemote: _FakeAuthRemote(),
      baseUrl: Uri.parse('http://127.0.0.1:8080'),
      realtime: _FakeRealtime(),
    );
    final repository = ChatRepositoryImpl(dataSource: source);

    final chats = await repository.watchConversations().first;
    expect(chats.single.title, 'lin');
    expect(chats.toString().contains('secret-token'), isFalse);

    final messages = await repository.watchMessages('cht_1').first;
    expect(messages.single.body, 'Hi');
    expect(messages.single.isFrom('usr_ada'), isFalse);

    final sent = await repository.sendMessage(
      conversationId: 'cht_1',
      body: 'Hello from Seyra',
    );
    expect((sent as Success<ChatMessage>).value.body, 'Hello from Seyra');
    expect(sent.value.isFrom('usr_ada'), isTrue);
    expect(sent.value.delivery, MessageDelivery.sent);
    expect(sent.toString().contains('secret-token'), isFalse);
  });

  test('search users maps public fields only', () async {
    final api = _FakeApiClient()
      ..responses.add(
        const ApiResponse(statusCode: 200, body: '{"chats":[]}'),
      )
      ..responses.add(
        const ApiResponse(
          statusCode: 200,
          body: '{"users":[{"id":"usr_lin","username":"lin"}]}',
        ),
      );
    final storage = MemorySecureStorage();
    await storage.write(key: AuthSecureStorageKeys.accessToken, value: 'token');
    final repository = ChatRepositoryImpl(
      dataSource: HttpChatDataSource(
        apiClient: api,
        secureStorage: storage,
        authRemote: _FakeAuthRemote(),
        baseUrl: Uri.parse('http://127.0.0.1:8080'),
        realtime: _FakeRealtime(),
      ),
    );
    await repository.watchConversations().first;
    final result = await repository.searchUsers('li');
    final users = (result as Success<List<UserPreview>>).value;
    expect(users.single.username, 'lin');
  });

  test('unread increments for peer messages when chat is not open', () async {
    final api = _FakeApiClient()
      ..responses.add(
        const ApiResponse(
          statusCode: 200,
          body:
              '{"chats":[{"id":"cht_1","peer":{"id":"usr_lin","username":"lin"},"last_message_preview":"Hi","last_message_at":"2026-09-08T00:00:00.000Z","unread_count":0}]}',
        ),
      );
    final realtime = _ControllableRealtime();
    final storage = MemorySecureStorage();
    await storage.write(key: AuthSecureStorageKeys.accessToken, value: 'token');
    final source = HttpChatDataSource(
      apiClient: api,
      secureStorage: storage,
      authRemote: _FakeAuthRemote(),
      baseUrl: Uri.parse('http://127.0.0.1:8080'),
      realtime: realtime,
    );
    final repository = ChatRepositoryImpl(dataSource: source);
    final seen = <List<Conversation>>[];
    final sub = repository.watchConversations().listen(seen.add);
    await Future<void>.delayed(Duration.zero);
    expect(seen.last.single.unreadCount, 0);
    realtime.emit({
      'type': 'message.created',
      'payload': {
        'id': 'msg_new',
        'conversation_id': 'cht_1',
        'sender_id': 'usr_lin',
        'body': 'ping',
        'created_at': '2026-09-08T00:02:00.000Z',
      },
    });
    await Future<void>.delayed(Duration.zero);
    expect(seen.last.single.unreadCount, 1);
    expect(seen.last.single.lastMessagePreview, 'ping');
    await sub.cancel();
  });

  test('failed send then retry replaces the local message', () async {
    final api = _FakeApiClient()
      ..responses.add(
        const ApiResponse(
          statusCode: 200,
          body:
              '{"chats":[{"id":"cht_1","peer":{"id":"usr_lin","username":"lin"},"last_message_preview":"Hi","last_message_at":"2026-09-08T00:00:00.000Z","unread_count":0}]}',
        ),
      )
      ..responses.add(
        const ApiResponse(
          statusCode: 200,
          body:
              '{"messages":[{"id":"msg_1","conversation_id":"cht_1","sender_id":"usr_lin","body":"Hi","created_at":"2026-09-08T00:00:00.000Z"}]}',
        ),
      )
      ..responses.add(const ApiResponse(statusCode: 500, body: '{}'))
      ..responses.add(
        const ApiResponse(
          statusCode: 201,
          body:
              '{"id":"msg_2","conversation_id":"cht_1","sender_id":"usr_ada","body":"retry me","created_at":"2026-09-08T00:03:00.000Z"}',
        ),
      );
    final storage = MemorySecureStorage();
    await storage.write(key: AuthSecureStorageKeys.accessToken, value: 'token');
    final source = HttpChatDataSource(
      apiClient: api,
      secureStorage: storage,
      authRemote: _FakeAuthRemote(),
      baseUrl: Uri.parse('http://127.0.0.1:8080'),
      realtime: _FakeRealtime(),
    );
    final repository = ChatRepositoryImpl(dataSource: source);
    await repository.watchConversations().first;
    await repository.watchMessages('cht_1').first;

    final failed = await repository.sendMessage(
      conversationId: 'cht_1',
      body: 'retry me',
    );
    expect(failed, isA<FailureResult<ChatMessage>>());
    expect(
      source.peekMessages('cht_1').singleWhere((item) => item.body == 'retry me').delivery,
      MessageDelivery.failed,
    );
    final pendingId = source
        .peekMessages('cht_1')
        .lastWhere((item) => item.body == 'retry me')
        .id;

    final retried = await repository.retryMessage(
      conversationId: 'cht_1',
      messageId: pendingId,
    );
    expect((retried as Success<ChatMessage>).value.id, 'msg_2');
    expect(retried.value.delivery, MessageDelivery.sent);
    expect(
      source.peekMessages('cht_1').where((item) => item.body == 'retry me').length,
      1,
    );
    expect(
      source.peekMessages('cht_1').any((item) => item.delivery == MessageDelivery.failed),
      isFalse,
    );
  });

  test('delete waits for success and keeps the message on failure', () async {
    final api = _FakeApiClient()
      ..responses.add(
        const ApiResponse(
          statusCode: 200,
          body:
              '{"chats":[{"id":"cht_1","peer":{"id":"usr_lin","username":"lin"},"last_message_preview":"Hi","last_message_at":"2026-09-08T00:00:00.000Z","unread_count":0}]}',
        ),
      )
      ..responses.add(
        const ApiResponse(
          statusCode: 200,
          body:
              '{"messages":[{"id":"msg_1","conversation_id":"cht_1","sender_id":"usr_ada","body":"Hi","created_at":"2026-09-08T00:00:00.000Z"}]}',
        ),
      )
      ..responses.add(
        const ApiResponse(
          statusCode: 403,
          body: '{"error":{"code":"forbidden","message":"Forbidden"}}',
        ),
      )
      ..responses.add(const ApiResponse(statusCode: 204, body: ''));
    final storage = MemorySecureStorage();
    await storage.write(key: AuthSecureStorageKeys.accessToken, value: 'token');
    final source = HttpChatDataSource(
      apiClient: api,
      secureStorage: storage,
      authRemote: _FakeAuthRemote(),
      baseUrl: Uri.parse('http://127.0.0.1:8080'),
      realtime: _FakeRealtime(),
    );
    final repository = ChatRepositoryImpl(dataSource: source);
    await repository.watchConversations().first;
    await repository.watchMessages('cht_1').first;

    final denied = await repository.deleteMessage(
      conversationId: 'cht_1',
      messageId: 'msg_1',
    );
    expect(denied, isA<FailureResult<void>>());
    expect(source.peekMessages('cht_1').any((item) => item.id == 'msg_1'), isTrue);

    final ok = await repository.deleteMessage(
      conversationId: 'cht_1',
      messageId: 'msg_1',
    );
    expect(ok, isA<Success<void>>());
    expect(source.peekMessages('cht_1').any((item) => item.id == 'msg_1'), isFalse);
  });

  test('reconnect merge does not duplicate messages', () async {
    final api = _FakeApiClient()
      ..responses.add(
        const ApiResponse(
          statusCode: 200,
          body:
              '{"chats":[{"id":"cht_1","peer":{"id":"usr_lin","username":"lin"},"last_message_preview":"Hi","last_message_at":"2026-09-08T00:00:00.000Z","unread_count":0}]}',
        ),
      )
      ..responses.add(
        const ApiResponse(
          statusCode: 200,
          body:
              '{"messages":[{"id":"msg_1","conversation_id":"cht_1","sender_id":"usr_lin","body":"Hi","created_at":"2026-09-08T00:00:00.000Z"}]}',
        ),
      )
      ..responses.add(
        const ApiResponse(
          statusCode: 200,
          body:
              '{"chats":[{"id":"cht_1","peer":{"id":"usr_lin","username":"lin"},"last_message_preview":"Later","last_message_at":"2026-09-08T00:04:00.000Z","unread_count":0}]}',
        ),
      )
      ..responses.add(
        const ApiResponse(
          statusCode: 200,
          body:
              '{"messages":[{"id":"msg_1","conversation_id":"cht_1","sender_id":"usr_lin","body":"Hi","created_at":"2026-09-08T00:00:00.000Z"},{"id":"msg_2","conversation_id":"cht_1","sender_id":"usr_lin","body":"Later","created_at":"2026-09-08T00:04:00.000Z"}]}',
        ),
      );
    final realtime = _ControllableRealtime();
    final storage = MemorySecureStorage();
    await storage.write(key: AuthSecureStorageKeys.accessToken, value: 'token');
    final repository = ChatRepositoryImpl(
      dataSource: HttpChatDataSource(
        apiClient: api,
        secureStorage: storage,
        authRemote: _FakeAuthRemote(),
        baseUrl: Uri.parse('http://127.0.0.1:8080'),
        realtime: realtime,
      ),
    );
    await repository.watchConversations().first;
    final seen = <List<ChatMessage>>[];
    final sub = repository.watchMessages('cht_1').listen(seen.add);
    for (var i = 0; i < 20 && seen.isEmpty; i++) {
      await Future<void>.delayed(Duration.zero);
    }
    expect(seen, isNotEmpty);
    await Future<void>.delayed(Duration.zero);
    expect(seen.last.map((item) => item.id), ['msg_1']);
    realtime.emit({'type': 'realtime.connected'});
    await Future<void>.delayed(const Duration(milliseconds: 20));
    expect(seen.last.map((item) => item.id), ['msg_1', 'msg_2']);
    await sub.cancel();
  });

  test('realtime message.deleted removes the message', () async {
    final api = _FakeApiClient()
      ..responses.add(
        const ApiResponse(
          statusCode: 200,
          body:
              '{"chats":[{"id":"cht_1","peer":{"id":"usr_lin","username":"lin"},"last_message_preview":"Hi","last_message_at":"2026-09-08T00:00:00.000Z","unread_count":0}]}',
        ),
      )
      ..responses.add(
        const ApiResponse(
          statusCode: 200,
          body:
              '{"messages":[{"id":"msg_1","conversation_id":"cht_1","sender_id":"usr_lin","body":"Hi","created_at":"2026-09-08T00:00:00.000Z"}]}',
        ),
      );
    final realtime = _ControllableRealtime();
    final storage = MemorySecureStorage();
    await storage.write(key: AuthSecureStorageKeys.accessToken, value: 'token');
    final repository = ChatRepositoryImpl(
      dataSource: HttpChatDataSource(
        apiClient: api,
        secureStorage: storage,
        authRemote: _FakeAuthRemote(),
        baseUrl: Uri.parse('http://127.0.0.1:8080'),
        realtime: realtime,
      ),
    );
    await repository.watchConversations().first;
    final seen = <List<ChatMessage>>[];
    final sub = repository.watchMessages('cht_1').listen(seen.add);
    for (var i = 0; i < 20 && seen.isEmpty; i++) {
      await Future<void>.delayed(Duration.zero);
    }
    expect(seen, isNotEmpty);
    await Future<void>.delayed(Duration.zero);
    expect(seen.last.single.id, 'msg_1');
    realtime.emit({
      'type': 'message.deleted',
      'payload': {'id': 'msg_1', 'conversation_id': 'cht_1'},
    });
    await Future<void>.delayed(Duration.zero);
    expect(seen.last, isEmpty);
    await sub.cancel();
  });

  test('maps missing username to ChatUserNotFoundFailure', () async {
    final api = _FakeApiClient()
      ..responses.add(
        const ApiResponse(statusCode: 200, body: '{"chats":[]}'),
      )
      ..responses.add(
        const ApiResponse(
          statusCode: 404,
          body: '{"error":{"code":"not_found","message":"Not found"}}',
        ),
      );
    final storage = MemorySecureStorage();
    await storage.write(key: AuthSecureStorageKeys.accessToken, value: 'token');
    final repository = ChatRepositoryImpl(
      dataSource: HttpChatDataSource(
        apiClient: api,
        secureStorage: storage,
        authRemote: _FakeAuthRemote(),
        baseUrl: Uri.parse('http://127.0.0.1:8080'),
        realtime: _FakeRealtime(),
      ),
    );

    await repository.watchConversations().first;
    final result = await repository.startDirectChat('ghost');
    expect(
      (result as FailureResult<Conversation>).failure,
      isA<ChatUserNotFoundFailure>(),
    );
  });
}

final class _FakeApiClient implements ApiClient {
  final List<ApiResponse> responses = [];

  @override
  Future<ApiResponse> send({
    required String method,
    required Uri uri,
    Map<String, String>? headers,
    Object? jsonBody,
  }) async {
    if (uri.path.contains('/v1/e2e/')) {
      return const ApiResponse(statusCode: 204, body: '{}');
    }
    if (responses.isEmpty) {
      return const ApiResponse(statusCode: 500, body: '{}');
    }
    return responses.removeAt(0);
  }
}

final class _ControllableRealtime implements ChatRealtimePort {
  final _controller = StreamController<Map<String, dynamic>>.broadcast();

  void emit(Map<String, dynamic> event) => _controller.add(event);

  @override
  Stream<Map<String, dynamic>> connect({
    required Uri uri,
    required String accessToken,
  }) {
    return _controller.stream;
  }

  @override
  Future<void> disconnect() async {}

  @override
  Future<void> send(Map<String, dynamic> event) async {}
}

final class _FakeRealtime implements ChatRealtimePort {
  @override
  Stream<Map<String, dynamic>> connect({
    required Uri uri,
    required String accessToken,
  }) {
    return const Stream.empty();
  }

  @override
  Future<void> disconnect() async {}

  @override
  Future<void> send(Map<String, dynamic> event) async {}
}

final class _FakeAuthRemote implements AuthRemoteDataSource {
  @override
  Future<CurrentAccountModel> getCurrentAccount() async {
    return CurrentAccountModel(
      id: 'usr_ada',
      username: 'ada',
      createdAt: DateTime.utc(2026, 1, 1),
    );
  }

  @override
  Future<AuthSessionModel> login({
    required String username,
    required String password,
  }) async {
    throw const AuthRemoteUnavailableException();
  }

  @override
  Future<AuthSessionModel> register({
    required String username,
    required String password,
  }) {
    return login(username: username, password: password);
  }

  @override
  Future<void> logout() async {}

  @override
  Future<AuthSessionModel?> restoreSession() async => null;

  @override
  Future<AuthSessionModel> refreshSession() async {
    throw const AuthRemoteUnavailableException();
  }

  @override
  Future<void> deleteAccount({required String password}) async {}
}
