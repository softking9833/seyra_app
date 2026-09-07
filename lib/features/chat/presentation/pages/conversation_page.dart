import 'package:flutter/material.dart';
import 'package:seyra/core/errors/result.dart';
import 'package:seyra/core/theme/app_colors.dart';
import 'package:seyra/features/chat/domain/entities/chat_message.dart';
import 'package:seyra/features/chat/domain/entities/conversation.dart';
import 'package:seyra/features/chat/domain/usecases/clear_conversation_use_case.dart';
import 'package:seyra/features/chat/domain/usecases/delete_message_use_case.dart';
import 'package:seyra/features/chat/domain/usecases/mark_conversation_read_use_case.dart';
import 'package:seyra/features/chat/domain/usecases/react_to_message_use_case.dart';
import 'package:seyra/features/chat/domain/usecases/send_message_use_case.dart';
import 'package:seyra/features/chat/domain/usecases/set_conversation_muted_use_case.dart';
import 'package:seyra/features/chat/domain/usecases/watch_conversations_use_case.dart';
import 'package:seyra/features/chat/domain/usecases/watch_messages_use_case.dart';
import 'package:seyra/features/chat/domain/usecases/watch_peer_typing_use_case.dart';
import 'package:seyra/features/chat/presentation/formatters/chat_time_format.dart';
import 'package:seyra/features/chat/presentation/widgets/conversation_avatar.dart';
import 'package:seyra/features/chat/presentation/widgets/message_bubble.dart';
import 'package:seyra/features/chat/presentation/widgets/message_composer.dart';

class ConversationPage extends StatefulWidget {
  const ConversationPage({
    super.key,
    required this.conversationId,
    required this.watchConversations,
    required this.watchMessages,
    required this.watchPeerTyping,
    required this.sendMessage,
    required this.deleteMessage,
    required this.reactToMessage,
    required this.markConversationRead,
    required this.clearConversation,
    required this.setMuted,
  });

  final String conversationId;
  final WatchConversationsUseCase watchConversations;
  final WatchMessagesUseCase watchMessages;
  final WatchPeerTypingUseCase watchPeerTyping;
  final SendMessageUseCase sendMessage;
  final DeleteMessageUseCase deleteMessage;
  final ReactToMessageUseCase reactToMessage;
  final MarkConversationReadUseCase markConversationRead;
  final ClearConversationUseCase clearConversation;
  final SetConversationMutedUseCase setMuted;

  @override
  State<ConversationPage> createState() => _ConversationPageState();
}

class _ConversationPageState extends State<ConversationPage> {
  final _composer = TextEditingController();
  bool _hasText = false;
  ChatMessage? _replyTo;
  late final Stream<List<Conversation>> _conversations;
  late final Stream<List<ChatMessage>> _messages;
  late final Stream<bool> _typing;

  static const _emojis = ['😀', '😂', '😍', '👍', '🔥', '🎉', '💙', '🙏'];

  @override
  void initState() {
    super.initState();
    _conversations = widget.watchConversations();
    _messages = widget.watchMessages(widget.conversationId);
    _typing = widget.watchPeerTyping(widget.conversationId);
    widget.markConversationRead(widget.conversationId);
  }

  @override
  void dispose() {
    _composer.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return StreamBuilder<List<Conversation>>(
      stream: _conversations,
      builder: (context, conversationSnapshot) {
        Conversation? conversation;
        for (final item in conversationSnapshot.data ?? const <Conversation>[]) {
          if (item.id == widget.conversationId) {
            conversation = item;
            break;
          }
        }

        return StreamBuilder<List<ChatMessage>>(
          stream: _messages,
          builder: (context, messageSnapshot) {
            return StreamBuilder<bool>(
              stream: _typing,
              builder: (context, typingSnapshot) {
                final typing = typingSnapshot.data ?? false;
                return Scaffold(
                  backgroundColor: const Color(0xFFE8EEF8),
                  appBar: _appBar(conversation, typing),
                  body: Column(
                    children: [
                      Expanded(
                        child: _MessageHistory(
                          messages: messageSnapshot.data ?? const [],
                          typing: typing,
                          onLongPress: _openMessageActions,
                        ),
                      ),
                      MessageComposer(
                        controller: _composer,
                        hasText: _hasText,
                        replyTo: _replyTo,
                        onChanged: (value) {
                          setState(() => _hasText = value.trim().isNotEmpty);
                        },
                        onSend: _send,
                        onAttach: _showAttachments,
                        onEmoji: _showEmojis,
                        onMic: () => _comingSoon('Voice messages'),
                        onCancelReply: () => setState(() => _replyTo = null),
                      ),
                    ],
                  ),
                );
              },
            );
          },
        );
      },
    );
  }

