import 'dart:async';
import 'dart:convert';

import 'package:seyra/core/network/api_client.dart';
import 'package:seyra/core/storage/secure_storage.dart';
import 'package:seyra/features/auth/data/datasources/auth_remote_data_source.dart';
import 'package:seyra/features/auth/data/exceptions/auth_remote_exceptions.dart';
import 'package:seyra/features/auth/data/storage/auth_secure_storage_keys.dart';
import 'package:seyra/features/chat/data/contracts/chat_api_endpoints.dart';
import 'package:seyra/features/chat/data/crypto/signal_e2e_service.dart';
import 'package:seyra/features/chat/data/datasources/chat_data_source.dart';
import 'package:seyra/features/chat/data/exceptions/chat_remote_exceptions.dart';
import 'package:seyra/features/chat/data/models/chat_message_model.dart';
import 'package:seyra/features/chat/data/models/conversation_summary_model.dart';
import 'package:seyra/features/chat/data/models/message_sync.dart';
import 'package:seyra/features/chat/data/realtime/chat_realtime_port.dart';
import 'package:seyra/features/chat/domain/entities/chat_message.dart';
import 'package:seyra/features/chat/domain/entities/conversation.dart';
import 'package:seyra/features/chat/domain/entities/user_preview.dart';
import 'package:seyra/features/chat/domain/entities/room_member.dart';
import 'package:seyra/features/notifications/domain/entities/notification_models.dart';

final class HttpChatDataSource implements ChatDataSource {
  HttpChatDataSource({
    required this.apiClient,
    required this.secureStorage,
    required this.authRemote,
    required this.baseUrl,
    required this.realtime,
  });

  final ApiClient apiClient;
  final SecureStorage secureStorage;
  final AuthRemoteDataSource authRemote;
  final Uri baseUrl;
  final ChatRealtimePort realtime;

  String _currentUserId = '';
  var _started = false;
  List<Conversation> _conversations = [];
  final Map<String, List<ChatMessage>> _messages = {};
  final _conversationsController =
      StreamController<List<Conversation>>.broadcast();
  final Map<String, StreamController<List<ChatMessage>>> _messageControllers =
      {};
  StreamSubscription<Map<String, dynamic>>? _realtimeSub;
  String? _activeConversationId;
  var _pendingSeq = 0;
  final _alertsController = StreamController<IncomingAlert>.broadcast();
  final _callSignals = StreamController<Map<String, dynamic>>.broadcast();
  final Map<String, StreamController<bool>> _typingControllers = {};
  SignalE2eService? _e2e;

  @override
  String get currentUserId => _currentUserId;

  @override
  Stream<List<Conversation>> watchConversations() async* {
    await _ensureStarted();
    yield List<Conversation>.from(_conversations);
    yield* _conversationsController.stream;
  }

  @override
  Stream<List<ChatMessage>> watchMessages(String conversationId) async* {
    await _ensureStarted();
    await _loadMessages(conversationId);
    yield List<ChatMessage>.from(_messages[conversationId] ?? const []);
    yield* _controllerFor(conversationId).stream;
  }

  @override
  Stream<bool> watchPeerTyping(String conversationId) async* {
    yield false;
    yield* _typingControllers
        .putIfAbsent(conversationId, StreamController<bool>.broadcast)
        .stream;
  }

  @override
  Stream<IncomingAlert> watchIncomingAlerts() => _alertsController.stream;

  @override
  Conversation? getConversation(String id) {
    for (final item in _conversations) {
      if (item.id == id) {
        return item;
      }
    }
    return null;
  }

  List<ChatMessage> peekMessages(String conversationId) {
    return List<ChatMessage>.from(_messages[conversationId] ?? const []);
  }

