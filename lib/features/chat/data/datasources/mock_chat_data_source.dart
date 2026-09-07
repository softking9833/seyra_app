import 'dart:async';

import 'package:seyra/features/chat/data/datasources/chat_data_source.dart';
import 'package:seyra/features/chat/domain/entities/chat_message.dart';
import 'package:seyra/features/chat/domain/entities/conversation.dart';

final class MockChatDataSource implements ChatDataSource {
  MockChatDataSource({
    this.deliveryDelay = Duration.zero,
    DateTime? now,
  }) : _now = now ?? DateTime.now() {
    _conversations = _seedConversations(_now);
    _messages = _seedMessages(_now);
  }

  /// When non-zero, mock delivery/read and optional peer replies are delayed.
  final Duration deliveryDelay;
  final DateTime _now;

  late List<Conversation> _conversations;
  late Map<String, List<ChatMessage>> _messages;
  final Map<String, bool> _typing = {};
  int _seq = 0;

  final _conversationsController =
      StreamController<List<Conversation>>.broadcast();
  final Map<String, StreamController<List<ChatMessage>>> _messageControllers =
      {};
  final Map<String, StreamController<bool>> _typingControllers = {};

  static const _me = ChatMessage.localUserId;

  @override
  Stream<List<Conversation>> watchConversations() async* {
    yield _sortedConversations();
    yield* _conversationsController.stream;
  }

  @override
  Stream<List<ChatMessage>> watchMessages(String conversationId) async* {
    yield List.unmodifiable(_messages[conversationId] ?? const []);
    yield* _controllerFor(conversationId).stream;
  }

  @override
  Stream<bool> watchPeerTyping(String conversationId) async* {
    yield _typing[conversationId] ?? false;
    yield* _typingControllerFor(conversationId).stream;
  }

  @override
  Conversation? getConversation(String id) {
    for (final item in _conversations) {
      if (item.id == id) {
        return item;
      }
    }
    return null;
  }

  @override
  ChatMessage sendMessage({
    required String conversationId,
    required String body,
    String? replyToId,
  }) {
    final conversation = getConversation(conversationId);
    if (conversation == null) {
      throw const ChatNotFoundException('Conversation not found');
    }

    String? replyPreview;
    if (replyToId != null) {
      for (final item in _messages[conversationId] ?? const <ChatMessage>[]) {
        if (item.id == replyToId) {
          replyPreview = item.body;
          break;
        }
      }
    }

    final message = ChatMessage(
      id: 'local-${++_seq}',
      conversationId: conversationId,
      senderId: _me,
      body: body,
      sentAt: DateTime.now(),
      delivery: deliveryDelay == Duration.zero
          ? MessageDelivery.read
          : MessageDelivery.sent,
      replyToId: replyToId,
      replyPreview: replyPreview,
    );

    _append(conversationId, message);
    _updateConversationPreview(conversationId, message.body, message.sentAt);

    if (deliveryDelay > Duration.zero) {
      unawaited(_progressDelivery(conversationId, message.id));
      if (conversation.kind == ConversationKind.direct) {
        unawaited(_mockPeerReply(conversationId));
      }
    }

    return message;
  }

  @override
  void deleteMessage({
    required String conversationId,
    required String messageId,
  }) {
    final items = _messages[conversationId];
    if (items == null) {
      throw const ChatNotFoundException('Conversation not found');
    }
    items.removeWhere((item) => item.id == messageId);
    _emitMessages(conversationId);
    final last = items.isEmpty ? null : items.last;
    _updateConversationPreview(
      conversationId,
      last?.body ?? 'No messages yet',
      last?.sentAt ?? DateTime.now(),
    );
  }

