String formatConversationTime(DateTime value, {DateTime? now}) {
  final current = now ?? DateTime.now();
  final local = value.toLocal();
  final today = DateTime(current.year, current.month, current.day);
  final date = DateTime(local.year, local.month, local.day);
  final difference = today.difference(date).inDays;

  if (difference == 0) {
    return _two(local.hour, local.minute);
  }
  if (difference == 1) {
    return 'Yesterday';
  }
  if (difference < 7) {
    const days = ['Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat', 'Sun'];
    return days[local.weekday - 1];
  }
  return '${local.day}/${local.month}/${local.year}';
}

String formatMessageTime(DateTime value) {
  final local = value.toLocal();
  return _two(local.hour, local.minute);
}

String formatDateSeparator(DateTime value, {DateTime? now}) {
  final current = now ?? DateTime.now();
  final local = value.toLocal();
  final today = DateTime(current.year, current.month, current.day);
  final date = DateTime(local.year, local.month, local.day);
  final difference = today.difference(date).inDays;
  if (difference == 0) {
    return 'Today';
  }
  if (difference == 1) {
    return 'Yesterday';
  }
  return '${local.day}/${local.month}/${local.year}';
}

bool isSameDay(DateTime a, DateTime b) {
  return a.year == b.year && a.month == b.month && a.day == b.day;
}

String _two(int hour, int minute) {
  final hh = hour.toString().padLeft(2, '0');
  final mm = minute.toString().padLeft(2, '0');
  return '$hh:$mm';
}