  @override
  Future<Conversation> startDirectChat(String username) async {
    await _ensureStarted();
    final response = await _authorized(
      method: 'POST',
      path: ChatApiEndpoints.chats,
      jsonBody: {'username': username},
    );
    final conversation = ConversationSummaryModel.fromJson(
      _decodeObject(response.body),
    ).toEntity();
    _upsertConversation(conversation);
    return conversation;
  }

  @override
  Future<Conversation> startGroup({
    required String title,
    required List<String> usernames,
  }) {
    return _startRoom(
      path: ChatApiEndpoints.groups,
      title: title,
      usernames: usernames,
    );
  }

  @override
  Future<Conversation> startChannel({
    required String title,
    required List<String> usernames,
    String visibility = 'private',
  }) {
    return _startRoom(
      path: ChatApiEndpoints.channels,
      title: title,
      usernames: usernames,
      visibility: visibility,
    );
  }

  Future<Conversation> _startRoom({
    required String path,
    required String title,
    required List<String> usernames,
    String visibility = 'private',
  }) async {
    await _ensureStarted();
    final response = await _authorized(
      method: 'POST',
      path: path,
      jsonBody: {
        'title': title,
        'usernames': usernames,
        if (visibility.isNotEmpty) 'visibility': visibility,
      },
    );
    final conversation = ConversationSummaryModel.fromJson(
      _decodeObject(response.body),
    ).toEntity();
    _upsertConversation(conversation);
    return conversation;
  }

  @override
  Future<List<RoomMember>> listMembers(String conversationId) async {
    await _ensureStarted();
    final response = await _authorized(
      method: 'GET',
      path: ChatApiEndpoints.members(conversationId),
    );
    return _parseMembers(_decodeObject(response.body));
  }

  @override
  Future<List<RoomMember>> addMembers({
    required String conversationId,
    required List<String> usernames,
  }) async {
    await _ensureStarted();
    final response = await _authorized(
      method: 'POST',
      path: ChatApiEndpoints.members(conversationId),
      jsonBody: {'usernames': usernames},
    );
    unawaited(refreshConversations());
    return _parseMembers(_decodeObject(response.body));
  }

  @override
  Future<void> removeMember({
    required String conversationId,
    required String userId,
  }) async {
    await _ensureStarted();
    await _authorized(
      method: 'DELETE',
      path: ChatApiEndpoints.member(conversationId, userId),
    );
    unawaited(refreshConversations());
  }

  @override
  Future<void> setMemberRole({
    required String conversationId,
    required String userId,
    required MemberRole role,
  }) async {
    await _ensureStarted();
    final value = switch (role) {
      MemberRole.owner => 'owner',
      MemberRole.admin => 'admin',
      MemberRole.member => 'member',
    };
    await _authorized(
      method: 'POST',
      path: ChatApiEndpoints.memberRole(conversationId, userId),
      jsonBody: {'role': value},
    );
  }

  @override
  Future<void> leaveConversation(String conversationId) async {
    await _ensureStarted();
    await _authorized(
      method: 'POST',
      path: ChatApiEndpoints.leave(conversationId),
    );
    _conversations = _conversations.where((item) => item.id != conversationId).toList();
    _conversationsController.add(List<Conversation>.from(_conversations));
  }

  List<RoomMember> _parseMembers(Map<String, dynamic> json) {
    final raw = json['members'];
    final out = <RoomMember>[];
    if (raw is! List) {
      return out;
    }
    for (final item in raw) {
      if (item is Map<String, dynamic>) {
        out.add(
          RoomMember(
            id: item['id'] as String? ?? '',
            username: item['username'] as String? ?? '',
            role: memberRoleFromApi(item['role'] as String?),
          ),
        );
      }
    }
    return out;
  }