  PreferredSizeWidget _appBar(Conversation? conversation, bool typing) {
    final title = conversation?.title ?? 'Seyra';
    final status = typing
        ? 'typing...'
        : (conversation?.statusText.isNotEmpty == true
              ? conversation!.statusText
              : '');

    return AppBar(
      backgroundColor: Colors.white,
      titleSpacing: 0,
      title: conversation == null
          ? Text(title)
          : Row(
              children: [
                ConversationAvatar(conversation: conversation, size: 40),
                const SizedBox(width: 10),
                Expanded(
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Text(
                        title,
                        maxLines: 1,
                        overflow: TextOverflow.ellipsis,
                        style: const TextStyle(
                          fontWeight: FontWeight.w700,
                          fontSize: 16,
                        ),
                      ),
                      if (status.isNotEmpty)
                        Text(
                          status,
                          maxLines: 1,
                          overflow: TextOverflow.ellipsis,
                          style: TextStyle(
                            fontSize: 12,
                            color: typing
                                ? AppColors.primary
                                : AppColors.textSecondary,
                            fontWeight: typing
                                ? FontWeight.w600
                                : FontWeight.w400,
                          ),
                        ),
                    ],
                  ),
                ),
              ],
            ),
      actions: [
        IconButton(
          tooltip: 'Voice call',
          onPressed: () => _comingSoon('Voice calls'),
          icon: const Icon(Icons.call_outlined),
        ),
        IconButton(
          tooltip: 'Video call',
          onPressed: () => _comingSoon('Video calls'),
          icon: const Icon(Icons.videocam_outlined),
        ),
        PopupMenuButton<String>(
          key: const Key('conversation_more_button'),
          onSelected: (value) {
            if (value == 'mute' && conversation != null) {
              widget.setMuted(
                conversationId: conversation.id,
                muted: !conversation.isMuted,
              );
            } else if (value == 'search') {
              _comingSoon('Search in chat');
            } else if (value == 'clear') {
              widget.clearConversation(widget.conversationId);
            }
          },
          itemBuilder: (context) => [
            PopupMenuItem(
              value: 'mute',
              child: Text(conversation?.isMuted == true ? 'Unmute' : 'Mute'),
            ),
            const PopupMenuItem(value: 'search', child: Text('Search')),
            const PopupMenuItem(value: 'clear', child: Text('Clear history')),
          ],
        ),
      ],
    );
  }

  Future<void> _send() async {
    final body = _composer.text;
    final replyToId = _replyTo?.id;
    final result = await widget.sendMessage(
      conversationId: widget.conversationId,
      body: body,
      replyToId: replyToId,
    );
    if (!mounted) {
      return;
    }
    if (result is Success<ChatMessage>) {
      _composer.clear();
      setState(() {
        _hasText = false;
        _replyTo = null;
      });
    }
  }

  Future<void> _openMessageActions(ChatMessage message) async {
    await showModalBottomSheet<void>(
      context: context,
      showDragHandle: true,
      builder: (context) {
        return SafeArea(
          child: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              Padding(
                padding: const EdgeInsets.symmetric(vertical: 8),
                child: Row(
                  mainAxisAlignment: MainAxisAlignment.spaceEvenly,
                  children: [
                    for (final emoji in ['❤️', '👍', '😂', '😮', '😢'])
                      IconButton(
                        onPressed: () {
                          widget.reactToMessage(
                            conversationId: widget.conversationId,
                            messageId: message.id,
                            emoji: emoji,
                          );
                          Navigator.pop(context);
                        },
                        icon: Text(emoji, style: const TextStyle(fontSize: 22)),
                      ),
                  ],
                ),
              ),
              ListTile(
                leading: const Icon(Icons.reply),
                title: const Text('Reply'),
                onTap: () {
                  Navigator.pop(context);
                  setState(() => _replyTo = message);
                },
              ),
              if (message.isMine)
                ListTile(
                  leading: const Icon(Icons.delete_outline),
                  title: const Text('Delete'),
                  onTap: () {
                    widget.deleteMessage(
                      conversationId: widget.conversationId,
                      messageId: message.id,
                    );
                    Navigator.pop(context);
                  },
                ),
            ],
          ),
        );
      },
    );
  }

  void _showAttachments() {
    showModalBottomSheet<void>(
      context: context,
      showDragHandle: true,
      builder: (context) {
        return SafeArea(
          child: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              ListTile(
                leading: const Icon(Icons.photo_outlined),
                title: const Text('Photo'),
                onTap: () {
                  Navigator.pop(context);
                  _comingSoon('Photo attachments');
                },
              ),
              ListTile(
                leading: const Icon(Icons.insert_drive_file_outlined),
                title: const Text('File'),
                onTap: () {
                  Navigator.pop(context);
                  _comingSoon('File attachments');
                },
              ),
              ListTile(
                leading: const Icon(Icons.location_on_outlined),
                title: const Text('Location'),
                onTap: () {
                  Navigator.pop(context);
                  _comingSoon('Location sharing');
                },
              ),
            ],
          ),
        );
      },
    );
  }

  void _showEmojis() {
    showModalBottomSheet<void>(
      context: context,
      showDragHandle: true,
      builder: (context) {
        return SafeArea(
          child: Padding(
            padding: const EdgeInsets.fromLTRB(16, 8, 16, 24),
            child: Wrap(
              spacing: 12,
              runSpacing: 12,
              children: [
                for (final emoji in _emojis)
                  GestureDetector(
                    onTap: () {
                      _composer.text += emoji;
                      _composer.selection = TextSelection.collapsed(
                        offset: _composer.text.length,
                      );
                      setState(() => _hasText = _composer.text.trim().isNotEmpty);
                      Navigator.pop(context);
                    },
                    child: Text(emoji, style: const TextStyle(fontSize: 28)),
                  ),
              ],
            ),
          ),
        );
      },
    );
  }

  void _comingSoon(String feature) {
    ScaffoldMessenger.of(context).showSnackBar(
      SnackBar(content: Text('$feature coming soon')),
    );
  }
}

class _MessageHistory extends StatelessWidget {
  const _MessageHistory({
    required this.messages,
    required this.typing,
    required this.onLongPress,
  });

  final List<ChatMessage> messages;
  final bool typing;
  final ValueChanged<ChatMessage> onLongPress;

  @override
  Widget build(BuildContext context) {
    final rows = <Widget>[];
    DateTime? lastDay;
    for (final message in messages) {
      if (lastDay == null || !isSameDay(lastDay, message.sentAt)) {
        rows.add(
          DateSeparatorChip(label: formatDateSeparator(message.sentAt)),
        );
        lastDay = message.sentAt;
      }
      rows.add(
        MessageBubble(
          message: message,
          onLongPress: () => onLongPress(message),
        ),
      );
    }
    if (typing) {
      rows.add(const TypingIndicatorBubble());
    }

    return ListView.builder(
      reverse: true,
      padding: const EdgeInsets.only(top: 8, bottom: 8),
      itemCount: rows.length,
      itemBuilder: (context, index) => rows[rows.length - 1 - index],
    );
  }
}
