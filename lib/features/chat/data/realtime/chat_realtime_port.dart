abstract interface class ChatRealtimePort {
  Stream<Map<String, dynamic>> connect({
    required Uri uri,
    required String accessToken,
  });

  Future<void> disconnect();

  Future<void> send(Map<String, dynamic> event);
}