  @override
  Future<List<UserPreview>> searchUsers(String query) async {
    await _ensureStarted();
    final encoded = Uri.encodeQueryComponent(query.trim());
    final response = await _authorized(
      method: 'GET',
      path: '${ChatApiEndpoints.userSearch}?q=$encoded',
    );
    final json = _decodeObject(response.body);
    final raw = json['users'];
    final users = <UserPreview>[];
    if (raw is List) {
      for (final item in raw) {
        if (item is Map<String, dynamic>) {
          final id = item['id'] as String? ?? '';
          final username = item['username'] as String? ?? '';
          if (id.isEmpty || username.isEmpty) {
            continue;
          }
          users.add(UserPreview(id: id, username: username));
        }
      }
    }
    return users;
  }

  @override
  Future<void> refreshConversations() async {
    await _ensureStarted();
    await _loadConversations();
  }

  @override
  void setActiveConversation(String? conversationId) {
    _activeConversationId = conversationId;
  }

  @override
  Future<ChatMessage> retryMessage({
    required String conversationId,
    required String messageId,
  }) async {
    await _ensureStarted();
    final items = _messages[conversationId] ?? const <ChatMessage>[];
    ChatMessage? failed;
    for (final item in items) {
      if (item.id == messageId) {
        failed = item;
        break;
      }
    }
    if (failed == null) {
      throw const ChatNotFoundException('Message not found');
    }
    return _sendWithLocal(
      conversationId: conversationId,
      body: failed.body,
      replaceId: messageId,
    );
  }

  @override
  Future<ChatMessage> sendMessage({
    required String conversationId,
    required String body,
    String? replyToId,
    String? attachmentId,
  }) async {
    await _ensureStarted();
    return _sendWithLocal(
      conversationId: conversationId,
      body: body,
      replyToId: replyToId,
      attachmentId: attachmentId,
    );
  }

  Future<ChatMessage> _sendWithLocal({
    required String conversationId,
    required String body,
    String? replyToId,
    String? replaceId,
    String? attachmentId,
  }) async {
    final localId = replaceId ?? 'pending_${++_pendingSeq}';
    var outbound = body;
    var e2e = false;
    final conversation = getConversation(conversationId);
    if (conversation?.kind == ConversationKind.direct && body.isNotEmpty) {
      try {
        await _ensureE2e();
        try {
          await publishLocalKeys();
        } catch (_) {}
        final peerId = conversation!.peerId;
        if (peerId.isNotEmpty) {
          await establishSession(peerId);
          final cipher = await _e2e!.encrypt(peerUserId: peerId, plaintext: body);
          if (cipher != null) {
            outbound = cipher;
            e2e = true;
          }
        }
      } catch (_) {}
    }
    if (_containsAttachmentKeys(body) && !e2e) {
      throw const ChatRemoteException(ChatRemoteErrorCode.invalidInput);
    }
    final pending = ChatMessage(
      id: localId,
      conversationId: conversationId,
      senderId: _currentUserId,
      body: body,
      sentAt: DateTime.now(),
      delivery: MessageDelivery.sending,
      attachmentId: attachmentId,
      e2e: e2e,
    );
    if (replaceId == null) {
      _appendMessage(pending, countUnread: false);
    } else {
      _replaceMessage(conversationId, localId, pending);
    }
    try {
      final response = await _authorized(
        method: 'POST',
        path: ChatApiEndpoints.messages(conversationId),
        jsonBody: {
          'body': outbound,
          if (replyToId != null && replyToId.isNotEmpty) 'reply_to_id': replyToId,
          if (attachmentId != null && attachmentId.isNotEmpty)
            'attachment_id': attachmentId,
          if (e2e) 'e2e': true,
        },
      );
      final message = await _hydrateMessage(
        ChatMessageModel.fromJson(_decodeObject(response.body)),
      );
      _removeMessage(conversationId, localId);
      _appendMessage(message, countUnread: false);
      return message;
    } catch (error) {
      _replaceMessage(
        conversationId,
        localId,
        pending.copyWith(delivery: MessageDelivery.failed),
      );
      rethrow;
    }
  }

