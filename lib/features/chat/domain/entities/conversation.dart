enum ConversationKind { direct, group, channel }

final class Conversation {
  const Conversation({
    required this.id,
    required this.title,
    required this.initials,
    required this.kind,
    required this.lastMessagePreview,
    required this.lastMessageAt,
    this.unreadCount = 0,
    this.isOnline = false,
    this.isPinned = false,
    this.isMuted = false,
    this.statusText = '',
  });

  final String id;
  final String title;
  final String initials;
  final ConversationKind kind;
  final String lastMessagePreview;
  final DateTime lastMessageAt;
  final int unreadCount;
  final bool isOnline;
  final bool isPinned;
  final bool isMuted;
  final String statusText;

  Conversation copyWith({
    String? lastMessagePreview,
    DateTime? lastMessageAt,
    int? unreadCount,
    bool? isMuted,
  }) {
    return Conversation(
      id: id,
      title: title,
      initials: initials,
      kind: kind,
      lastMessagePreview: lastMessagePreview ?? this.lastMessagePreview,
      lastMessageAt: lastMessageAt ?? this.lastMessageAt,
      unreadCount: unreadCount ?? this.unreadCount,
      isOnline: isOnline,
      isPinned: isPinned,
      isMuted: isMuted ?? this.isMuted,
      statusText: statusText,
    );
  }
}
