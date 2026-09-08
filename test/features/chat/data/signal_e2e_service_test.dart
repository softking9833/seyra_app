import 'package:flutter_test/flutter_test.dart';
import 'package:seyra/core/storage/memory_secure_storage.dart';
import 'package:seyra/features/chat/data/crypto/signal_e2e_service.dart';

void main() {
  test('Signal prekeys and sessions survive a new service instance', () async {
    final aliceStore = MemorySecureStorage();
    final bobStore = MemorySecureStorage();
    final alice = SignalE2eService(aliceStore);
    final bob = SignalE2eService(bobStore);
    final aliceBundle = await alice.exportPublicBundle();
    final bobBundle = await bob.exportPublicBundle();

    final aliceAgain = SignalE2eService(aliceStore);
    final republished = await aliceAgain.exportPublicBundle();
    expect(
      republished['signed_prekey_public'],
      aliceBundle['signed_prekey_public'],
    );
    expect(
      (republished['prekeys'] as List).length,
      (aliceBundle['prekeys'] as List).length,
    );

    final firstPre = (aliceBundle['prekeys'] as List).first as Map;
    await bob.processRemoteBundle(
      userId: 'alice',
      bundle: {
        ...aliceBundle,
        'one_time_prekey_id': firstPre['key_id'],
        'one_time_prekey_public': firstPre['public_key'],
      },
    );
    final cipher = await bob.encrypt(peerUserId: 'alice', plaintext: 'hello');
    expect(cipher, isNotNull);
    final plain = await alice.decrypt(peerUserId: 'bob', ciphertext: cipher!);
    expect(plain, 'hello');

    final aliceRestored = SignalE2eService(aliceStore);
    final bobRestored = SignalE2eService(bobStore);
    final next = await bobRestored.encrypt(peerUserId: 'alice', plaintext: 'again');
    expect(next, isNotNull);
    final again = await aliceRestored.decrypt(
      peerUserId: 'bob',
      ciphertext: next!,
    );
    expect(again, 'again');
    expect(bobBundle['device_id'], '1');
  });
}
