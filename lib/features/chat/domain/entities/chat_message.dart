enum MessageDelivery { sending, sent, delivered, read, failed }

final class MessageReaction {
  const MessageReaction({
    required this.emoji,
    required this.count,
    required this.reactedByMe,
  });

  final String emoji;
  final int count;
  final bool reactedByMe;
}

final class ChatMessage {
  const ChatMessage({
    required this.id,
    required this.conversationId,
    required this.senderId,
    required this.body,
    required this.sentAt,
    required this.delivery,
    this.replyToId,
    this.replyPreview,
    this.reactions = const [],
  });

  final String id;
  final String conversationId;
  final String senderId;
  final String body;
  final DateTime sentAt;
  final MessageDelivery delivery;
  final String? replyToId;
  final String? replyPreview;
  final List<MessageReaction> reactions;

  bool get isMine => senderId == ChatMessage.localUserId;

  static const localUserId = 'local-user';

  ChatMessage copyWith({
    MessageDelivery? delivery,
    List<MessageReaction>? reactions,
  }) {
    return ChatMessage(
      id: id,
      conversationId: conversationId,
      senderId: senderId,
      body: body,
      sentAt: sentAt,
      delivery: delivery ?? this.delivery,
      replyToId: replyToId,
      replyPreview: replyPreview,
      reactions: reactions ?? this.reactions,
    );
  }
}
