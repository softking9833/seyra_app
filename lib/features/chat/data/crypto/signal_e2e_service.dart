import 'dart:convert';
import 'dart:typed_data';

import 'package:libsignal_protocol_dart/libsignal_protocol_dart.dart';
import 'package:seyra/core/storage/secure_storage.dart';

/// Signal Protocol (X3DH + Double Ratchet) via libsignal_protocol_dart.
/// Private identity material stays in SecureStorage and is never POSTed.
/// Session/ratchet state, one-time prekeys, and signed prekeys are also
/// persisted in SecureStorage so encrypt/decrypt survives process death.
final class SignalE2eService {
  SignalE2eService(this._storage, {this.userId = ''});

  final SecureStorage _storage;
  final String userId;
  InMemorySignalProtocolStore? _store;
  int _registrationId = 0;
  String _deviceId = '1';
  var _identityWasNew = false;

  String get deviceId => _deviceId;

  Future<void> install() async {
    if (_store != null) {
      return;
    }
    var existing = await _storage.read(_identityKey);
    if ((existing == null || existing.isEmpty) && userId.isNotEmpty) {
      final owner = await _storage.read('seyra.e2e.owner');
      if (owner == null || owner == userId) {
        existing = await _storage.read('seyra.e2e.identity');
        if (existing != null && existing.isNotEmpty) {
          await _storage.write(key: _identityKey, value: existing);
          await _storage.write(
            key: _regKey,
            value: await _storage.read('seyra.e2e.registration') ?? '1',
          );
          await _storage.write(
            key: _deviceKey,
            value: await _storage.read('seyra.e2e.device') ?? '1',
          );
          final legacyState = await _storage.read('seyra.e2e.protocol_state');
          if (legacyState != null && legacyState.isNotEmpty) {
            await _storage.write(key: _stateKey, value: legacyState);
          }
          await _storage.write(key: 'seyra.e2e.owner', value: userId);
        }
      }
    }
    final IdentityKeyPair identity;
    if (existing != null && existing.isNotEmpty) {
      identity = IdentityKeyPair.fromSerialized(base64Decode(existing));
      _registrationId = int.parse(await _storage.read(_regKey) ?? '1');
      _deviceId = await _storage.read(_deviceKey) ?? '1';
      _identityWasNew = false;
    } else {
      _identityWasNew = true;
      identity = generateIdentityKeyPair();
      _registrationId = generateRegistrationId(false);
      _deviceId = '1';
      await _storage.write(
        key: _identityKey,
        value: base64Encode(identity.serialize()),
      );
      await _storage.write(key: _regKey, value: '$_registrationId');
      await _storage.write(key: _deviceKey, value: _deviceId);
      if (userId.isNotEmpty) {
        final owner = await _storage.read('seyra.e2e.owner');
        if (owner == null) {
          await _storage.write(key: 'seyra.e2e.owner', value: userId);
        }
      }
    }
    _store = InMemorySignalProtocolStore(identity, _registrationId);
    await _restoreState();
  }

  Future<Map<String, dynamic>> exportPublicBundle() async {
    await install();
    final store = _store!;
    final identity = await store.getIdentityKeyPair();
    late final SignedPreKeyRecord signed;
    if (await store.containsSignedPreKey(1)) {
      signed = await store.loadSignedPreKey(1);
    } else {
      signed = generateSignedPreKey(identity, 1);
      await store.storeSignedPreKey(signed.id, signed);
    }
    if (store.preKeyStore.store.isEmpty && !_identityWasNew) {
      await _restoreState();
    }
    if (store.preKeyStore.store.length < 20) {
      var nextId = 1;
      for (final id in store.preKeyStore.store.keys) {
        if (id >= nextId) {
          nextId = id + 1;
        }
      }
      if (nextId == 1 && !_identityWasNew) {
        nextId = 1000;
      }
      final need = 80 - store.preKeyStore.store.length;
      if (need > 0) {
        final preKeys = generatePreKeys(nextId, need);
        for (final key in preKeys) {
          await store.storePreKey(key.id, key);
        }
      }
    }
    await _persistState();
    final prekeys = <Map<String, dynamic>>[];
    for (final id in store.preKeyStore.store.keys.toList()..sort()) {
      final record = await store.loadPreKey(id);
      prekeys.add({
        'key_id': record.id,
        'public_key': base64Encode(record.getKeyPair().publicKey.serialize()),
      });
    }
    return {
      'device_id': _deviceId,
      'registration_id': _registrationId,
      'identity_public': base64Encode(identity.getPublicKey().serialize()),
      'signed_prekey_id': signed.id,
      'signed_prekey_public':
          base64Encode(signed.getKeyPair().publicKey.serialize()),
      'signed_prekey_sig': base64Encode(signed.signature),
      'prekeys': prekeys,
    };
  }

  Future<bool> hasSession(String peerUserId) async {
    await install();
    return _store!.containsSession(SignalProtocolAddress(peerUserId, 1));
  }