  @override
  void reactToMessage({
    required String conversationId,
    required String messageId,
    required String emoji,
  }) {
    final items = _messages[conversationId];
    if (items == null) {
      throw const ChatNotFoundException('Conversation not found');
    }
    final index = items.indexWhere((item) => item.id == messageId);
    if (index < 0) {
      throw const ChatNotFoundException('Message not found');
    }
    final current = items[index];
    final existing = [...current.reactions];
    final reactionIndex = existing.indexWhere((item) => item.emoji == emoji);
    if (reactionIndex >= 0) {
      final reaction = existing[reactionIndex];
      if (reaction.reactedByMe) {
        if (reaction.count <= 1) {
          existing.removeAt(reactionIndex);
        } else {
          existing[reactionIndex] = MessageReaction(
            emoji: emoji,
            count: reaction.count - 1,
            reactedByMe: false,
          );
        }
      } else {
        existing[reactionIndex] = MessageReaction(
          emoji: emoji,
          count: reaction.count + 1,
          reactedByMe: true,
        );
      }
    } else {
      existing.add(
        MessageReaction(emoji: emoji, count: 1, reactedByMe: true),
      );
    }
    items[index] = current.copyWith(reactions: existing);
    _emitMessages(conversationId);
  }

  @override
  void markConversationRead(String conversationId) {
    final conversation = getConversation(conversationId);
    if (conversation == null) {
      throw const ChatNotFoundException('Conversation not found');
    }
    _replaceConversation(conversation.copyWith(unreadCount: 0));
  }

  @override
  void clearConversation(String conversationId) {
    if (getConversation(conversationId) == null) {
      throw const ChatNotFoundException('Conversation not found');
    }
    _messages[conversationId] = [];
    _emitMessages(conversationId);
    _updateConversationPreview(
      conversationId,
      'No messages yet',
      DateTime.now(),
    );
  }

  @override
  void setMuted({required String conversationId, required bool muted}) {
    final conversation = getConversation(conversationId);
    if (conversation == null) {
      throw const ChatNotFoundException('Conversation not found');
    }
    _replaceConversation(conversation.copyWith(isMuted: muted));
  }

  void dispose() {
    unawaited(_conversationsController.close());
    for (final controller in _messageControllers.values) {
      unawaited(controller.close());
    }
    for (final controller in _typingControllers.values) {
      unawaited(controller.close());
    }
  }

  List<Conversation> _sortedConversations() {
    final items = [..._conversations];
    items.sort((a, b) {
      if (a.isPinned != b.isPinned) {
        return a.isPinned ? -1 : 1;
      }
      return b.lastMessageAt.compareTo(a.lastMessageAt);
    });
    return List.unmodifiable(items);
  }

  void _append(String conversationId, ChatMessage message) {
    final items = _messages.putIfAbsent(conversationId, () => []);
    items.add(message);
    _emitMessages(conversationId);
  }

  void _replaceMessage(
    String conversationId,
    String messageId,
    ChatMessage Function(ChatMessage current) update,
  ) {
    final items = _messages[conversationId];
    if (items == null) {
      return;
    }
    final index = items.indexWhere((item) => item.id == messageId);
    if (index < 0) {
      return;
    }
    items[index] = update(items[index]);
    _emitMessages(conversationId);
  }

  void _replaceConversation(Conversation conversation) {
    _conversations = [
      for (final item in _conversations)
        if (item.id == conversation.id) conversation else item,
    ];
    _conversationsController.add(_sortedConversations());
  }

  void _updateConversationPreview(
    String conversationId,
    String preview,
    DateTime at,
  ) {
    final conversation = getConversation(conversationId);
    if (conversation == null) {
      return;
    }
    _replaceConversation(
      conversation.copyWith(
        lastMessagePreview: preview,
        lastMessageAt: at,
        unreadCount: 0,
      ),
    );
  }

  StreamController<List<ChatMessage>> _controllerFor(String id) {
    return _messageControllers.putIfAbsent(
      id,
      StreamController<List<ChatMessage>>.broadcast,
    );
  }

  StreamController<bool> _typingControllerFor(String id) {
    return _typingControllers.putIfAbsent(
      id,
      StreamController<bool>.broadcast,
    );
  }

  void _emitMessages(String conversationId) {
    _controllerFor(conversationId).add(
      List.unmodifiable(_messages[conversationId] ?? const []),
    );
  }