  @override
  Future<void> deleteMessage({
    required String conversationId,
    required String messageId,
  }) async {
    await _authorized(
      method: 'DELETE',
      path: ChatApiEndpoints.message(conversationId, messageId),
    );
    final items = _messages[conversationId];
    if (items == null) {
      return;
    }
    items.removeWhere((item) => item.id == messageId);
    _emitMessages(conversationId);
  }

  @override
  Future<void> reactToMessage({
    required String conversationId,
    required String messageId,
    required String emoji,
  }) async {
    await _ensureStarted();
    await _authorized(
      method: 'PUT',
      path: ChatApiEndpoints.reactions(conversationId, messageId),
      jsonBody: {'emoji': emoji},
    );
  }

  @override
  Future<void> markConversationRead(String conversationId) async {
    final conversation = getConversation(conversationId);
    if (conversation == null) {
      return;
    }
    _upsertConversation(conversation.copyWith(unreadCount: 0));
  }

  @override
  Future<void> clearConversation(String conversationId) async {
    _messages[conversationId] = [];
    _emitMessages(conversationId);
  }

  @override
  Future<void> setMuted({
    required String conversationId,
    required bool muted,
  }) async {
    await _authorized(
      method: 'PUT',
      path: '/v1/chats/$conversationId/mute',
      jsonBody: {'value': muted},
    );
    final conversation = getConversation(conversationId);
    if (conversation == null) {
      return;
    }
    _upsertConversation(conversation.copyWith(isMuted: muted));
  }

  Future<void> _ensureStarted() async {
    if (_started) {
      return;
    }
    _started = true;
    try {
      final account = await authRemote.getCurrentAccount();
      _currentUserId = account.id;
    } on AuthRemoteException catch (error) {
      throw ChatRemoteException(_fromAuth(error.code));
    }
    await _loadConversations();
    await _connectRealtime();
  }

  Future<void> _loadConversations() async {
    final response = await _authorized(
      method: 'GET',
      path: ChatApiEndpoints.chats,
    );
    final json = _decodeObject(response.body);
    final raw = json['chats'];
    final chats = <Conversation>[];
    if (raw is List) {
      for (final item in raw) {
        if (item is Map<String, dynamic>) {
          chats.add(ConversationSummaryModel.fromJson(item).toEntity());
        }
      }
    }
    _conversations = chats;
    _conversationsController.add(List<Conversation>.from(_conversations));
  }

  Future<void> _loadMessages(String conversationId) async {
    final response = await _authorized(
      method: 'GET',
      path: ChatApiEndpoints.messages(conversationId),
    );
    final json = _decodeObject(response.body);
    final raw = json['messages'];
    final items = <ChatMessage>[];
    if (raw is List) {
      for (final item in raw) {
        if (item is Map<String, dynamic>) {
          items.add(await _hydrateMessage(ChatMessageModel.fromJson(item)));
        }
      }
    }
    final keptFailed = (_messages[conversationId] ?? const <ChatMessage>[])
        .where((item) => item.delivery == MessageDelivery.failed)
        .toList();
    _messages[conversationId] = mergeMessagesById(
      local: keptFailed,
      remote: items,
    );
    _emitMessages(conversationId);
  }

  Future<void> _connectRealtime() async {
    final access = await secureStorage.read(AuthSecureStorageKeys.accessToken);
    if (access == null || access.isEmpty) {
      return;
    }
    final wsUrl = _wsUri(baseUrl.resolve(ChatApiEndpoints.realtime));
    await _realtimeSub?.cancel();
    _realtimeSub = realtime.connect(uri: wsUrl, accessToken: access).listen(
      _onRealtime,
      onError: (_) {
        unawaited(_resyncAfterReconnect());
      },
    );
  }

  Future<void> _resyncAfterReconnect() async {
    try {
      await _loadConversations();
      for (final id in _messages.keys.toList()) {
        await _loadMessages(id);
      }
    } catch (_) {}
  }

  void _onRealtime(Map<String, dynamic> event) {
    unawaited(_handleRealtime(event));
  }