  Future<void> processRemoteBundle({
    required String userId,
    required Map<String, dynamic> bundle,
  }) async {
    await install();
    final address = SignalProtocolAddress(userId, 1);
    final identityPub = IdentityKey.fromBytes(
      base64Decode(bundle['identity_public'] as String),
      0,
    );
    final signedPub = Curve.decodePoint(
      base64Decode(bundle['signed_prekey_public'] as String),
      0,
    );
    ECPublicKey? oneTime;
    final ot = bundle['one_time_prekey_public'] as String?;
    if (ot != null && ot.isNotEmpty) {
      oneTime = Curve.decodePoint(base64Decode(ot), 0);
    }
    final preKeyId = oneTime == null ? null : _jsonInt(bundle['one_time_prekey_id']);
    final preKeyBundle = PreKeyBundle(
      _jsonInt(bundle['registration_id']),
      1,
      preKeyId,
      oneTime,
      _jsonInt(bundle['signed_prekey_id']),
      signedPub,
      base64Decode(bundle['signed_prekey_sig'] as String),
      identityPub,
    );
    final builder = SessionBuilder.fromSignalStore(_store!, address);
    await builder.processPreKeyBundle(preKeyBundle);
    await _persistState();
  }

  Future<String?> encrypt({
    required String peerUserId,
    required String plaintext,
  }) async {
    await install();
    final address = SignalProtocolAddress(peerUserId, 1);
    if (!await _store!.containsSession(address)) {
      return null;
    }
    final cipher = SessionCipher.fromStore(_store!, address);
    final message = await cipher.encrypt(
      Uint8List.fromList(utf8.encode(plaintext)),
    );
    await _persistState();
    return base64Encode(message.serialize());
  }

  Future<String> decrypt({
    required String peerUserId,
    required String ciphertext,
  }) async {
    await install();
    final address = SignalProtocolAddress(peerUserId, 1);
    final cipher = SessionCipher.fromStore(_store!, address);
    final raw = base64Decode(ciphertext);
    final hasSession = await _store!.containsSession(address);
    if (hasSession) {
      try {
        final bytes = await cipher.decryptFromSignal(
          SignalMessage.fromSerialized(raw),
        );
        await _persistState();
        return utf8.decode(bytes);
      } catch (_) {}
    }
    final bytes = await cipher.decrypt(PreKeySignalMessage(raw));
    await _persistState();
    return utf8.decode(bytes);
  }

  static int _jsonInt(Object? value) {
    if (value is int) {
      return value;
    }
    if (value is num) {
      return value.toInt();
    }
    return int.tryParse('$value') ?? 0;
  }

  Future<void> _persistState() async {
    final store = _store;
    if (store == null) {
      return;
    }
    final sessions = <String, String>{};
    store.sessionStore.sessions.forEach((address, bytes) {
      sessions['${address.getName()}:${address.getDeviceId()}'] =
          base64Encode(bytes);
    });
    final prekeys = <String, String>{};
    store.preKeyStore.store.forEach((id, bytes) {
      prekeys['$id'] = base64Encode(bytes);
    });
    final signed = <String, String>{};
    store.signedPreKeyStore.store.forEach((id, bytes) {
      signed['$id'] = base64Encode(bytes);
    });
    final trusted = <String, String>{};
    for (final address in store.sessionStore.sessions.keys) {
      try {
        final identity = await store.getIdentity(address);
        if (identity != null) {
          trusted['${address.getName()}:${address.getDeviceId()}'] =
              base64Encode(identity.serialize());
        }
      } catch (_) {}
    }
    await _storage.write(
      key: _stateKey,
      value: jsonEncode({
        'sessions': sessions,
        'prekeys': prekeys,
        'signed': signed,
        'trusted': trusted,
      }),
    );
  }

  Future<void> _restoreState() async {
    final store = _store!;
    final raw = await _storage.read(_stateKey);
    if (raw == null || raw.isEmpty) {
      return;
    }
    final map = jsonDecode(raw) as Map<String, dynamic>;
    final prekeys = map['prekeys'] as Map<String, dynamic>? ?? {};
    for (final entry in prekeys.entries) {
      final id = int.tryParse(entry.key);
      final value = entry.value as String?;
      if (id == null || value == null) {
        continue;
      }
      store.preKeyStore.store[id] = base64Decode(value);
    }
    final signed = map['signed'] as Map<String, dynamic>? ?? {};
    for (final entry in signed.entries) {
      final id = int.tryParse(entry.key);
      final value = entry.value as String?;
      if (id == null || value == null) {
        continue;
      }
      store.signedPreKeyStore.store[id] = base64Decode(value);
    }
    final trusted = map['trusted'] as Map<String, dynamic>? ?? {};
    for (final entry in trusted.entries) {
      final parts = entry.key.split(':');
      final value = entry.value as String?;
      if (parts.length != 2 || value == null) {
        continue;
      }
      final deviceId = int.tryParse(parts[1]) ?? 1;
      await store.saveIdentity(
        SignalProtocolAddress(parts[0], deviceId),
        IdentityKey.fromBytes(base64Decode(value), 0),
      );
    }
    final sessions = map['sessions'] as Map<String, dynamic>? ?? {};
    for (final entry in sessions.entries) {
      final parts = entry.key.split(':');
      final value = entry.value as String?;
      if (parts.length != 2 || value == null) {
        continue;
      }
      final deviceId = int.tryParse(parts[1]) ?? 1;
      store.sessionStore.sessions[SignalProtocolAddress(parts[0], deviceId)] =
          base64Decode(value);
    }
  }

  String get _identityKey => _scoped('seyra.e2e.identity');
  String get _regKey => _scoped('seyra.e2e.registration');
  String get _deviceKey => _scoped('seyra.e2e.device');
  String get _stateKey => _scoped('seyra.e2e.protocol_state');

  String _scoped(String base) {
    if (userId.isEmpty) {
      return base;
    }
    return '$base.$userId';
  }
}
