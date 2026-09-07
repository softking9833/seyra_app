import 'package:seyra/core/errors/failures.dart';
import 'package:seyra/core/errors/result.dart';
import 'package:seyra/features/chat/data/datasources/chat_data_source.dart';
import 'package:seyra/features/chat/domain/entities/chat_message.dart';
import 'package:seyra/features/chat/domain/entities/conversation.dart';
import 'package:seyra/features/chat/domain/repositories/chat_repository.dart';

final class ChatRepositoryImpl implements ChatRepository {
  const ChatRepositoryImpl({required this.dataSource});

  final ChatDataSource dataSource;

  @override
  Stream<List<Conversation>> watchConversations() {
    return dataSource.watchConversations();
  }

  @override
  Stream<List<ChatMessage>> watchMessages(String conversationId) {
    return dataSource.watchMessages(conversationId);
  }

  @override
  Stream<bool> watchPeerTyping(String conversationId) {
    return dataSource.watchPeerTyping(conversationId);
  }

  @override
  Future<Result<Conversation>> getConversation(String id) async {
    final conversation = dataSource.getConversation(id);
    if (conversation == null) {
      return const FailureResult(UnexpectedFailure('Conversation not found'));
    }
    return Success(conversation);
  }

  @override
  Future<Result<ChatMessage>> sendMessage({
    required String conversationId,
    required String body,
    String? replyToId,
  }) async {
    try {
      return Success(
        dataSource.sendMessage(
          conversationId: conversationId,
          body: body,
          replyToId: replyToId,
        ),
      );
    } on ChatNotFoundException catch (error) {
      return FailureResult(UnexpectedFailure(error.message));
    }
  }

  @override
  Future<Result<void>> deleteMessage({
    required String conversationId,
    required String messageId,
  }) async {
    try {
      dataSource.deleteMessage(
        conversationId: conversationId,
        messageId: messageId,
      );
      return const Success(null);
    } on ChatNotFoundException catch (error) {
      return FailureResult(UnexpectedFailure(error.message));
    }
  }

  @override
  Future<Result<void>> reactToMessage({
    required String conversationId,
    required String messageId,
    required String emoji,
  }) async {
    try {
      dataSource.reactToMessage(
        conversationId: conversationId,
        messageId: messageId,
        emoji: emoji,
      );
      return const Success(null);
    } on ChatNotFoundException catch (error) {
      return FailureResult(UnexpectedFailure(error.message));
    }
  }

  @override
  Future<Result<void>> markConversationRead(String conversationId) async {
    try {
      dataSource.markConversationRead(conversationId);
      return const Success(null);
    } on ChatNotFoundException catch (error) {
      return FailureResult(UnexpectedFailure(error.message));
    }
  }

  @override
  Future<Result<void>> clearConversation(String conversationId) async {
    try {
      dataSource.clearConversation(conversationId);
      return const Success(null);
    } on ChatNotFoundException catch (error) {
      return FailureResult(UnexpectedFailure(error.message));
    }
  }

  @override
  Future<Result<void>> setMuted({
    required String conversationId,
    required bool muted,
  }) async {
    try {
      dataSource.setMuted(conversationId: conversationId, muted: muted);
      return const Success(null);
    } on ChatNotFoundException catch (error) {
      return FailureResult(UnexpectedFailure(error.message));
    }
  }
}
