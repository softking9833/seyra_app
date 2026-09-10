import 'package:flutter/material.dart';
import 'package:seyra/core/theme/app_colors.dart';
import 'package:seyra/features/chat/domain/entities/chat_message.dart';

class MessageComposer extends StatelessWidget {
  const MessageComposer({
    super.key,
    required this.controller,
    required this.hasText,
    required this.replyTo,
    required this.onChanged,
    required this.onSend,
    required this.onAttach,
    required this.onEmoji,
    required this.onMic,
    required this.onCancelReply,
  });

  final TextEditingController controller;
  final bool hasText;
  final ChatMessage? replyTo;
  final ValueChanged<String> onChanged;
  final VoidCallback onSend;
  final VoidCallback onAttach;
  final VoidCallback onEmoji;
  final VoidCallback onMic;
  final VoidCallback onCancelReply;

  @override
  Widget build(BuildContext context) {
    final bottom = MediaQuery.paddingOf(context).bottom;
    return Material(
      color: AppColors.scaffoldOf(context),
      elevation: 8,
      shadowColor: const Color(0x1A000000),
      child: Padding(
        padding: EdgeInsets.fromLTRB(6, 8, 6, 8 + bottom),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            if (replyTo != null)
              Container(
                margin: const EdgeInsets.fromLTRB(8, 0, 8, 8),
                padding: const EdgeInsets.all(10),
                decoration: BoxDecoration(
                  color: AppColors.mutedOf(context),
                  borderRadius: BorderRadius.circular(12),
                  border: Border(
                    left: BorderSide(color: AppColors.accentOf(context), width: 3),
                  ),
                ),
                child: Row(
                  children: [
                    Icon(Icons.reply, size: 18, color: AppColors.accentOf(context)),
                    const SizedBox(width: 8),
                    Expanded(
                      child: Text(
                        replyTo!.body,
                        maxLines: 1,
                        overflow: TextOverflow.ellipsis,
                      ),
                    ),
                    IconButton(
                      key: const Key('composer_cancel_reply'),
                      onPressed: onCancelReply,
                      icon: const Icon(Icons.close, size: 18),
                    ),
                  ],
                ),
              ),
            Row(
              crossAxisAlignment: CrossAxisAlignment.end,
              children: [
                IconButton(
                  key: const Key('composer_attach_button'),
                  tooltip: 'Attach',
                  onPressed: onAttach,
                  icon: const Icon(Icons.add_circle_outline),
                ),
                Expanded(
                  child: Container(
                    padding: const EdgeInsets.symmetric(horizontal: 4),
                    decoration: BoxDecoration(
                      color: AppColors.mutedOf(context),
                      borderRadius: BorderRadius.circular(24),
                      border: Border.all(color: AppColors.borderOf(context)),
                    ),
                    child: Row(
                      children: [
                        IconButton(
                          key: const Key('composer_emoji_button'),
                          tooltip: 'Emoji',
                          onPressed: onEmoji,
                          icon: const Icon(Icons.emoji_emotions_outlined),
                        ),
                        Expanded(
                          child: TextField(
                            key: const Key('composer_text_field'),
                            controller: controller,
                            minLines: 1,
                            maxLines: 5,
                            onChanged: onChanged,
                            textCapitalization: TextCapitalization.sentences,
                            decoration: const InputDecoration(
                              hintText: 'Message',
                              border: InputBorder.none,
                              enabledBorder: InputBorder.none,
                              focusedBorder: InputBorder.none,
                              filled: false,
                              contentPadding: EdgeInsets.symmetric(
                                vertical: 12,
                              ),
                            ),
                          ),
                        ),
                      ],
                    ),
                  ),
                ),
                const SizedBox(width: 4),
                Padding(
                  padding: const EdgeInsets.only(bottom: 2),
                  child: hasText
                      ? IconButton.filled(
                          key: const Key('composer_send_button'),
                          tooltip: 'Send',
                          onPressed: onSend,
                          style: IconButton.styleFrom(
                            backgroundColor: AppColors.accentOf(context),
                            foregroundColor: Colors.white,
                          ),
                          icon: const Icon(Icons.send_rounded),
                        )
                      : IconButton(
                          key: const Key('composer_mic_button'),
                          tooltip: 'Voice message',
                          onPressed: onMic,
                          icon: const Icon(Icons.mic_none_outlined),
                        ),
                ),
              ],
            ),
          ],
        ),
      ),
    );
  }
}
