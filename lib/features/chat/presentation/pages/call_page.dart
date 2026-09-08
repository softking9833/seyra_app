import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter_webrtc/flutter_webrtc.dart';
import 'package:permission_handler/permission_handler.dart';
import 'package:seyra/core/errors/result.dart';
import 'package:seyra/core/theme/app_colors.dart';
import 'package:seyra/features/chat/domain/entities/social_models.dart';
import 'package:seyra/features/chat/domain/repositories/chat_social_repository.dart';

class CallPage extends StatefulWidget {
  const CallPage({
    super.key,
    required this.social,
    required this.conversationId,
    required this.video,
    required this.outgoing,
    this.incomingPayload,
  });

  final ChatSocialRepository social;
  final String conversationId;
  final bool video;
  final bool outgoing;
  final Map<String, dynamic>? incomingPayload;

  @override
  State<CallPage> createState() => _CallPageState();
}

class _CallPageState extends State<CallPage> {
  final _local = RTCVideoRenderer();
  final _remote = RTCVideoRenderer();
  RTCPeerConnection? _pc;
  MediaStream? _localStream;
  String? _callId;
  var _muted = false;
  var _cameraOff = false;
  var _status = 'Connecting';
  var _disposed = false;
  var _released = false;
  StreamSubscription<Map<String, dynamic>>? _signals;

  @override
  void initState() {
    super.initState();
    unawaited(_start());
  }

  @override
  void dispose() {
    _disposed = true;
    unawaited(_release());
    super.dispose();
  }

  Future<void> _release({bool hangup = true}) async {
    if (_released) {
      return;
    }
    _released = true;
    final callId = _callId;
    _callId = null;
    if (hangup && callId != null) {
      await widget.social.signalCall(callId: callId, action: 'hangup');
    }
    await _signals?.cancel();
    _signals = null;
    final stream = _localStream;
    _localStream = null;
    if (stream != null) {
      for (final track in stream.getTracks()) {
        await track.stop();
      }
      await stream.dispose();
    }
    final pc = _pc;
    _pc = null;
    await pc?.close();
    await pc?.dispose();
    await _local.dispose();
    await _remote.dispose();
  }

