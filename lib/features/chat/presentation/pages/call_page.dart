import 'dart:async';
import 'dart:ui';

import 'package:flutter/material.dart';
import 'package:flutter_webrtc/flutter_webrtc.dart';
import 'package:permission_handler/permission_handler.dart';
import 'package:seyra/core/errors/result.dart';
import 'package:seyra/features/chat/domain/entities/social_models.dart';
import 'package:seyra/features/chat/domain/repositories/chat_social_repository.dart';
import 'package:seyra/features/profile/presentation/widgets/user_avatar.dart';

class CallPage extends StatefulWidget {
  const CallPage({
    super.key,
    required this.social,
    required this.conversationId,
    required this.video,
    required this.outgoing,
    this.incomingPayload,
    this.peerTitle = '',
    this.peerId = '',
    this.peerInitials = '',
  });

  final ChatSocialRepository social;
  final String conversationId;
  final bool video;
  final bool outgoing;
  final Map<String, dynamic>? incomingPayload;
  final String peerTitle;
  final String peerId;
  final String peerInitials;

  @override
  State<CallPage> createState() => _CallPageState();
}

class _CallPageState extends State<CallPage>
    with SingleTickerProviderStateMixin {
  final _local = RTCVideoRenderer();
  final _remote = RTCVideoRenderer();
  RTCPeerConnection? _pc;
  MediaStream? _localStream;
  String? _callId;
  var _muted = false;
  var _cameraOff = false;
  var _speaker = false;
  var _more = false;
  var _status = 'Calling';
  var _connected = false;
  var _disposed = false;
  var _released = false;
  DateTime? _connectedAt;
  Timer? _clock;
  late final AnimationController _pulse;
  StreamSubscription<Map<String, dynamic>>? _signals;

  @override
  void initState() {
    super.initState();
    _pulse = AnimationController(
      vsync: this,
      duration: const Duration(milliseconds: 2200),
    )..repeat();
    unawaited(_start());
  }

  @override
  void dispose() {
    _disposed = true;
    _clock?.cancel();
    _pulse.dispose();
    unawaited(_release());
    super.dispose();
  }

  String get _displayName {
    final title = widget.peerTitle.trim();
    if (title.isNotEmpty) {
      return title;
    }
    final fromPayload = widget.incomingPayload?['username'] as String?;
    if (fromPayload != null && fromPayload.trim().isNotEmpty) {
      return fromPayload.trim();
    }
    return 'Seyra';
  }

  String get _handle {
    final title = widget.peerTitle.trim();
    if (title.startsWith('@')) {
      return title;
    }
    if (title.isNotEmpty) {
      return '@$title';
    }
    return '';
  }

  String get _elapsed {
    final start = _connectedAt;
    if (start == null) {
      return '00:00';
    }
    final seconds = DateTime.now().difference(start).inSeconds;
    final mm = (seconds ~/ 60).toString().padLeft(2, '0');
    final ss = (seconds % 60).toString().padLeft(2, '0');
    return '$mm:$ss';
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
        _markConnected();
      }
    };
    _pc!.onConnectionState = (state) {
      if (state == RTCPeerConnectionState.RTCPeerConnectionStateConnected) {
        _markConnected();
      } else if (mounted && !_connected) {
        setState(() => _status = widget.outgoing ? 'Calling' : 'Connecting');
      }
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
        if (mounted) {
          setState(() => _status = 'Calling');
        }
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
        _markConnected();
      }
    }
    if (mounted) {
      setState(() {});
    }
  }

  void _markConnected() {
    if (_disposed || !mounted) {
      return;
    }
    _connectedAt ??= DateTime.now();
    _clock ??= Timer.periodic(const Duration(seconds: 1), (_) {
      if (mounted) {
        setState(() {});
      }
    });
    setState(() {
      _connected = true;
      _status = 'Connected';
    });
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
        _markConnected();
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

  Future<void> _toggleSpeaker() async {
    _speaker = !_speaker;
    try {
      await Helper.setSpeakerphoneOn(_speaker);
    } catch (_) {}
    if (mounted) {
      setState(() {});
    }
  }

  @override
  Widget build(BuildContext context) {
    const navy = Color(0xFF070B14);
    return Scaffold(
      backgroundColor: navy,
      body: GestureDetector(
        onVerticalDragEnd: (details) {
          if ((details.primaryVelocity ?? 0) < -240) {
            setState(() => _more = true);
          } else if ((details.primaryVelocity ?? 0) > 240) {
            setState(() => _more = false);
          }
        },
        child: Stack(
          children: [
            const Positioned(
              top: -80,
              left: -60,
              child: _GlowBlob(color: Color(0xFF3B4FD4), size: 280),
            ),
            const Positioned(
              bottom: 40,
              right: -90,
              child: _GlowBlob(color: Color(0xFF6B3CC9), size: 320),
            ),
            if (widget.video)
              Positioned.fill(
                child: RTCVideoView(_remote, objectFit: RTCVideoViewObjectFit.RTCVideoViewObjectFitCover),
              ),
            if (widget.video)
              Positioned(
                top: 64,
                right: 16,
                child: ClipRRect(
                  borderRadius: BorderRadius.circular(16),
                  child: SizedBox(
                    width: 108,
                    height: 148,
                    child: RTCVideoView(_local, mirror: true),
                  ),
                ),
              ),
            SafeArea(
              child: Column(
                children: [
                  const SizedBox(height: 12),
                  _StatusPill(
                    live: _connected || widget.outgoing,
                    label: _status.contains('denied') ||
                            _status.contains('unavailable')
                        ? _status
                        : _connected
                            ? 'In call'
                            : (widget.outgoing ? 'Calling...' : 'Connecting...'),
                  ),
                  const Spacer(),
                  if (!widget.video) _PulseAvatar(
                    pulse: _pulse,
                    peerId: widget.peerId,
                    initials: widget.peerInitials.isEmpty
                        ? _displayName
                        : widget.peerInitials,
                  ),
                  const SizedBox(height: 28),
                  Text(
                    _displayName,
                    style: const TextStyle(
                      color: Colors.white,
                      fontSize: 34,
                      fontWeight: FontWeight.w700,
                    ),
                  ),
                  if (_handle.isNotEmpty) ...[
                    const SizedBox(height: 6),
                    Text(
                      _handle,
                      style: const TextStyle(
                        color: Color(0xFF9AA3B5),
                        fontSize: 16,
                      ),
                    ),
                  ],
                  const SizedBox(height: 10),
                  Text(
                    _elapsed,
                    style: const TextStyle(
                      color: Colors.white,
                      fontSize: 16,
                      fontWeight: FontWeight.w500,
                    ),
                  ),
                  const SizedBox(height: 18),
                  _ConnectionChip(good: _connected),
                  const Spacer(),
                  if (_more) ...[
                    Row(
                      mainAxisAlignment: MainAxisAlignment.center,
                      children: [
                        if (widget.video)
                          _RoundControl(
                            icon: _cameraOff ? Icons.videocam_off : Icons.videocam,
                            label: 'Camera',
                            onTap: () {
                              _cameraOff = !_cameraOff;
                              _localStream?.getVideoTracks().forEach((track) {
                                track.enabled = !_cameraOff;
                              });
                              setState(() {});
                            },
                          ),
                      ],
                    ),
                    const SizedBox(height: 16),
                  ],
                  Padding(
                    padding: const EdgeInsets.symmetric(horizontal: 36),
                    child: Row(
                      mainAxisAlignment: MainAxisAlignment.spaceBetween,
                      children: [
                        _RoundControl(
                          icon: _muted ? Icons.mic_off : Icons.mic_none,
                          label: 'Mute',
                          onTap: () {
                            _muted = !_muted;
                            _localStream?.getAudioTracks().forEach((track) {
                              track.enabled = !_muted;
                            });
                            setState(() {});
                          },
                        ),
                        _EndCallButton(onTap: _hangup),
                        _RoundControl(
                          icon: _speaker ? Icons.volume_up : Icons.volume_up_outlined,
                          label: 'Speaker',
                          onTap: _toggleSpeaker,
                        ),
                      ],
                    ),
                  ),
                  const SizedBox(height: 28),
                  GestureDetector(
                    onTap: () => setState(() => _more = !_more),
                    child: Column(
                      children: [
                        Icon(
                          _more ? Icons.keyboard_arrow_down : Icons.keyboard_arrow_up,
                          color: const Color(0xFF8B93A7),
                        ),
                        Text(
                          _more ? 'Hide extras' : 'Swipe up for more',
                          style: const TextStyle(
                            color: Color(0xFF8B93A7),
                            fontSize: 13,
                          ),
                        ),
                      ],
                    ),
                  ),
                  const SizedBox(height: 16),
                ],
              ),
            ),
          ],
        ),
      ),
    );
  }
}