  Future<void> _handleRealtime(Map<String, dynamic> event) async {
    final type = event['type'] as String?;
    if (type == 'realtime.connected') {
      unawaited(_resyncAfterReconnect());
      return;
    }
    final payload = event['payload'];
    if (payload is! Map<String, dynamic>) {
      return;
    }
    if (type == 'message.created') {
      final message = await _hydrateMessage(ChatMessageModel.fromJson(payload));
      _appendMessage(message);
    } else if (type == 'message.updated') {
      unawaited(_loadMessages(payload['conversation_id'] as String? ?? ''));
    } else if (type == 'message.deleted') {
      final id = payload['id'] as String?;
      final conversationId = payload['conversation_id'] as String?;
      if (id == null || conversationId == null) {
        return;
      }
      _messages[conversationId]?.removeWhere((item) => item.id == id);
      _emitMessages(conversationId);
    } else if (type == 'call.signal') {
      _callSignals.add(payload);
    } else if (type == 'typing') {
      final conversationId = payload['conversation_id'] as String? ?? '';
      final userId = payload['user_id'] as String? ?? '';
      if (conversationId.isEmpty || userId == _currentUserId) {
        return;
      }
      _typingControllers
          .putIfAbsent(conversationId, StreamController<bool>.broadcast)
          .add(payload['typing'] == true);
    } else if (type == 'message.pinned' ||
        type == 'message.unpinned' ||
        type == 'member.joined' ||
        type == 'member.left' ||
        type == 'member.updated') {
      unawaited(refreshConversations());
    }
  }

  void _appendMessage(ChatMessage message, {bool countUnread = true}) {
    final items = _messages.putIfAbsent(message.conversationId, () => []);
    if (items.any((item) => item.id == message.id)) {
      return;
    }
    items.add(message);
    items.sort((a, b) => a.sentAt.compareTo(b.sentAt));
    _emitMessages(message.conversationId);
    final conversation = getConversation(message.conversationId);
    if (conversation != null) {
      final fromPeer = message.senderId != _currentUserId;
      final incrementUnread = countUnread &&
          fromPeer &&
          message.conversationId != _activeConversationId;
      final unread = incrementUnread
          ? conversation.unreadCount + 1
          : message.conversationId == _activeConversationId
          ? 0
          : conversation.unreadCount;
      _upsertConversation(
        conversation.copyWith(
          lastMessagePreview: message.body,
          lastMessageAt: message.sentAt,
          unreadCount: unread,
        ),
      );
    }
    if (countUnread &&
        message.senderId != _currentUserId &&
        message.conversationId != _activeConversationId) {
      final title = conversation?.title ?? 'Seyra';
      _alertsController.add(
        IncomingAlert(
          conversationId: message.conversationId,
          title: title,
          body: message.body,
          messageId: message.id,
        ),
      );
    }
  }

  void _replaceMessage(
    String conversationId,
    String messageId,
    ChatMessage next,
  ) {
    final items = _messages[conversationId];
    if (items == null) {
      return;
    }
    final index = items.indexWhere((item) => item.id == messageId);
    if (index < 0) {
      items.add(next);
    } else {
      items[index] = next;
    }
    _emitMessages(conversationId);
  }

  void _removeMessage(String conversationId, String messageId) {
    _messages[conversationId]?.removeWhere((item) => item.id == messageId);
  }

  void _upsertConversation(Conversation conversation) {
    final exists = _conversations.any((item) => item.id == conversation.id);
    if (exists) {
      _conversations = [
        for (final item in _conversations)
          if (item.id == conversation.id) conversation else item,
      ];
    } else {
      _conversations = [..._conversations, conversation];
    }
    _conversations.sort((a, b) => b.lastMessageAt.compareTo(a.lastMessageAt));
    _conversationsController.add(List<Conversation>.from(_conversations));
  }

