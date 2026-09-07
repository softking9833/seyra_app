import 'package:flutter/material.dart';
import 'package:seyra/core/theme/app_colors.dart';
import 'package:seyra/features/chat/domain/entities/conversation.dart';
import 'package:seyra/features/chat/domain/usecases/watch_conversations_use_case.dart';
import 'package:seyra/features/chat/presentation/widgets/conversation_tile.dart';

enum ChatFilter { chats, groups, channels }

class ChatsHomeView extends StatefulWidget {
  const ChatsHomeView({
    super.key,
    required this.query,
    required this.filter,
    required this.onFilterChanged,
    required this.watchConversations,
    required this.onOpenConversation,
  });

  final String query;
  final ChatFilter filter;
  final ValueChanged<ChatFilter> onFilterChanged;
  final WatchConversationsUseCase watchConversations;
  final ValueChanged<String> onOpenConversation;

  @override
  State<ChatsHomeView> createState() => _ChatsHomeViewState();
}

class _ChatsHomeViewState extends State<ChatsHomeView> {
  late final Stream<List<Conversation>> _conversations;

  @override
  void initState() {
    super.initState();
    _conversations = widget.watchConversations();
  }

  @override
  Widget build(BuildContext context) {
    return Column(
      children: [
        Padding(
          padding: const EdgeInsets.fromLTRB(8, 8, 8, 0),
          child: Row(
            children: [
              _FilterTab(
                label: 'Chats',
                selected: widget.filter == ChatFilter.chats,
                onTap: () => widget.onFilterChanged(ChatFilter.chats),
              ),
              _FilterTab(
                label: 'Groups',
                selected: widget.filter == ChatFilter.groups,
                onTap: () => widget.onFilterChanged(ChatFilter.groups),
              ),
              _FilterTab(
                label: 'Channels',
                selected: widget.filter == ChatFilter.channels,
                onTap: () => widget.onFilterChanged(ChatFilter.channels),
              ),
            ],
          ),
        ),
        Expanded(
          child: StreamBuilder<List<Conversation>>(
            stream: _conversations,
            builder: (context, snapshot) {
              final items = _visible(
                snapshot.data ?? const [],
                widget.query,
                widget.filter,
              );
              if (items.isEmpty) {
                return const _EmptyConversations();
              }
              return ListView.separated(
                itemCount: items.length,
                separatorBuilder: (_, _) => const Divider(
                  height: 1,
                  indent: 80,
                  color: AppColors.fieldBorder,
                ),
                itemBuilder: (context, index) {
                  final conversation = items[index];
                  return ConversationTile(
                    conversation: conversation,
                    onTap: () => widget.onOpenConversation(conversation.id),
                  );
                },
              );
            },
          ),
        ),
      ],
    );
  }

  List<Conversation> _visible(
    List<Conversation> conversations,
    String query,
    ChatFilter filter,
  ) {
    final normalized = query.trim().toLowerCase();
    return conversations.where((item) {
      final matchesFilter = switch (filter) {
        ChatFilter.chats => true,
        ChatFilter.groups => item.kind == ConversationKind.group,
        ChatFilter.channels => item.kind == ConversationKind.channel,
      };
      final matchesQuery =
          normalized.isEmpty ||
          item.title.toLowerCase().contains(normalized) ||
          item.lastMessagePreview.toLowerCase().contains(normalized);
      return matchesFilter && matchesQuery;
    }).toList();
  }
}

class _FilterTab extends StatelessWidget {
  const _FilterTab({
    required this.label,
    required this.selected,
    required this.onTap,
  });

  final String label;
  final bool selected;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.symmetric(horizontal: 8),
      child: InkWell(
        onTap: onTap,
        child: Padding(
          padding: const EdgeInsets.symmetric(vertical: 10, horizontal: 8),
          child: Column(
            children: [
              Text(
                label,
                style: TextStyle(
                  fontWeight: FontWeight.w600,
                  color: selected ? AppColors.primary : AppColors.textSecondary,
                ),
              ),
              const SizedBox(height: 6),
              Container(
                height: 2,
                width: 28,
                color: selected ? AppColors.primary : Colors.transparent,
              ),
            ],
          ),
        ),
      ),
    );
  }
}

class _EmptyConversations extends StatelessWidget {
  const _EmptyConversations();

  @override
  Widget build(BuildContext context) {
    return Center(
      child: Padding(
        padding: const EdgeInsets.all(32),
        child: Column(
          mainAxisAlignment: MainAxisAlignment.center,
          children: [
            const Icon(
              Icons.chat_bubble_outline,
              size: 48,
              color: AppColors.textSecondary,
            ),
            const SizedBox(height: 12),
            Text(
              'No conversations yet',
              style: Theme.of(context).textTheme.titleMedium,
            ),
            const SizedBox(height: 8),
            Text(
              'Start a private chat when you\'re ready.',
              textAlign: TextAlign.center,
              style: Theme.of(context).textTheme.bodyMedium,
            ),
          ],
        ),
      ),
    );
  }
}