class _GlowBlob extends StatelessWidget {
  const _GlowBlob({required this.color, required this.size});

  final Color color;
  final double size;

  @override
  Widget build(BuildContext context) {
    return ImageFiltered(
      imageFilter: ImageFilter.blur(sigmaX: 48, sigmaY: 48),
      child: Container(
        width: size,
        height: size,
        decoration: BoxDecoration(
          shape: BoxShape.circle,
          color: color.withValues(alpha: 0.45),
        ),
      ),
    );
  }
}

class _StatusPill extends StatelessWidget {
  const _StatusPill({required this.live, required this.label});

  final bool live;
  final String label;

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 14, vertical: 8),
      decoration: BoxDecoration(
        color: Colors.white.withValues(alpha: 0.08),
        borderRadius: BorderRadius.circular(24),
      ),
      child: Row(
        mainAxisSize: MainAxisSize.min,
        children: [
          Container(
            width: 8,
            height: 8,
            decoration: BoxDecoration(
              color: live ? const Color(0xFF3DDC84) : const Color(0xFF9AA3B5),
              shape: BoxShape.circle,
            ),
          ),
          const SizedBox(width: 8),
          Text(label, style: const TextStyle(color: Colors.white, fontSize: 13)),
        ],
      ),
    );
  }
}