  StreamController<List<ChatMessage>> _controllerFor(String id) {
    return _messageControllers.putIfAbsent(
      id,
      StreamController<List<ChatMessage>>.broadcast,
    );
  }

  void _emitMessages(String conversationId) {
    _controllerFor(conversationId).add(
      List<ChatMessage>.from(_messages[conversationId] ?? const []),
    );
  }

  Future<ApiResponse> _authorized({
    required String method,
    required String path,
    Object? jsonBody,
  }) async {
    var access = await secureStorage.read(AuthSecureStorageKeys.accessToken);
    if (access == null || access.isEmpty) {
      throw const ChatRemoteException(ChatRemoteErrorCode.unauthorized);
    }
    try {
      return await _send(
        method: method,
        path: path,
        headers: {'Authorization': 'Bearer $access'},
        jsonBody: jsonBody,
      );
    } on ChatRemoteException catch (error) {
      if (error.code != ChatRemoteErrorCode.sessionExpired) {
        rethrow;
      }
      try {
        await authRemote.refreshSession();
      } catch (_) {
        throw const ChatRemoteException(ChatRemoteErrorCode.sessionExpired);
      }
      access = await secureStorage.read(AuthSecureStorageKeys.accessToken);
      return _send(
        method: method,
        path: path,
        headers: {'Authorization': 'Bearer $access'},
        jsonBody: jsonBody,
      );
    }
  }

  Future<ApiResponse> _send({
    required String method,
    required String path,
    Map<String, String>? headers,
    Object? jsonBody,
  }) async {
    try {
      final response = await apiClient.send(
        method: method,
        uri: baseUrl.resolve(path),
        headers: headers,
        jsonBody: jsonBody,
      );
      if (response.statusCode >= 200 && response.statusCode < 300) {
        return response;
      }
      throw ChatRemoteException(
        _codeFor(response),
        statusCode: response.statusCode,
      );
    } on ChatRemoteException {
      rethrow;
    } catch (_) {
      throw const ChatRemoteException(ChatRemoteErrorCode.network);
    }
  }

  ChatRemoteErrorCode _codeFor(ApiResponse response) {
    final code = _errorCode(response.body);
    return switch (code) {
      'not_found' => ChatRemoteErrorCode.notFound,
      'forbidden' => ChatRemoteErrorCode.forbidden,
      'cannot_message_self' => ChatRemoteErrorCode.cannotMessageSelf,
      'session_expired' => ChatRemoteErrorCode.sessionExpired,
      'unauthorized' => ChatRemoteErrorCode.unauthorized,
      'invalid_input' => ChatRemoteErrorCode.invalidInput,
      'already_member' => ChatRemoteErrorCode.alreadyMember,
      'owner_protected' => ChatRemoteErrorCode.ownerProtected,
      _ => switch (response.statusCode) {
        404 => ChatRemoteErrorCode.notFound,
        403 => ChatRemoteErrorCode.forbidden,
        401 => ChatRemoteErrorCode.unauthorized,
        400 => ChatRemoteErrorCode.invalidInput,
        _ => ChatRemoteErrorCode.network,
      },
    };
  }

  String? _errorCode(String body) {
    if (body.trim().isEmpty) {
      return null;
    }
    try {
      final json = jsonDecode(body);
      if (json is Map<String, dynamic>) {
        final error = json['error'];
        if (error is Map<String, dynamic>) {
          return error['code'] as String?;
        }
      }
    } catch (_) {
      return null;
    }
    return null;
  }

  Map<String, dynamic> _decodeObject(String body) {
    final json = jsonDecode(body);
    if (json is Map<String, dynamic>) {
      return json;
    }
    throw const ChatRemoteException(ChatRemoteErrorCode.network);
  }

