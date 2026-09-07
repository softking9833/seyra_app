import 'package:seyra/core/errors/result.dart';
import 'package:seyra/features/chat/domain/entities/chat_message.dart';
import 'package:seyra/features/chat/domain/entities/conversation.dart';

abstract interface class ChatRepository {
  Stream<List<Conversation>> watchConversations();

  Stream<List<ChatMessage>> watchMessages(String conversationId);

  Stream<bool> watchPeerTyping(String conversationId);

  Future<Result<Conversation>> getConversation(String id);

  Future<Result<ChatMessage>> sendMessage({
    required String conversationId,
    required String body,
    String? replyToId,
  });

  Future<Result<void>> deleteMessage({
    required String conversationId,
    required String messageId,
  });

  Future<Result<void>> reactToMessage({
    required String conversationId,
    required String messageId,
    required String emoji,
  });

  Future<Result<void>> markConversationRead(String conversationId);

  Future<Result<void>> clearConversation(String conversationId);

  Future<Result<void>> setMuted({
    required String conversationId,
    required bool muted,
  });
}