class _ConnectionChip extends StatelessWidget {
  const _ConnectionChip({required this.good});

  final bool good;

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 14, vertical: 8),
      decoration: BoxDecoration(
        color: Colors.white.withValues(alpha: 0.08),
        borderRadius: BorderRadius.circular(24),
      ),
      child: Row(
        mainAxisSize: MainAxisSize.min,
        children: [
          Icon(
            Icons.signal_cellular_alt,
            size: 16,
            color: good ? const Color(0xFF5B8CFF) : const Color(0xFF9AA3B5),
          ),
          const SizedBox(width: 8),
          Text(
            good ? 'Good connection' : 'Connecting',
            style: const TextStyle(color: Colors.white, fontSize: 13),
          ),
        ],
      ),
    );
  }
}

class _PulseAvatar extends StatelessWidget {
  const _PulseAvatar({
    required this.pulse,
    required this.peerId,
    required this.initials,
  });

  final Animation<double> pulse;
  final String peerId;
  final String initials;

  @override
  Widget build(BuildContext context) {
    return SizedBox(
      width: 220,
      height: 220,
      child: AnimatedBuilder(
        animation: pulse,
        builder: (context, child) {
          return CustomPaint(
            painter: _RingPainter(pulse.value),
            child: Center(child: child),
          );
        },
        child: Container(
          width: 128,
          height: 128,
          decoration: const BoxDecoration(
            shape: BoxShape.circle,
            gradient: LinearGradient(
              begin: Alignment.topLeft,
              end: Alignment.bottomRight,
              colors: [Color(0xFFD7E0FF), Color(0xFF8B86FF)],
            ),
          ),
          child: peerId.isEmpty
              ? const Icon(Icons.person, size: 72, color: Colors.white)
              : Center(
                  child: UserAvatar(
                    userId: peerId,
                    initials: initials,
                    radius: 62,
                    hasAvatar: true,
                  ),
                ),
        ),
      ),
    );
  }
}

class _RingPainter extends CustomPainter {
  _RingPainter(this.t);

  final double t;

  @override
  void paint(Canvas canvas, Size size) {
    final center = Offset(size.width / 2, size.height / 2);
    for (var i = 0; i < 3; i++) {
      final progress = (t + i / 3) % 1.0;
      final radius = 58 + progress * 48;
      final paint = Paint()
        ..style = PaintingStyle.stroke
        ..strokeWidth = 2
        ..color = const Color(0xFF6EA8FF).withValues(alpha: (1 - progress) * 0.45);
      canvas.drawCircle(center, radius, paint);
    }
  }

  @override
  bool shouldRepaint(covariant _RingPainter oldDelegate) => oldDelegate.t != t;
}

class _RoundControl extends StatelessWidget {
  const _RoundControl({
    required this.icon,
    required this.label,
    required this.onTap,
  });

  final IconData icon;
  final String label;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    return Column(
      children: [
        Material(
          color: const Color(0xFF2A3140),
          shape: const CircleBorder(),
          child: InkWell(
            customBorder: const CircleBorder(),
            onTap: onTap,
            child: SizedBox(
              width: 64,
              height: 64,
              child: Icon(icon, color: Colors.white),
            ),
          ),
        ),
        const SizedBox(height: 8),
        Text(label, style: const TextStyle(color: Colors.white, fontSize: 13)),
      ],
    );
  }
}

class _EndCallButton extends StatelessWidget {
  const _EndCallButton({required this.onTap});

  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    return Material(
      color: const Color(0xFFE53935),
      shape: const CircleBorder(),
      child: InkWell(
        customBorder: const CircleBorder(),
        onTap: onTap,
        child: const SizedBox(
          width: 76,
          height: 76,
          child: Icon(Icons.call_end, color: Colors.white, size: 32),
        ),
      ),
    );
  }
}