  ChatRemoteErrorCode _fromAuth(AuthRemoteErrorCode code) {
    return switch (code) {
      AuthRemoteErrorCode.sessionExpired => ChatRemoteErrorCode.sessionExpired,
      AuthRemoteErrorCode.unauthorized => ChatRemoteErrorCode.unauthorized,
      AuthRemoteErrorCode.network => ChatRemoteErrorCode.network,
      _ => ChatRemoteErrorCode.network,
    };
  }

  Uri _wsUri(Uri httpUri) {
    final scheme = httpUri.scheme == 'https' ? 'wss' : 'ws';
    return httpUri.replace(scheme: scheme);
  }

  Future<ApiResponse> authorized({
    required String method,
    required String path,
    Object? jsonBody,
  }) {
    return _authorized(method: method, path: path, jsonBody: jsonBody);
  }

  Stream<Map<String, dynamic>> watchCallSignals() => _callSignals.stream;

  Future<void> sendRealtime(Map<String, dynamic> event) {
    return realtime.send(event);
  }

  Future<ChatMessage> hydrateMessage(ChatMessageModel model) =>
      _hydrateMessage(model);

  Future<ChatMessage> _hydrateMessage(ChatMessageModel model) async {
    if (!model.e2e || model.body.isEmpty) {
      return model.toEntity();
    }
    try {
      await _ensureE2e();
      final conversation = getConversation(model.conversationId);
      final peerId = model.senderId == _currentUserId
          ? (conversation?.peerId ?? '')
          : model.senderId;
      if (peerId.isEmpty) {
        return model.toEntity();
      }
      final plain = await _e2e!.decrypt(
        peerUserId: peerId,
        ciphertext: model.body,
      );
      return _entityFromPlain(model, plain);
    } catch (_) {
      return model.toEntity(decryptedBody: 'Encrypted message');
    }
  }

  ChatMessage _entityFromPlain(ChatMessageModel model, String plain) {
    try {
      final decoded = jsonDecode(plain);
      if (decoded is Map<String, dynamic> && decoded['v'] == 1) {
        final kind = decoded['kind'] as String?;
        if (kind == 'file') {
          List<int>? key;
          List<int>? nonce;
          try {
            final encodedKey = decoded['k'] as String?;
            final encodedNonce = decoded['n'] as String?;
            if (encodedKey != null && encodedNonce != null) {
              key = base64Decode(encodedKey);
              nonce = base64Decode(encodedNonce);
            }
          } catch (_) {}
          return model.toEntity(
            decryptedBody: decoded['name'] as String? ?? 'Encrypted file',
          ).copyWith(
            attachmentId: decoded['att'] as String? ?? model.attachmentId,
            contentType: decoded['mime'] as String?,
            fileKey: key,
            fileNonce: nonce,
            e2e: true,
          );
        }
        if (kind == 'sticker') {
          return model.toEntity(
            decryptedBody: decoded['emoji'] as String? ?? plain,
          );
        }
      }
    } catch (_) {}
    return model.toEntity(decryptedBody: plain);
  }

  bool _containsAttachmentKeys(String body) {
    try {
      final decoded = jsonDecode(body);
      return decoded is Map<String, dynamic> &&
          decoded['kind'] == 'file' &&
          decoded['k'] is String &&
          decoded['n'] is String;
    } catch (_) {
      return false;
    }
  }

  Future<void> _ensureE2e() async {
    _e2e ??= SignalE2eService(secureStorage);
    await _e2e!.install();
  }

  Future<void> publishLocalKeys() async {
    await _ensureE2e();
    final bundle = await _e2e!.exportPublicBundle();
    await _authorized(method: 'POST', path: '/v1/e2e/keys', jsonBody: bundle);
  }

  Future<void> establishSession(String peerUserId) async {
    await _ensureE2e();
    final response = await _authorized(
      method: 'GET',
      path: '/v1/e2e/bundle/$peerUserId',
    );
    await _e2e!.processRemoteBundle(
      userId: peerUserId,
      bundle: _decodeObject(response.body),
    );
  }
}
