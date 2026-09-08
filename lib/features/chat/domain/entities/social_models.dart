import 'package:seyra/features/chat/domain/entities/chat_message.dart';
import 'package:seyra/features/chat/domain/entities/conversation.dart';
import 'package:seyra/features/chat/domain/entities/user_preview.dart';

class PrivacySettings {
  const PrivacySettings({
    this.lastSeenVisible = true,
    this.readReceipts = true,
    this.typingVisible = true,
    this.profileVisible = true,
    this.notificationPreview = true,
  });

  final bool lastSeenVisible;
  final bool readReceipts;
  final bool typingVisible;
  final bool profileVisible;
  final bool notificationPreview;

  PrivacySettings copyWith({
    bool? lastSeenVisible,
    bool? readReceipts,
    bool? typingVisible,
    bool? profileVisible,
    bool? notificationPreview,
  }) {
    return PrivacySettings(
      lastSeenVisible: lastSeenVisible ?? this.lastSeenVisible,
      readReceipts: readReceipts ?? this.readReceipts,
      typingVisible: typingVisible ?? this.typingVisible,
      profileVisible: profileVisible ?? this.profileVisible,
      notificationPreview: notificationPreview ?? this.notificationPreview,
    );
  }
}

class AuthDeviceSession {
  const AuthDeviceSession({
    required this.id,
    required this.expiresAt,
    this.revoked = false,
  });

  final String id;
  final DateTime expiresAt;
  final bool revoked;
}

class CallRecord {
  const CallRecord({
    required this.id,
    required this.conversationId,
    required this.callerId,
    required this.kind,
    required this.state,
    required this.createdAt,
    this.endedAt,
    this.durationSeconds,
  });

  final String id;
  final String conversationId;
  final String callerId;
  final String kind;
  final String state;
  final DateTime createdAt;
  final DateTime? endedAt;
  final int? durationSeconds;
}

class BotAccount {
  const BotAccount({
    required this.id,
    required this.userId,
    required this.username,
    this.token,
  });

  final String id;
  final String userId;
  final String username;
  final String? token;
}

class IceServers {
  const IceServers({required this.servers});

  final List<Map<String, dynamic>> servers;
}

class RoomAttachment {
  const RoomAttachment({
    required this.id,
    required this.filename,
    required this.contentType,
    required this.byteSize,
    required this.e2e,
    required this.createdAt,
  });

  final String id;
  final String filename;
  final String contentType;
  final int byteSize;
  final bool e2e;
  final DateTime createdAt;
}

class StickerItem {
  const StickerItem({
    required this.pack,
    required this.id,
    required this.emoji,
    required this.name,
  });

  final String pack;
  final String id;
  final String emoji;
  final String name;
}

class UploadCancelToken {
  void Function()? _onCancel;
  var cancelled = false;

  void bind(void Function() onCancel) {
    _onCancel = onCancel;
  }

  void cancel() {
    cancelled = true;
    _onCancel?.call();
  }
}

class GlobalSearchResult {
  const GlobalSearchResult({
    this.users = const [],
    this.conversations = const [],
    this.messages = const [],
  });

  final List<UserPreview> users;
  final List<Conversation> conversations;
  final List<ChatMessage> messages;
}