  void _setTyping(String conversationId, bool value) {
    _typing[conversationId] = value;
    _typingControllerFor(conversationId).add(value);
  }

  Future<void> _progressDelivery(
    String conversationId,
    String messageId,
  ) async {
    await Future<void>.delayed(deliveryDelay);
    _replaceMessage(
      conversationId,
      messageId,
      (current) => current.copyWith(delivery: MessageDelivery.delivered),
    );
    await Future<void>.delayed(deliveryDelay);
    _replaceMessage(
      conversationId,
      messageId,
      (current) => current.copyWith(delivery: MessageDelivery.read),
    );
  }

  Future<void> _mockPeerReply(String conversationId) async {
    await Future<void>.delayed(deliveryDelay);
    _setTyping(conversationId, true);
    await Future<void>.delayed(deliveryDelay);
    _setTyping(conversationId, false);
    final conversation = getConversation(conversationId);
    if (conversation == null) {
      return;
    }
    final reply = ChatMessage(
      id: 'peer-${++_seq}',
      conversationId: conversationId,
      senderId: 'peer-$conversationId',
      body: 'Got it — talking on Seyra.',
      sentAt: DateTime.now(),
      delivery: MessageDelivery.read,
    );
    _append(conversationId, reply);
    _updateConversationPreview(conversationId, reply.body, reply.sentAt);
  }

  static List<Conversation> _seedConversations(DateTime now) {
    final today = DateTime(now.year, now.month, now.day, 12, 4);
    return [
      Conversation(
        id: '1',
        title: 'Alex Carter',
        initials: 'AC',
        kind: ConversationKind.direct,
        lastMessagePreview: 'Did you get the encrypted files?',
        lastMessageAt: today,
        unreadCount: 2,
        isOnline: true,
        isPinned: true,
        statusText: 'online',
      ),
      Conversation(
        id: '2',
        title: 'Design Team',
        initials: 'DT',
        kind: ConversationKind.group,
        lastMessagePreview: 'Maya: Let\'s ship the new header tonight.',
        lastMessageAt: DateTime(now.year, now.month, now.day, 11, 41),
        isPinned: true,
        statusText: '12 members',
      ),
      Conversation(
        id: '3',
        title: 'Seyra News',
        initials: 'SN',
        kind: ConversationKind.channel,
        lastMessagePreview: 'Privacy update is live for everyone.',
        lastMessageAt: DateTime(now.year, now.month, now.day - 1, 18, 20),
        unreadCount: 12,
        statusText: 'Channel',
      ),
      Conversation(
        id: '4',
        title: 'Jordan Lee',
        initials: 'JL',
        kind: ConversationKind.direct,
        lastMessagePreview: 'Voice message',
        lastMessageAt: DateTime(now.year, now.month, now.day, 10, 12),
        isMuted: true,
        statusText: 'last seen recently',
      ),
      Conversation(
        id: '5',
        title: 'Maya Chen',
        initials: 'MC',
        kind: ConversationKind.direct,
        lastMessagePreview: 'See you at 6. I\'ll bring the keys.',
        lastMessageAt: DateTime(now.year, now.month, now.day, 9, 18),
        unreadCount: 1,
        isOnline: true,
        statusText: 'online',
      ),
      Conversation(
        id: '6',
        title: 'Weekend Hikers',
        initials: 'WH',
        kind: ConversationKind.group,
        lastMessagePreview: 'Sam: Trail photos are in the album.',
        lastMessageAt: now.subtract(const Duration(days: 2, hours: 3)),
        statusText: '8 members',
      ),
      Conversation(
        id: '7',
        title: 'Security Tips',
        initials: 'ST',
        kind: ConversationKind.channel,
        lastMessagePreview: 'How to lock down your account in 2 minutes.',
        lastMessageAt: now.subtract(const Duration(days: 3, hours: 5)),
        statusText: 'Channel',
      ),
      Conversation(
        id: '8',
        title: 'Noah Patel',
        initials: 'NP',
        kind: ConversationKind.direct,
        lastMessagePreview: 'Thanks — received and deleted on my side.',
        lastMessageAt: now.subtract(const Duration(days: 4, hours: 2)),
        statusText: 'last seen Saturday',
      ),
    ];
  }

