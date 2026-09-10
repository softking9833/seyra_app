import 'package:seyra/features/chat/domain/entities/chat_message.dart';

/// Shown when Signal decrypt fails. Must stay in sync with hydrate fallback.
const e2eDecryptPlaceholder = 'Encrypted message';

bool isE2eDecryptPlaceholder(ChatMessage message) =>
    message.e2e && message.body == e2eDecryptPlaceholder;

/// Merges REST history with local/realtime messages by id.
/// Pending/failed local rows are kept until a server id replaces them.
/// A successful local decrypt is not replaced by a later decrypt failure.
List<ChatMessage> mergeMessagesById({
  required List<ChatMessage> local,
  required List<ChatMessage> remote,
}) {
  final byId = <String, ChatMessage>{
    for (final item in local) item.id: item,
  };
  for (final item in remote) {
    final previous = byId[item.id];
    if (previous != null &&
        isE2eDecryptPlaceholder(item) &&
        !isE2eDecryptPlaceholder(previous)) {
      continue;
    }
    byId[item.id] = item;
  }
  final out = byId.values.toList()
    ..sort((a, b) => a.sentAt.compareTo(b.sentAt));
  return out;
}
