import 'package:flutter_test/flutter_test.dart';
import 'package:seyra/core/errors/failures.dart';
import 'package:seyra/core/errors/result.dart';
import 'package:seyra/features/chat/domain/entities/chat_message.dart';
import 'package:seyra/features/chat/domain/repositories/chat_repository.dart';
import 'package:seyra/features/chat/domain/usecases/send_message_use_case.dart';
import 'package:seyra/features/chat/domain/entities/conversation.dart';

void main() {
  late _FakeChatRepository repository;
  late SendMessageUseCase useCase;

  setUp(() {
    repository = _FakeChatRepository();
    useCase = SendMessageUseCase(repository);
  });

  test('rejects blank messages without calling the repository', () async {
    final result = await useCase(conversationId: '1', body: '   ');

    expect(result, isA<FailureResult<ChatMessage>>());
    expect(
      (result as FailureResult<ChatMessage>).failure,
      isA<ValidationFailure>(),
    );
    expect(repository.sendCalls, 0);
  });

  test('trims body and delegates to the repository', () async {
    final result = await useCase(
      conversationId: '1',
      body: '  Hello  ',
      replyToId: '1-a',
    );

    expect(result, isA<Success<ChatMessage>>());
    expect(repository.lastBody, 'Hello');
    expect(repository.lastReplyToId, '1-a');
  });
}

final class _FakeChatRepository implements ChatRepository {
  int sendCalls = 0;
  String? lastBody;
  String? lastReplyToId;

  @override
  Future<Result<ChatMessage>> sendMessage({
    required String conversationId,
    required String body,
    String? replyToId,
  }) async {
    sendCalls += 1;
    lastBody = body;
    lastReplyToId = replyToId;
    return Success(
      ChatMessage(
        id: 'm1',
        conversationId: conversationId,
        senderId: ChatMessage.localUserId,
        body: body,
        sentAt: DateTime(2026, 9, 7, 21, 0),
        delivery: MessageDelivery.sent,
        replyToId: replyToId,
      ),
    );
  }

  @override
  Stream<List<Conversation>> watchConversations() => const Stream.empty();

  @override
  Stream<List<ChatMessage>> watchMessages(String conversationId) {
    return const Stream.empty();
  }

  @override
  Stream<bool> watchPeerTyping(String conversationId) => const Stream.empty();

  @override
  Future<Result<Conversation>> getConversation(String id) async {
    throw UnimplementedError();
  }

  @override
  Future<Result<void>> deleteMessage({
    required String conversationId,
    required String messageId,
  }) async {
    throw UnimplementedError();
  }

  @override
  Future<Result<void>> reactToMessage({
    required String conversationId,
    required String messageId,
    required String emoji,
  }) async {
    throw UnimplementedError();
  }

  @override
  Future<Result<void>> markConversationRead(String conversationId) async {
    throw UnimplementedError();
  }

  @override
  Future<Result<void>> clearConversation(String conversationId) async {
    throw UnimplementedError();
  }

  @override
  Future<Result<void>> setMuted({
    required String conversationId,
    required bool muted,
  }) async {
    throw UnimplementedError();
  }
}