  Future<void> _start() async {
    await _local.initialize();
    await _remote.initialize();
    if (_disposed) {
      return;
    }
    final mic = await Permission.microphone.request();
    if (!mic.isGranted) {
      if (mounted) {
        setState(() => _status = 'Microphone permission denied');
      }
      return;
    }
    if (widget.video) {
      final cam = await Permission.camera.request();
      if (!cam.isGranted) {
        if (mounted) {
          setState(() => _status = 'Camera permission denied');
        }
        return;
      }
    }
    if (_disposed) {
      return;
    }
    final ice = await widget.social.iceServers();
    final servers = switch (ice) {
      Success(:final value) => value.servers,
      FailureResult() => [
        {
          'urls': ['stun:stun.l.google.com:19302'],
        },
      ],
    };
    _pc = await createPeerConnection({
      'iceServers': servers,
      'sdpSemantics': 'unified-plan',
    });
    _pc!.onIceCandidate = (candidate) {
      if (_callId == null || candidate.candidate == null) {
        return;
      }
      unawaited(
        widget.social.signalCall(
          callId: _callId!,
          action: 'ice',
          payload: {
            'candidate': candidate.candidate,
            'sdpMid': candidate.sdpMid,
            'sdpMLineIndex': candidate.sdpMLineIndex,
          },
        ),
      );
    };
    _pc!.onTrack = (event) {
      if (event.streams.isNotEmpty) {
        _remote.srcObject = event.streams.first;
        setState(() => _status = 'Connected');
      }
    };
    _pc!.onConnectionState = (state) {
      setState(() => _status = state.toString());
    };
    try {
      _localStream = await navigator.mediaDevices.getUserMedia({
        'audio': true,
        'video': widget.video,
      });
    } catch (_) {
      if (mounted) {
        setState(() => _status = 'Camera or microphone unavailable');
      }
      return;
    }
    if (_disposed) {
      for (final track in _localStream!.getTracks()) {
        await track.stop();
      }
      await _localStream!.dispose();
      _localStream = null;
      return;
    }
    _local.srcObject = _localStream;
    for (final track in _localStream!.getTracks()) {
      await _pc!.addTrack(track, _localStream!);
    }
    _signals = widget.social.watchCallSignals().listen(_onSignal);
    if (widget.outgoing) {
      final offer = await _pc!.createOffer();
      await _pc!.setLocalDescription(offer);
      final started = await widget.social.startCall(
        conversationId: widget.conversationId,
        kind: widget.video ? 'video' : 'voice',
        payload: {'sdp': offer.sdp, 'type': offer.type},
      );
      if (started is Success<CallRecord>) {
        _callId = started.value.id;
        setState(() => _status = 'Ringing');
      } else if (started is FailureResult<CallRecord> && mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(content: Text(started.failure.message)),
        );
      }
    } else {
      final payload = widget.incomingPayload ?? const {};
      _callId = payload['call_id'] as String?;
      final sdp = payload['sdp'] as String?;
      final type = payload['type'] as String? ?? 'offer';
      if (sdp != null) {
        await _pc!.setRemoteDescription(RTCSessionDescription(sdp, type));
        final answer = await _pc!.createAnswer();
        await _pc!.setLocalDescription(answer);
        if (_callId != null) {
          await widget.social.signalCall(
            callId: _callId!,
            action: 'answer',
            payload: {'sdp': answer.sdp, 'type': answer.type},
          );
        }
        setState(() => _status = 'Connected');
      }
    }
    setState(() {});
  }

  Future<void> _onSignal(Map<String, dynamic> payload) async {
    final action = payload['action'] as String?;
    final callId = payload['call_id'] as String?;
    if (_callId != null && callId != null && callId != _callId) {
      return;
    }
    if (action == 'answer') {
      final sdp = payload['sdp'] as String?;
      if (sdp != null) {
        await _pc?.setRemoteDescription(
          RTCSessionDescription(sdp, payload['type'] as String? ?? 'answer'),
        );
        setState(() => _status = 'Connected');
      }
    } else if (action == 'ice') {
      final candidate = payload['candidate'] as String?;
      if (candidate != null) {
        await _pc?.addCandidate(
          RTCIceCandidate(
            candidate,
            payload['sdpMid'] as String?,
            payload['sdpMLineIndex'] as int?,
          ),
        );
      }
    } else if (action == 'reject' || action == 'hangup' || action == 'missed') {
      _callId = null;
      if (mounted) {
        Navigator.pop(context);
      }
    }
  }

  Future<void> _hangup() async {
    await _release(hangup: true);
    if (mounted) {
      Navigator.pop(context);
    }
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      backgroundColor: Colors.black,
      body: SafeArea(
        child: Column(
          children: [
            Expanded(
              child: widget.video
                  ? Stack(
                      children: [
                        RTCVideoView(_remote),
                        Align(
                          alignment: Alignment.topRight,
                          child: SizedBox(
                            width: 120,
                            height: 160,
                            child: RTCVideoView(_local, mirror: true),
                          ),
                        ),
                      ],
                    )
                  : Center(
                      child: Column(
                        mainAxisSize: MainAxisSize.min,
                        children: [
                          const Icon(Icons.call, color: Colors.white, size: 72),
                          const SizedBox(height: 16),
                          Text(
                            _status,
                            style: const TextStyle(color: Colors.white),
                          ),
                        ],
                      ),
                    ),
            ),
            Padding(
              padding: const EdgeInsets.all(24),
              child: Row(
                mainAxisAlignment: MainAxisAlignment.spaceEvenly,
                children: [
                  IconButton.filled(
                    onPressed: () async {
                      _muted = !_muted;
                      _localStream?.getAudioTracks().forEach((track) {
                        track.enabled = !_muted;
                      });
                      setState(() {});
                    },
                    icon: Icon(_muted ? Icons.mic_off : Icons.mic),
                  ),
                  if (widget.video)
                    IconButton.filled(
                      onPressed: () {
                        _cameraOff = !_cameraOff;
                        _localStream?.getVideoTracks().forEach((track) {
                          track.enabled = !_cameraOff;
                        });
                        setState(() {});
                      },
                      icon: Icon(_cameraOff ? Icons.videocam_off : Icons.videocam),
                    ),
                  IconButton.filled(
                    style: IconButton.styleFrom(backgroundColor: Colors.red),
                    onPressed: _hangup,
                    icon: const Icon(Icons.call_end),
                  ),
                ],
              ),
            ),
            Text(_status, style: const TextStyle(color: AppColors.wave)),
            const SizedBox(height: 12),
          ],
        ),
      ),
    );
  }
}