  static Map<String, List<ChatMessage>> _seedMessages(DateTime now) {
    DateTime today(int hour, int minute) =>
        DateTime(now.year, now.month, now.day, hour, minute);
    DateTime daysAgo(int days, int hour, int minute) =>
        DateTime(now.year, now.month, now.day - days, hour, minute);

    return {
      '1': [
        ChatMessage(
          id: '1-a',
          conversationId: '1',
          senderId: 'peer-1',
          body: 'Are you around this afternoon?',
          sentAt: today(11, 52),
          delivery: MessageDelivery.read,
        ),
        ChatMessage(
          id: '1-b',
          conversationId: '1',
          senderId: _me,
          body: 'Yes — finishing the Seyra chat screen now.',
          sentAt: today(11, 54),
          delivery: MessageDelivery.read,
          replyToId: '1-a',
          replyPreview: 'Are you around this afternoon?',
        ),
        ChatMessage(
          id: '1-c',
          conversationId: '1',
          senderId: 'peer-1',
          body: 'Did you get the encrypted files?',
          sentAt: today(12, 4),
          delivery: MessageDelivery.read,
          reactions: const [
            MessageReaction(emoji: '👍', count: 1, reactedByMe: false),
          ],
        ),
      ],
      '2': [
        ChatMessage(
          id: '2-a',
          conversationId: '2',
          senderId: 'maya',
          body: 'Header mock is in Figma.',
          sentAt: today(11, 20),
          delivery: MessageDelivery.read,
        ),
        ChatMessage(
          id: '2-b',
          conversationId: '2',
          senderId: _me,
          body: 'Looks clean. I\'ll match the bubbles.',
          sentAt: today(11, 28),
          delivery: MessageDelivery.read,
        ),
        ChatMessage(
          id: '2-c',
          conversationId: '2',
          senderId: 'maya',
          body: 'Let\'s ship the new header tonight.',
          sentAt: today(11, 41),
          delivery: MessageDelivery.read,
        ),
      ],
      '3': [
        ChatMessage(
          id: '3-a',
          conversationId: '3',
          senderId: 'channel-3',
          body: 'Privacy update is live for everyone.',
          sentAt: daysAgo(1, 18, 20),
          delivery: MessageDelivery.read,
        ),
      ],
      '4': [
        ChatMessage(
          id: '4-a',
          conversationId: '4',
          senderId: 'peer-4',
          body: 'Voice message',
          sentAt: today(10, 12),
          delivery: MessageDelivery.read,
        ),
      ],
      '5': [
        ChatMessage(
          id: '5-a',
          conversationId: '5',
          senderId: 'peer-5',
          body: 'See you at 6. I\'ll bring the keys.',
          sentAt: today(9, 18),
          delivery: MessageDelivery.read,
        ),
      ],
      '6': [
        ChatMessage(
          id: '6-a',
          conversationId: '6',
          senderId: 'sam',
          body: 'Trail photos are in the album.',
          sentAt: daysAgo(2, 16, 10),
          delivery: MessageDelivery.read,
        ),
      ],
      '7': [
        ChatMessage(
          id: '7-a',
          conversationId: '7',
          senderId: 'channel-7',
          body: 'How to lock down your account in 2 minutes.',
          sentAt: daysAgo(3, 9, 0),
          delivery: MessageDelivery.read,
        ),
      ],
      '8': [
        ChatMessage(
          id: '8-a',
          conversationId: '8',
          senderId: _me,
          body: 'Files received. Deleting my local copy.',
          sentAt: daysAgo(4, 14, 2),
          delivery: MessageDelivery.read,
        ),
        ChatMessage(
          id: '8-b',
          conversationId: '8',
          senderId: 'peer-8',
          body: 'Thanks — received and deleted on my side.',
          sentAt: daysAgo(4, 14, 8),
          delivery: MessageDelivery.read,
        ),
      ],
    };
  }
}
