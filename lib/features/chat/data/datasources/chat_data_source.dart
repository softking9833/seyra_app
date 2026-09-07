import 'package:seyra/features/chat/domain/entities/chat_message.dart';
import 'package:seyra/features/chat/domain/entities/conversation.dart';

/// Local/mock or future remote adapter. Domain stays independent of storage.
abstract interface class ChatDataSource {
  Stream<List<Conversation>> watchConversations();

  Stream<List<ChatMessage>> watchMessages(String conversationId);

  Stream<bool> watchPeerTyping(String conversationId);

  Conversation? getConversation(String id);

  ChatMessage sendMessage({
    required String conversationId,
    required String body,
    String? replyToId,
  });

  void deleteMessage({
    required String conversationId,
    required String messageId,
  });

  void reactToMessage({
    required String conversationId,
    required String messageId,
    required String emoji,
  });

  void markConversationRead(String conversationId);

  void clearConversation(String conversationId);

  void setMuted({required String conversationId, required bool muted});
}

final class ChatNotFoundException implements Exception {
  const ChatNotFoundException(this.message);

  final String message;
}
