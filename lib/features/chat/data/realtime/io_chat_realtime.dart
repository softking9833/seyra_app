import 'dart:async';
import 'dart:convert';
import 'dart:io';

import 'package:seyra/features/chat/data/realtime/chat_realtime_port.dart';

/// dart:io WebSocket client. Tokens are sent only as an Authorization header.
/// Reconnects with backoff so a later Redis-backed hub can keep the same events.
final class IoChatRealtime implements ChatRealtimePort {
  WebSocket? _socket;
  StreamController<Map<String, dynamic>>? _events;
  var _closed = false;

  @override
  Stream<Map<String, dynamic>> connect({
    required Uri uri,
    required String accessToken,
  }) {
    final events = StreamController<Map<String, dynamic>>.broadcast();
    _events = events;
    _closed = false;
    unawaited(_listen(uri, accessToken, events));
    return events.stream;
  }

  Future<void> _listen(
    Uri uri,
    String accessToken,
    StreamController<Map<String, dynamic>> events,
  ) async {
    var delay = const Duration(milliseconds: 400);
    while (!events.isClosed && !_closed) {
      try {
        final socket = await WebSocket.connect(
          uri.toString(),
          headers: {'Authorization': 'Bearer $accessToken'},
        );
        _socket = socket;
        delay = const Duration(milliseconds: 400);
        if (!events.isClosed) {
          events.add(const {'type': 'realtime.connected'});
        }
        await for (final raw in socket) {
          if (raw is! String || events.isClosed) {
            continue;
          }
          final decoded = jsonDecode(raw);
          if (decoded is Map<String, dynamic>) {
            events.add(decoded);
          }
        }
      } catch (_) {
        if (!events.isClosed) {
          events.add(const {'type': 'realtime.disconnected'});
        }
      }
      _socket = null;
      if (events.isClosed || _closed) {
        return;
      }
      await Future<void>.delayed(delay);
      final nextMs = delay.inMilliseconds * 2;
      delay = Duration(milliseconds: nextMs > 8000 ? 8000 : nextMs);
    }
  }

  @override
  Future<void> disconnect() async {
    _closed = true;
    await _socket?.close();
    _socket = null;
    await _events?.close();
    _events = null;
  }

  @override
  Future<void> send(Map<String, dynamic> event) async {
    final socket = _socket;
    if (socket == null) {
      return;
    }
    socket.add(jsonEncode(event));
  }
}
