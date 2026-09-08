import 'package:seyra/features/chat/domain/entities/chat_message.dart';

/// Merges REST history with local/realtime messages by id.
/// Pending/failed local rows are kept until a server id replaces them.
List<ChatMessage> mergeMessagesById({
  required List<ChatMessage> local,
  required List<ChatMessage> remote,
}) {
  final byId = <String, ChatMessage>{
    for (final item in local) item.id: item,
  };
  for (final item in remote) {
    byId[item.id] = item;
  }
  final out = byId.values.toList()
    ..sort((a, b) => a.sentAt.compareTo(b.sentAt));
  return out;
}
