import 'package:flutter/material.dart';
import 'package:seyra/features/chat/domain/entities/conversation.dart';

class ConversationAvatar extends StatelessWidget {
  const ConversationAvatar({
    super.key,
    required this.conversation,
    this.size = 48,
  });

  final Conversation conversation;
  final double size;

  @override
  Widget build(BuildContext context) {
    final colors = <Color>[
      const Color(0xFF2F6FED),
      const Color(0xFF3B82F6),
      const Color(0xFF1D4ED8),
      const Color(0xFF0F766E),
      const Color(0xFF4F46E5),
    ];
    final color = colors[conversation.title.hashCode.abs() % colors.length];
    final radius = size / 2;

    return SizedBox(
      width: size,
      height: size,
      child: Stack(
        children: [
          CircleAvatar(
            radius: radius,
            backgroundColor: color,
            child: conversation.kind == ConversationKind.direct
                ? Text(
                    conversation.initials,
                    style: TextStyle(
                      color: Colors.white,
                      fontWeight: FontWeight.w700,
                      fontSize: size * 0.32,
                    ),
                  )
                : Icon(
                    conversation.kind == ConversationKind.group
                        ? Icons.group_outlined
                        : Icons.campaign_outlined,
                    color: Colors.white,
                    size: size * 0.46,
                  ),
          ),
          if (conversation.isOnline)
            Positioned(
              right: 0,
              bottom: 0,
              child: Container(
                width: size * 0.24,
                height: size * 0.24,
                decoration: BoxDecoration(
                  color: const Color(0xFF22C55E),
                  shape: BoxShape.circle,
                  border: Border.all(color: Colors.white, width: 2),
                ),
              ),
            ),
        ],
      ),
    );
  }
}
